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
	AccountID string         `json:"accountId"`
	Name      string         `json:"name"`
	Type      SubaccountType `json:"type,omitempty"`
	Note      string         `json:"note,omitempty"`
	// GoalCents is the pocket's savings target.
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

// UpdateSubaccountInput are the parameters for UpdateSubaccount. This is
// also how a pocket's savings goal is set — SetTargetBalance is a separate,
// account-level feature.
type UpdateSubaccountInput struct {
	SubaccountID string         `json:"subaccountId"`
	Name         string         `json:"name,omitempty"`
	Type         SubaccountType `json:"type,omitempty"`
	Note         string         `json:"note,omitempty"`
	// GoalCents is the pocket's savings target.
	GoalCents *int64 `json:"goal,omitempty"`
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

const targetBalanceSettingFields = `id targetBalance buffer direction enabled`

// SetTargetBalanceInput are the parameters for SetTargetBalance. Note that
// target balances apply to an *account*, not a subaccount.
type SetTargetBalanceInput struct {
	AccountID          string `json:"accountId"`
	TargetBalanceCents int64  `json:"targetBalance"`
	// BufferCents is the slack allowed around the target before money is
	// swept, or nil to let Crew choose.
	BufferCents *int64 `json:"buffer,omitempty"`
	// Direction limits which way money may be swept.
	Direction TargetBalanceSettingDirection `json:"direction,omitempty"`
}

const mutationSetTargetBalance = `mutation SetTargetBalance($input: SetTargetBalanceInput!) {
  setTargetBalance(input: $input) {
    result { ` + targetBalanceSettingFields + ` }
  }
}`

// SetTargetBalance sets an account's target balance, sweeping money to or
// from its overflow account to maintain it.
//
// This is not a pocket savings goal: to set one of those, pass GoalCents to
// UpdateSubaccount.
func (c *Client) SetTargetBalance(ctx context.Context, in SetTargetBalanceInput) (*TargetBalanceSetting, error) {
	var out struct {
		SetTargetBalance struct {
			Result TargetBalanceSetting `json:"result"`
		} `json:"setTargetBalance"`
	}
	if err := c.Execute(ctx, mutationSetTargetBalance, map[string]any{"input": in}, &out); err != nil {
		return nil, err
	}
	return &out.SetTargetBalance.Result, nil
}

const mutationRemoveTargetBalance = `mutation RemoveTargetBalance($input: RemoveTargetBalanceInput!) {
  removeTargetBalance(input: $input) {
    result { ` + targetBalanceSettingFields + ` }
  }
}`

// RemoveTargetBalance clears an account's target balance.
func (c *Client) RemoveTargetBalance(ctx context.Context, accountID string) (*TargetBalanceSetting, error) {
	var out struct {
		RemoveTargetBalance struct {
			Result TargetBalanceSetting `json:"result"`
		} `json:"removeTargetBalance"`
	}
	input := map[string]any{"accountId": accountID}
	if err := c.Execute(ctx, mutationRemoveTargetBalance, map[string]any{"input": input}, &out); err != nil {
		return nil, err
	}
	return &out.RemoveTargetBalance.Result, nil
}
