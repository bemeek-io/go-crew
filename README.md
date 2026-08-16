# go-crew

[![Go Reference](https://pkg.go.dev/badge/github.com/bemeek-io/go-crew.svg)](https://pkg.go.dev/github.com/bemeek-io/go-crew)
[![Go Report Card](https://goreportcard.com/badge/github.com/bemeek-io/go-crew)](https://goreportcard.com/report/github.com/bemeek-io/go-crew)
[![Go](https://github.com/bemeek-io/go-crew/actions/workflows/go.yml/badge.svg)](https://github.com/bemeek-io/go-crew/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/bemeek-io/go-crew/graph/badge.svg)](https://codecov.io/gh/bemeek-io/go-crew)

An unofficial Go client for the [Crew Finance](https://www.trycrew.com) consumer API: accounts, pockets (subaccounts), transactions, transfers, and debit cards, plus a polling watcher that notifies you when transactions occur or clear.

> **Disclaimer:** This project is not affiliated with, endorsed by, or supported by Crew Finance, Inc. It is built against Crew's publicly reachable GraphQL API, which may change without notice. **This SDK can move real money — you are responsible for anything you automate with it.** Use at your own risk.

## Raw API Reference

Crew publishes a GraphQL reference at [docs.trycrew.com](https://docs.trycrew.com/), but its field names don't always match the live schema. The protocol notes in [docs/api.md](docs/api.md) document the auth flow, the GraphQL operations this SDK uses, and which parts have been validated against the live API — useful even if you're not using Go.

## Installation

```bash
go get github.com/bemeek-io/go-crew
```

## Quickstart

```go
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	crew "github.com/bemeek-io/go-crew"
)

func main() {
	ctx := context.Background()
	client := crew.NewClient(
		// Persist the token so future runs can skip the OTP login.
		crew.WithTokenCallback(func(token string) {
			_ = os.WriteFile("crew-token.txt", []byte(token), 0o600)
		}),
	)

	// Resume a saved session, or log in interactively.
	if saved, err := os.ReadFile("crew-token.txt"); err == nil {
		client.SetToken(strings.TrimSpace(string(saved)))
	} else {
		login(ctx, client)
	}

	// List pockets.
	subs, err := client.Subaccounts(ctx)
	if err != nil {
		panic(err)
	}
	for _, s := range subs {
		fmt.Printf("%-20s $%.2f\n", s.Name, float64(s.BalanceCents)/100)
	}

	// Get notified when transactions occur or clear.
	client.OnTransaction(func(tx crew.CashTransaction) {
		fmt.Printf("new: %s $%.2f\n", tx.Payee(), float64(-tx.AmountCents)/100)
	})
	client.OnTransactionUpdate(func(tx crew.CashTransaction) {
		fmt.Printf("update: %s is now %s\n", tx.Payee(), tx.Status)
	})
	if err := client.StartWatching(ctx); err != nil {
		panic(err)
	}
	<-client.WatchDone()
}

func login(ctx context.Context, client *crew.Client) {
	stdin := bufio.NewReader(os.Stdin)
	read := func(prompt string) string {
		fmt.Print(prompt)
		line, _ := stdin.ReadString('\n')
		return strings.TrimSpace(line)
	}

	phoneID, err := client.SendSMSOTP(ctx, read("Phone number: "))
	if err != nil {
		panic(err)
	}
	res, err := client.AuthSMSOTP(ctx, phoneID, read("SMS code: "))
	if err != nil {
		panic(err)
	}
	if !res.SingleFactor {
		emailID, err := client.SendEmailOTP(ctx, res.Email)
		if err != nil {
			panic(err)
		}
		if err := client.AuthEmailOTP(ctx, emailID, read("Email code: ")); err != nil {
			panic(err)
		}
	}
}
```

## Authentication

Crew authenticates individual users with a four-step one-time-passcode flow:

1. `SendSMSOTP(ctx, phone)` — Crew texts a code to the phone number; returns a `phoneID`.
2. `AuthSMSOTP(ctx, phoneID, code)` — verifies the code and stores a bearer token. If the result reports `SingleFactor`, you're done.
3. `SendEmailOTP(ctx, email)` — Crew emails a code; returns an `emailID`.
4. `AuthEmailOTP(ctx, emailID, code)` — verifies the code and stores the final bearer token.

The server may rotate the token on any response; the client captures rotations automatically and reports them through `WithTokenCallback`, so persist the latest token there and resume later with `WithToken` (or `SetToken`). Token lifetime is undocumented — when a request fails with `ErrUnauthorized`, the session has expired and the user must log in again.

## Watching for Transactions

The Crew API has no push channel for individual users, so the watcher polls the newest transactions on an interval (default 60s) and fires handlers on changes:

- `OnTransaction` — a transaction not seen before.
- `OnTransactionUpdate` — a previously seen transaction changed status or amount (e.g. pending → cleared).
- `OnWatchError` — a poll failed. Polling continues with exponential backoff (capped at 15 minutes), **except** on `ErrUnauthorized`, where the watcher stops (re-login is interactive) and `WatchDone()` closes.

Handlers run synchronously on the watch goroutine — spawn your own goroutine for slow work. The first poll seeds the baseline and does not fire handlers.

## API Reference

### Client Lifecycle

| Method | Description |
| --- | --- |
| `NewClient(opts...)` | Create a client (no network IO). |
| `Token()` / `SetToken(tok)` | Read or replace the current bearer token. |
| `Execute(ctx, query, vars, out)` | Run any raw GraphQL query or mutation. |

### Authentication

| Method | Description |
| --- | --- |
| `SendSMSOTP(ctx, phone)` | Step 1: text a login code. |
| `AuthSMSOTP(ctx, phoneID, otp)` | Step 2: verify the SMS code. |
| `SendEmailOTP(ctx, email)` | Step 3: email a login code. |
| `AuthEmailOTP(ctx, emailID, otp)` | Step 4: verify the email code. |

### Accounts & Pockets

| Method | Description |
| --- | --- |
| `CurrentUser(ctx)` | The user with accounts and subaccounts. |
| `Accounts(ctx)` | All accounts. |
| `SpendAccount(ctx)` / `SaveAccount(ctx)` | The primary spending / savings account. |
| `Subaccounts(ctx)` | All pockets across accounts. |
| `CreateSubaccount(ctx, in)` | Create a pocket. |
| `UpdateSubaccount(ctx, in)` | Rename a pocket. |
| `DeleteSubaccount(ctx, id)` | Delete a pocket. |
| `SetTargetBalance(ctx, id, cents)` | Set a pocket's savings goal. |
| `RemoveTargetBalance(ctx, id)` | Clear a pocket's savings goal. |

### Transactions

| Method | Description |
| --- | --- |
| `CashTransactions(ctx, opts)` | One page of transactions (Relay cursors). |
| `AllCashTransactions(ctx, filter)` | Iterator over every transaction. |
| `UpdateCashTransaction(ctx, in)` | Edit a transaction's description. |
| `ReassignCashTransaction(ctx, txID, subID)` | Move a transaction to another pocket. |
| `SplitCashTransaction(ctx, in)` | Split a transaction across pockets. |

Each `CashTransaction` carries rich detail: use **`Payee()`** for the display name (the enriched `Title`, e.g. "Costco" — `MerchantName` and `Description` are usually null on card transactions), plus `MCC`, merchant address fields, `ImageURL` (merchant logo), `Note`/`Memo`, `OccurredAt`/`ClearedAt`, running balance totals, and the associated `Subaccount` and `DebitCard`.

### Transfers

| Method | Description |
| --- | --- |
| `Transfers(ctx, opts)` | One page of transfers. |
| `AllTransfers(ctx, filter)` | Iterator over every transfer. |
| `InitiateTransfer(ctx, in)` | Move money. **Not safely retryable** — see godoc. |
| `CancelTransfer(ctx, id)` | Cancel a pending transfer. |
| `UpdateTransfer(ctx, in)` | Edit a transfer's memo. |

### Debit Cards

| Method | Description |
| --- | --- |
| `DebitCards(ctx)` / `VirtualDebitCards(ctx)` | List physical / virtual cards. |
| `FreezeDebitCard(ctx, id)` / `UnfreezeDebitCard(ctx, id)` | Freeze / unfreeze a card. |
| `CancelDebitCard(ctx, id)` | Permanently cancel a card. |
| `ActivateDebitCards(ctx, in)` | Activate new physical cards. |
| `CreateVirtualDebitCard(ctx, in)` / `UpdateVirtualDebitCard(ctx, in)` | Create / edit a virtual card. |

### Watching

| Method | Description |
| --- | --- |
| `OnTransaction(h)` | Handler for new transactions. |
| `OnTransactionUpdate(h)` | Handler for status/amount changes. |
| `OnWatchError(h)` | Handler for poll failures. |
| `StartWatching(ctx)` | Baseline poll, then poll in the background. |
| `StopWatching()` | Stop polling (idempotent). |
| `WatchDone()` | Channel closed when the watch loop exits. |

### Options

| Option | Default | Description |
| --- | --- | --- |
| `WithHTTPClient(hc)` | 30s timeout | Custom HTTP client. |
| `WithToken(tok)` | — | Resume a saved session. |
| `WithTokenCallback(fn)` | — | Observe token rotations for persistence. |
| `WithLogger(l)` | discard | Structured logger (adapt slog/zap). |
| `WithAPIURL(u)` | Crew production | Override the GraphQL endpoint. |
| `WithAuthURL(u)` | Crew production | Override the auth REST base URL. |
| `WithWatchInterval(d)` | 60s | Watcher poll interval. |
| `WithWatchPageSize(n)` | 50 | Transactions fetched per poll. |
| `WithWatchFilter(f)` | none | Server-side filter for watched transactions. |

## Errors

All errors are prefixed `crew:` and support `errors.Is`/`errors.As`:

- `ErrUnauthorized` — token expired or invalid; log in again.
- `ErrNoToken` — a request was made before logging in.
- `*APIError` — non-2xx HTTP response (`StatusCode`, `Body`).
- `GraphQLErrors` — the API returned GraphQL-level errors.
