package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/transport"
)

// Doer is the interface each client subpackage consumes. It is satisfied
// by *zeam.Client (via an adapter in the top-level package) and by the
// httptest-based fakes in test/fake/.
type Doer interface {
	BaseURL() *url.URL
	HTTPClient() *http.Client
	VerboseErrors() bool
}

// Call performs an HTTP call against path, attaching auth headers for the
// supplied session (if any), JSON-marshalling body, decoding the SPEC §18
// envelope, and writing the inner `data` into out. `connectSecret` is the
// x-zeam-auth header value for Connect endpoints; leave empty otherwise.
func Call(
	ctx context.Context,
	d Doer,
	method, path string,
	query url.Values,
	session *auth.Session,
	requireTrack auth.Track,
	connectSecret string,
	body any,
	out any,
) error {
	if session != nil && requireTrack != auth.TrackUnknown && session.Track() != requireTrack {
		return auth.ErrWrongTrack
	}
	base := d.BaseURL()
	if base == nil {
		return errors.New("client: nil base URL")
	}
	u := *base
	u.Path = strings.TrimRight(u.Path, "/") + path
	if query != nil {
		u.RawQuery = query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("client: marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("client: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if session != nil {
		req.Header.Set("Authorization", "Bearer "+session.IDToken())
	}
	if connectSecret != "" {
		req.Header.Set("x-zeam-auth", connectSecret)
	}
	resp, err := d.HTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("client: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return fmt.Errorf("client: read response: %w", err)
	}

	data, appErr := transport.Unwrap(resp.StatusCode, raw)
	if appErr != nil {
		return &CallError{
			Code:      appErr.Code,
			Status:    appErr.Status,
			RequestID: appErr.RequestID,
			Message:   appErr.Message,
			Details:   appErr.Details,
			verbose:   d.VerboseErrors(),
		}
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("client: decode response: %w", err)
	}
	return nil
}

// CallError is the error returned by [Call] for non-2xx responses. It
// carries the SPEC §18 fields partner code can inspect; the top-level
// package converts it into a public *zeam.Error in recipes.
type CallError struct {
	Code      string
	Status    int
	RequestID string
	Message   string
	Details   map[string]any

	verbose bool
}

// Error satisfies the error interface. Safe for partner-facing logs; the
// gateway's verbose message is only appended when the client was
// constructed with WithVerboseErrors.
func (e *CallError) Error() string {
	if e == nil {
		return "<nil>"
	}
	base := fmt.Sprintf("client: %s (status=%d request_id=%s)", ellipsis(e.Code, "error"), e.Status, e.RequestID)
	if e.verbose && e.Message != "" {
		base += ": " + e.Message
	}
	return base
}

func ellipsis(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
