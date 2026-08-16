//go:build live

package crew

import (
	"context"
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
	t.Logf("%d transactions, hasNextPage=%v", len(page.Transactions), page.PageInfo.HasNextPage)
	for _, tx := range page.Transactions {
		t.Logf("  %s %d cents %s (%s)", tx.ID, tx.AmountCents, tx.Description, tx.Status)
	}
}
