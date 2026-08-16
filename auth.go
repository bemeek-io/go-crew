package crew

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SMSAuthResult is the outcome of AuthSMSOTP.
type SMSAuthResult struct {
	// Email is the (partially masked) email address on the account, to be
	// confirmed in the email OTP steps.
	Email string
	// SingleFactor reports whether the account only requires SMS
	// verification; when true, the email OTP steps can be skipped.
	SingleFactor bool
}

type smsOTPRequest struct {
	Phone string `json:"phone"`
}

type smsOTPResponse struct {
	PhoneID string `json:"phone_id"`
}

type smsAuthRequest struct {
	OTP     string `json:"otp"`
	PhoneID string `json:"phone_id"`
}

type smsAuthResponse struct {
	Email          string `json:"email"`
	IsSingleFactor bool   `json:"isSingleFactor"`
}

type emailOTPRequest struct {
	Email string `json:"email"`
}

type emailOTPResponse struct {
	EmailID string `json:"email_id"`
}

type emailAuthRequest struct {
	OTP     string `json:"otp"`
	EmailID string `json:"email_id"`
}

// SendSMSOTP starts the login flow (step 1 of 4) by sending a one-time
// passcode to the given phone number (digits only, e.g. "5555555555").
// The returned phone ID is required by AuthSMSOTP.
func (c *Client) SendSMSOTP(ctx context.Context, phone string) (string, error) {
	var out smsOTPResponse
	if err := c.postAuth(ctx, "/send_sms_otp", smsOTPRequest{Phone: phone}, &out); err != nil {
		return "", err
	}
	return out.PhoneID, nil
}

// AuthSMSOTP verifies the SMS passcode (step 2 of 4) and stores the bearer
// token returned by the server. If the result reports SingleFactor, the
// email steps can be skipped and the client is ready for API calls.
func (c *Client) AuthSMSOTP(ctx context.Context, phoneID, otp string) (*SMSAuthResult, error) {
	var out smsAuthResponse
	if err := c.postAuth(ctx, "/auth_sms_otp", smsAuthRequest{OTP: otp, PhoneID: phoneID}, &out); err != nil {
		return nil, err
	}
	return &SMSAuthResult{Email: out.Email, SingleFactor: out.IsSingleFactor}, nil
}

// SendEmailOTP sends a one-time passcode to the account email (step 3 of 4).
// It requires the token stored by AuthSMSOTP. The returned email ID is
// required by AuthEmailOTP.
func (c *Client) SendEmailOTP(ctx context.Context, email string) (string, error) {
	var out emailOTPResponse
	if err := c.postAuth(ctx, "/send_email_otp", emailOTPRequest{Email: email}, &out); err != nil {
		return "", err
	}
	return out.EmailID, nil
}

// AuthEmailOTP verifies the email passcode (step 4 of 4) and stores the
// final bearer token. The client is then ready for API calls.
func (c *Client) AuthEmailOTP(ctx context.Context, emailID, otp string) error {
	return c.postAuth(ctx, "/auth_email_otp", emailAuthRequest{OTP: otp, EmailID: emailID}, nil)
}

// postAuth POSTs a JSON body to an auth endpoint, decodes the JSON response
// into out (which may be nil), and captures any rotated bearer token from
// the Authorization response header.
func (c *Client) postAuth(ctx context.Context, path string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("crew: marshal auth request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("crew: create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", userAgent)
	if token := c.Token(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("crew: auth request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	c.captureToken(resp.Header.Get("Authorization"))

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("crew: auth: %w", ErrUnauthorized)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return fmt.Errorf("crew: auth: %w", &APIError{StatusCode: resp.StatusCode, Body: string(respBody)})
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("crew: decode auth response: %w", err)
		}
	}
	return nil
}
