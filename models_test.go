package crew

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDateUnmarshal(t *testing.T) {
	var d Date
	if err := json.Unmarshal([]byte(`"2026-08-16"`), &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	want := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	if !d.Equal(want) {
		t.Errorf("date = %v, want %v", d.Time, want)
	}
}

func TestDateUnmarshalNull(t *testing.T) {
	var d Date
	if err := json.Unmarshal([]byte(`null`), &d); err != nil {
		t.Fatalf("Unmarshal null: %v", err)
	}
	if !d.IsZero() {
		t.Errorf("date = %v, want zero", d.Time)
	}
}

func TestDateUnmarshalInvalid(t *testing.T) {
	var d Date
	if err := json.Unmarshal([]byte(`"not-a-date"`), &d); err == nil {
		t.Fatal("Unmarshal invalid date: want error, got nil")
	}
}

func TestDateMarshal(t *testing.T) {
	d := Date{time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(b) != `"2026-08-16"` {
		t.Errorf("marshaled = %s, want \"2026-08-16\"", b)
	}
}

func TestEnumStrings(t *testing.T) {
	if got := TransferStatusCompleted.String(); got != "COMPLETED" {
		t.Errorf("TransferStatusCompleted = %q", got)
	}
	if got := SubaccountTypeBillReserve.String(); got != "BILL_RESERVE" {
		t.Errorf("SubaccountTypeBillReserve = %q", got)
	}
	if got := AccountTypeSpend.String(); got != "SPEND" {
		t.Errorf("AccountTypeSpend = %q", got)
	}
	if got := TransferTypeACH.String(); got != "ACH" {
		t.Errorf("TransferTypeACH = %q", got)
	}
}

func TestUnknownEnumValuePassesThrough(t *testing.T) {
	var tr Transfer
	if err := json.Unmarshal([]byte(`{"status":"SOME_NEW_STATUS"}`), &tr); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if tr.Status.String() != "SOME_NEW_STATUS" {
		t.Errorf("status = %q, want SOME_NEW_STATUS", tr.Status)
	}
}

func TestCashTransactionUnmarshal(t *testing.T) {
	raw := `{
		"id": "tx-1",
		"amount": -1250,
		"title": "Cafe",
		"description": null,
		"status": "CLEARED",
		"type": "CARD",
		"mcc": "5812",
		"merchantName": null,
		"merchantCity": "Lehi",
		"merchantState": "UT",
		"occurredAt": "2026-08-16T12:30:00.551000Z",
		"clearedAt": null,
		"subaccountRunningTotal": 24500,
		"accountRunningTotal": 900000,
		"subaccount": {"id": "sub-1", "name": "Groceries"},
		"debitCard": {"id": "card-1"}
	}`
	var tx CashTransaction
	if err := json.Unmarshal([]byte(raw), &tx); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if tx.ID != "tx-1" || tx.AmountCents != -1250 || tx.Status != "CLEARED" {
		t.Errorf("tx = %+v", tx)
	}
	if tx.OccurredAt.Hour() != 12 || tx.ClearedAt != nil {
		t.Errorf("timestamps = %v / %v", tx.OccurredAt, tx.ClearedAt)
	}
	if tx.Subaccount == nil || tx.Subaccount.ID != "sub-1" {
		t.Errorf("subaccount = %+v", tx.Subaccount)
	}
	if tx.DebitCard == nil || tx.DebitCard.ID != "card-1" {
		t.Errorf("debitCard = %+v", tx.DebitCard)
	}
	if tx.Payee() != "Cafe" {
		t.Errorf("Payee() = %q, want Cafe", tx.Payee())
	}
	if tx.SubaccountRunningTotalCents != 24500 || tx.AccountRunningTotalCents != 900000 {
		t.Errorf("running totals = %d / %d", tx.SubaccountRunningTotalCents, tx.AccountRunningTotalCents)
	}
}

func TestPayeeFallbacks(t *testing.T) {
	if got := (CashTransaction{Title: "T", MerchantName: "M", Description: "D"}).Payee(); got != "T" {
		t.Errorf("Payee() = %q, want T", got)
	}
	if got := (CashTransaction{MerchantName: "M", Description: "D"}).Payee(); got != "M" {
		t.Errorf("Payee() = %q, want M", got)
	}
	if got := (CashTransaction{Description: "D"}).Payee(); got != "D" {
		t.Errorf("Payee() = %q, want D", got)
	}
}

func TestCashTransactionFilterOmitsEmpty(t *testing.T) {
	b, err := json.Marshal(CashTransactionFilter{SubaccountID: "sub-1"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(b) != `{"subaccountId":"sub-1"}` {
		t.Errorf("marshaled = %s, want only subaccountId", b)
	}
}
