package crew

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ── Test helpers ─────────────────────────────────────────────────────────

type capturedReq struct {
	Method    string
	Path      string
	Auth      string
	Query     string
	Variables map[string]any
}

// fakeServer fakes the Crew auth REST endpoints and GraphQL endpoint.
// GraphQL responses are selected by matching each key of gqlResponses as a
// substring of the request's query plus marshaled variables.
type fakeServer struct {
	t   *testing.T
	srv *httptest.Server

	mu           sync.Mutex
	gqlResponses map[string]string
	gqlStatus    int    // when non-zero, force this HTTP status on /graphql
	rotateToken  string // when set, sent as the Authorization response header
	requests     []capturedReq
}

func (f *fakeServer) setGQL(key, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gqlResponses[key] = body
}

func (f *fakeServer) setGQLStatus(code int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gqlStatus = code
}

func (f *fakeServer) setRotateToken(tok string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rotateToken = tok
}

func (f *fakeServer) lastRequest() capturedReq {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		f.t.Fatal("no requests captured")
	}
	return f.requests[len(f.requests)-1]
}

func (f *fakeServer) record(r *http.Request, query string, vars map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, capturedReq{
		Method:    r.Method,
		Path:      r.URL.Path,
		Auth:      r.Header.Get("Authorization"),
		Query:     query,
		Variables: vars,
	})
}

func (f *fakeServer) maybeRotate(w http.ResponseWriter) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.rotateToken != "" {
		w.Header().Set("Authorization", f.rotateToken)
	}
}

// newTestServer starts a fake Crew server and returns a client wired to it
// with a token already set.
func newTestServer(t *testing.T, opts ...Option) (*Client, *fakeServer) {
	t.Helper()
	f := &fakeServer{t: t, gqlResponses: make(map[string]string)}

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/send_sms_otp", func(w http.ResponseWriter, r *http.Request) {
		f.record(r, "", nil)
		f.maybeRotate(w)
		writeJSON(w, `{"phone_id":"phone-number-live-abc"}`)
	})
	mux.HandleFunc("/auth/auth_sms_otp", func(w http.ResponseWriter, r *http.Request) {
		f.record(r, "", nil)
		var body struct {
			OTP string `json:"otp"`
		}
		decodeJSON(t, r, &body)
		if body.OTP == "bad" {
			http.Error(w, `{"error":"invalid otp"}`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Authorization", "token-after-sms")
		writeJSON(w, `{"email":"b***@example.com","isSingleFactor":false}`)
	})
	mux.HandleFunc("/auth/send_email_otp", func(w http.ResponseWriter, r *http.Request) {
		f.record(r, "", nil)
		if r.Header.Get("Authorization") == "" {
			http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Authorization", "token-after-email-send")
		writeJSON(w, `{"email_id":"email-live-def"}`)
	})
	mux.HandleFunc("/auth/auth_email_otp", func(w http.ResponseWriter, r *http.Request) {
		f.record(r, "", nil)
		w.Header().Set("Authorization", "Bearer token-final")
		writeJSON(w, `{}`)
	})
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		var req gqlRequest
		decodeJSON(t, r, &req)
		f.record(r, req.Query, req.Variables)
		f.maybeRotate(w)

		f.mu.Lock()
		status := f.gqlStatus
		vars, _ := json.Marshal(req.Variables)
		// A key matching the variables (e.g. a cursor like `"after":"c1"`)
		// beats one matching only the query text; longest match wins within
		// each tier, keeping selection deterministic.
		var response, matched string
		for _, haystack := range []string{string(vars), req.Query} {
			for key, body := range f.gqlResponses {
				if strings.Contains(haystack, key) && len(key) > len(matched) {
					matched, response = key, body
				}
			}
			if response != "" {
				break
			}
		}
		f.mu.Unlock()

		if status != 0 {
			http.Error(w, `{"error":"forced status"}`, status)
			return
		}
		if response == "" {
			t.Errorf("no fake response matches query: %s", req.Query)
			http.Error(w, `{"error":"no fake response"}`, http.StatusInternalServerError)
			return
		}
		writeJSON(w, response)
	})

	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)

	base := []Option{
		WithToken("test-token"),
		WithAPIURL(f.srv.URL + "/graphql"),
		WithAuthURL(f.srv.URL + "/auth"),
	}
	return NewClient(append(base, opts...)...), f
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

func decodeJSON(t *testing.T, r *http.Request, out any) {
	t.Helper()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		t.Errorf("decode request body: %v", err)
	}
}

// ── Client tests ─────────────────────────────────────────────────────────

func TestNewClientDefaults(t *testing.T) {
	c := NewClient()
	if c.apiURL != defaultAPIURL {
		t.Errorf("apiURL = %q, want %q", c.apiURL, defaultAPIURL)
	}
	if c.authURL != defaultAuthURL {
		t.Errorf("authURL = %q, want %q", c.authURL, defaultAuthURL)
	}
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("http timeout = %v, want 30s", c.httpClient.Timeout)
	}
	if c.watchInterval != 60*time.Second {
		t.Errorf("watchInterval = %v, want 60s", c.watchInterval)
	}
	if c.watchPageSize != 50 {
		t.Errorf("watchPageSize = %d, want 50", c.watchPageSize)
	}
	if c.token != "" {
		t.Errorf("token = %q, want empty", c.token)
	}
	if _, ok := c.logger.(nopLogger); !ok {
		t.Errorf("logger = %T, want nopLogger", c.logger)
	}
}

func TestWithHTTPClient(t *testing.T) {
	hc := &http.Client{Timeout: time.Second}
	c := NewClient(WithHTTPClient(hc))
	if c.httpClient != hc {
		t.Error("WithHTTPClient not applied")
	}
}

func TestWithToken(t *testing.T) {
	c := NewClient(WithToken("abc"))
	if got := c.Token(); got != "abc" {
		t.Errorf("Token() = %q, want %q", got, "abc")
	}
}

func TestWithWatchOptions(t *testing.T) {
	filter := CashTransactionFilter{SubaccountID: "sub-1"}
	c := NewClient(
		WithWatchInterval(5*time.Second),
		WithWatchPageSize(7),
		WithWatchFilter(filter),
	)
	if c.watchInterval != 5*time.Second {
		t.Errorf("watchInterval = %v, want 5s", c.watchInterval)
	}
	if c.watchPageSize != 7 {
		t.Errorf("watchPageSize = %d, want 7", c.watchPageSize)
	}
	if c.watchFilter == nil || c.watchFilter.SubaccountID != "sub-1" {
		t.Errorf("watchFilter = %+v, want SubaccountID sub-1", c.watchFilter)
	}
}

type testLogger struct {
	mu     sync.Mutex
	debugs int
	infos  int
	errors int
}

func (l *testLogger) Debug(string, ...any) { l.mu.Lock(); l.debugs++; l.mu.Unlock() }
func (l *testLogger) Info(string, ...any)  { l.mu.Lock(); l.infos++; l.mu.Unlock() }
func (l *testLogger) Error(string, ...any) { l.mu.Lock(); l.errors++; l.mu.Unlock() }

func TestWithLogger(t *testing.T) {
	l := &testLogger{}
	c := NewClient(WithLogger(l))
	if c.logger != Logger(l) {
		t.Error("WithLogger not applied")
	}
}

func TestSetTokenDoesNotFireCallback(t *testing.T) {
	fired := false
	c := NewClient(WithTokenCallback(func(string) { fired = true }))
	c.SetToken("abc")
	if fired {
		t.Error("SetToken fired the token callback")
	}
	if got := c.Token(); got != "abc" {
		t.Errorf("Token() = %q, want %q", got, "abc")
	}
}

func TestCaptureTokenStripsBearerPrefix(t *testing.T) {
	c := NewClient()
	c.captureToken("Bearer abc123")
	if got := c.Token(); got != "abc123" {
		t.Errorf("Token() = %q, want %q", got, "abc123")
	}
}

func TestCaptureTokenCallbackOnlyOnChange(t *testing.T) {
	var got []string
	c := NewClient(WithTokenCallback(func(tok string) { got = append(got, tok) }))
	c.captureToken("tok-1")
	c.captureToken("tok-1")
	c.captureToken("tok-2")
	c.captureToken("")
	if len(got) != 2 || got[0] != "tok-1" || got[1] != "tok-2" {
		t.Errorf("callback tokens = %v, want [tok-1 tok-2]", got)
	}
}
