package crew

import "context"

const queryAccounts = `query Accounts {
  currentUser {
    accounts { ` + accountFields + ` }
  }
}`

const querySpendAccount = `query SpendAccount {
  currentUser {
    spendAccount { ` + accountFields + ` }
  }
}`

const querySaveAccount = `query SaveAccount {
  currentUser {
    saveAccount { ` + accountFields + ` }
  }
}`

// Accounts fetches all of the user's accounts with their subaccounts.
func (c *Client) Accounts(ctx context.Context) ([]Account, error) {
	var out struct {
		CurrentUser struct {
			Accounts []Account `json:"accounts"`
		} `json:"currentUser"`
	}
	if err := c.Execute(ctx, queryAccounts, nil, &out); err != nil {
		return nil, err
	}
	return out.CurrentUser.Accounts, nil
}

// SpendAccount fetches the user's primary spending account.
func (c *Client) SpendAccount(ctx context.Context) (*Account, error) {
	var out struct {
		CurrentUser struct {
			SpendAccount *Account `json:"spendAccount"`
		} `json:"currentUser"`
	}
	if err := c.Execute(ctx, querySpendAccount, nil, &out); err != nil {
		return nil, err
	}
	return out.CurrentUser.SpendAccount, nil
}

// SaveAccount fetches the user's savings account.
func (c *Client) SaveAccount(ctx context.Context) (*Account, error) {
	var out struct {
		CurrentUser struct {
			SaveAccount *Account `json:"saveAccount"`
		} `json:"currentUser"`
	}
	if err := c.Execute(ctx, querySaveAccount, nil, &out); err != nil {
		return nil, err
	}
	return out.CurrentUser.SaveAccount, nil
}

// Subaccounts fetches all subaccounts ("pockets") across the user's
// accounts.
func (c *Client) Subaccounts(ctx context.Context) ([]Subaccount, error) {
	accounts, err := c.Accounts(ctx)
	if err != nil {
		return nil, err
	}
	var subs []Subaccount
	for _, a := range accounts {
		subs = append(subs, a.Subaccounts...)
	}
	return subs, nil
}

// CreateSubaccountInput are the parameters for CreateSubaccount.
type CreateSubaccountInput struct {
	AccountID string `json:"accountId"`
	Name      string `json:"name"`
	GoalCents *int64 `json:"goal,omitempty"`
}

const mutationCreateSubaccount = `mutation CreateSubaccount($input: CreateSubaccountInput!) {
  createSubaccount(input: $input) {
    result { ` + subaccountFields + ` }
  }
}`

// CreateSubaccount creates a new subaccount ("pocket").
func (c *Client) CreateSubaccount(ctx context.Context, in CreateSubaccountInput) (*Subaccount, error) {
	var out struct {
		CreateSubaccount struct {
			Result Subaccount `json:"result"`
		} `json:"createSubaccount"`
	}
	if err := c.Execute(ctx, mutationCreateSubaccount, map[string]any{"input": in}, &out); err != nil {
		return nil, err
	}
	return &out.CreateSubaccount.Result, nil
}

// UpdateSubaccountInput are the parameters for UpdateSubaccount.
type UpdateSubaccountInput struct {
	SubaccountID string `json:"subaccountId"`
	Name         string `json:"name,omitempty"`
}

const mutationUpdateSubaccount = `mutation UpdateSubaccount($input: UpdateSubaccountInput!) {
  updateSubaccount(input: $input) {
    result { ` + subaccountFields + ` }
  }
}`

// UpdateSubaccount updates a subaccount's attributes.
func (c *Client) UpdateSubaccount(ctx context.Context, in UpdateSubaccountInput) (*Subaccount, error) {
	var out struct {
		UpdateSubaccount struct {
			Result Subaccount `json:"result"`
		} `json:"updateSubaccount"`
	}
	if err := c.Execute(ctx, mutationUpdateSubaccount, map[string]any{"input": in}, &out); err != nil {
		return nil, err
	}
	return &out.UpdateSubaccount.Result, nil
}

const mutationDeleteSubaccount = `mutation DeleteSubaccount($input: DeleteSubaccountInput!) {
  deleteSubaccount(input: $input) {
    result { id }
  }
}`

// DeleteSubaccount deletes a subaccount.
func (c *Client) DeleteSubaccount(ctx context.Context, subaccountID string) error {
	input := map[string]any{"subaccountId": subaccountID}
	return c.Execute(ctx, mutationDeleteSubaccount, map[string]any{"input": input}, nil)
}

const mutationSetTargetBalance = `mutation SetTargetBalance($input: SetTargetBalanceInput!) {
  setTargetBalance(input: $input) {
    result { ` + subaccountFields + ` }
  }
}`

// SetTargetBalance sets a subaccount's savings target, in cents.
func (c *Client) SetTargetBalance(ctx context.Context, subaccountID string, amountCents int64) (*Subaccount, error) {
	var out struct {
		SetTargetBalance struct {
			Result Subaccount `json:"result"`
		} `json:"setTargetBalance"`
	}
	input := map[string]any{"subaccountId": subaccountID, "targetBalance": amountCents}
	if err := c.Execute(ctx, mutationSetTargetBalance, map[string]any{"input": input}, &out); err != nil {
		return nil, err
	}
	return &out.SetTargetBalance.Result, nil
}

const mutationRemoveTargetBalance = `mutation RemoveTargetBalance($input: RemoveTargetBalanceInput!) {
  removeTargetBalance(input: $input) {
    result { ` + subaccountFields + ` }
  }
}`

// RemoveTargetBalance clears a subaccount's savings target.
func (c *Client) RemoveTargetBalance(ctx context.Context, subaccountID string) (*Subaccount, error) {
	var out struct {
		RemoveTargetBalance struct {
			Result Subaccount `json:"result"`
		} `json:"removeTargetBalance"`
	}
	input := map[string]any{"subaccountId": subaccountID}
	if err := c.Execute(ctx, mutationRemoveTargetBalance, map[string]any{"input": input}, &out); err != nil {
		return nil, err
	}
	return &out.RemoveTargetBalance.Result, nil
}
