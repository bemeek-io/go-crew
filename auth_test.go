package crew

import (
	"context"
	"errors"
	"testing"
)

func TestSendSMSOTPReturnsPhoneID(t *testing.T) {
	c, f := newTestServer(t)
	c.SetToken("")

	phoneID, err := c.SendSMSOTP(context.Background(), "5555555555")
	if err != nil {
		t.Fatalf("SendSMSOTP: %v", err)
	}
	if phoneID != "phone-number-live-abc" {
		t.Errorf("phoneID = %q, want phone-number-live-abc", phoneID)
	}
	req := f.lastRequest()
	if req.Path != "/auth/send_sms_otp" {
		t.Errorf("path = %q, want /auth/send_sms_otp", req.Path)
	}
	if req.Auth != "" {
		t.Errorf("Authorization = %q, want empty before login", req.Auth)
	}
}

func TestAuthSMSOTPStoresToken(t *testing.T) {
	c, _ := newTestServer(t)
	c.SetToken("")

	res, err := c.AuthSMSOTP(context.Background(), "phone-number-live-abc", "123456")
	if err != nil {
		t.Fatalf("AuthSMSOTP: %v", err)
	}
	if res.Email != "b***@example.com" {
		t.Errorf("Email = %q, want b***@example.com", res.Email)
	}
	if res.SingleFactor {
		t.Error("SingleFactor = true, want false")
	}
	if got := c.Token(); got != "token-after-sms" {
		t.Errorf("Token() = %q, want token-after-sms", got)
	}
}

func TestAuthSMSOTPBadCodeReturnsUnauthorized(t *testing.T) {
	c, _ := newTestServer(t)
	c.SetToken("")

	_, err := c.AuthSMSOTP(context.Background(), "phone-number-live-abc", "bad")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestSendEmailOTPSendsBearer(t *testing.T) {
	c, f := newTestServer(t)
	c.SetToken("token-after-sms")

	emailID, err := c.SendEmailOTP(context.Background(), "ben@example.com")
	if err != nil {
		t.Fatalf("SendEmailOTP: %v", err)
	}
	if emailID != "email-live-def" {
		t.Errorf("emailID = %q, want email-live-def", emailID)
	}
	if got := f.lastRequest().Auth; got != "Bearer token-after-sms" {
		t.Errorf("Authorization = %q, want Bearer token-after-sms", got)
	}
	if got := c.Token(); got != "token-after-email-send" {
		t.Errorf("Token() = %q, want rotated token-after-email-send", got)
	}
}

func TestAuthEmailOTPStoresFinalToken(t *testing.T) {
	tokens := []string{}
	c, _ := newTestServer(t, WithTokenCallback(func(tok string) { tokens = append(tokens, tok) }))
	c.SetToken("token-after-email-send")

	if err := c.AuthEmailOTP(context.Background(), "email-live-def", "654321"); err != nil {
		t.Fatalf("AuthEmailOTP: %v", err)
	}
	// The fake sends "Bearer token-final"; the prefix must be stripped.
	if got := c.Token(); got != "token-final" {
		t.Errorf("Token() = %q, want token-final", got)
	}
	if len(tokens) != 1 || tokens[0] != "token-final" {
		t.Errorf("callback tokens = %v, want [token-final]", tokens)
	}
}

func TestFullLoginFlow(t *testing.T) {
	c, _ := newTestServer(t)
	c.SetToken("")
	ctx := context.Background()

	phoneID, err := c.SendSMSOTP(ctx, "5555555555")
	if err != nil {
		t.Fatalf("SendSMSOTP: %v", err)
	}
	if _, err := c.AuthSMSOTP(ctx, phoneID, "111111"); err != nil {
		t.Fatalf("AuthSMSOTP: %v", err)
	}
	emailID, err := c.SendEmailOTP(ctx, "ben@example.com")
	if err != nil {
		t.Fatalf("SendEmailOTP: %v", err)
	}
	if err := c.AuthEmailOTP(ctx, emailID, "222222"); err != nil {
		t.Fatalf("AuthEmailOTP: %v", err)
	}
	if got := c.Token(); got != "token-final" {
		t.Errorf("Token() = %q, want token-final", got)
	}
}
