//go:build live

package crew

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

// Live tests hit the real Crew API. Run them with:
//
//	CREW_TOKEN=<bearer token> go test -tags live -run TestLive -v
//
// The token can be obtained by running the interactive login flow (see the
// README quickstart) or captured from the Crew mobile app.
func liveClient(t *testing.T) *Client {
	t.Helper()
	token := os.Getenv("CREW_TOKEN")
	if token == "" {
		t.Skip("CREW_TOKEN not set; skipping live test")
	}
	return NewClient(
		WithToken(token),
		WithTokenCallback(func(tok string) {
			t.Logf("token rotated; update CREW_TOKEN: %s", tok)
		}),
	)
}

func TestLiveCurrentUser(t *testing.T) {
	c := liveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := c.CurrentUser(ctx)
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}
	if user.ID == "" {
		t.Error("user.ID is empty")
	}
	t.Logf("user %s with %d accounts", user.ID, len(user.Accounts))
	for _, a := range user.Accounts {
		t.Logf("  account %s (%s) balance %d cents, %d subaccounts", a.Name, a.Type, a.BalanceCents, len(a.Subaccounts))
	}
}

func TestLiveCashTransactions(t *testing.T) {
	c := liveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	page, err := c.CashTransactions(ctx, CashTransactionsOptions{First: 5})
	if err != nil {
		t.Fatalf("CashTransactions: %v", err)
	}
	t.Logf("%d of %d transactions, hasNextPage=%v", len(page.Transactions), page.Total, page.PageInfo.HasNextPage)
	for _, tx := range page.Transactions {
		t.Logf("  %s %d cents %s (%s)", tx.ID, tx.AmountCents, tx.Payee(), tx.Status)
	}

	// Page forward, then back to the same window, to prove both cursor
	// directions survive the move onto Account.
	next, err := c.CashTransactions(ctx, CashTransactionsOptions{First: 2, After: page.PageInfo.EndCursor})
	if err != nil {
		t.Fatalf("CashTransactions(after): %v", err)
	}
	t.Logf("after %q: %d transactions", page.PageInfo.EndCursor, len(next.Transactions))

	back, err := c.CashTransactions(ctx, CashTransactionsOptions{Last: 2, Before: next.PageInfo.StartCursor})
	if err != nil {
		t.Fatalf("CashTransactions(last+before): %v", err)
	}
	t.Logf("before %q: %d transactions", next.PageInfo.StartCursor, len(back.Transactions))

	// Half a backward page is refused client-side; Crew answers it with a 500.
	if _, err := c.CashTransactions(ctx, CashTransactionsOptions{Last: 2}); !errors.Is(err, ErrBackwardPagination) {
		t.Errorf("Last without Before: err = %v, want ErrBackwardPagination", err)
	}
}

func TestLiveCashTransactionsByAccount(t *testing.T) {
	c := liveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accounts, err := c.Accounts(ctx)
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	for _, a := range accounts {
		page, err := c.CashTransactions(ctx, CashTransactionsOptions{AccountID: a.ID, First: 1})
		if err != nil {
			t.Fatalf("CashTransactions(%s): %v", a.Type, err)
		}
		t.Logf("account %s (%s): total %d", a.Name, a.Type, page.Total)
	}
}
