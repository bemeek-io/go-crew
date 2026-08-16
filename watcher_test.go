package crew

import (
	"context"
	"errors"
	"testing"
	"time"
)

func watchTestServer(t *testing.T, opts ...Option) (*Client, *fakeServer) {
	t.Helper()
	base := []Option{WithWatchInterval(5 * time.Millisecond)}
	return newTestServer(t, append(base, opts...)...)
}

func waitFor[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
		panic("unreachable")
	}
}

func TestStartWatchingRequiresToken(t *testing.T) {
	c := NewClient()
	if err := c.StartWatching(context.Background()); !errors.Is(err, ErrNoToken) {
		t.Fatalf("err = %v, want ErrNoToken", err)
	}
}

func TestStartWatchingBaselineFailureReturned(t *testing.T) {
	c, f := watchTestServer(t)
	f.setGQLStatus(500)

	err := c.StartWatching(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError from baseline poll", err)
	}
	// A failed start must leave the watcher restartable.
	f.setGQLStatus(0)
	f.setGQL("cashTransactions", txPageJSON(false, "c1"))
	if err := c.StartWatching(context.Background()); err != nil {
		t.Fatalf("restart after failed baseline: %v", err)
	}
	c.StopWatching()
}

func TestStartWatchingTwiceReturnsErrAlreadyWatching(t *testing.T) {
	c, f := watchTestServer(t)
	f.setGQL("cashTransactions", txPageJSON(false, "c1"))

	if err := c.StartWatching(context.Background()); err != nil {
		t.Fatalf("StartWatching: %v", err)
	}
	defer c.StopWatching()
	if err := c.StartWatching(context.Background()); !errors.Is(err, ErrAlreadyWatching) {
		t.Fatalf("second start err = %v, want ErrAlreadyWatching", err)
	}
}

func TestWatcherBaselineDoesNotFire(t *testing.T) {
	c, f := watchTestServer(t)
	f.setGQL("cashTransactions", txPageJSON(false, "c1",
		`{"id":"tx-1","amount":-500,"status":"CLEARED"}`))

	fired := make(chan CashTransaction, 10)
	c.OnTransaction(func(tx CashTransaction) { fired <- tx })

	if err := c.StartWatching(context.Background()); err != nil {
		t.Fatalf("StartWatching: %v", err)
	}
	defer c.StopWatching()

	// Let several polls elapse; the baseline transaction must never fire.
	time.Sleep(50 * time.Millisecond)
	select {
	case tx := <-fired:
		t.Fatalf("handler fired for baseline transaction %s", tx.ID)
	default:
	}
}

func TestWatcherFiresOnNewTransaction(t *testing.T) {
	c, f := watchTestServer(t)
	f.setGQL("cashTransactions", txPageJSON(false, "c1",
		`{"id":"tx-1","amount":-500,"status":"CLEARED"}`))

	fired := make(chan CashTransaction, 10)
	c.OnTransaction(func(tx CashTransaction) { fired <- tx })

	if err := c.StartWatching(context.Background()); err != nil {
		t.Fatalf("StartWatching: %v", err)
	}
	defer c.StopWatching()

	f.setGQL("cashTransactions", txPageJSON(false, "c2",
		`{"id":"tx-2","amount":-750,"status":"PENDING","description":"Lunch"}`,
		`{"id":"tx-1","amount":-500,"status":"CLEARED"}`))

	tx := waitFor(t, fired, "new transaction")
	if tx.ID != "tx-2" || tx.AmountCents != -750 {
		t.Errorf("tx = %+v, want tx-2", tx)
	}
}

func TestWatcherFiresUpdateOnStatusChange(t *testing.T) {
	c, f := watchTestServer(t)
	f.setGQL("cashTransactions", txPageJSON(false, "c1",
		`{"id":"tx-1","amount":-500,"status":"PENDING"}`))

	newTx := make(chan CashTransaction, 10)
	updated := make(chan CashTransaction, 10)
	c.OnTransaction(func(tx CashTransaction) { newTx <- tx })
	c.OnTransactionUpdate(func(tx CashTransaction) { updated <- tx })

	if err := c.StartWatching(context.Background()); err != nil {
		t.Fatalf("StartWatching: %v", err)
	}
	defer c.StopWatching()

	f.setGQL("cashTransactions", txPageJSON(false, "c1",
		`{"id":"tx-1","amount":-500,"status":"CLEARED"}`))

	tx := waitFor(t, updated, "transaction update")
	if tx.ID != "tx-1" || tx.Status != "CLEARED" {
		t.Errorf("tx = %+v, want tx-1 CLEARED", tx)
	}
	select {
	case tx := <-newTx:
		t.Errorf("OnTransaction fired for updated transaction %s", tx.ID)
	default:
	}
}

func TestWatcherRecoversAfterError(t *testing.T) {
	c, f := watchTestServer(t)
	f.setGQL("cashTransactions", txPageJSON(false, "c1"))

	errs := make(chan error, 10)
	fired := make(chan CashTransaction, 10)
	c.OnWatchError(func(err error) { errs <- err })
	c.OnTransaction(func(tx CashTransaction) { fired <- tx })

	if err := c.StartWatching(context.Background()); err != nil {
		t.Fatalf("StartWatching: %v", err)
	}
	defer c.StopWatching()

	f.setGQLStatus(500)
	err := waitFor(t, errs, "watch error")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("watch error = %v, want *APIError", err)
	}

	f.setGQLStatus(0)
	f.setGQL("cashTransactions", txPageJSON(false, "c2",
		`{"id":"tx-2","amount":-100,"status":"PENDING"}`))
	tx := waitFor(t, fired, "recovery transaction")
	if tx.ID != "tx-2" {
		t.Errorf("tx = %+v, want tx-2 after recovery", tx)
	}
}

func TestWatcherStopsOnUnauthorized(t *testing.T) {
	c, f := watchTestServer(t)
	f.setGQL("cashTransactions", txPageJSON(false, "c1"))

	errs := make(chan error, 10)
	c.OnWatchError(func(err error) { errs <- err })

	if err := c.StartWatching(context.Background()); err != nil {
		t.Fatalf("StartWatching: %v", err)
	}

	f.setGQLStatus(401)
	err := waitFor(t, errs, "unauthorized error")
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("watch error = %v, want ErrUnauthorized", err)
	}
	waitFor(t, c.WatchDone(), "WatchDone close")

	// The watcher must be restartable after an unauthorized exit.
	f.setGQLStatus(0)
	if err := c.StartWatching(context.Background()); err != nil {
		t.Fatalf("restart after unauthorized: %v", err)
	}
	c.StopWatching()
}

func TestStopWatchingIsIdempotent(t *testing.T) {
	c, f := watchTestServer(t)
	f.setGQL("cashTransactions", txPageJSON(false, "c1"))

	c.StopWatching() // never started: no panic

	if err := c.StartWatching(context.Background()); err != nil {
		t.Fatalf("StartWatching: %v", err)
	}
	c.StopWatching()
	c.StopWatching()
	waitFor(t, c.WatchDone(), "WatchDone close")
}

func TestWatcherRespectsContextCancel(t *testing.T) {
	c, f := watchTestServer(t)
	f.setGQL("cashTransactions", txPageJSON(false, "c1"))

	ctx, cancel := context.WithCancel(context.Background())
	if err := c.StartWatching(ctx); err != nil {
		t.Fatalf("StartWatching: %v", err)
	}
	cancel()
	waitFor(t, c.WatchDone(), "WatchDone close after cancel")
}

func TestWatchDoneNilBeforeStart(t *testing.T) {
	c := NewClient()
	if c.WatchDone() != nil {
		t.Error("WatchDone() before start should be nil")
	}
}

func TestWatcherSendsFilterAndPageSize(t *testing.T) {
	c, f := watchTestServer(t,
		WithWatchPageSize(7),
		WithWatchFilter(CashTransactionFilter{SubaccountID: "sub-1"}),
	)
	f.setGQL("cashTransactions", txPageJSON(false, "c1"))

	if err := c.StartWatching(context.Background()); err != nil {
		t.Fatalf("StartWatching: %v", err)
	}
	c.StopWatching()

	vars := f.lastRequest().Variables
	if vars["first"] != float64(7) {
		t.Errorf("first = %v, want 7", vars["first"])
	}
	filter, _ := vars["filter"].(map[string]any)
	if filter["subaccountId"] != "sub-1" {
		t.Errorf("filter = %v", filter)
	}
}
