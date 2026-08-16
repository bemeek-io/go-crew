package crew

import "context"

const debitCardFields = `id last4 status frozen nickname`

const queryDebitCards = `query DebitCards {
  currentUser {
    debitCards { ` + debitCardFields + ` }
  }
}`

const queryVirtualDebitCards = `query VirtualDebitCards {
  currentUser {
    virtualDebitCards { ` + debitCardFields + ` }
  }
}`

// DebitCards fetches the user's physical debit cards.
func (c *Client) DebitCards(ctx context.Context) ([]DebitCard, error) {
	var out struct {
		CurrentUser struct {
			DebitCards []DebitCard `json:"debitCards"`
		} `json:"currentUser"`
	}
	if err := c.Execute(ctx, queryDebitCards, nil, &out); err != nil {
		return nil, err
	}
	return out.CurrentUser.DebitCards, nil
}

// VirtualDebitCards fetches the user's virtual debit cards.
func (c *Client) VirtualDebitCards(ctx context.Context) ([]VirtualDebitCard, error) {
	var out struct {
		CurrentUser struct {
			VirtualDebitCards []VirtualDebitCard `json:"virtualDebitCards"`
		} `json:"currentUser"`
	}
	if err := c.Execute(ctx, queryVirtualDebitCards, nil, &out); err != nil {
		return nil, err
	}
	return out.CurrentUser.VirtualDebitCards, nil
}

const mutationFreezeDebitCard = `mutation FreezeDebitCard($input: FreezeDebitCardInput!) {
  freezeDebitCard(input: $input) {
    result { ` + debitCardFields + ` }
  }
}`

// FreezeDebitCard freezes a debit card, blocking new authorizations.
func (c *Client) FreezeDebitCard(ctx context.Context, cardID string) (*DebitCard, error) {
	return c.cardMutation(ctx, mutationFreezeDebitCard, "freezeDebitCard", cardID)
}

const mutationUnfreezeDebitCard = `mutation UnfreezeDebitCard($input: UnfreezeDebitCardInput!) {
  unfreezeDebitCard(input: $input) {
    result { ` + debitCardFields + ` }
  }
}`

// UnfreezeDebitCard unfreezes a previously frozen debit card.
func (c *Client) UnfreezeDebitCard(ctx context.Context, cardID string) (*DebitCard, error) {
	return c.cardMutation(ctx, mutationUnfreezeDebitCard, "unfreezeDebitCard", cardID)
}

const mutationCancelDebitCard = `mutation CancelDebitCard($input: CancelDebitCardInput!) {
  cancelDebitCard(input: $input) {
    result { ` + debitCardFields + ` }
  }
}`

// CancelDebitCard permanently cancels a debit card.
func (c *Client) CancelDebitCard(ctx context.Context, cardID string) (*DebitCard, error) {
	return c.cardMutation(ctx, mutationCancelDebitCard, "cancelDebitCard", cardID)
}

// cardMutation runs a single-card mutation whose payload wraps a result
// debit card under the given field name.
func (c *Client) cardMutation(ctx context.Context, mutation, field, cardID string) (*DebitCard, error) {
	var out map[string]struct {
		Result DebitCard `json:"result"`
	}
	input := map[string]any{"debitCardId": cardID}
	if err := c.Execute(ctx, mutation, map[string]any{"input": input}, &out); err != nil {
		return nil, err
	}
	payload := out[field]
	return &payload.Result, nil
}

// ActivateDebitCardsInput are the parameters for ActivateDebitCards.
type ActivateDebitCardsInput struct {
	DebitCardIDs []string `json:"debitCardIds"`
	Last4        string   `json:"last4,omitempty"`
}

const mutationActivateDebitCards = `mutation ActivateDebitCards($input: ActivateDebitCardsInput!) {
  activateDebitCards(input: $input) {
    result { ` + debitCardFields + ` }
  }
}`

// ActivateDebitCards activates newly received physical debit cards.
func (c *Client) ActivateDebitCards(ctx context.Context, in ActivateDebitCardsInput) ([]DebitCard, error) {
	var out struct {
		ActivateDebitCards struct {
			Result []DebitCard `json:"result"`
		} `json:"activateDebitCards"`
	}
	if err := c.Execute(ctx, mutationActivateDebitCards, map[string]any{"input": in}, &out); err != nil {
		return nil, err
	}
	return out.ActivateDebitCards.Result, nil
}

// CreateVirtualDebitCardInput are the parameters for CreateVirtualDebitCard.
type CreateVirtualDebitCardInput struct {
	Nickname     string `json:"nickname,omitempty"`
	SubaccountID string `json:"subaccountId,omitempty"`
}

const mutationCreateVirtualDebitCard = `mutation CreateVirtualDebitCard($input: CreateVirtualDebitCardInput!) {
  createVirtualDebitCard(input: $input) {
    result { ` + debitCardFields + ` }
  }
}`

// CreateVirtualDebitCard creates a new virtual debit card.
func (c *Client) CreateVirtualDebitCard(ctx context.Context, in CreateVirtualDebitCardInput) (*VirtualDebitCard, error) {
	var out struct {
		CreateVirtualDebitCard struct {
			Result VirtualDebitCard `json:"result"`
		} `json:"createVirtualDebitCard"`
	}
	if err := c.Execute(ctx, mutationCreateVirtualDebitCard, map[string]any{"input": in}, &out); err != nil {
		return nil, err
	}
	return &out.CreateVirtualDebitCard.Result, nil
}

// UpdateVirtualDebitCardInput are the parameters for UpdateVirtualDebitCard.
type UpdateVirtualDebitCardInput struct {
	VirtualDebitCardID string `json:"virtualDebitCardId"`
	Nickname           string `json:"nickname,omitempty"`
}

const mutationUpdateVirtualDebitCard = `mutation UpdateVirtualDebitCard($input: UpdateVirtualDebitCardInput!) {
  updateVirtualDebitCard(input: $input) {
    result { ` + debitCardFields + ` }
  }
}`

// UpdateVirtualDebitCard updates a virtual debit card's attributes.
func (c *Client) UpdateVirtualDebitCard(ctx context.Context, in UpdateVirtualDebitCardInput) (*VirtualDebitCard, error) {
	var out struct {
		UpdateVirtualDebitCard struct {
			Result VirtualDebitCard `json:"result"`
		} `json:"updateVirtualDebitCard"`
	}
	if err := c.Execute(ctx, mutationUpdateVirtualDebitCard, map[string]any{"input": in}, &out); err != nil {
		return nil, err
	}
	return &out.UpdateVirtualDebitCard.Result, nil
}
