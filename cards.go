package crew

import (
	"context"
	"time"
)

const debitCardFields = `id name lastFour status formFactor frozenStatus frozenReason color monthlyLimit monthlySpendToDate`

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

// VirtualDebitCards fetches the user's virtual debit cards. Crew has no
// distinct virtual card type — these are DebitCards whose FormFactor is
// VIRTUAL or SINGLE_USE.
func (c *Client) VirtualDebitCards(ctx context.Context) ([]DebitCard, error) {
	var out struct {
		CurrentUser struct {
			VirtualDebitCards []DebitCard `json:"virtualDebitCards"`
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

// FreezeDebitCard freezes a debit card, blocking new authorizations. Crew
// requires a reason; pass CardFrozenReasonUserRequested for an ordinary
// user-initiated freeze.
func (c *Client) FreezeDebitCard(ctx context.Context, cardID string, reason CardFrozenReason) (*DebitCard, error) {
	input := map[string]any{"debitCardId": cardID, "reason": reason}
	return c.cardMutation(ctx, mutationFreezeDebitCard, "freezeDebitCard", input)
}

const mutationUnfreezeDebitCard = `mutation UnfreezeDebitCard($input: UnfreezeDebitCardInput!) {
  unfreezeDebitCard(input: $input) {
    result { ` + debitCardFields + ` }
  }
}`

// UnfreezeDebitCard unfreezes a previously frozen debit card.
func (c *Client) UnfreezeDebitCard(ctx context.Context, cardID string) (*DebitCard, error) {
	input := map[string]any{"debitCardId": cardID}
	return c.cardMutation(ctx, mutationUnfreezeDebitCard, "unfreezeDebitCard", input)
}

const mutationCancelDebitCard = `mutation CancelDebitCard($input: CancelDebitCardInput!) {
  cancelDebitCard(input: $input) {
    result { ` + debitCardFields + ` }
  }
}`

// CancelDebitCard permanently cancels a debit card.
func (c *Client) CancelDebitCard(ctx context.Context, cardID string) (*DebitCard, error) {
	input := map[string]any{"debitCardId": cardID}
	return c.cardMutation(ctx, mutationCancelDebitCard, "cancelDebitCard", input)
}

// cardMutation runs a single-card mutation whose payload wraps a result
// debit card under the given field name.
func (c *Client) cardMutation(ctx context.Context, mutation, field string, input map[string]any) (*DebitCard, error) {
	var out map[string]struct {
		Result DebitCard `json:"result"`
	}
	if err := c.Execute(ctx, mutation, map[string]any{"input": input}, &out); err != nil {
		return nil, err
	}
	payload := out[field]
	return &payload.Result, nil
}

const mutationActivateDebitCards = `mutation ActivateDebitCards($input: ActivateDebitCardsInput!) {
  activateDebitCards(input: $input) {
    result { ` + debitCardFields + ` }
  }
}`

// ActivateDebitCards activates newly received physical debit cards.
func (c *Client) ActivateDebitCards(ctx context.Context, cardIDs []string) ([]DebitCard, error) {
	var out struct {
		ActivateDebitCards struct {
			Result []DebitCard `json:"result"`
		} `json:"activateDebitCards"`
	}
	// The schema names this list field debitCardId, singular.
	input := map[string]any{"debitCardId": cardIDs}
	if err := c.Execute(ctx, mutationActivateDebitCards, map[string]any{"input": input}, &out); err != nil {
		return nil, err
	}
	return out.ActivateDebitCards.Result, nil
}

// CreateVirtualDebitCardInput are the parameters for CreateVirtualDebitCard.
// UserID is required and identifies the card's owner — use CurrentUser to
// get your own ID.
type CreateVirtualDebitCardInput struct {
	UserID       string `json:"userId"`
	Name         string `json:"name,omitempty"`
	SubaccountID string `json:"subaccountId,omitempty"`
	// MonthlyLimitCents caps spend per calendar month, or nil for no cap.
	MonthlyLimitCents *int64                `json:"monthlyLimit,omitempty"`
	CardColor         DebitCardColor        `json:"cardColor,omitempty"`
	FormFactor        VirtualCardFormFactor `json:"formFactor,omitempty"`
	// CancelAfter auto-cancels the card at the given time, or nil to leave
	// it open-ended.
	CancelAfter *time.Time `json:"cancelAfter,omitempty"`
}

const mutationCreateVirtualDebitCard = `mutation CreateVirtualDebitCard($input: CreateVirtualDebitCardInput!) {
  createVirtualDebitCard(input: $input) {
    result { ` + debitCardFields + ` }
  }
}`

// CreateVirtualDebitCard creates a new virtual debit card.
func (c *Client) CreateVirtualDebitCard(ctx context.Context, in CreateVirtualDebitCardInput) (*DebitCard, error) {
	var out struct {
		CreateVirtualDebitCard struct {
			Result DebitCard `json:"result"`
		} `json:"createVirtualDebitCard"`
	}
	if err := c.Execute(ctx, mutationCreateVirtualDebitCard, map[string]any{"input": in}, &out); err != nil {
		return nil, err
	}
	return &out.CreateVirtualDebitCard.Result, nil
}

// UpdateVirtualDebitCardInput are the parameters for UpdateVirtualDebitCard.
type UpdateVirtualDebitCardInput struct {
	DebitCardID  string `json:"debitCardId"`
	Name         string `json:"name,omitempty"`
	SubaccountID string `json:"subaccountId,omitempty"`
	// MonthlyLimitCents caps spend per calendar month, or nil to leave the
	// existing limit unchanged.
	MonthlyLimitCents *int64         `json:"monthlyLimit,omitempty"`
	CardColor         DebitCardColor `json:"cardColor,omitempty"`
	// CancelAfter auto-cancels the card at the given time, or nil to leave
	// the existing setting unchanged.
	CancelAfter *time.Time `json:"cancelAfter,omitempty"`
}

const mutationUpdateVirtualDebitCard = `mutation UpdateVirtualDebitCard($input: UpdateVirtualDebitCardInput!) {
  updateVirtualDebitCard(input: $input) {
    result { ` + debitCardFields + ` }
  }
}`

// UpdateVirtualDebitCard updates a virtual debit card's attributes.
func (c *Client) UpdateVirtualDebitCard(ctx context.Context, in UpdateVirtualDebitCardInput) (*DebitCard, error) {
	var out struct {
		UpdateVirtualDebitCard struct {
			Result DebitCard `json:"result"`
		} `json:"updateVirtualDebitCard"`
	}
	if err := c.Execute(ctx, mutationUpdateVirtualDebitCard, map[string]any{"input": in}, &out); err != nil {
		return nil, err
	}
	return &out.UpdateVirtualDebitCard.Result, nil
}
