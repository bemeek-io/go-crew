package crew

import "context"

// subaccountType, not type: on a Subaccount, type is the parent account's
// AccountType, while subaccountType is the pocket's own kind.
const subaccountFields = `id name subaccountType overallBalance goal`

const accountFields = `id type name overallBalance subaccounts { ` + subaccountFields + ` } primarySubaccount { id name }`

// Crew's published docs also list selectedSpendSubaccount and
// selectedSpendSubaccountIsExpired on User, but the deployed schema
// currently rejects both, so they are deliberately not requested here.
// Read the selected pocket from Account.PrimarySubaccount instead.
const userFields = `id firstName lastName email phone accounts { ` + accountFields + ` }`

const queryCurrentUser = `query CurrentUser {
  currentUser { ` + userFields + ` }
}`

// CurrentUser fetches the authenticated user with their accounts and
// subaccounts.
func (c *Client) CurrentUser(ctx context.Context) (*User, error) {
	var out struct {
		CurrentUser User `json:"currentUser"`
	}
	if err := c.Execute(ctx, queryCurrentUser, nil, &out); err != nil {
		return nil, err
	}
	return &out.CurrentUser, nil
}

const mutationSetSpendSubaccount = `mutation SetSpendSubaccount($input: SetSpendSubaccountInput!) {
  setSpendSubaccount(input: $input) {
    result { ` + userFields + ` }
  }
}`

// SetSpendSubaccount chooses the pocket a user's physical card swipes spend
// from, returning the updated user. Pass an empty subaccountID to fall back
// to the account's default pocket.
//
// This is the only way to repoint a physical card: UpdateVirtualDebitCard's
// SubaccountID applies to per-merchant virtual cards only. The result is
// visible as Account.PrimarySubaccount.
func (c *Client) SetSpendSubaccount(ctx context.Context, userID, subaccountID string) (*User, error) {
	var out struct {
		SetSpendSubaccount struct {
			Result User `json:"result"`
		} `json:"setSpendSubaccount"`
	}
	// Sent explicitly as null rather than omitted, so that clearing the
	// selection reaches the server as a value.
	input := map[string]any{"userId": userID, "selectedSpendSubaccountId": nil}
	if subaccountID != "" {
		input["selectedSpendSubaccountId"] = subaccountID
	}
	if err := c.Execute(ctx, mutationSetSpendSubaccount, map[string]any{"input": input}, &out); err != nil {
		return nil, err
	}
	return &out.SetSpendSubaccount.Result, nil
}
