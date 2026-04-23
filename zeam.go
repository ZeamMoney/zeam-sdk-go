package zeam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/ZeamMoney/zeam-sdk-go/transport"
)

// Version is the current SDK version. Populated at release time by the
// build tooling; overridden by linker flags for tagged builds.
const Version = "0.0.0-unreleased"

// MinGatewayVersion is the lowest gateway version this SDK has been
// verified against. The runtime handshake in [Client.Ping] refuses gateways
// older than this.
const MinGatewayVersion = "0.1.0"

// SpecHash is the SHA-256 of the OpenAPI spec the SDK was generated from.
// A compile-time assertion in internal/wire verifies this matches the
// checked-in spec to prevent silent drift.
const SpecHash = "0000000000000000000000000000000000000000000000000000000000000000"

// Client is the top-level SDK entry point. Construct one via [New], keep a
// single instance per process, and pass it to the client/ sub-packages or
// to a recipes/ workflow.
type Client struct {
	cfg *config

	httpClient *http.Client
	baseURL    *url.URL

	versionOnce sync.Once
	versionErr  error

	lz lazy
}

// New constructs a [Client]. Returns an error for invalid option
// combinations (e.g. plain HTTP without [WithInsecureTransport]).
func New(opts ...Option) (*Client, error) {
	cfg := defaultConfig()
	for _, o := range opts {
		if o == nil {
			continue
		}
		if err := o(cfg); err != nil {
			return nil, fmt.Errorf("zeam: apply option: %w", err)
		}
	}

	if cfg.env.BaseURL() == "" {
		return nil, errors.New("zeam: environment has empty base URL")
	}
	u, err := url.Parse(cfg.env.BaseURL())
	if err != nil {
		return nil, fmt.Errorf("zeam: parse base URL: %w", err)
	}
	if u.Scheme == "http" {
		if !cfg.insecureTrans {
			return nil, errors.New("zeam: http:// base URLs require WithInsecureTransport()")
		}
		if os.Getenv("ZEAM_SDK_ALLOW_INSECURE") != "1" {
			return nil, errors.New("zeam: set ZEAM_SDK_ALLOW_INSECURE=1 to permit plain http")
		}
		fmt.Fprintln(os.Stderr, "zeam-sdk: WARNING — plain HTTP transport in use; never use this in production")
	} else if u.Scheme != "https" {
		return nil, fmt.Errorf("zeam: unsupported URL scheme %q", u.Scheme)
	}

	if cfg.insecureFile {
		fmt.Fprintln(os.Stderr, "zeam-sdk: WARNING — insecure file-backed token store enabled")
	}

	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.timeout}
	}

	base := transport.Base(httpClient.Transport, cfg.tlsConfig())
	decorated := transport.Decorate(base, transport.Options{
		UserAgent:     cfg.userAgent,
		Observer:      transport.ObservabilityHook(cfg.observer),
		VerboseErrors: cfg.verboseErrors,
	})
	for _, mw := range cfg.transportMWs {
		if mw != nil {
			decorated = mw(decorated)
		}
	}
	httpClient.Transport = decorated

	c := &Client{
		cfg:        cfg,
		httpClient: httpClient,
		baseURL:    u,
	}
	return c, nil
}

// BaseURL returns the gateway base URL this client is bound to.
func (c *Client) BaseURL() *url.URL { return c.baseURL }

// HTTPClient returns the underlying *http.Client (already decorated with
// the SDK's transport stack). Intended for the client/ sub-packages and
// contract tests, not for partner use.
func (c *Client) HTTPClient() *http.Client { return c.httpClient }

// UserAgent returns the User-Agent string the client sends on every call.
func (c *Client) UserAgent() string { return c.cfg.userAgent }

// Environment returns the [Environment] this client was constructed with.
func (c *Client) Environment() Environment { return c.cfg.env }

// VerboseErrors reports whether the client is configured to include
// upstream gateway messages in the [*Error] output.
func (c *Client) VerboseErrors() bool { return c.cfg.verboseErrors }

// StellarNetwork returns the configured Stellar network passphrase.
func (c *Client) StellarNetwork() string { return c.cfg.stellarNetwork }

// Ping performs the version-compatibility handshake against /healthz.
// It runs at most once per client; callers can invoke it explicitly to
// surface [ErrIncompatibleGateway] at start-up rather than on the first
// domain call.
func (c *Client) Ping(ctx context.Context) error {
	if c.cfg.skipVerCheck {
		return nil
	}
	c.versionOnce.Do(func() {
		c.versionErr = c.ping(ctx)
	})
	return c.versionErr
}

type healthzBody struct {
	Status   string `json:"status"`
	UptimeMS int64  `json:"uptime_ms"`
	Version  string `json:"version"`
}

func (c *Client) ping(ctx context.Context) error {
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + "/healthz"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("zeam: build /healthz request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("zeam: /healthz call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &Error{
			Code:   "healthz_failed",
			Kind:   KindFromStatus(resp.StatusCode),
			Status: resp.StatusCode,
		}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("zeam: read /healthz body: %w", err)
	}
	var h healthzBody
	if err := json.Unmarshal(body, &h); err != nil {
		// Gateways older than the version we require may not have a version
		// field at all; they are incompatible by definition.
		return ErrIncompatibleGateway
	}
	if h.Version == "" {
		return ErrIncompatibleGateway
	}
	if compareSemver(h.Version, MinGatewayVersion) < 0 {
		return fmt.Errorf("%w: gateway %s < %s", ErrIncompatibleGateway, h.Version, MinGatewayVersion)
	}
	return nil
}

// compareSemver returns negative / zero / positive comparing a and b as
// `MAJOR.MINOR.PATCH`. Missing segments are treated as zero.
func compareSemver(a, b string) int {
	split := func(s string) [3]int {
		s = strings.TrimPrefix(s, "v")
		var out [3]int
		parts := strings.SplitN(s, ".", 3)
		for i := 0; i < len(parts) && i < 3; i++ {
			var n int
			_, _ = fmt.Sscanf(parts[i], "%d", &n)
			out[i] = n
		}
		return out
	}
	as, bs := split(a), split(b)
	for i := 0; i < 3; i++ {
		if as[i] != bs[i] {
			return as[i] - bs[i]
		}
	}
	return 0
}

// Raw returns a handle for direct HTTP calls against the gateway. Useful
// for endpoints the typed wrappers don't cover yet. Still goes through the
// SDK's transport stack (retry, redaction, envelope unwrap).
func (c *Client) Raw() *RawClient { return &RawClient{c: c} }

// RawClient exposes raw method-level HTTP against the gateway. Obtained
// via [Client.Raw].
type RawClient struct{ c *Client }

// GET performs a GET against path + query. path must be an absolute path
// beginning with "/".
func (rc *RawClient) GET(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	return rc.do(ctx, http.MethodGet, path, query, nil)
}

// POST performs a POST with a JSON-encoded body.
func (rc *RawClient) POST(ctx context.Context, path string, body any) (json.RawMessage, error) {
	return rc.do(ctx, http.MethodPost, path, nil, body)
}

// PUT performs a PUT with a JSON-encoded body.
func (rc *RawClient) PUT(ctx context.Context, path string, body any) (json.RawMessage, error) {
	return rc.do(ctx, http.MethodPut, path, nil, body)
}

// DELETE performs a DELETE against path.
func (rc *RawClient) DELETE(ctx context.Context, path string) (json.RawMessage, error) {
	return rc.do(ctx, http.MethodDelete, path, nil, nil)
}

func (rc *RawClient) do(ctx context.Context, method, path string, query url.Values, body any) (json.RawMessage, error) {
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("zeam: raw path must be absolute, got %q", path)
	}
	u := *rc.c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + path
	if query != nil {
		u.RawQuery = query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("zeam: marshal body: %w", err)
		}
		bodyReader = io.NopCloser(strings.NewReader(string(buf)))
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, rc.c.cfg.timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("zeam: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := rc.c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zeam: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("zeam: read response: %w", err)
	}

	data, appErr := transport.Unwrap(resp.StatusCode, raw)
	if appErr != nil {
		return nil, toError(appErr, rc.c.cfg.verboseErrors)
	}
	return data, nil
}

// toError converts a transport-layer envelope error into the public Error.
func toError(in *transport.EnvelopeError, verbose bool) *Error {
	out := &Error{
		Code:      in.Code,
		Kind:      KindFromStatus(in.Status),
		Status:    in.Status,
		RequestID: in.RequestID,
		Details:   in.Details,
	}
	if verbose {
		out.Message = in.Message
	}
	return out
}

// deadlineCeiling returns a context bounded by either ctx's deadline or by
// the client-wide default timeout. Used by the client/ sub-packages.
func (c *Client) deadlineCeiling(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.cfg.timeout)
}
