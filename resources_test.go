package crew

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const fakeUserJSON = `{
	"id": "user-1", "firstName": "Ben", "lastName": "Meeker",
	"email": "ben@example.com", "phone": "5555555555",
	"accounts": [{
		"id": "acct-1", "type": "SPEND", "name": "Spending", "overallBalance": 100000,
		"subaccounts": [
			{"id": "sub-1", "name": "Groceries", "subaccountType": "SPENDING", "overallBalance": 25000, "goal": null},
			{"id": "sub-2", "name": "Rent", "subaccountType": "BILL", "overallBalance": 150000, "goal": 200000}
		],
		"primarySubaccount": {"id": "sub-1", "name": "Groceries"},
		"family": {"id": "fam-1"}
	}],
	"userSpendConfig": {"id": "usc-1", "selectedSpendSubaccount": {"id": "sub-2", "name": "Rent"}}
}`

func TestCurrentUser(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("query CurrentUser", `{"data":{"currentUser":`+fakeUserJSON+`}}`)

	u, err := c.CurrentUser(context.Background())
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}
	if u.ID != "user-1" || u.FirstName != "Ben" {
		t.Errorf("user = %+v", u)
	}
	if len(u.Accounts) != 1 || len(u.Accounts[0].Subaccounts) != 2 {
		t.Fatalf("accounts = %+v", u.Accounts)
	}
	sub := u.Accounts[0].Subaccounts[1]
	if sub.GoalCents == nil || *sub.GoalCents != 200000 {
		t.Errorf("sub-2 goal = %v, want 200000", sub.GoalCents)
	}
	primary := u.Accounts[0].PrimarySubaccount
	if primary == nil || primary.ID != "sub-1" || primary.Name != "Groceries" {
		t.Errorf("primarySubaccount = %+v", primary)
	}
	if fam := u.Accounts[0].Family; fam == nil || fam.ID != "fam-1" {
		t.Errorf("family = %+v", fam)
	}
	if selected := u.SelectedSpendSubaccount(); selected == nil || selected.ID != "sub-2" {
		t.Errorf("SelectedSpendSubaccount = %+v, want sub-2", selected)
	}
	// selectedSpendSubaccount must be requested under userSpendConfig; the
	// deployed schema rejects it directly on User.
	req := f.lastRequest()
	if !strings.Contains(req.Query, "userSpendConfig { id selectedSpendSubaccount") {
		t.Errorf("query does not request selectedSpendSubaccount under userSpendConfig: %s", req.Query)
	}
}

func TestSelectedSpendSubaccountWithoutConfig(t *testing.T) {
	// A user who has never chosen a pocket has no selection, and callers
	// fall back to the account default.
	if got := (User{}).SelectedSpendSubaccount(); got != nil {
		t.Errorf("SelectedSpendSubaccount = %+v, want nil", got)
	}
	u := User{SpendConfig: &UserSpendConfig{ID: "usc-1"}}
	if got := u.SelectedSpendSubaccount(); got != nil {
		t.Errorf("SelectedSpendSubaccount = %+v, want nil", got)
	}
}

func TestCurrentUserFamilyID(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("CurrentUserFamilyID", `{"data":{"currentUser":{"accounts":[{"family":{"id":"fam-1"}}]}}}`)

	id, err := c.CurrentUserFamilyID(context.Background())
	if err != nil {
		t.Fatalf("CurrentUserFamilyID: %v", err)
	}
	if id != "fam-1" {
		t.Errorf("family ID = %q, want fam-1", id)
	}
}

func TestCurrentUserFamilyIDSkipsAccountsWithoutFamily(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("CurrentUserFamilyID", `{"data":{"currentUser":{"accounts":[{"family":null},{"family":{"id":"fam-2"}}]}}}`)

	id, err := c.CurrentUserFamilyID(context.Background())
	if err != nil {
		t.Fatalf("CurrentUserFamilyID: %v", err)
	}
	if id != "fam-2" {
		t.Errorf("family ID = %q, want fam-2", id)
	}
}

func TestCurrentUserFamilyIDWithoutFamilyIsNotAnError(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("CurrentUserFamilyID", `{"data":{"currentUser":{"accounts":[]}}}`)

	// A user with no household is a normal state, not a failure: callers
	// use a blank ID to mean "nothing to link".
	id, err := c.CurrentUserFamilyID(context.Background())
	if err != nil {
		t.Fatalf("CurrentUserFamilyID: %v", err)
	}
	if id != "" {
		t.Errorf("family ID = %q, want empty", id)
	}
}

func TestSetSpendSubaccount(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("setSpendSubaccount", `{"data":{"setSpendSubaccount":{"result":`+fakeUserJSON+`}}}`)

	u, err := c.SetSpendSubaccount(context.Background(), "user-1", "sub-2")
	if err != nil {
		t.Fatalf("SetSpendSubaccount: %v", err)
	}
	if u.ID != "user-1" {
		t.Errorf("user = %+v", u)
	}
	input := f.lastRequest().Variables["input"].(map[string]any)
	if input["userId"] != "user-1" || input["selectedSpendSubaccountId"] != "sub-2" {
		t.Errorf("input = %v", input)
	}
}

func TestSetSpendSubaccountClearsWithExplicitNull(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("setSpendSubaccount", `{"data":{"setSpendSubaccount":{"result":`+fakeUserJSON+`}}}`)

	if _, err := c.SetSpendSubaccount(context.Background(), "user-1", ""); err != nil {
		t.Fatalf("SetSpendSubaccount: %v", err)
	}
	// Clearing must send null, not omit the key: an absent field leaves the
	// existing selection in place.
	input := f.lastRequest().Variables["input"].(map[string]any)
	v, ok := input["selectedSpendSubaccountId"]
	if !ok {
		t.Fatalf("selectedSpendSubaccountId was omitted: %v", input)
	}
	if v != nil {
		t.Errorf("selectedSpendSubaccountId = %v, want nil", v)
	}
}

func TestAccounts(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("query Accounts", `{"data":{"currentUser":`+fakeUserJSON+`}}`)

	accounts, err := c.Accounts(context.Background())
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	if len(accounts) != 1 || accounts[0].BalanceCents != 100000 {
		t.Errorf("accounts = %+v", accounts)
	}
	if accounts[0].Type != AccountTypeSpend {
		t.Errorf("type = %q, want SPEND", accounts[0].Type)
	}
}

func TestSpendAccount(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("spendAccount", `{"data":{"currentUser":{"spendAccount":{"id":"acct-1","type":"SPEND","name":"Spending","overallBalance":100000}}}}`)

	acct, err := c.SpendAccount(context.Background())
	if err != nil {
		t.Fatalf("SpendAccount: %v", err)
	}
	if acct.ID != "acct-1" {
		t.Errorf("account = %+v", acct)
	}
}

func TestSaveAccount(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("saveAccount", `{"data":{"currentUser":{"saveAccount":{"id":"acct-2","type":"SAVE","name":"Savings","overallBalance":500000}}}}`)

	acct, err := c.SaveAccount(context.Background())
	if err != nil {
		t.Fatalf("SaveAccount: %v", err)
	}
	if acct.ID != "acct-2" || acct.Type != AccountTypeSave {
		t.Errorf("account = %+v", acct)
	}
}

func TestSubaccountsFlattens(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("query Accounts", `{"data":{"currentUser":`+fakeUserJSON+`}}`)

	subs, err := c.Subaccounts(context.Background())
	if err != nil {
		t.Fatalf("Subaccounts: %v", err)
	}
	if len(subs) != 2 || subs[0].ID != "sub-1" || subs[1].ID != "sub-2" {
		t.Errorf("subs = %+v", subs)
	}
}

func TestCreateSubaccount(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("createSubaccount", `{"data":{"createSubaccount":{"result":{"id":"sub-3","name":"Vacation","subaccountType":"SAVINGS","overallBalance":0}}}}`)

	sub, err := c.CreateSubaccount(context.Background(), CreateSubaccountInput{AccountID: "acct-1", Name: "Vacation"})
	if err != nil {
		t.Fatalf("CreateSubaccount: %v", err)
	}
	if sub.ID != "sub-3" || sub.Type != SubaccountTypeSavings {
		t.Errorf("sub = %+v", sub)
	}
	input := f.lastRequest().Variables["input"].(map[string]any)
	if input["accountId"] != "acct-1" || input["name"] != "Vacation" {
		t.Errorf("input = %v", input)
	}
}

func TestUpdateSubaccount(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("updateSubaccount", `{"data":{"updateSubaccount":{"result":{"id":"sub-1","name":"Food","subaccountType":"SPENDING","overallBalance":25000,"goal":40000}}}}`)

	goal := int64(40000)
	sub, err := c.UpdateSubaccount(context.Background(), UpdateSubaccountInput{SubaccountID: "sub-1", Name: "Food", GoalCents: &goal})
	if err != nil {
		t.Fatalf("UpdateSubaccount: %v", err)
	}
	if sub.Name != "Food" || sub.GoalCents == nil || *sub.GoalCents != 40000 {
		t.Errorf("sub = %+v", sub)
	}
	// A pocket's savings goal is set here, not via SetTargetBalance.
	input := f.lastRequest().Variables["input"].(map[string]any)
	if input["goal"] != float64(40000) {
		t.Errorf("input = %v", input)
	}
}

func TestDeleteSubaccount(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("deleteSubaccount", `{"data":{"deleteSubaccount":{"result":{"id":"sub-1"}}}}`)

	if err := c.DeleteSubaccount(context.Background(), "sub-1"); err != nil {
		t.Fatalf("DeleteSubaccount: %v", err)
	}
	input := f.lastRequest().Variables["input"].(map[string]any)
	if input["subaccountId"] != "sub-1" {
		t.Errorf("input = %v", input)
	}
}

func TestSetTargetBalance(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("setTargetBalance", `{"data":{"setTargetBalance":{"result":{"id":"tbs-1","targetBalance":50000,"buffer":2500,"direction":"BOTH","enabled":true}}}}`)

	setting, err := c.SetTargetBalance(context.Background(), SetTargetBalanceInput{
		AccountID:          "acct-1",
		TargetBalanceCents: 50000,
	})
	if err != nil {
		t.Fatalf("SetTargetBalance: %v", err)
	}
	if setting.TargetBalanceCents != 50000 || setting.Direction != TargetBalanceDirectionBoth || !setting.Enabled {
		t.Errorf("setting = %+v", setting)
	}
	// Target balances are set on an account, not a subaccount.
	input := f.lastRequest().Variables["input"].(map[string]any)
	if input["accountId"] != "acct-1" || input["targetBalance"] != float64(50000) {
		t.Errorf("input = %v", input)
	}
	if _, ok := input["buffer"]; ok {
		t.Errorf("unset buffer was sent: %v", input)
	}
}

func TestRemoveTargetBalance(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("removeTargetBalance", `{"data":{"removeTargetBalance":{"result":{"id":"tbs-1","enabled":false}}}}`)

	setting, err := c.RemoveTargetBalance(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("RemoveTargetBalance: %v", err)
	}
	if setting.Enabled {
		t.Errorf("setting = %+v, want disabled", setting)
	}
	input := f.lastRequest().Variables["input"].(map[string]any)
	if input["accountId"] != "acct-1" {
		t.Errorf("input = %v", input)
	}
}

func txPageJSON(hasNext bool, endCursor string, txs ...string) string {
	edges := make([]string, len(txs))
	for i, tx := range txs {
		edges[i] = `{"node":` + tx + `}`
	}
	return `{"data":{"currentUser":{"cashTransactions":{
		"edges":[` + strings.Join(edges, ",") + `],
		"pageInfo":{"startCursor":"s","endCursor":"` + endCursor + `","hasNextPage":` + boolStr(hasNext) + `,"hasPreviousPage":false}
	}}}}`
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestCashTransactions(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("cashTransactions", txPageJSON(false, "c1",
		`{"id":"tx-1","amount":-500,"status":"PENDING","description":"Coffee"}`))

	page, err := c.CashTransactions(context.Background(), CashTransactionsOptions{
		First:  10,
		Filter: &CashTransactionFilter{SubaccountID: "sub-1"},
	})
	if err != nil {
		t.Fatalf("CashTransactions: %v", err)
	}
	if len(page.Transactions) != 1 || page.Transactions[0].ID != "tx-1" {
		t.Errorf("page = %+v", page)
	}
	if page.PageInfo.EndCursor != "c1" || page.PageInfo.HasNextPage {
		t.Errorf("pageInfo = %+v", page.PageInfo)
	}
	vars := f.lastRequest().Variables
	if vars["first"] != float64(10) {
		t.Errorf("first = %v, want 10", vars["first"])
	}
	filter := vars["filter"].(map[string]any)
	if filter["subaccountId"] != "sub-1" {
		t.Errorf("filter = %v", filter)
	}
}

func TestAllCashTransactionsPaginates(t *testing.T) {
	c, f := newTestServer(t)
	// Page 1 (no "after" variable), then page 2 (after=c1).
	f.setGQL(`"after":"c1"`, txPageJSON(false, "c2", `{"id":"tx-2","amount":-200}`))
	f.setGQL("cashTransactions", txPageJSON(true, "c1", `{"id":"tx-1","amount":-100}`))

	var ids []string
	for tx, err := range c.AllCashTransactions(context.Background(), nil) {
		if err != nil {
			t.Fatalf("iterate: %v", err)
		}
		ids = append(ids, tx.ID)
	}
	if len(ids) != 2 || ids[0] != "tx-1" || ids[1] != "tx-2" {
		t.Errorf("ids = %v, want [tx-1 tx-2]", ids)
	}
}

func TestAllCashTransactionsYieldsError(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQLStatus(500)

	var count int
	var lastErr error
	for _, err := range c.AllCashTransactions(context.Background(), nil) {
		count++
		lastErr = err
	}
	if count != 1 {
		t.Fatalf("yielded %d times, want 1", count)
	}
	var apiErr *APIError
	if !errors.As(lastErr, &apiErr) {
		t.Errorf("err = %v, want *APIError", lastErr)
	}
}

func TestUpdateCashTransaction(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("updateCashTransaction", `{"data":{"updateCashTransaction":{"result":{"id":"tx-1","note":"Espresso"}}}}`)

	tx, err := c.UpdateCashTransaction(context.Background(), UpdateCashTransactionInput{CashTransactionID: "tx-1", Note: "Espresso"})
	if err != nil {
		t.Fatalf("UpdateCashTransaction: %v", err)
	}
	if tx.Note != "Espresso" {
		t.Errorf("tx = %+v", tx)
	}
	// Crew's UpdateCashTransactionInput accepts only cashTransactionId and
	// note; any other key fails GraphQL input validation server-side.
	input := f.lastRequest().Variables["input"].(map[string]any)
	if input["cashTransactionId"] != "tx-1" || input["note"] != "Espresso" {
		t.Errorf("input = %v", input)
	}
	for key := range input {
		if key != "cashTransactionId" && key != "note" {
			t.Errorf("input contains unsupported field %q: %v", key, input)
		}
	}
}

func TestReassignCashTransaction(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("reassignCashTransaction", `{"data":{"reassignCashTransaction":{"result":{"id":"tx-1","subaccount":{"id":"sub-2"}}}}}`)

	tx, err := c.ReassignCashTransaction(context.Background(), "tx-1", "sub-2")
	if err != nil {
		t.Fatalf("ReassignCashTransaction: %v", err)
	}
	if tx.Subaccount == nil || tx.Subaccount.ID != "sub-2" {
		t.Errorf("tx = %+v", tx)
	}
	input := f.lastRequest().Variables["input"].(map[string]any)
	if input["cashTransactionId"] != "tx-1" || input["subaccountId"] != "sub-2" {
		t.Errorf("input = %v", input)
	}
}

func TestSplitCashTransaction(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("splitCashTransaction", `{"data":{"splitCashTransaction":{"result":[{"id":"tx-1a","amount":-300},{"id":"tx-1b","amount":-200}]}}}`)

	txs, err := c.SplitCashTransaction(context.Background(), SplitCashTransactionInput{
		CashTransactionID: "tx-1",
		Splits: []TransactionSplit{
			{SubaccountID: "sub-1", AmountCents: -300},
			{SubaccountID: "sub-2", AmountCents: -200},
		},
	})
	if err != nil {
		t.Fatalf("SplitCashTransaction: %v", err)
	}
	if len(txs) != 2 || txs[0].ID != "tx-1a" {
		t.Errorf("txs = %+v", txs)
	}
}

func TestTransfers(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("transfers", `{"data":{"currentUser":{"transfers":{
		"edges":[{"node":{"id":"tr-1","amount":10000,"status":"COMPLETED","type":"BOOK","occurredAt":"2026-08-01T00:00:00Z"}}],
		"pageInfo":{"startCursor":"s","endCursor":"e","hasNextPage":false,"hasPreviousPage":false}
	}}}}`)

	page, err := c.Transfers(context.Background(), TransfersOptions{First: 5})
	if err != nil {
		t.Fatalf("Transfers: %v", err)
	}
	if len(page.Transfers) != 1 || page.Transfers[0].Status != TransferStatusCompleted {
		t.Errorf("page = %+v", page)
	}
	// currentUser.transfers accepts no searchFilters argument, so the query
	// must not declare or pass a filter variable.
	req := f.lastRequest()
	if strings.Contains(req.Query, "searchFilters") {
		t.Errorf("query passes searchFilters to currentUser.transfers: %s", req.Query)
	}
	if _, ok := req.Variables["filter"]; ok {
		t.Errorf("variables = %v, want no filter", req.Variables)
	}
}

func TestInitiateTransfer(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("initiateTransfer", `{"data":{"initiateTransfer":{"result":{"id":"tr-2","amount":2500,"status":"DECISION_PENDING","type":"BOOK"}}}}`)

	tr, err := c.InitiateTransfer(context.Background(), InitiateTransferInput{
		AccountFromID: "sub-1",
		AccountToID:   "sub-2",
		AmountCents:   2500,
		Memo:          "rebalance",
	})
	if err != nil {
		t.Fatalf("InitiateTransfer: %v", err)
	}
	if tr.ID != "tr-2" || tr.Status != TransferStatusDecisionPending {
		t.Errorf("transfer = %+v", tr)
	}
	input := f.lastRequest().Variables["input"].(map[string]any)
	if input["amount"] != float64(2500) || input["accountFromId"] != "sub-1" || input["memo"] != "rebalance" {
		t.Errorf("input = %v", input)
	}
}

func TestCancelTransfer(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("cancelTransfer", `{"data":{"cancelTransfer":{"result":{"id":"tr-2","status":"CANCELED"}}}}`)

	tr, err := c.CancelTransfer(context.Background(), "tr-2")
	if err != nil {
		t.Fatalf("CancelTransfer: %v", err)
	}
	if tr.Status != TransferStatusCanceled {
		t.Errorf("transfer = %+v", tr)
	}
}

func TestUpdateTransfer(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("updateTransfer", `{"data":{"updateTransfer":{"result":{"id":"tr-2","memo":"fixed"}}}}`)

	tr, err := c.UpdateTransfer(context.Background(), UpdateTransferInput{TransferID: "tr-2", Memo: "fixed"})
	if err != nil {
		t.Fatalf("UpdateTransfer: %v", err)
	}
	if tr.Memo != "fixed" {
		t.Errorf("transfer = %+v", tr)
	}
}

func TestDebitCards(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("debitCards", `{"data":{"currentUser":{"debitCards":[{"id":"card-1","lastFour":"1234","name":"Everyday","status":"ACTIVATED","formFactor":"PHYSICAL","frozenStatus":"UNFROZEN","subaccount":null,"account":{"id":"acct-1"}}]}}}`)

	cards, err := c.DebitCards(context.Background())
	if err != nil {
		t.Fatalf("DebitCards: %v", err)
	}
	if len(cards) != 1 || cards[0].LastFour != "1234" || cards[0].Status != DebitCardStatusActivated {
		t.Errorf("cards = %+v", cards)
	}
	if cards[0].IsFrozen() || cards[0].IsVirtual() {
		t.Errorf("card = %+v, want unfrozen physical", cards[0])
	}
	if cards[0].Account == nil || cards[0].Account.ID != "acct-1" {
		t.Errorf("account = %+v", cards[0].Account)
	}
	// Physical cards carry no pinned pocket of their own.
	if cards[0].Subaccount != nil {
		t.Errorf("subaccount = %+v, want nil on a physical card", cards[0].Subaccount)
	}
	if got := cards[0].SpendSubaccount(nil, nil); got != nil {
		t.Errorf("SpendSubaccount(nil, nil) = %+v, want nil", got)
	}
}

func TestSpendSubaccountPrefersTheMembersChoice(t *testing.T) {
	physical := DebitCard{ID: "card-1", FormFactor: DebitCardFormFactorPhysical}
	account := &Account{ID: "acct-1", PrimarySubaccount: &Subaccount{ID: "sub-1", Name: "Groceries"}}

	// With no explicit choice, a physical card follows the account default.
	noChoice := &User{}
	if got := physical.SpendSubaccount(noChoice, account); got == nil || got.ID != "sub-1" {
		t.Errorf("SpendSubaccount = %+v, want the account default sub-1", got)
	}

	// Once the member picks a pocket, that wins — PrimarySubaccount does
	// not move when SetSpendSubaccount is called, so consulting it alone
	// reports the stale pocket.
	chose := &User{SpendConfig: &UserSpendConfig{
		ID:                      "usc-1",
		SelectedSpendSubaccount: &Subaccount{ID: "sub-2", Name: "Rent"},
	}}
	if got := physical.SpendSubaccount(chose, account); got == nil || got.ID != "sub-2" {
		t.Errorf("SpendSubaccount = %+v, want the member's choice sub-2", got)
	}

	// A virtual card's own pin outranks the member-level choice.
	virtual := DebitCard{
		ID:         "vcard-1",
		FormFactor: DebitCardFormFactorVirtual,
		Subaccount: &Subaccount{ID: "sub-3", Name: "Fun Money"},
	}
	if got := virtual.SpendSubaccount(chose, account); got == nil || got.ID != "sub-3" {
		t.Errorf("SpendSubaccount = %+v, want the card's own pin sub-3", got)
	}
}

func TestVirtualDebitCards(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("virtualDebitCards", `{"data":{"currentUser":{"virtualDebitCards":[{"id":"vcard-1","lastFour":"9876","name":"Streaming","status":"ACTIVATED","formFactor":"VIRTUAL","monthlyLimit":5000,"subaccount":{"id":"sub-3","name":"Fun Money"},"account":{"id":"acct-1"}}]}}}`)

	cards, err := c.VirtualDebitCards(context.Background())
	if err != nil {
		t.Fatalf("VirtualDebitCards: %v", err)
	}
	if len(cards) != 1 || cards[0].Name != "Streaming" || !cards[0].IsVirtual() {
		t.Errorf("cards = %+v", cards)
	}
	if cards[0].MonthlyLimitCents == nil || *cards[0].MonthlyLimitCents != 5000 {
		t.Errorf("monthlyLimit = %v", cards[0].MonthlyLimitCents)
	}
	if cards[0].Subaccount == nil || cards[0].Subaccount.Name != "Fun Money" {
		t.Errorf("subaccount = %+v", cards[0].Subaccount)
	}
	// A virtual card's own pinned pocket wins over the account's primary.
	account := &Account{ID: "acct-1", PrimarySubaccount: &Subaccount{ID: "sub-1"}}
	if got := cards[0].SpendSubaccount(nil, account); got == nil || got.ID != "sub-3" {
		t.Errorf("SpendSubaccount = %+v, want sub-3", got)
	}
}

func TestFreezeDebitCard(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("freezeDebitCard", `{"data":{"freezeDebitCard":{"result":{"id":"card-1","frozenStatus":"FROZEN","frozenReason":"LOST_OR_STOLEN"}}}}`)

	card, err := c.FreezeDebitCard(context.Background(), "card-1", CardFrozenReasonLostOrStolen)
	if err != nil {
		t.Fatalf("FreezeDebitCard: %v", err)
	}
	if !card.IsFrozen() || card.FrozenReason != CardFrozenReasonLostOrStolen {
		t.Errorf("card = %+v, want frozen", card)
	}
	// Crew requires a reason on this input; omitting it fails validation.
	input := f.lastRequest().Variables["input"].(map[string]any)
	if input["debitCardId"] != "card-1" || input["reason"] != "LOST_OR_STOLEN" {
		t.Errorf("input = %v", input)
	}
}

func TestUnfreezeDebitCard(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("unfreezeDebitCard", `{"data":{"unfreezeDebitCard":{"result":{"id":"card-1","frozenStatus":"UNFROZEN"}}}}`)

	card, err := c.UnfreezeDebitCard(context.Background(), "card-1")
	if err != nil {
		t.Fatalf("UnfreezeDebitCard: %v", err)
	}
	if card.IsFrozen() {
		t.Errorf("card = %+v, want unfrozen", card)
	}
}

func TestFreezingCardIsNotYetFrozen(t *testing.T) {
	card := DebitCard{FrozenStatus: CardFrozenStatusFreezing}
	if card.IsFrozen() {
		t.Error("a FREEZING card reported IsFrozen")
	}
}

func TestCancelDebitCard(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("cancelDebitCard", `{"data":{"cancelDebitCard":{"result":{"id":"card-1","status":"DEACTIVATED"}}}}`)

	card, err := c.CancelDebitCard(context.Background(), "card-1")
	if err != nil {
		t.Fatalf("CancelDebitCard: %v", err)
	}
	if card.Status != DebitCardStatusDeactivated {
		t.Errorf("card = %+v", card)
	}
}

func TestActivateDebitCards(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("activateDebitCards", `{"data":{"activateDebitCards":{"result":[{"id":"card-1","status":"ACTIVATED"}]}}}`)

	cards, err := c.ActivateDebitCards(context.Background(), []string{"card-1", "card-2"})
	if err != nil {
		t.Fatalf("ActivateDebitCards: %v", err)
	}
	if len(cards) != 1 || cards[0].Status != DebitCardStatusActivated {
		t.Errorf("cards = %+v", cards)
	}
	// The schema names this list field debitCardId, singular.
	input := f.lastRequest().Variables["input"].(map[string]any)
	ids, ok := input["debitCardId"].([]any)
	if !ok || len(ids) != 2 || ids[0] != "card-1" {
		t.Errorf("input = %v", input)
	}
}

func TestCreateVirtualDebitCard(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("createVirtualDebitCard", `{"data":{"createVirtualDebitCard":{"result":{"id":"vcard-2","name":"Trial subscriptions","formFactor":"VIRTUAL"}}}}`)

	limit := int64(2500)
	card, err := c.CreateVirtualDebitCard(context.Background(), CreateVirtualDebitCardInput{
		UserID:            "user-1",
		Name:              "Trial subscriptions",
		MonthlyLimitCents: &limit,
	})
	if err != nil {
		t.Fatalf("CreateVirtualDebitCard: %v", err)
	}
	if card.ID != "vcard-2" || card.Name != "Trial subscriptions" {
		t.Errorf("card = %+v", card)
	}
	input := f.lastRequest().Variables["input"].(map[string]any)
	if input["userId"] != "user-1" || input["name"] != "Trial subscriptions" || input["monthlyLimit"] != float64(2500) {
		t.Errorf("input = %v", input)
	}
}

func TestUpdateVirtualDebitCard(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("updateVirtualDebitCard", `{"data":{"updateVirtualDebitCard":{"result":{"id":"vcard-2","name":"Renamed"}}}}`)

	card, err := c.UpdateVirtualDebitCard(context.Background(), UpdateVirtualDebitCardInput{DebitCardID: "vcard-2", Name: "Renamed"})
	if err != nil {
		t.Fatalf("UpdateVirtualDebitCard: %v", err)
	}
	if card.Name != "Renamed" {
		t.Errorf("card = %+v", card)
	}
	input := f.lastRequest().Variables["input"].(map[string]any)
	if input["debitCardId"] != "vcard-2" || input["name"] != "Renamed" {
		t.Errorf("input = %v", input)
	}
}
