package transport

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ObservabilityHook mirrors the public top-level hook. Aliased here so the
// transport package has no dependency on the top-level package (breaking
// the import cycle).
type ObservabilityHook func(event string, attrs map[string]any)

// Options configures [Decorate].
type Options struct {
	UserAgent     string
	Observer      ObservabilityHook
	VerboseErrors bool
}

// Base returns an *http.Transport pre-configured for TLS 1.3 and sensible
// timeouts. If rt is non-nil it is used verbatim — callers can wrap their
// own transport and still benefit from [Decorate].
func Base(rt http.RoundTripper, tlsCfg *tls.Config) http.RoundTripper {
	if rt != nil {
		return rt
	}
	return &http.Transport{
		TLSClientConfig:       tlsCfg,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// Decorate wraps base with the SDK's request-id capture, idempotency-key
// injection, retry, and redaction-aware observability layers.
func Decorate(base http.RoundTripper, opts Options) http.RoundTripper {
	return &decoratedRT{
		base:  withRetry(base),
		opts:  opts,
		now:   time.Now,
		newID: func() string { return uuid.NewString() },
	}
}

type decoratedRT struct {
	base  http.RoundTripper
	opts  Options
	now   func() time.Time
	newID func() string
}

func (d *decoratedRT) RoundTrip(req *http.Request) (*http.Response, error) {
	if d.opts.UserAgent != "" {
		req.Header.Set("User-Agent", d.opts.UserAgent)
	}

	// Generate an Idempotency-Key on mutating verbs if the caller didn't set
	// one. GETs never get a key (ADR 0008 R2).
	if isMutating(req.Method) && req.Header.Get("Idempotency-Key") == "" {
		req.Header.Set("Idempotency-Key", d.newID())
	}

	start := d.now()
	resp, err := d.base.RoundTrip(req)
	latency := d.now().Sub(start)

	attrs := map[string]any{
		"method":     req.Method,
		"host":       req.URL.Host,
		"path":       req.URL.Path,
		"latency_ms": latency.Milliseconds(),
	}
	if resp != nil {
		attrs["status"] = resp.StatusCode
		attrs["x_request_id"] = resp.Header.Get("X-Request-Id")
	}
	if err != nil {
		attrs["error"] = err.Error()
	}
	if d.opts.Observer != nil {
		d.opts.Observer("http.request", Redact(attrs))
	}

	return resp, err
}

// isMutating reports whether the verb counts as a mutation under ADR 0008 R2.
func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}
