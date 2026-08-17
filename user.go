package crew

import "context"

// subaccountType, not type: on a Subaccount, type is the parent account's
// AccountType, while subaccountType is the pocket's own kind.
const subaccountFields = `id name subaccountType overallBalance goal`

const accountFields = `id type name overallBalance subaccounts { ` + subaccountFields + ` }`

const queryCurrentUser = `query CurrentUser {
  currentUser {
    id firstName lastName email phone
    accounts { ` + accountFields + ` }
  }
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
