package crew

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrUnauthorized is returned when the API rejects the auth token.
	// The token has expired or was revoked; the user must log in again.
	ErrUnauthorized = errors.New("crew: unauthorized")

	// ErrNoToken is returned when an operation requires an auth token
	// but none has been set.
	ErrNoToken = errors.New("crew: no auth token set")

	// ErrAlreadyWatching is returned by StartWatching when the watcher
	// is already running.
	ErrAlreadyWatching = errors.New("crew: watcher already started")

	// ErrNotWatching is returned when an operation requires a running
	// watcher.
	ErrNotWatching = errors.New("crew: watcher not started")

	// ErrBackwardPagination is returned when only one of Last and Before is
	// set. Crew answers a half-specified backward page with HTTP 500 rather
	// than a GraphQL error, so the SDK rejects it before sending.
	ErrBackwardPagination = errors.New("crew: backward pagination requires both Last and Before")
)

// APIError wraps a non-2xx HTTP response from the Crew API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("crew: API error %d: %s", e.StatusCode, e.Body)
}

// GraphQLError is a single error from a GraphQL response.
type GraphQLError struct {
	Message string `json:"message"`
	Path    []any  `json:"path,omitempty"`
}

// GraphQLErrors is the list of errors from a GraphQL response.
type GraphQLErrors []GraphQLError

func (e GraphQLErrors) Error() string {
	msgs := make([]string, len(e))
	for i, err := range e {
		msgs[i] = err.Message
	}
	return "crew: graphql: " + strings.Join(msgs, "; ")
}
