package crew

import "context"

// subaccountType, not type: on a Subaccount, type is the parent account's
// AccountType, while subaccountType is the pocket's own kind.
const subaccountFields = `id name subaccountType overallBalance goal`

const accountFields = `id type name overallBalance subaccounts { ` + subaccountFields + ` } primarySubaccount { id name } family { id }`

// selectedSpendSubaccount hangs off userSpendConfig, not off User — asking
// for it directly on User is what the deployed schema rejects.
const userFields = `id firstName lastName email phone accounts { ` + accountFields + ` } userSpendConfig { id selectedSpendSubaccount { id name } }`

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

// Deliberately narrower than queryCurrentUser: this runs on the login path,
// so it fetches only what the family lookup needs.
const queryCurrentUserFamilyID = `query CurrentUserFamilyID {
  currentUser {
    accounts { family { id } }
  }
}`

// CurrentUserFamilyID returns the ID of the Crew household the user's
// accounts belong to. Every account in a household shares one, so two
// members can be linked by comparing this value — no invite exchange
// needed.
//
// It returns an empty ID and a nil error when the user has no accounts or
// no family, so a blank result means "nothing to link", not a failure.
// Callers using this for optional linking should treat any error the same
// way and fall back to their own path rather than failing the caller.
func (c *Client) CurrentUserFamilyID(ctx context.Context) (string, error) {
	var out struct {
		CurrentUser struct {
			Accounts []Account `json:"accounts"`
		} `json:"currentUser"`
	}
	if err := c.Execute(ctx, queryCurrentUserFamilyID, nil, &out); err != nil {
		return "", err
	}
	for _, a := range out.CurrentUser.Accounts {
		if a.Family != nil && a.Family.ID != "" {
			return a.Family.ID, nil
		}
	}
	return "", nil
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
// visible as User.SelectedSpendSubaccount(); it does not change the
// account default, Account.PrimarySubaccount.
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
