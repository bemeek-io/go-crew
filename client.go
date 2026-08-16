package crew

import (
	"net/http"
	"sync"
	"time"
)

const (
	defaultAPIURL  = "https://api.trycrew.com/willow/graphql"
	defaultAuthURL = "https://api.trycrew.com/willow/auth"
	userAgent      = "Crew/1 CFNetwork/3860.300.31 Darwin/25.2.0"
)

// Logger is a pluggable logging interface. Users can adapt slog, zap, etc.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// nopLogger is the default logger that discards all output.
type nopLogger struct{}

func (nopLogger) Debug(string, ...any) {}
func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}

// TransactionHandler is called by the watcher with a new or updated
// transaction.
type TransactionHandler func(tx CashTransaction)

// Client is the Crew Finance API client.
type Client struct {
	httpClient *http.Client
	logger     Logger

	mu      sync.RWMutex
	token   string
	onToken func(token string)

	watchInterval    time.Duration
	watchPageSize    int
	watchFilter      *CashTransactionFilter
	txHandlers       []TransactionHandler
	txUpdateHandlers []TransactionHandler
	errHandlers      []func(error)
	watching         bool
	watchStop        chan struct{}
	watchDone        chan struct{}

	// Internal overrides for testing.
	apiURL  string
	authURL string
}

// Option configures the Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client for API requests.
// The default has a 30-second timeout.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithToken resumes a previously saved session with a bearer token,
// skipping the OTP login flow. Default is empty (log in first).
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

// WithTokenCallback registers a function called whenever the server rotates
// the bearer token. Use it to persist the current token so a session can be
// resumed later with WithToken. Default is nil.
func WithTokenCallback(fn func(token string)) Option {
	return func(c *Client) {
		c.onToken = fn
	}
}

// WithLogger sets a structured logger for the client.
// The default discards all logs.
func WithLogger(l Logger) Option {
	return func(c *Client) {
		c.logger = l
	}
}

// WithAPIURL overrides the GraphQL endpoint URL.
// Default is https://api.trycrew.com/willow/graphql.
func WithAPIURL(u string) Option {
	return func(c *Client) {
		c.apiURL = u
	}
}

// WithAuthURL overrides the base URL for the auth REST endpoints.
// Default is https://api.trycrew.com/willow/auth.
func WithAuthURL(u string) Option {
	return func(c *Client) {
		c.authURL = u
	}
}

// WithWatchInterval sets the watcher poll interval. Default is 60 seconds.
func WithWatchInterval(d time.Duration) Option {
	return func(c *Client) {
		c.watchInterval = d
	}
}

// WithWatchPageSize sets how many transactions the watcher fetches per poll.
// Default is 50.
func WithWatchPageSize(n int) Option {
	return func(c *Client) {
		c.watchPageSize = n
	}
}

// WithWatchFilter sets a server-side filter for the transactions the watcher
// polls. Default is none (all transactions).
func WithWatchFilter(f CashTransactionFilter) Option {
	return func(c *Client) {
		c.watchFilter = &f
	}
}

// NewClient creates a new Crew client. It performs no network IO; log in
// with the OTP methods or resume a session with WithToken.
func NewClient(opts ...Option) *Client {
	c := &Client{
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		logger:        nopLogger{},
		watchInterval: 60 * time.Second,
		watchPageSize: 50,
		apiURL:        defaultAPIURL,
		authURL:       defaultAuthURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Token returns the current bearer token, or an empty string if the client
// has not authenticated.
func (c *Client) Token() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// SetToken replaces the current bearer token. Unlike server-side rotation,
// it does not invoke the WithTokenCallback function.
func (c *Client) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

// captureToken stores a token rotated by the server and notifies the
// WithTokenCallback function. The Authorization response header may or may
// not carry a "Bearer " prefix; strip it either way.
func (c *Client) captureToken(header string) {
	if header == "" {
		return
	}
	token := header
	if len(token) > 7 && (token[:7] == "Bearer " || token[:7] == "bearer ") {
		token = token[7:]
	}
	c.mu.Lock()
	changed := token != c.token
	if changed {
		c.token = token
	}
	onToken := c.onToken
	c.mu.Unlock()
	if changed {
		c.logger.Debug("crew: token rotated")
		if onToken != nil {
			onToken(token)
		}
	}
}
