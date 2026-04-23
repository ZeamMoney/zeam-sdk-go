package zeam

import (
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net/http"
	"time"

	"github.com/ZeamMoney/zeam-sdk-go/auth"
)

// Option configures a [Client]. Options are evaluated in the order they are
// passed to [New]; later options win. Unknown or zero-valued options are
// ignored.
type Option func(*config) error

// ObservabilityHook is invoked by the SDK on notable events. It is always
// called with pre-redacted payloads; implementations must never store
// pointers to the supplied maps.
type ObservabilityHook func(event string, attrs map[string]any)

// config is the private bag populated by [Option] functions.
type config struct {
	env            Environment
	httpClient     *http.Client
	tokenStore     auth.TokenStore
	logger         *slog.Logger
	userAgent      string
	verboseErrors  bool
	insecureTrans  bool
	insecureFile   bool
	pinnedRoots    *x509.CertPool
	skipVerCheck   bool
	timeout        time.Duration
	observer       ObservabilityHook
	transportMWs   []func(http.RoundTripper) http.RoundTripper
	stellarNetwork string
}

func defaultConfig() *config {
	return &config{
		env:            EnvironmentProduction,
		timeout:        30 * time.Second,
		userAgent:      "zeam-sdk-go/" + Version,
		stellarNetwork: "Public Global Stellar Network ; September 2015",
	}
}

// WithEnvironment sets the target gateway environment.
func WithEnvironment(env Environment) Option {
	return func(c *config) error {
		c.env = env
		return nil
	}
}

// WithHTTPClient supplies a custom *http.Client. The supplied client is
// decorated with the SDK's transport middleware (retry, redaction,
// observability); callers retain ownership of any long-lived connection pool.
func WithHTTPClient(client *http.Client) Option {
	return func(c *config) error {
		c.httpClient = client
		return nil
	}
}

// WithTokenStore configures the token persistence backend used by the
// authentication flows. Defaults to an in-memory store if unset.
func WithTokenStore(store auth.TokenStore) Option {
	return func(c *config) error {
		c.tokenStore = store
		return nil
	}
}

// WithLogger wires a slog logger. The SDK's redaction layer runs before the
// logger receives any payload.
func WithLogger(logger *slog.Logger) Option {
	return func(c *config) error {
		c.logger = logger
		return nil
	}
}

// WithUserAgent overrides the User-Agent header. The default is
// "zeam-sdk-go/<Version>".
func WithUserAgent(ua string) Option {
	return func(c *config) error {
		c.userAgent = ua
		return nil
	}
}

// WithVerboseErrors enables inclusion of upstream gateway messages in the
// [*Error.Error] output. Disabled by default so partners do not surface
// upstream copy to end users unintentionally.
func WithVerboseErrors() Option {
	return func(c *config) error {
		c.verboseErrors = true
		return nil
	}
}

// WithInsecureTransport permits plain-HTTP base URLs. Intended for local
// development only. Must be paired with ZEAM_SDK_ALLOW_INSECURE=1.
func WithInsecureTransport() Option {
	return func(c *config) error {
		c.insecureTrans = true
		return nil
	}
}

// WithInsecureFileStore permits the optional file-backed token store.
// Strongly discouraged outside of local development; prints a warning to
// stderr on construction.
func WithInsecureFileStore() Option {
	return func(c *config) error {
		c.insecureFile = true
		return nil
	}
}

// WithPinnedRootCAs pins the set of acceptable root CAs for TLS connections
// to the gateway.
func WithPinnedRootCAs(pool *x509.CertPool) Option {
	return func(c *config) error {
		c.pinnedRoots = pool
		return nil
	}
}

// WithSkipVersionCheck disables the runtime /healthz handshake. Intended for
// sandbox / pre-release development only.
func WithSkipVersionCheck() Option {
	return func(c *config) error {
		c.skipVerCheck = true
		return nil
	}
}

// WithTimeout sets the default per-call deadline applied when the caller's
// context has no deadline of its own.
func WithTimeout(d time.Duration) Option {
	return func(c *config) error {
		c.timeout = d
		return nil
	}
}

// WithObservability registers an observability hook. The hook is called with
// already-redacted attributes; implementations must not retain references to
// the supplied map.
func WithObservability(hook ObservabilityHook) Option {
	return func(c *config) error {
		c.observer = hook
		return nil
	}
}

// WithTransportMiddleware wraps the SDK's base RoundTripper. Middlewares run
// after redaction and retry, so they see only redacted events.
func WithTransportMiddleware(mw func(http.RoundTripper) http.RoundTripper) Option {
	return func(c *config) error {
		c.transportMWs = append(c.transportMWs, mw)
		return nil
	}
}

// WithStellarNetwork overrides the SEP-10 network passphrase. Defaults to
// the Stellar Public Main Network.
func WithStellarNetwork(passphrase string) Option {
	return func(c *config) error {
		c.stellarNetwork = passphrase
		return nil
	}
}

// tlsConfig returns the TLS config the SDK uses by default: TLS 1.3 min,
// optional pinned roots.
func (c *config) tlsConfig() *tls.Config {
	cfg := &tls.Config{MinVersion: tls.VersionTLS13}
	if c.pinnedRoots != nil {
		cfg.RootCAs = c.pinnedRoots
	}
	return cfg
}
