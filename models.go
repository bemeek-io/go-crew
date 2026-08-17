package crew

import (
	"fmt"
	"strings"
	"time"
)

// Date is Crew's date scalar, serialized as "2006-01-02".
type Date struct {
	time.Time
}

// MarshalJSON implements json.Marshaler.
func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Format("2006-01-02") + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("crew: parse date %q: %w", s, err)
	}
	d.Time = t
	return nil
}

// AccountType classifies an account. Unknown values from the API pass
// through unchanged.
type AccountType string

// Known account types.
const (
	AccountTypeCredit        AccountType = "CREDIT"
	AccountTypeExternalOther AccountType = "EXTERNAL_OTHER"
	AccountTypeExternalSave  AccountType = "EXTERNAL_SAVE"
	AccountTypeExternalSpend AccountType = "EXTERNAL_SPEND"
	AccountTypeSave          AccountType = "SAVE"
	AccountTypeSpend         AccountType = "SPEND"
)

func (t AccountType) String() string { return string(t) }

// SubaccountType classifies a subaccount ("pocket"). Unknown values from
// the API pass through unchanged.
type SubaccountType string

// Known subaccount types.
const (
	SubaccountTypeBill          SubaccountType = "BILL"
	SubaccountTypeBillReserve   SubaccountType = "BILL_RESERVE"
	SubaccountTypeCredit        SubaccountType = "CREDIT"
	SubaccountTypeCreditReserve SubaccountType = "CREDIT_RESERVE"
	SubaccountTypeSavings       SubaccountType = "SAVINGS"
	SubaccountTypeSpending      SubaccountType = "SPENDING"
)

func (t SubaccountType) String() string { return string(t) }

// TransferType classifies a transfer. Unknown values from the API pass
// through unchanged.
type TransferType string

// Known transfer types.
const (
	TransferTypeACH            TransferType = "ACH"
	TransferTypeAdjustment     TransferType = "ADJUSTMENT"
	TransferTypeAllowance      TransferType = "ALLOWANCE"
	TransferTypeBillSubaccount TransferType = "BILL_SUBACCOUNT"
	TransferTypeBonus          TransferType = "BONUS"
	TransferTypeBook           TransferType = "BOOK"
	TransferTypeCashDeposit    TransferType = "CASH_DEPOSIT"
	TransferTypeCheck          TransferType = "CHECK"
)

func (t TransferType) String() string { return string(t) }

// TransferStatus is the state of a transfer. Unknown values from the API
// pass through unchanged.
type TransferStatus string

// Known transfer statuses.
const (
	TransferStatusCanceled             TransferStatus = "CANCELED"
	TransferStatusCanceling            TransferStatus = "CANCELING"
	TransferStatusCompleted            TransferStatus = "COMPLETED"
	TransferStatusDecisionAccepted     TransferStatus = "DECISION_ACCEPTED"
	TransferStatusDecisionManualReview TransferStatus = "DECISION_MANUAL_REVIEW"
	TransferStatusDecisionPending      TransferStatus = "DECISION_PENDING"
	TransferStatusDecisionRejected     TransferStatus = "DECISION_REJECTED"
	TransferStatusDecisionRetrying     TransferStatus = "DECISION_RETRYING"
)

func (s TransferStatus) String() string { return string(s) }

// TransferSide filters transactions by direction.
type TransferSide string

// Transfer sides.
const (
	TransferSideCredit TransferSide = "CREDIT"
	TransferSideDebit  TransferSide = "DEBIT"
)

func (s TransferSide) String() string { return string(s) }

// User is the authenticated Crew user.
type User struct {
	ID        string    `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Accounts  []Account `json:"accounts"`
}

// Account is a Crew account (spend, save, or external).
type Account struct {
	ID           string       `json:"id"`
	Type         AccountType  `json:"type"`
	Name         string       `json:"name"`
	BalanceCents int64        `json:"overallBalance"`
	Subaccounts  []Subaccount `json:"subaccounts"`
}

// Subaccount is a "pocket" within an account.
type Subaccount struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Type         SubaccountType `json:"type"`
	BalanceCents int64          `json:"overallBalance"`
	// GoalCents is the savings target ("goal"), or nil if none is set.
	GoalCents *int64 `json:"goal"`
}

// CashTransaction is a money movement on a Crew account.
type CashTransaction struct {
	ID          string `json:"id"`
	AmountCents int64  `json:"amount"`
	// Title is the enriched payee/merchant display name (e.g. "Costco") —
	// the name the Crew app shows. MerchantName and Description are often
	// null on card transactions; prefer Title for display.
	Title string `json:"title"`
	// Deprecated: Crew's schema marks CashTransaction.description as
	// deprecated ("use memo instead"). It is still queried and populated,
	// but prefer Memo (or Payee for display) in new code.
	Description      string `json:"description"`
	Status           string `json:"status"`
	Type             string `json:"type"`
	MCC              string `json:"mcc"`
	MerchantName     string `json:"merchantName"`
	MerchantAddress1 string `json:"merchantAddress1"`
	MerchantCity     string `json:"merchantCity"`
	MerchantState    string `json:"merchantState"`
	MerchantZip      string `json:"merchantZip"`
	MerchantCountry  string `json:"merchantCountry"`
	ImageURL         string `json:"imageUrl"`
	// Note is the user's free-text annotation, set by UpdateCashTransaction.
	Note string `json:"note"`
	// Memo is the statement memo; it supersedes the deprecated Description.
	Memo       string     `json:"memo"`
	ExternalID string     `json:"externalId"`
	OccurredAt time.Time  `json:"occurredAt"`
	ClearedAt  *time.Time `json:"clearedAt"`
	// Running balances (in cents) after this transaction settled.
	SubaccountRunningTotalCents int64 `json:"subaccountRunningTotal"`
	AccountRunningTotalCents    int64 `json:"accountRunningTotal"`
	// Subaccount and DebitCard carry only the fields the SDK queries
	// (id and name); they are nil when the transaction has none.
	Subaccount *Subaccount `json:"subaccount"`
	DebitCard  *DebitCard  `json:"debitCard"`
}

// Payee returns the best available display name for the transaction:
// Title, then MerchantName, then Description.
func (t CashTransaction) Payee() string {
	switch {
	case t.Title != "":
		return t.Title
	case t.MerchantName != "":
		return t.MerchantName
	default:
		return t.Description
	}
}

// Transfer is a movement of funds between accounts or subaccounts.
type Transfer struct {
	ID            string         `json:"id"`
	AmountCents   int64          `json:"amount"`
	Status        TransferStatus `json:"status"`
	Type          TransferType   `json:"type"`
	Memo          string         `json:"memo"`
	ErrorCode     string         `json:"errorCode"`
	IsCancellable bool           `json:"isCancellable"`
	OccurredAt    time.Time      `json:"occurredAt"`
}

// DebitCard is a physical Crew debit card.
type DebitCard struct {
	ID       string `json:"id"`
	Last4    string `json:"last4"`
	Status   string `json:"status"`
	Frozen   bool   `json:"frozen"`
	Nickname string `json:"nickname"`
}

// VirtualDebitCard is a virtual Crew debit card.
type VirtualDebitCard struct {
	ID       string `json:"id"`
	Last4    string `json:"last4"`
	Status   string `json:"status"`
	Frozen   bool   `json:"frozen"`
	Nickname string `json:"nickname"`
}

// PageInfo is Relay-style pagination metadata.
type PageInfo struct {
	StartCursor     string `json:"startCursor"`
	EndCursor       string `json:"endCursor"`
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
}

// IntegerRange bounds an integer filter field.
type IntegerRange struct {
	Min *int64 `json:"min,omitempty"`
	Max *int64 `json:"max,omitempty"`
}

// CashTransactionFilter narrows a cash transaction query server-side.
type CashTransactionFilter struct {
	Amount        *IntegerRange `json:"amount,omitempty"`
	DebitCardID   string        `json:"debitCardId,omitempty"`
	FuzzySearch   string        `json:"fuzzySearch,omitempty"`
	SubaccountID  string        `json:"subaccountId,omitempty"`
	SubaccountIDs []string      `json:"subaccountIds,omitempty"`
	TransferSide  TransferSide  `json:"transferSide,omitempty"`
	Type          string        `json:"type,omitempty"`
}

// TransferFilter narrows a transfer query server-side.
type TransferFilter struct {
	Status TransferStatus `json:"status,omitempty"`
}
