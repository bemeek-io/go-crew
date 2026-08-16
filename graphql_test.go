package crew

import (
	"context"
	"errors"
	"testing"
)

func TestExecuteSendsQueryAndVariables(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("hello", `{"data":{"hello":"world"}}`)

	var out struct {
		Hello string `json:"hello"`
	}
	err := c.Execute(context.Background(), "query { hello }", map[string]any{"a": float64(1)}, &out)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Hello != "world" {
		t.Errorf("out.Hello = %q, want world", out.Hello)
	}
	req := f.lastRequest()
	if req.Query != "query { hello }" {
		t.Errorf("query = %q", req.Query)
	}
	if req.Variables["a"] != float64(1) {
		t.Errorf("variables = %v, want a=1", req.Variables)
	}
	if req.Auth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want Bearer test-token", req.Auth)
	}
}

func TestExecuteNilOutDiscardsData(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("hello", `{"data":{"hello":"world"}}`)

	if err := c.Execute(context.Background(), "query { hello }", nil, nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestExecuteNoTokenReturnsErrNoToken(t *testing.T) {
	c, _ := newTestServer(t)
	c.SetToken("")

	err := c.Execute(context.Background(), "query { hello }", nil, nil)
	if !errors.Is(err, ErrNoToken) {
		t.Fatalf("err = %v, want ErrNoToken", err)
	}
}

func TestExecuteGraphQLErrors(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQL("hello", `{"data":null,"errors":[{"message":"boom","path":["currentUser"]},{"message":"bang"}]}`)

	err := c.Execute(context.Background(), "query { hello }", nil, nil)
	var gqlErrs GraphQLErrors
	if !errors.As(err, &gqlErrs) {
		t.Fatalf("err = %T (%v), want GraphQLErrors", err, err)
	}
	if len(gqlErrs) != 2 || gqlErrs[0].Message != "boom" {
		t.Errorf("errors = %+v", gqlErrs)
	}
	want := "crew: graphql: boom; bang"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestExecuteHTTP500ReturnsAPIError(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQLStatus(500)

	err := c.Execute(context.Background(), "query { hello }", nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T (%v), want *APIError", err, err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}

func TestExecute401ReturnsErrUnauthorized(t *testing.T) {
	c, f := newTestServer(t)
	f.setGQLStatus(401)

	err := c.Execute(context.Background(), "query { hello }", nil, nil)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestExecuteCapturesRotatedToken(t *testing.T) {
	var rotated []string
	c, f := newTestServer(t, WithTokenCallback(func(tok string) { rotated = append(rotated, tok) }))
	f.setGQL("hello", `{"data":{"hello":"world"}}`)
	f.setRotateToken("fresh-token")

	if err := c.Execute(context.Background(), "query { hello }", nil, nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got := c.Token(); got != "fresh-token" {
		t.Errorf("Token() = %q, want fresh-token", got)
	}
	if len(rotated) != 1 || rotated[0] != "fresh-token" {
		t.Errorf("callback tokens = %v, want [fresh-token]", rotated)
	}
}

func TestAPIErrorString(t *testing.T) {
	err := &APIError{StatusCode: 503, Body: "unavailable"}
	want := "crew: API error 503: unavailable"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}
