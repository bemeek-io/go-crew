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
			{"id": "sub-1", "name": "Groceries", "type": "SPENDING", "overallBalance": 25000, "goal": null},
			{"id": "sub-2", "name": "Rent", "type": "BILL", "overallBalance": 150000, "goal": 200000}
		]
	}]
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
	f.setGQL("createSubaccount", `{"data":{"createSubaccount":{"result":{"id":"sub-3","name":"Vacation","type":"SAVINGS","balance":0}}}}`)

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
	f.setGQL("updateSubaccount", `{"data":{"updateSubaccount":{"result":{"id":"sub-1","name":"Food","type":"SPENDING","balance":25000}}}}`)

	sub, err := c.UpdateSubaccount(context.Background(), UpdateSubaccountInput{SubaccountID: "sub-1", Name: "Food"})
	if err != nil {
		t.Fatalf("UpdateSubaccount: %v", err)
	}
	if sub.Name != "Food" {
		t.Errorf("sub = %+v", sub)
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
	f.setGQL("setTargetBalance", `{"data":{"setTargetBalance":{"result":{"id":"sub-1","goal":50000}}}}`)

	sub, err := c.SetTargetBalance(context.Background(), "sub-1", 50000)
	if err != nil {
		t.Fatalf("SetTargetBalance: %v", err)
	}
	if sub.GoalCents == nil || *sub.GoalCents != 50000 {
		t.Errorf("goal = %v", sub.GoalCents)
	}
	input := f.lastRequest().Variables["input"].(map[string]any)
	if input["targetBalance"] != float64(50000) {
		t.Errorf("input = %v", input)
	}
}

func TestRemoveTargetBalance(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("removeTargetBalance", `{"data":{"removeTargetBalance":{"result":{"id":"sub-1","goal":null}}}}`)

	sub, err := c.RemoveTargetBalance(context.Background(), "sub-1")
	if err != nil {
		t.Fatalf("RemoveTargetBalance: %v", err)
	}
	if sub.GoalCents != nil {
		t.Errorf("goal = %v, want nil", sub.GoalCents)
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
	f.setGQL("updateCashTransaction", `{"data":{"updateCashTransaction":{"result":{"id":"tx-1","description":"Espresso"}}}}`)

	tx, err := c.UpdateCashTransaction(context.Background(), UpdateCashTransactionInput{CashTransactionID: "tx-1", Description: "Espresso"})
	if err != nil {
		t.Fatalf("UpdateCashTransaction: %v", err)
	}
	if tx.Description != "Espresso" {
		t.Errorf("tx = %+v", tx)
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
	f.setGQL("debitCards", `{"data":{"currentUser":{"debitCards":[{"id":"card-1","last4":"1234","status":"ACTIVE","frozen":false}]}}}`)

	cards, err := c.DebitCards(context.Background())
	if err != nil {
		t.Fatalf("DebitCards: %v", err)
	}
	if len(cards) != 1 || cards[0].Last4 != "1234" {
		t.Errorf("cards = %+v", cards)
	}
}

func TestVirtualDebitCards(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("virtualDebitCards", `{"data":{"currentUser":{"virtualDebitCards":[{"id":"vcard-1","last4":"9876","status":"ACTIVE","frozen":false,"nickname":"Streaming"}]}}}`)

	cards, err := c.VirtualDebitCards(context.Background())
	if err != nil {
		t.Fatalf("VirtualDebitCards: %v", err)
	}
	if len(cards) != 1 || cards[0].Nickname != "Streaming" {
		t.Errorf("cards = %+v", cards)
	}
}

func TestFreezeDebitCard(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("freezeDebitCard", `{"data":{"freezeDebitCard":{"result":{"id":"card-1","frozen":true}}}}`)

	card, err := c.FreezeDebitCard(context.Background(), "card-1")
	if err != nil {
		t.Fatalf("FreezeDebitCard: %v", err)
	}
	if !card.Frozen {
		t.Errorf("card = %+v, want frozen", card)
	}
	input := f.lastRequest().Variables["input"].(map[string]any)
	if input["debitCardId"] != "card-1" {
		t.Errorf("input = %v", input)
	}
}

func TestUnfreezeDebitCard(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("unfreezeDebitCard", `{"data":{"unfreezeDebitCard":{"result":{"id":"card-1","frozen":false}}}}`)

	card, err := c.UnfreezeDebitCard(context.Background(), "card-1")
	if err != nil {
		t.Fatalf("UnfreezeDebitCard: %v", err)
	}
	if card.Frozen {
		t.Errorf("card = %+v, want unfrozen", card)
	}
}

func TestCancelDebitCard(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("cancelDebitCard", `{"data":{"cancelDebitCard":{"result":{"id":"card-1","status":"CANCELED"}}}}`)

	card, err := c.CancelDebitCard(context.Background(), "card-1")
	if err != nil {
		t.Fatalf("CancelDebitCard: %v", err)
	}
	if card.Status != "CANCELED" {
		t.Errorf("card = %+v", card)
	}
}

func TestActivateDebitCards(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("activateDebitCards", `{"data":{"activateDebitCards":{"result":[{"id":"card-1","status":"ACTIVE"}]}}}`)

	cards, err := c.ActivateDebitCards(context.Background(), ActivateDebitCardsInput{DebitCardIDs: []string{"card-1"}, Last4: "1234"})
	if err != nil {
		t.Fatalf("ActivateDebitCards: %v", err)
	}
	if len(cards) != 1 || cards[0].Status != "ACTIVE" {
		t.Errorf("cards = %+v", cards)
	}
}

func TestCreateVirtualDebitCard(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("createVirtualDebitCard", `{"data":{"createVirtualDebitCard":{"result":{"id":"vcard-2","nickname":"Trial subscriptions"}}}}`)

	card, err := c.CreateVirtualDebitCard(context.Background(), CreateVirtualDebitCardInput{Nickname: "Trial subscriptions"})
	if err != nil {
		t.Fatalf("CreateVirtualDebitCard: %v", err)
	}
	if card.ID != "vcard-2" {
		t.Errorf("card = %+v", card)
	}
}

func TestUpdateVirtualDebitCard(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("updateVirtualDebitCard", `{"data":{"updateVirtualDebitCard":{"result":{"id":"vcard-2","nickname":"Renamed"}}}}`)

	card, err := c.UpdateVirtualDebitCard(context.Background(), UpdateVirtualDebitCardInput{VirtualDebitCardID: "vcard-2", Nickname: "Renamed"})
	if err != nil {
		t.Fatalf("UpdateVirtualDebitCard: %v", err)
	}
	if card.Nickname != "Renamed" {
		t.Errorf("card = %+v", card)
	}
}
