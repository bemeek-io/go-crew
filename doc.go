// Package crew is an unofficial Go client for the Crew Finance
// (https://www.trycrew.com) consumer API.
//
// The Crew API is GraphQL. This package exposes typed methods for the core
// resources — the current user, accounts and subaccounts ("pockets"), cash
// transactions, transfers, and debit cards — plus Execute, a raw GraphQL
// escape hatch for everything else. It also provides a polling watcher that
// invokes handlers when new transactions appear or existing ones change
// (for example, when a pending transaction clears).
//
// # Authentication
//
// Crew authenticates individual users with a four-step one-time-passcode
// flow: SendSMSOTP, AuthSMSOTP, SendEmailOTP, AuthEmailOTP. Each step
// returns a rotated bearer token which the client captures automatically.
// Tokens may also be rotated by the server on any later response; register
// WithTokenCallback to persist the current token and resume a session later
// with WithToken. When a token expires, requests fail with ErrUnauthorized
// and the user must log in again.
package crew
