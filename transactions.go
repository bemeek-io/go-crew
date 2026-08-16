package crew

import (
	"context"
	"iter"
)

const cashTransactionFields = `id amount description status type merchantName subaccountId debitCardId date createdAt`

const queryCashTransactions = `query CashTransactions($first: Int, $last: Int, $after: String, $before: String, $filter: CashTransactionFilter) {
  currentUser {
    cashTransactions(first: $first, last: $last, after: $after, before: $before, filter: $filter) {
      edges { node { ` + cashTransactionFields + ` } }
      pageInfo { startCursor endCursor hasNextPage hasPreviousPage }
    }
  }
}`

// CashTransactionsOptions control pagination and filtering for
// CashTransactions.
type CashTransactionsOptions struct {
	First  int
	Last   int
	After  string
	Before string
	Filter *CashTransactionFilter
}

// CashTransactionPage is one page of cash transactions.
type CashTransactionPage struct {
	Transactions []CashTransaction
	PageInfo     PageInfo
}

type connection[T any] struct {
	Edges []struct {
		Node T `json:"node"`
	} `json:"edges"`
	PageInfo PageInfo `json:"pageInfo"`
}

func (c connection[T]) nodes() []T {
	nodes := make([]T, len(c.Edges))
	for i, e := range c.Edges {
		nodes[i] = e.Node
	}
	return nodes
}

// connectionVariables builds the standard Relay pagination variables,
// omitting unset values.
func connectionVariables(first, last int, after, before string, filter any) map[string]any {
	vars := map[string]any{}
	if first > 0 {
		vars["first"] = first
	}
	if last > 0 {
		vars["last"] = last
	}
	if after != "" {
		vars["after"] = after
	}
	if before != "" {
		vars["before"] = before
	}
	if filter != nil {
		vars["filter"] = filter
	}
	return vars
}

// CashTransactions fetches one page of the user's cash transactions,
// newest first.
func (c *Client) CashTransactions(ctx context.Context, opts CashTransactionsOptions) (*CashTransactionPage, error) {
	var out struct {
		CurrentUser struct {
			CashTransactions connection[CashTransaction] `json:"cashTransactions"`
		} `json:"currentUser"`
	}
	var filter any
	if opts.Filter != nil {
		filter = opts.Filter
	}
	vars := connectionVariables(opts.First, opts.Last, opts.After, opts.Before, filter)
	if err := c.Execute(ctx, queryCashTransactions, vars, &out); err != nil {
		return nil, err
	}
	conn := out.CurrentUser.CashTransactions
	return &CashTransactionPage{Transactions: conn.nodes(), PageInfo: conn.PageInfo}, nil
}

// AllCashTransactions iterates every cash transaction, fetching pages of 100
// as needed. Iteration stops after yielding a non-nil error.
func (c *Client) AllCashTransactions(ctx context.Context, filter *CashTransactionFilter) iter.Seq2[CashTransaction, error] {
	return func(yield func(CashTransaction, error) bool) {
		after := ""
		for {
			page, err := c.CashTransactions(ctx, CashTransactionsOptions{First: 100, After: after, Filter: filter})
			if err != nil {
				yield(CashTransaction{}, err)
				return
			}
			for _, tx := range page.Transactions {
				if !yield(tx, nil) {
					return
				}
			}
			if !page.PageInfo.HasNextPage {
				return
			}
			after = page.PageInfo.EndCursor
		}
	}
}

// UpdateCashTransactionInput are the parameters for UpdateCashTransaction.
type UpdateCashTransactionInput struct {
	CashTransactionID string `json:"cashTransactionId"`
	Description       string `json:"description,omitempty"`
}

const mutationUpdateCashTransaction = `mutation UpdateCashTransaction($input: UpdateCashTransactionInput!) {
  updateCashTransaction(input: $input) {
    result { ` + cashTransactionFields + ` }
  }
}`

// UpdateCashTransaction updates a transaction's attributes.
func (c *Client) UpdateCashTransaction(ctx context.Context, in UpdateCashTransactionInput) (*CashTransaction, error) {
	var out struct {
		UpdateCashTransaction struct {
			Result CashTransaction `json:"result"`
		} `json:"updateCashTransaction"`
	}
	if err := c.Execute(ctx, mutationUpdateCashTransaction, map[string]any{"input": in}, &out); err != nil {
		return nil, err
	}
	return &out.UpdateCashTransaction.Result, nil
}

const mutationReassignCashTransaction = `mutation ReassignCashTransaction($input: ReassignCashTransactionInput!) {
  reassignCashTransaction(input: $input) {
    result { ` + cashTransactionFields + ` }
  }
}`

// ReassignCashTransaction moves a transaction to a different subaccount.
func (c *Client) ReassignCashTransaction(ctx context.Context, transactionID, subaccountID string) (*CashTransaction, error) {
	var out struct {
		ReassignCashTransaction struct {
			Result CashTransaction `json:"result"`
		} `json:"reassignCashTransaction"`
	}
	input := map[string]any{"cashTransactionId": transactionID, "subaccountId": subaccountID}
	if err := c.Execute(ctx, mutationReassignCashTransaction, map[string]any{"input": input}, &out); err != nil {
		return nil, err
	}
	return &out.ReassignCashTransaction.Result, nil
}

// TransactionSplit assigns part of a transaction's amount to a subaccount.
type TransactionSplit struct {
	SubaccountID string `json:"subaccountId"`
	AmountCents  int64  `json:"amount"`
}

// SplitCashTransactionInput are the parameters for SplitCashTransaction.
type SplitCashTransactionInput struct {
	CashTransactionID string             `json:"cashTransactionId"`
	Splits            []TransactionSplit `json:"splits"`
}

const mutationSplitCashTransaction = `mutation SplitCashTransaction($input: SplitCashTransactionInput!) {
  splitCashTransaction(input: $input) {
    result { ` + cashTransactionFields + ` }
  }
}`

// SplitCashTransaction divides a transaction across multiple subaccounts.
func (c *Client) SplitCashTransaction(ctx context.Context, in SplitCashTransactionInput) ([]CashTransaction, error) {
	var out struct {
		SplitCashTransaction struct {
			Result []CashTransaction `json:"result"`
		} `json:"splitCashTransaction"`
	}
	if err := c.Execute(ctx, mutationSplitCashTransaction, map[string]any{"input": in}, &out); err != nil {
		return nil, err
	}
	return out.SplitCashTransaction.Result, nil
}
