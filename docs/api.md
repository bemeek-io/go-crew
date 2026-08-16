# Crew Finance API — Raw Protocol Notes

Reverse-engineered notes on the Crew Finance consumer API, for anyone building against it without this SDK. Sources: Crew's GraphQL reference at `https://docs.trycrew.com/` (SpectaQL, v1.0.0) and a working third-party integration. Crew issues no public schema file; treat everything here as subject to change.

**Validation status:** the GraphQL endpoint, bearer auth, response envelope, and the subaccount fields `id`, `name`, `overallBalance`, `goal` are confirmed against a working integration. The OTP auth flow, mutation shapes, and remaining field names come from Crew's docs and have not been exercised live. The docs are internally inconsistent about the auth base path (`/willow/auth/...` in curl examples vs `/willow/graphql/auth/...` in prose); this SDK defaults to `/willow/auth`.

## Base URLs

| Purpose | URL |
| --- | --- |
| GraphQL endpoint | `https://api.trycrew.com/willow/graphql` |
| Auth REST base | `https://api.trycrew.com/willow/auth` |

Recommended headers on every request:

```
Content-Type: application/json
Accept: */*
User-Agent: Crew/1 CFNetwork/3860.300.31 Darwin/25.2.0
Authorization: Bearer <token>        (once authenticated)
```

## Authentication (individual user)

Four steps, each a `POST` of JSON. **Each step returns a rotated bearer token in the `authorization` response header** — capture it after every request, including ordinary GraphQL calls. Token lifetime is undocumented; a 401 means the session is dead and the flow must be rerun.

1. `POST /send_sms_otp` — body `{"phone":"5555555555"}` → `{"phone_id":"phone-number-live-<uuid>"}`
2. `POST /auth_sms_otp` — body `{"otp":"123456","phone_id":"..."}` → header `authorization: <token>`, body `{"email":"b***@x.com","isSingleFactor":false}`
3. `POST /send_email_otp` — bearer token from step 2; body `{"email":"..."}` → header `authorization: <token>`, body `{"email_id":"email-live-<uuid>"}`
4. `POST /auth_email_otp` — body `{"otp":"654321","email_id":"..."}` → header `authorization: <token>` (the session token)

If step 2 reports `"isSingleFactor": true`, steps 3–4 are unnecessary.

## GraphQL conventions

- Single endpoint; request envelope `{"query": "...", "variables": {...}}`.
- Responses are HTTP 200 with `{"data": ..., "errors": [{"message", "path"}]}` — check `errors` even on 200.
- Root query field: `currentUser`. There is no user-level subscription or webhook channel (webhooks are integrator-only).
- Mutations take a single `input` object (`<Name>Input`) and return a payload wrapping `result` (e.g. `{"initiateTransfer": {"result": {...}}}`).
- IDs are opaque Relay global IDs (base64 of `Type:uuid`). Treat as strings.
- Money is integer **cents**. Balance fields observed live: `overallBalance` (subaccounts and accounts), `goal` (subaccount savings target, nullable).
- Scalars: `Date` = `"2007-12-03"`, `DateTime` = `"2007-12-03T10:15:30Z"`.

## Pagination

Relay cursor connections: arguments `first`, `last`, `after`, `before`, plus an optional `filter` input; responses carry `edges { node }` and `pageInfo { startCursor endCursor hasNextPage hasPreviousPage }`.

Known filters:

- `CashTransactionFilter`: `amount` (range), `debitCardId`, `fuzzySearch`, `subaccountId`, `subaccountIds`, `transferSide`, `type`
- `TransferFilter`: `status`

## Operations used by this SDK

Queries (all rooted at `currentUser`):

```graphql
query CurrentUser { currentUser { id firstName lastName email phone accounts { ...account } } }
query Accounts { currentUser { accounts { ...account } } }
query SpendAccount { currentUser { spendAccount { ...account } } }
query SaveAccount { currentUser { saveAccount { ...account } } }
query DebitCards { currentUser { debitCards { id last4 status frozen nickname } } }
query VirtualDebitCards { currentUser { virtualDebitCards { id last4 status frozen nickname } } }

query CashTransactions($first: Int, $last: Int, $after: String, $before: String, $filter: CashTransactionFilter) {
  currentUser {
    cashTransactions(first: $first, last: $last, after: $after, before: $before, filter: $filter) {
      edges { node { id amount description status type merchantName subaccountId debitCardId date createdAt } }
      pageInfo { startCursor endCursor hasNextPage hasPreviousPage }
    }
  }
}

query Transfers($first: Int, $last: Int, $after: String, $before: String, $filter: TransferFilter) {
  currentUser {
    transfers(first: $first, last: $last, after: $after, before: $before, filter: $filter) {
      edges { node { id amount status type memo errorCode isCancellable occurredAt } }
      pageInfo { startCursor endCursor hasNextPage hasPreviousPage }
    }
  }
}
```

Account/subaccount fields used: `id type name overallBalance subaccounts { id name type overallBalance goal }`.

Mutations (each `(input: $input)`, returning `{ result { ...fields } }`):

- Subaccounts: `createSubaccount`, `updateSubaccount`, `deleteSubaccount`, `setTargetBalance`, `removeTargetBalance`
- Transactions: `updateCashTransaction`, `reassignCashTransaction`, `splitCashTransaction`
- Transfers: `initiateTransfer` (`accountFromId`, `accountToId`, `amount` in cents, `memo`, `note`), `cancelTransfer`, `updateTransfer`
- Cards: `freezeDebitCard`, `unfreezeDebitCard`, `cancelDebitCard`, `activateDebitCards`, `createVirtualDebitCard`, `updateVirtualDebitCard`

Enums (observed/documented values): `SubaccountType` = BILL, BILL_RESERVE, CREDIT, CREDIT_RESERVE, SAVINGS, SPENDING; `TransferStatus` = CANCELED, CANCELING, COMPLETED, DECISION_ACCEPTED, DECISION_MANUAL_REVIEW, DECISION_PENDING, DECISION_REJECTED, DECISION_RETRYING; `TransferType` = ACH, ADJUSTMENT, ALLOWANCE, BILL_SUBACCOUNT, BONUS, BOOK, CASH_DEPOSIT, CHECK, …

## Known unknowns

- **Rate limits:** undocumented. This SDK polls conservatively (60s default).
- **Idempotency:** undocumented; `initiateTransfer` has no idempotency key. A timed-out transfer may still have executed — never blind-retry money movement.
- **Token TTL:** undocumented. Assume it can die at any time and handle 401 by re-running the login flow.
- **Auth path discrepancy:** `/willow/auth` vs `/willow/graphql/auth` (see header note). Both bases are configurable in this SDK.
