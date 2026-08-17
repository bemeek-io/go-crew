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

// DebitCardStatus is the lifecycle state of a debit card. Unknown values
// from the API pass through unchanged.
type DebitCardStatus string

// Known debit card statuses.
const (
	DebitCardStatusActivated    DebitCardStatus = "ACTIVATED"
	DebitCardStatusActivating   DebitCardStatus = "ACTIVATING"
	DebitCardStatusDeactivated  DebitCardStatus = "DEACTIVATED"
	DebitCardStatusDeactivating DebitCardStatus = "DEACTIVATING"
	DebitCardStatusEmancipated  DebitCardStatus = "EMANCIPATED"
	DebitCardStatusExpired      DebitCardStatus = "EXPIRED"
	DebitCardStatusFailed       DebitCardStatus = "FAILED"
	DebitCardStatusIssued       DebitCardStatus = "ISSUED"
	DebitCardStatusIssuing      DebitCardStatus = "ISSUING"
)

func (s DebitCardStatus) String() string { return string(s) }

// DebitCardFormFactor distinguishes physical cards from virtual ones.
// Unknown values from the API pass through unchanged.
type DebitCardFormFactor string

// Known debit card form factors.
const (
	DebitCardFormFactorPhysical    DebitCardFormFactor = "PHYSICAL"
	DebitCardFormFactorProvisional DebitCardFormFactor = "PROVISIONAL"
	DebitCardFormFactorSingleUse   DebitCardFormFactor = "SINGLE_USE"
	DebitCardFormFactorVirtual     DebitCardFormFactor = "VIRTUAL"
)

func (f DebitCardFormFactor) String() string { return string(f) }

// CardFrozenStatus is a debit card's freeze state. Note that Crew models
// freezing as a state machine, not a boolean: a card may be mid-transition
// (FREEZING, UNFREEZING). Unknown values pass through unchanged.
type CardFrozenStatus string

// Known card freeze states.
const (
	CardFrozenStatusFailed     CardFrozenStatus = "FAILED"
	CardFrozenStatusFreezing   CardFrozenStatus = "FREEZING"
	CardFrozenStatusFrozen     CardFrozenStatus = "FROZEN"
	CardFrozenStatusUnfreezing CardFrozenStatus = "UNFREEZING"
	CardFrozenStatusUnfrozen   CardFrozenStatus = "UNFROZEN"
)

func (s CardFrozenStatus) String() string { return string(s) }

// CardFrozenReason explains why a debit card was frozen. FreezeDebitCard
// requires one. Unknown values from the API pass through unchanged.
type CardFrozenReason string

// Known freeze reasons.
const (
	CardFrozenReasonFraudDetected   CardFrozenReason = "FRAUD_DETECTED"
	CardFrozenReasonFrozenByBank    CardFrozenReason = "FROZEN_BY_BANK"
	CardFrozenReasonLostOrStolen    CardFrozenReason = "LOST_OR_STOLEN"
	CardFrozenReasonParentRequested CardFrozenReason = "PARENT_REQUESTED"
	CardFrozenReasonUserFrozen      CardFrozenReason = "USER_FROZEN"
	CardFrozenReasonUserRequested   CardFrozenReason = "USER_REQUESTED"
)

func (r CardFrozenReason) String() string { return string(r) }

// DebitCardColor is the card's appearance in the Crew app. Unknown values
// from the API pass through unchanged.
type DebitCardColor string

// Known card colors.
const (
	DebitCardColorBanana DebitCardColor = "BANANA"
	DebitCardColorBeige  DebitCardColor = "BEIGE"
	DebitCardColorBlack  DebitCardColor = "BLACK"
	DebitCardColorCoral  DebitCardColor = "CORAL"
	DebitCardColorDenim  DebitCardColor = "DENIM"
	DebitCardColorJade   DebitCardColor = "JADE"
	DebitCardColorLilac  DebitCardColor = "LILAC"
	DebitCardColorMetal  DebitCardColor = "METAL"
	DebitCardColorTeal   DebitCardColor = "TEAL"
)

func (c DebitCardColor) String() string { return string(c) }

// VirtualCardFormFactor is the form factor requested when creating a
// virtual card. Unknown values from the API pass through unchanged.
type VirtualCardFormFactor string

// Known virtual card form factors.
const (
	VirtualCardFormFactorSingleUse VirtualCardFormFactor = "SINGLE_USE"
	VirtualCardFormFactorVirtual   VirtualCardFormFactor = "VIRTUAL"
)

func (f VirtualCardFormFactor) String() string { return string(f) }

// TargetBalanceSettingDirection controls which way money moves between an
// account and its overflow account. Unknown values pass through unchanged.
type TargetBalanceSettingDirection string

// Known target balance directions.
const (
	TargetBalanceDirectionBoth         TargetBalanceSettingDirection = "BOTH"
	TargetBalanceDirectionFromOverflow TargetBalanceSettingDirection = "FROM_OVERFLOW"
	TargetBalanceDirectionToOverflow   TargetBalanceSettingDirection = "TO_OVERFLOW"
)

func (d TargetBalanceSettingDirection) String() string { return string(d) }

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
	// PrimarySubaccount is the pocket the account's physical cards spend
	// from, carrying only ID and Name. Change it with SetSpendSubaccount.
	// Virtual cards are pinned per-card instead — see DebitCard.Subaccount.
	PrimarySubaccount *Subaccount `json:"primarySubaccount"`
	// Family is the household this account belongs to. Every account in a
	// Crew household shares one, so it identifies members of the same
	// household without any exchange between them. It is nil only when the
	// field wasn't requested — see CurrentUserFamilyID.
	Family *Family `json:"family"`
}

// Family is a Crew household. Crew's schema exposes no name for it, so the
// ID is all there is to carry.
type Family struct {
	ID string `json:"id"`
}

// Subaccount is a "pocket" within an account.
type Subaccount struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Type is the pocket's kind. It maps the schema's subaccountType field —
	// Subaccount.type is an AccountType (the parent account's kind), not a
	// SubaccountType.
	Type         SubaccountType `json:"subaccountType"`
	BalanceCents int64          `json:"overallBalance"`
	// GoalCents is the savings target ("goal"), or nil if none is set. Set
	// it with UpdateSubaccount; SetTargetBalance is a different,
	// account-level feature.
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

// DebitCard is a Crew debit card. Crew has no separate virtual card type:
// virtual cards are DebitCards whose FormFactor is VIRTUAL or SINGLE_USE.
type DebitCard struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// LastFour is the last four digits of the card number.
	LastFour     string              `json:"lastFour"`
	Status       DebitCardStatus     `json:"status"`
	FormFactor   DebitCardFormFactor `json:"formFactor"`
	FrozenStatus CardFrozenStatus    `json:"frozenStatus"`
	FrozenReason CardFrozenReason    `json:"frozenReason"`
	Color        DebitCardColor      `json:"color"`
	// MonthlyLimitCents is the card's monthly spend cap, or nil if uncapped.
	MonthlyLimitCents *int64 `json:"monthlyLimit"`
	// MonthlySpendToDateCents is spend so far in the current calendar month.
	MonthlySpendToDateCents int64 `json:"monthlySpendToDate"`
	// Subaccount is the pocket this card is pinned to, carrying only ID and
	// Name. It is always nil for physical cards, which spend from their
	// account's PrimarySubaccount instead — use SpendSubaccount to resolve
	// either kind.
	Subaccount *Subaccount `json:"subaccount"`
	// Account is the card's owning account, carrying only ID.
	Account *Account `json:"account"`
}

// SpendSubaccount returns the pocket the card spends from: its own pinned
// Subaccount for virtual cards, or the given account's PrimarySubaccount
// for physical ones. It returns nil if that pocket wasn't queried — pass
// the Account this card belongs to, which DebitCards does not fetch.
func (c DebitCard) SpendSubaccount(account *Account) *Subaccount {
	if c.Subaccount != nil {
		return c.Subaccount
	}
	if account == nil {
		return nil
	}
	return account.PrimarySubaccount
}

// IsFrozen reports whether the card is fully frozen. Cards mid-transition
// (FREEZING, UNFREEZING) report false — inspect FrozenStatus for those.
func (c DebitCard) IsFrozen() bool { return c.FrozenStatus == CardFrozenStatusFrozen }

// IsVirtual reports whether the card is virtual rather than physical.
func (c DebitCard) IsVirtual() bool {
	return c.FormFactor == DebitCardFormFactorVirtual || c.FormFactor == DebitCardFormFactorSingleUse
}

// TargetBalanceSetting is an account-level rule that keeps an account at a
// target balance by sweeping money to or from an overflow account. It is
// not a subaccount savings goal — see Subaccount.GoalCents for that.
type TargetBalanceSetting struct {
	ID                 string                        `json:"id"`
	TargetBalanceCents int64                         `json:"targetBalance"`
	BufferCents        int64                         `json:"buffer"`
	Direction          TargetBalanceSettingDirection `json:"direction"`
	Enabled            bool                          `json:"enabled"`
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
	Amount *IntegerRange `json:"amount,omitempty"`
	// DebitCardIDs serializes to the schema's list-typed debitCardId field.
	DebitCardIDs  []string     `json:"debitCardId,omitempty"`
	FuzzySearch   string       `json:"fuzzySearch,omitempty"`
	SubaccountID  string       `json:"subaccountId,omitempty"`
	SubaccountIDs []string     `json:"subaccountIds,omitempty"`
	TransferSide  TransferSide `json:"transferSide,omitempty"`
	Type          string       `json:"type,omitempty"`
}
