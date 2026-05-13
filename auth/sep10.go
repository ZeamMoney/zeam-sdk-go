package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ZeamMoney/zeam-sdk-go/stellar"
)

// SEP10Flow runs the Connect SEP-10 authentication flow. Unlike the
// Business OTP surface, the gateway's `/auth-connect*` endpoints return
// BARE JSON (not the SPEC §18 envelope) — this flow decodes bare JSON
// accordingly.
type SEP10Flow struct {
	HTTP    HTTPDoer
	BaseURL *url.URL
	Signer  stellar.ChallengeSigner
}

// SEP10Challenge is the bare-JSON response of GET /auth-connect.
type SEP10Challenge struct {
	Transaction       string `json:"transaction"`
	NetworkPassphrase string `json:"network_passphrase"`
}

// SEP10TokenResponse is the bare-JSON response of POST /auth-connect.
type SEP10TokenResponse struct {
	IDToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"`
}

// GetChallenge fetches the SEP-10 challenge XDR for the given account.
func (f *SEP10Flow) GetChallenge(ctx context.Context, account string) (*SEP10Challenge, error) {
	if account == "" {
		return nil, errors.New("auth: account is required")
	}
	u := *f.BaseURL
	u.Path = strings.TrimRight(u.Path, "/") + "/v1/public/auth-connect"
	q := u.Query()
	q.Set("account", account)
	u.RawQuery = q.Encode()

	var out SEP10Challenge
	if err := f.getBareJSON(ctx, u.String(), &out); err != nil {
		return nil, err
	}
	if out.Transaction == "" {
		return nil, errors.New("auth: empty challenge XDR")
	}
	return &out, nil
}

// SubmitSigned submits a signed challenge XDR and returns the issued
// session (TrackConnect).
func (f *SEP10Flow) SubmitSigned(ctx context.Context, signedXDR, subject string) (*Session, error) {
	if signedXDR == "" {
		return nil, errors.New("auth: signed XDR is required")
	}
	u := *f.BaseURL
	u.Path = strings.TrimRight(u.Path, "/") + "/v1/public/auth-connect"

	body, err := json.Marshal(SEP10SubmitRequest{Transaction: signedXDR})
	if err != nil {
		return nil, fmt.Errorf("auth: marshal submit body: %w", err)
	}
	var resp SEP10TokenResponse
	if err := f.postBareJSON(ctx, u.String(), body, &resp); err != nil {
		return nil, err
	}
	expiresAt, err := expiryFromExpiresIn(resp.ExpiresIn)
	if err != nil {
		return nil, err
	}
	return NewSession(TrackConnect, resp.IDToken, resp.RefreshToken, expiresAt, subject), nil
}

// Login performs the full SEP-10 round trip using the supplied keypair:
// fetch challenge, sign locally, submit. Used by the ConnectLogin recipe.
func (f *SEP10Flow) Login(ctx context.Context, kp *stellar.Keypair) (*Session, error) {
	if f.Signer == nil {
		return nil, errors.New("auth: SEP10 signer is not configured")
	}
	if !kp.CanSign() {
		return nil, errors.New("auth: keypair cannot sign (no seed)")
	}
	challenge, err := f.GetChallenge(ctx, kp.PublicKey())
	if err != nil {
		return nil, err
	}
	signed, err := f.Signer.Sign(challenge.Transaction, kp)
	if err != nil {
		return nil, fmt.Errorf("auth: sign challenge: %w", err)
	}
	return f.SubmitSigned(ctx, signed, kp.PublicKey())
}

// Refresh exchanges the current Connect refresh token for a new pair.
// Satisfies [Refresher].
func (f *SEP10Flow) Refresh(ctx context.Context, sess *Session) (*Session, error) {
	if sess == nil {
		return nil, errors.New("auth: nil session")
	}
	if sess.Track() != TrackConnect {
		return nil, ErrWrongTrack
	}
	u := *f.BaseURL
	u.Path = strings.TrimRight(u.Path, "/") + "/v1/public/auth-connect/refresh"

	body, err := json.Marshal(RefreshRequest{RefreshToken: sess.RefreshToken()})
	if err != nil {
		return nil, fmt.Errorf("auth: marshal refresh body: %w", err)
	}
	var resp SEP10TokenResponse
	if err := f.postBareJSON(ctx, u.String(), body, &resp); err != nil {
		return nil, err
	}
	expiresAt, err := expiryFromExpiresIn(resp.ExpiresIn)
	if err != nil {
		return nil, err
	}
	sess.Update(resp.IDToken, resp.RefreshToken, expiresAt)
	return sess, nil
}

func (f *SEP10Flow) getBareJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("auth: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	return f.doAndDecode(req, out)
}

func (f *SEP10Flow) postBareJSON(ctx context.Context, url string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("auth: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return f.doAndDecode(req, out)
}

func (f *SEP10Flow) doAndDecode(req *http.Request, out any) error {
	resp, err := f.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("auth: %s %s: %w", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("auth: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("auth: %s %s status %d", req.Method, req.URL.Path, resp.StatusCode)
	}
	return json.Unmarshal(raw, out)
}
