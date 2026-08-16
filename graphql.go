package crew

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// maxErrorBody caps how much of an error response body is retained.
const maxErrorBody = 8 << 10

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors GraphQLErrors   `json:"errors"`
}

// Execute runs a raw GraphQL query or mutation against the Crew API and
// unmarshals the response's "data" object into out (which may be nil to
// discard it). It is the escape hatch for operations the typed methods do
// not cover.
func (c *Client) Execute(ctx context.Context, query string, variables map[string]any, out any) error {
	if c.Token() == "" {
		return fmt.Errorf("crew: graphql: %w", ErrNoToken)
	}

	payload, err := json.Marshal(gqlRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("crew: marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("crew: create graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Authorization", "Bearer "+c.Token())
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("crew: graphql request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	c.captureToken(resp.Header.Get("Authorization"))

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("crew: graphql: %w", ErrUnauthorized)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return fmt.Errorf("crew: graphql: %w", &APIError{StatusCode: resp.StatusCode, Body: string(body)})
	}

	var gr gqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return fmt.Errorf("crew: decode graphql response: %w", err)
	}
	if len(gr.Errors) > 0 {
		return gr.Errors
	}
	if out != nil {
		if err := json.Unmarshal(gr.Data, out); err != nil {
			return fmt.Errorf("crew: decode graphql data: %w", err)
		}
	}
	return nil
}
