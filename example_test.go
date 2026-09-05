package crew_test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	crew "github.com/bemeek-io/go-crew"
)

// Example shows creating a client and resuming a saved session.
func Example() {
	client := crew.NewClient(
		crew.WithToken(os.Getenv("CREW_TOKEN")),
		crew.WithTokenCallback(func(token string) {
			// Persist rotated tokens so the session survives restarts.
			_ = os.WriteFile("crew-token.txt", []byte(token), 0o600)
		}),
	)

	user, err := client.CurrentUser(context.Background())
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(user.FirstName)
}

// Example_login walks the interactive four-step OTP login flow.
func Example_login() {
	client := crew.NewClient()
	ctx := context.Background()
	stdin := bufio.NewReader(os.Stdin)

	phoneID, err := client.SendSMSOTP(ctx, "5555555555")
	if err != nil {
		panic(err)
	}
	fmt.Print("SMS code: ")
	smsCode, _ := stdin.ReadString('\n')
	res, err := client.AuthSMSOTP(ctx, phoneID, strings.TrimSpace(smsCode))
	if err != nil {
		panic(err)
	}

	if !res.SingleFactor {
		emailID, err := client.SendEmailOTP(ctx, res.Email)
		if err != nil {
			panic(err)
		}
		fmt.Print("Email code: ")
		emailCode, _ := stdin.ReadString('\n')
		if err := client.AuthEmailOTP(ctx, emailID, strings.TrimSpace(emailCode)); err != nil {
			panic(err)
		}
	}

	fmt.Println("logged in; token:", client.Token())
}

// Example_transactions lists recent transactions and iterates all of them.
func Example_transactions() {
	client := crew.NewClient(crew.WithToken(os.Getenv("CREW_TOKEN")))
	ctx := context.Background()

	page, err := client.CashTransactions(ctx, crew.CashTransactionsOptions{First: 10})
	if err != nil {
		panic(err)
	}
	for _, tx := range page.Transactions {
		fmt.Printf("%s %d %s\n", tx.ID, tx.AmountCents, tx.Payee())
	}

	for tx, err := range client.AllCashTransactions(ctx, &crew.CashTransactionFilter{TransferSide: crew.TransferSideDebit}) {
		if err != nil {
			panic(err)
		}
		fmt.Println(tx.Payee())
	}
}

// Example_cardPockets prints the pocket each card spends from. Virtual
// cards are pinned individually; physical ones follow the member's own
// choice, falling back to the account default — so both the user and the
// owning account have to be on hand.
func Example_cardPockets() {
	client := crew.NewClient(crew.WithToken(os.Getenv("CREW_TOKEN")))
	ctx := context.Background()

	cards, err := client.DebitCards(ctx)
	if err != nil {
		panic(err)
	}
	user, err := client.CurrentUser(ctx)
	if err != nil {
		panic(err)
	}

	byID := make(map[string]*crew.Account, len(user.Accounts))
	for i := range user.Accounts {
		byID[user.Accounts[i].ID] = &user.Accounts[i]
	}
	for _, card := range cards {
		var owner *crew.Account
		if card.Account != nil {
			owner = byID[card.Account.ID]
		}
		if pocket := card.SpendSubaccount(user, owner); pocket != nil {
			fmt.Printf("%s spends from %s\n", card.Name, pocket.Name)
		}
	}
}

// Example_setSpendSubaccount repoints physical card swipes at another
// pocket. UpdateVirtualDebitCard cannot do this — it only affects
// per-merchant virtual cards.
func Example_setSpendSubaccount() {
	client := crew.NewClient(crew.WithToken(os.Getenv("CREW_TOKEN")))
	ctx := context.Background()

	user, err := client.CurrentUser(ctx)
	if err != nil {
		panic(err)
	}
	updated, err := client.SetSpendSubaccount(ctx, user.ID, "sub-groceries")
	if err != nil {
		panic(err)
	}
	// The choice lands in the spend config, not on the account default.
	fmt.Println(updated.SelectedSpendSubaccount().Name)

	// Passing an empty ID falls back to the account's default pocket.
	if _, err := client.SetSpendSubaccount(ctx, user.ID, ""); err != nil {
		panic(err)
	}
}

// Example_familyLink links household members during login. Every account
// in a Crew household shares a family ID, so two members who both sign in
// land on the same value without exchanging anything. The lookup is
// best-effort: any error just means "no link".
func Example_familyLink() {
	client := crew.NewClient(crew.WithToken(os.Getenv("CREW_TOKEN")))

	familyID, err := client.CurrentUserFamilyID(context.Background())
	if err != nil || familyID == "" {
		// Non-fatal: fall back to whatever pairing path exists.
		fmt.Println("no household link; using an invite code")
		return
	}
	fmt.Println("household:", familyID)
}

// Example_watcher polls for transactions and reacts as they appear or clear.
func Example_watcher() {
	client := crew.NewClient(
		crew.WithToken(os.Getenv("CREW_TOKEN")),
		crew.WithWatchInterval(30*time.Second),
	)

	client.OnTransaction(func(tx crew.CashTransaction) {
		fmt.Printf("new transaction: %s (%d cents)\n", tx.Payee(), tx.AmountCents)
	})
	client.OnTransactionUpdate(func(tx crew.CashTransaction) {
		fmt.Printf("transaction %s is now %s\n", tx.ID, tx.Status)
	})
	client.OnWatchError(func(err error) {
		fmt.Println("watch error:", err)
	})

	if err := client.StartWatching(context.Background()); err != nil {
		panic(err)
	}
	<-client.WatchDone()
}

// ExampleClient_Execute runs a raw GraphQL query for fields the typed
// methods do not cover.
func ExampleClient_Execute() {
	client := crew.NewClient(crew.WithToken(os.Getenv("CREW_TOKEN")))

	var out struct {
		CurrentUser struct {
			Referrals struct {
				Edges []struct {
					Node struct {
						ID string `json:"id"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"referrals"`
		} `json:"currentUser"`
	}
	query := `query { currentUser { referrals(first: 10) { edges { node { id } } } } }`
	if err := client.Execute(context.Background(), query, nil, &out); err != nil {
		panic(err)
	}
	fmt.Println(len(out.CurrentUser.Referrals.Edges))
}

// Example_freezeCard reads a card's freeze state and toggles it. Freezing
// is a state machine, not a boolean: a card reports FREEZING or UNFREEZING
// while the network catches up, so IsFrozen only reports the settled state.
func Example_freezeCard() {
	client := crew.NewClient(crew.WithToken(os.Getenv("CREW_TOKEN")))
	ctx := context.Background()

	// Card IDs come from the card listings — there is no lookup by number.
	cards, err := client.DebitCards(ctx)
	if err != nil {
		panic(err)
	}
	if len(cards) == 0 {
		return
	}
	card := cards[0]
	fmt.Printf("%s ...%s: %s (%s)\n", card.Name, card.LastFour, card.FrozenStatus, card.FrozenReason)

	if card.IsFrozen() {
		unfrozen, err := client.UnfreezeDebitCard(ctx, card.ID)
		if err != nil {
			panic(err)
		}
		fmt.Println("now", unfrozen.FrozenStatus)
		return
	}

	// A reason is required; USER_REQUESTED is the ordinary "I lost sight of
	// my card" freeze. Report it lost or stolen only if it really is — Crew
	// treats those reasons differently.
	frozen, err := client.FreezeDebitCard(ctx, card.ID, crew.CardFrozenReasonUserRequested)
	if err != nil {
		panic(err)
	}
	fmt.Println("now", frozen.FrozenStatus)
}
