package crew

import (
	"context"
	"errors"
	"time"
)

// maxWatchBackoff caps the delay between polls after repeated errors.
const maxWatchBackoff = 15 * time.Minute

// txState is what the watcher remembers about a transaction to detect
// updates (e.g. pending → cleared).
type txState struct {
	status      string
	amountCents int64
}

// OnTransaction registers a handler called when the watcher sees a new
// transaction. Handlers run synchronously on the watch goroutine; spawn a
// goroutine for slow work.
func (c *Client) OnTransaction(h TransactionHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.txHandlers = append(c.txHandlers, h)
}

// OnTransactionUpdate registers a handler called when a previously seen
// transaction changes status or amount (e.g. a pending transaction clears).
// Handlers run synchronously on the watch goroutine.
func (c *Client) OnTransactionUpdate(h TransactionHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.txUpdateHandlers = append(c.txUpdateHandlers, h)
}

// OnWatchError registers a handler called when a watcher poll fails.
// Handlers run synchronously on the watch goroutine.
func (c *Client) OnWatchError(h func(error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errHandlers = append(c.errHandlers, h)
}

// StartWatching polls for transactions in a background goroutine, invoking
// OnTransaction and OnTransactionUpdate handlers, until ctx is canceled or
// StopWatching is called. It first runs a synchronous baseline poll (whose
// transactions do not fire handlers); a baseline failure is returned
// directly. The watcher stops on its own if the token expires
// (ErrUnauthorized), since re-login is interactive; wait on WatchDone to
// detect that.
func (c *Client) StartWatching(ctx context.Context) error {
	c.mu.Lock()
	if c.watching {
		c.mu.Unlock()
		return ErrAlreadyWatching
	}
	if c.token == "" {
		c.mu.Unlock()
		return ErrNoToken
	}
	c.watching = true
	c.watchStop = make(chan struct{})
	c.watchDone = make(chan struct{})
	stop, done := c.watchStop, c.watchDone
	c.mu.Unlock()

	seen := make(map[string]txState)
	var order []string

	if err := c.pollTransactions(ctx, seen, &order, false); err != nil {
		c.mu.Lock()
		c.watching = false
		c.mu.Unlock()
		close(done)
		return err
	}

	go c.watchLoop(ctx, stop, done, seen, order)
	return nil
}

// StopWatching stops the watcher. It is idempotent and safe to call even if
// the watcher never started.
func (c *Client) StopWatching() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.watching {
		return
	}
	c.watching = false
	close(c.watchStop)
}

// WatchDone returns a channel closed when the watch loop exits (via
// StopWatching, context cancellation, or an expired token). It returns nil
// if StartWatching has never been called.
func (c *Client) WatchDone() <-chan struct{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.watchDone
}

func (c *Client) watchLoop(ctx context.Context, stop, done chan struct{}, seen map[string]txState, order []string) {
	defer close(done)
	defer func() {
		c.mu.Lock()
		c.watching = false
		c.mu.Unlock()
	}()

	delay := c.watchInterval
	timer := time.NewTimer(delay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-timer.C:
		}

		err := c.pollTransactions(ctx, seen, &order, true)
		switch {
		case err == nil:
			delay = c.watchInterval
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return
		case errors.Is(err, ErrUnauthorized):
			c.logger.Error("crew: watcher stopping: token expired", "error", err)
			c.fireWatchError(err)
			return
		default:
			c.logger.Error("crew: watcher poll failed", "error", err)
			c.fireWatchError(err)
			delay *= 2
			if delay > maxWatchBackoff {
				delay = maxWatchBackoff
			}
		}
		timer.Reset(delay)
	}
}

// pollTransactions fetches the newest page of transactions and updates the
// seen map. When fire is false (the baseline poll), handlers are not
// invoked.
func (c *Client) pollTransactions(ctx context.Context, seen map[string]txState, order *[]string, fire bool) error {
	c.mu.RLock()
	pageSize := c.watchPageSize
	filter := c.watchFilter
	newHandlers := append([]TransactionHandler(nil), c.txHandlers...)
	updateHandlers := append([]TransactionHandler(nil), c.txUpdateHandlers...)
	c.mu.RUnlock()

	page, err := c.CashTransactions(ctx, CashTransactionsOptions{First: pageSize, Filter: filter})
	if err != nil {
		return err
	}

	for _, tx := range page.Transactions {
		state := txState{status: tx.Status, amountCents: tx.AmountCents}
		prev, ok := seen[tx.ID]
		switch {
		case !ok:
			seen[tx.ID] = state
			*order = append(*order, tx.ID)
			if fire {
				c.logger.Debug("crew: new transaction", "id", tx.ID)
				for _, h := range newHandlers {
					h(tx)
				}
			}
		case prev != state:
			seen[tx.ID] = state
			if fire {
				c.logger.Debug("crew: transaction updated", "id", tx.ID)
				for _, h := range updateHandlers {
					h(tx)
				}
			}
		}
	}

	// Bound memory: evict the oldest seen transactions once the map grows
	// well past a page.
	for len(*order) > 10*pageSize {
		delete(seen, (*order)[0])
		*order = (*order)[1:]
	}

	return nil
}

func (c *Client) fireWatchError(err error) {
	c.mu.RLock()
	handlers := append([]func(error){}, c.errHandlers...)
	c.mu.RUnlock()
	for _, h := range handlers {
		h(err)
	}
}
