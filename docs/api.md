# Crew Finance API — Raw Protocol Notes

Reverse-engineered notes on the Crew Finance consumer API, for anyone building against it without this SDK. Sources: Crew's GraphQL reference at `https://docs.trycrew.com/` (SpectaQL, v1.0.0) and a working third-party integration. Crew issues no public schema file; treat everything here as subject to change.

**Validation status:** confirmed against the live API (2026-08-16): the GraphQL endpoint, the full 4-step OTP auth flow at the `/willow/auth` base (settling the docs' `/willow/auth` vs `/willow/graphql/auth` inconsistency), bearer auth, the response envelope, the `currentUser` accounts/subaccounts query (`overallBalance`, `goal`), and the `cashTransactions` field names below. `CashTransaction` notably has **no** `createdAt` or `date` field — timestamps are `occurredAt` and nullable `clearedAt`; `subaccount` and `debitCard` are nested objects, not ID scalars; and the connection's filter argument is `searchFilters`, not `filter`. Mutation input shapes and transfer fields still come from Crew's docs and have not been exercised live — but they were reconciled field-by-field against that reference on 2026-08-17, which corrected a number of names this SDK had guessed wrong (see "Input shapes" below).

**Schema drift, 2026-09-05:** Crew moved `cashTransactions` off `User` (see "Type-name traps"), dropped `transferSide` from `CashTransactionFilter`, and `IntegerRange` now takes `gt`/`gte`/`lt`/`lte` instead of `min`/`max`. The published docs at `docs.trycrew.com` still describe the old shape, so they lag the live schema; where the two disagree, the live endpoint wins. Introspection stays disabled with a valid token (`{"errors":[{"message":"forbidden"}]}`), but the server's validation errors carry `Did you mean …` suggestions, which is enough to enumerate a type's fields by probing bogus ones.

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

Relay cursor connections: arguments `first`, `last`, `after`, `before`, plus an optional `searchFilters` input (validated on `cashTransactions`; assumed on `transfers`); responses carry `edges { cursor node }`, `pageInfo { startCursor endCursor hasNextPage hasPreviousPage }`, and `total` (the count matching the filter, not the page — note it is `total`, not the Relay-conventional `totalCount`).

**Backward pagination needs both `last` and `before`.** Supplying either alone returns HTTP 500 with the non-GraphQL body `{"errors":{"detail":"Internal Server Error"}}` — the server does not answer with a GraphQL error, so there is nothing useful to parse. Confirmed 2026-09-05.

Known filters:

- `CashTransactionFilter`: `amount` (`IntegerRange`), `amount_v2` (`[IntegerRange]`), `debitCardId` (`[ID!]`), `fuzzySearch`, `matching_name` (`StringFilter`), `occurred_at` (`DateRange`), `subaccountId`, `subaccountIds`, `type` (`CashTransactionType`), `user_id` (`[ID!]`)
- `IntegerRange`: `gt`, `gte`, `lt`, `lte` — **not** `min`/`max`
- `TransferFilter`: `status`

`CashTransactionFilter` no longer has a `transferSide` member, and no replacement direction filter exists — filter debits client-side on the sign of `amount`.

## Operations used by this SDK

Queries (all rooted at `currentUser`):

```graphql
query CurrentUser { currentUser { id firstName lastName email phone accounts { ...account } userSpendConfig { id selectedSpendSubaccount { id name } } } }
query Accounts { currentUser { accounts { ...account } } }
query SpendAccount { currentUser { spendAccount { ...account } } }
query SaveAccount { currentUser { saveAccount { ...account } } }
query DebitCards { currentUser { debitCards { ...card } } }
query VirtualDebitCards { currentUser { virtualDebitCards { ...card } } }
query CurrentUserFamilyID { currentUser { accounts { family { id } } } }

query CashTransactions($first: Int, $last: Int, $after: String, $before: String, $filter: CashTransactionFilter) {
  currentUser {
    spendAccount {
      cashTransactions(first: $first, last: $last, after: $after, before: $before, searchFilters: $filter) {
        total
        edges { node { id amount title description status type mcc merchantName merchantAddress1 merchantCity merchantState merchantZip merchantCountry imageUrl note memo externalId occurredAt clearedAt subaccountRunningTotal accountRunningTotal subaccount { id name } debitCard { id } } }
        pageInfo { startCursor endCursor hasNextPage hasPreviousPage }
      }
    }
  }
}

query AccountCashTransactions($accountId: ID!, $first: Int, $last: Int, $after: String, $before: String, $filter: CashTransactionFilter) {
  node(id: $accountId) {
    ... on Account {
      cashTransactions(first: $first, last: $last, after: $after, before: $before, searchFilters: $filter) { ...same as above }
    }
  }
}

query Transfers($first: Int, $last: Int, $after: String, $before: String) {
  currentUser {
    transfers(first: $first, last: $last, after: $after, before: $before) {
      edges { node { id amount status type memo errorCode isCancellable occurredAt } }
      pageInfo { startCursor endCursor hasNextPage hasPreviousPage }
    }
  }
}
```

Account/subaccount fields used: `id type name overallBalance subaccounts { id name subaccountType overallBalance goal } primarySubaccount { id name } family { id }`.

Debit card fields used (`...card` above): `id name lastFour status formFactor frozenStatus frozenReason color monthlyLimit monthlySpendToDate subaccount { id name } account { id }`.

Mutations (each `(input: $input)`, returning `{ result { ...fields } }`):

- Subaccounts: `createSubaccount`, `updateSubaccount`, `deleteSubaccount`, `setTargetBalance`, `removeTargetBalance`, `setSpendSubaccount`
- Transactions: `updateCashTransaction` (`cashTransactionId`, `note` — those are the **only** two input fields; merchant details, amount, and status are not editable), `reassignCashTransaction` (`cashTransactionId`, `subaccountId`), `splitCashTransaction` (`cashTransactionId`, `splits`)
- Transfers: `initiateTransfer` (`accountFromId`, `accountToId`, `amount` in cents, `memo`, `note`), `cancelTransfer`, `updateTransfer`
- Cards: `freezeDebitCard`, `unfreezeDebitCard`, `cancelDebitCard`, `activateDebitCards`, `createVirtualDebitCard`, `updateVirtualDebitCard`

## Input shapes

GraphQL validation rejects an input object containing any field the schema does not declare, so a mutation carrying an extra key fails outright rather than ignoring it. These shapes come from the published reference and are **not** live-validated. Traps worth knowing:

| Input | Fields | Trap |
| --- | --- | --- |
| `UpdateCashTransactionInput` | `cashTransactionId: ID!`, `note: String` | Only the note is editable. |
| `ReassignCashTransactionInput` | `cashTransactionId: ID!`, `subaccountId: ID` | — |
| `SplitCashTransactionInput` | `cashTransactionId: ID!`, `splits: [CashTransactionSplitInput]` | Split entries are `{subaccountId, amount}`. |
| `CreateSubaccountInput` | `accountId: ID!`, `name: String!`, `type`, `note`, `goal`, `targetAmount`, `initialTransferAmount`, `piggyBanked` | `goal` is the pocket's savings target. |
| `UpdateSubaccountInput` | `subaccountId: ID!`, `name`, `type`, `note`, `goal`, `targetAmount`, `piggyBanked` | This is how a savings goal is set. |
| `SetTargetBalanceInput` | `accountId: ID!`, `targetBalance: Int!`, `buffer`, `direction` | Keyed on an **account**, not a subaccount, and returns a `TargetBalanceSetting`, not a `Subaccount`. Unrelated to pocket goals. |
| `RemoveTargetBalanceInput` | `accountId: ID!` | Same — account, not subaccount. |
| `SetSpendSubaccountInput` | `userId: ID!`, `selectedSpendSubaccountId: ID` | Chooses the pocket physical card swipes spend from. Blank/null means the default subaccount, so send an explicit `null` to clear rather than omitting the key. Payload is `{result: User!}`, not a Subaccount — read the new value back at `result { userSpendConfig { selectedSpendSubaccount } }`. |
| `InitiateTransferInput` | `accountFromId: ID!`, `accountToId: ID!`, `amount: Int!`, `memo`, `note`, `type` | IDs may be an account **or** a subaccount. |
| `UpdateTransferInput` | `transferId: ID!`, `memo`, `note` | — |
| `CancelTransferInput` | `transferId: ID!` | — |
| `FreezeDebitCardInput` | `debitCardId: ID!`, `reason: CardFrozenReason!` | The reason is **required**. |
| `UnfreezeDebitCardInput` / `CancelDebitCardInput` | `debitCardId: ID!` | — |
| `ActivateDebitCardsInput` | `debitCardId: [ID!]!` | Singular name, list type. |
| `CreateVirtualDebitCardInput` | `userId: ID!`, `name`, `subaccountId`, `monthlyLimit`, `cardColor`, `formFactor`, `cancelAfter`, `type` | `userId` is required; the label field is `name`, not `nickname`. |
| `UpdateVirtualDebitCardInput` | `debitCardId: ID!`, `name`, `subaccountId`, `monthlyLimit`, `cardColor`, `cancelAfter` | `debitCardId`, not `virtualDebitCardId`. |

Enum values: `CardFrozenReason` = FRAUD_DETECTED, FROZEN_BY_BANK, LOST_OR_STOLEN, PARENT_REQUESTED, USER_FROZEN, USER_REQUESTED; `CardFrozenStatus` = FAILED, FREEZING, FROZEN, UNFREEZING, UNFROZEN; `DebitCardStatus` = ACTIVATED, ACTIVATING, DEACTIVATED, DEACTIVATING, EMANCIPATED, EXPIRED, FAILED, ISSUED, ISSUING; `DebitCardFormFactor` = PHYSICAL, PROVISIONAL, SINGLE_USE, VIRTUAL; `TargetBalanceSettingDirection` = BOTH, FROM_OVERFLOW, TO_OVERFLOW.

## Type-name traps

- **`cashTransactions` is not on `User`.** It hangs off `Account` and `Subaccount` (`Account.cashTransactions`, `Subaccount.cashTransactions`), plus `DebitCard.transactions` — which returns the same `CashTransactionConnection` but takes no `searchFilters`. `currentUser { cashTransactions }` is a hard schema error: *Cannot query field "cashTransactions" on type "User". Did you mean "cashTransactionDeclines"?* Reach for `currentUser.spendAccount`; the other accounts report `total: 0`. There is no root `account(id:)` field, so a specific account is fetched through `node(id:)` with an `... on Account` fragment.
- **There is no `VirtualDebitCard` type.** `currentUser.virtualDebitCards` returns `[DebitCard!]!`, as do all the card mutations. Virtual cards are `DebitCard`s whose `formFactor` is `VIRTUAL` or `SINGLE_USE`.
- **`DebitCard` field names**: `lastFour` (not `last4`), `name` (not `nickname`), and freezing is `frozenStatus` + `frozenReason` (not a `frozen` boolean).
- **`Subaccount.type` is an `AccountType`** — the parent account's kind. The pocket's own kind is `subaccountType: SubaccountType!`.
- **`currentUser.transfers` takes no `searchFilters` argument**, only `first`/`last`/`after`/`before`. `TransferFilter` applies to the account-level `transfersFrom`/`transfersTo` connections. By contrast `Account.cashTransactions` *does* accept `searchFilters`.
- **`CashTransactionFilter.debitCardId` is `[ID!]`**, a list despite the singular name.
- **`Account.family` is `Family!`** and is the same object for every account in one household, which makes its ID a household identifier. `Family` exposes no `name` — only `id` is worth carrying. `User.family` also exists but is nullable (`Family`), so the account route is the reliable one.
- **A card's spending pocket lives in three different places**, consulted in this order:
  1. `DebitCard.subaccount` — a *virtual* card's own pin; always null on physical cards.
  2. `User.userSpendConfig.selectedSpendSubaccount` — the member's explicit choice, and what physical card swipes follow.
  3. `Account.primarySubaccount` — the account default, used only when (2) is null.

  `setSpendSubaccount` writes (2) and leaves (3) untouched, so reading `primarySubaccount` alone reports a stale pocket for any member who has made a choice. `updateVirtualDebitCard` writes (1) only and cannot repoint a physical card.
- **`selectedSpendSubaccount` hangs off `userSpendConfig`, not off `User`.** Requesting `selectedSpendSubaccount` or `selectedSpendSubaccountIsExpired` directly on `User` is rejected by the live endpoint (2026-08-17) — the working selection is `currentUser { userSpendConfig { id selectedSpendSubaccount { id name } } }`. Introspection is disabled server-side (`forbidden`), so this was established by probing field names against the live endpoint. `UserSpendConfig` also exposes `allowOverspending`, `postSpendPocketAssignment`, and `selectedSpendSubaccountIsExpired`, none of which this SDK requests.

Enums (observed/documented values): `SubaccountType` = BILL, BILL_RESERVE, CREDIT, CREDIT_RESERVE, SAVINGS, SPENDING; `TransferStatus` = CANCELED, CANCELING, COMPLETED, DECISION_ACCEPTED, DECISION_MANUAL_REVIEW, DECISION_PENDING, DECISION_REJECTED, DECISION_RETRYING; `TransferType` = ACH, ADJUSTMENT, ALLOWANCE, BILL_SUBACCOUNT, BONUS, BOOK, CASH_DEPOSIT, CHECK, …

## CashTransaction schema notes (live-validated 2026-08-16)

- Schema introspection (`__type`) is **disabled** in production (returns `forbidden`); the fields below were discovered by probing and reading the server's "did you mean" validation errors.
- `title` is the enriched payee/merchant display name the Crew app shows (e.g. "Costco"); `merchantName` and `description` are typically **null** on card transactions. `imageUrl` is an enriched merchant logo (served from spadeapi.com).
- `mcc` is the card-network merchant category code (string); `externalId` looks like `ttx_...`.
- `subaccountRunningTotal` / `accountRunningTotal` are the pocket/account balances (cents) after the transaction.
- Fields confirmed to exist but not yet mapped by the SDK (query via `Execute`): `account`, `transfer`, `checkDeposit`, `originalCheckDepositTransaction`, `fundingEvent`, `seasoning`, `disputeReasons`, `enrichmentId`, `relatedTransactions`, `splitTransactions`, `permittedActions`.
- Confirmed NOT to exist: `payee`, `counterparty`, `category`, `location`, `createdAt`, `date`, `pending`, `direction`, `statementDescriptor`.
- `description` is **deprecated** in the published schema ("use memo instead"). It is still queryable and this SDK still maps it, but new code should read `memo`.

## Known unknowns

- **Rate limits:** undocumented. This SDK polls conservatively (60s default).
- **Idempotency:** undocumented; `initiateTransfer` has no idempotency key. A timed-out transfer may still have executed — never blind-retry money movement.
- **Token TTL:** undocumented. Assume it can die at any time and handle 401 by re-running the login flow.
- **Auth path discrepancy:** `/willow/auth` vs `/willow/graphql/auth` (see header note). Both bases are configurable in this SDK.
