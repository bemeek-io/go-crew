package crew

import (
	"context"
	"iter"
)

const transferFields = `id amount status type memo errorCode isCancellable occurredAt`

const queryTransfers = `query Transfers($first: Int, $last: Int, $after: String, $before: String, $filter: TransferFilter) {
  currentUser {
    transfers(first: $first, last: $last, after: $after, before: $before, filter: $filter) {
      edges { node { ` + transferFields + ` } }
      pageInfo { startCursor endCursor hasNextPage hasPreviousPage }
    }
  }
}`

// TransfersOptions control pagination and filtering for Transfers.
type TransfersOptions struct {
	First  int
	Last   int
	After  string
	Before string
	Filter *TransferFilter
}

// TransferPage is one page of transfers.
type TransferPage struct {
	Transfers []Transfer
	PageInfo  PageInfo
}

// Transfers fetches one page of the user's transfers, newest first.
func (c *Client) Transfers(ctx context.Context, opts TransfersOptions) (*TransferPage, error) {
	var out struct {
		CurrentUser struct {
			Transfers connection[Transfer] `json:"transfers"`
		} `json:"currentUser"`
	}
	var filter any
	if opts.Filter != nil {
		filter = opts.Filter
	}
	vars := connectionVariables(opts.First, opts.Last, opts.After, opts.Before, filter)
	if err := c.Execute(ctx, queryTransfers, vars, &out); err != nil {
		return nil, err
	}
	conn := out.CurrentUser.Transfers
	return &TransferPage{Transfers: conn.nodes(), PageInfo: conn.PageInfo}, nil
}

// AllTransfers iterates every transfer, fetching pages of 100 as needed.
// Iteration stops after yielding a non-nil error.
func (c *Client) AllTransfers(ctx context.Context, filter *TransferFilter) iter.Seq2[Transfer, error] {
	return func(yield func(Transfer, error) bool) {
		after := ""
		for {
			page, err := c.Transfers(ctx, TransfersOptions{First: 100, After: after, Filter: filter})
			if err != nil {
				yield(Transfer{}, err)
				return
			}
			for _, tr := range page.Transfers {
				if !yield(tr, nil) {
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

// InitiateTransferInput are the parameters for InitiateTransfer. IDs may
// reference an account or a subaccount. AmountCents is in cents.
type InitiateTransferInput struct {
	AccountFromID string `json:"accountFromId"`
	AccountToID   string `json:"accountToId"`
	AmountCents   int64  `json:"amount"`
	Memo          string `json:"memo,omitempty"`
	Note          string `json:"note,omitempty"`
}

const mutationInitiateTransfer = `mutation InitiateTransfer($input: InitiateTransferInput!) {
  initiateTransfer(input: $input) {
    result { ` + transferFields + ` }
  }
}`

// InitiateTransfer moves money between accounts or subaccounts.
//
// It is NOT safely retryable: the Crew API documents no idempotency
// mechanism, so a timeout or network error may mean the transfer went
// through anyway. Never retry automatically; check Transfers first.
func (c *Client) InitiateTransfer(ctx context.Context, in InitiateTransferInput) (*Transfer, error) {
	var out struct {
		InitiateTransfer struct {
			Result Transfer `json:"result"`
		} `json:"initiateTransfer"`
	}
	if err := c.Execute(ctx, mutationInitiateTransfer, map[string]any{"input": in}, &out); err != nil {
		return nil, err
	}
	return &out.InitiateTransfer.Result, nil
}

const mutationCancelTransfer = `mutation CancelTransfer($input: CancelTransferInput!) {
  cancelTransfer(input: $input) {
    result { ` + transferFields + ` }
  }
}`

// CancelTransfer cancels a pending transfer.
func (c *Client) CancelTransfer(ctx context.Context, transferID string) (*Transfer, error) {
	var out struct {
		CancelTransfer struct {
			Result Transfer `json:"result"`
		} `json:"cancelTransfer"`
	}
	input := map[string]any{"transferId": transferID}
	if err := c.Execute(ctx, mutationCancelTransfer, map[string]any{"input": input}, &out); err != nil {
		return nil, err
	}
	return &out.CancelTransfer.Result, nil
}

// UpdateTransferInput are the parameters for UpdateTransfer.
type UpdateTransferInput struct {
	TransferID string `json:"transferId"`
	Memo       string `json:"memo,omitempty"`
}

const mutationUpdateTransfer = `mutation UpdateTransfer($input: UpdateTransferInput!) {
  updateTransfer(input: $input) {
    result { ` + transferFields + ` }
  }
}`

// UpdateTransfer updates a transfer's attributes.
func (c *Client) UpdateTransfer(ctx context.Context, in UpdateTransferInput) (*Transfer, error) {
	var out struct {
		UpdateTransfer struct {
			Result Transfer `json:"result"`
		} `json:"updateTransfer"`
	}
	if err := c.Execute(ctx, mutationUpdateTransfer, map[string]any{"input": in}, &out); err != nil {
		return nil, err
	}
	return &out.UpdateTransfer.Result, nil
}
