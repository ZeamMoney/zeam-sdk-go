package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// HTTPDoer is the narrow subset of *http.Client that the auth flows need.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// OTPFlow runs the Business OTP login against the gateway's public
// `/v1/public/auth/*` surface. All responses from the gateway for these
// routes are SPEC §18 envelopes; OTPFlow unwraps them internally.
type OTPFlow struct {
	HTTP    HTTPDoer
	BaseURL *url.URL
}

// OTPRequest is the challenge phase of the Business flow.
type OTPRequest struct {
	MobileNumber string `json:"mobileNumber"`
}

// OTPChallenge is returned by [OTPFlow.RequestOTP].
type OTPChallenge struct {
	RequestID         string    `json:"requestId"`
	MaskedDestination string    `json:"maskedDestination"`
	ExpiresAt         time.Time `json:"expiresAt"`
}

// OTPVerify is the verification phase of the Business flow.
type OTPVerify struct {
	RequestID string `json:"requestId"`
	Code      string `json:"code"`
}

// RequestOTP starts the OTP flow. The returned [OTPChallenge] carries
// the opaque requestId the caller must echo back in [OTPFlow.VerifyOTP].
func (f *OTPFlow) RequestOTP(ctx context.Context, req OTPRequest) (*OTPChallenge, error) {
	if req.MobileNumber == "" {
		return nil, errors.New("auth: mobile number is required")
	}
	var out OTPChallenge
	if err := f.post(ctx, "/v1/public/auth/request-otp", req, &out); err != nil {
		return nil, err
	}
	if out.RequestID == "" {
		return nil, errors.New("auth: upstream did not return a requestId")
	}
	return &out, nil
}

// verifyOTPResponse mirrors the Business verify response. `expiresIn` is
// emitted as a numeric string by the upstream; we parse it defensively.
type verifyOTPResponse struct {
	IDToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"`
	CustomToken  string `json:"customToken"`
}

// VerifyOTP exchanges the OTP code for a session. The returned Session
// carries [TrackBusiness].
func (f *OTPFlow) VerifyOTP(ctx context.Context, v OTPVerify, subject string) (*Session, error) {
	if v.RequestID == "" || v.Code == "" {
		return nil, errors.New("auth: requestId and code are required")
	}
	var resp verifyOTPResponse
	if err := f.post(ctx, "/v1/public/auth/verify-otp", v, &resp); err != nil {
		return nil, err
	}
	expiresAt, err := expiryFromExpiresIn(resp.ExpiresIn)
	if err != nil {
		return nil, err
	}
	sess := NewSession(TrackBusiness, resp.IDToken, resp.RefreshToken, expiresAt, subject)
	return sess, nil
}

// Refresh exchanges the current refresh token for a new pair. Satisfies
// the [Refresher] interface so [AutoRefresher] can drive the Business
// track directly.
func (f *OTPFlow) Refresh(ctx context.Context, sess *Session) (*Session, error) {
	if sess == nil {
		return nil, errors.New("auth: nil session")
	}
	if sess.Track() != TrackBusiness {
		return nil, ErrWrongTrack
	}
	req := RefreshRequest{RefreshToken: sess.RefreshToken()}
	var resp verifyOTPResponse
	if err := f.post(ctx, "/v1/public/auth/refresh", req, &resp); err != nil {
		return nil, err
	}
	expiresAt, err := expiryFromExpiresIn(resp.ExpiresIn)
	if err != nil {
		return nil, err
	}
	sess.Update(resp.IDToken, resp.RefreshToken, expiresAt)
	return sess, nil
}

// post does a JSON POST and unwraps the SPEC §18 envelope.
func (f *OTPFlow) post(ctx context.Context, path string, body any, out any) error {
	u := *f.BaseURL
	u.Path = strings.TrimRight(u.Path, "/") + path

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("auth: marshal body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("auth: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := f.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("auth: %s: %w", path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("auth: read response: %w", err)
	}


	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var envelope struct {
			Data    json.RawMessage `json:"data"`
			Status  int             `json:"status"`
			Message string          `json:"message"`
		}
		if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Status >= 200 && envelope.Status < 300 && len(envelope.Data) > 0 {
			return json.Unmarshal(envelope.Data, out)
		}
		return json.Unmarshal(raw, out)
	}

	return fmt.Errorf("auth: %s returned status %d", path, resp.StatusCode)
}

// expiryFromExpiresIn parses `expiresIn` (seconds, supplied either as a
// number or as a numeric string — the upstream returns a string).
func expiryFromExpiresIn(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("auth: missing expiresIn")
	}
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("auth: parse expiresIn %q: %w", s, err)
	}
	if n <= 0 {
		return time.Time{}, errors.New("auth: non-positive expiresIn")
	}
	return time.Now().Add(time.Duration(n) * time.Second), nil
}
