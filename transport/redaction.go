package transport

import (
	"fmt"
	"regexp"
	"strings"
)

// sensitiveHeaderKeys are always redacted before logging.
var sensitiveHeaderKeys = map[string]struct{}{
	"authorization":     {},
	"x-zeam-auth":       {},
	"set-cookie":        {},
	"cookie":            {},
	"x-idempotency-key": {}, // treated as sensitive for replay audits
	"x-zeam-otp-token":  {},
}

// sensitiveBodyKeys are redacted when encountered in attribute maps or
// structured events. Compared case-insensitively (callers lower-case the
// key before lookup).
var sensitiveBodyKeys = map[string]struct{}{
	"idtoken":        {},
	"refreshtoken":   {},
	"accesstoken":    {},
	"customtoken":    {},
	"otp":            {},
	"code":           {},
	"secret":         {},
	"stellarsecret":  {},
	"connectsecret":  {},
	"apikey":         {},
	"apikeysecret":   {},
	"webhooksecret":  {},
	"password":       {},
	"pin":            {},
	"otp_session_id": {},
	"otpsessionid":   {},
	"requestid":      {}, // webhook request IDs can be sensitive
	"bearer":         {},
	"seed":           {},
	"privatekey":     {},
}

// patterns redacts known sensitive value shapes even when the key doesn't
// tip us off: Stellar seeds, Stellar public keys, JWT tokens, API keys in
// the `1_...` form used by the platform.
var patterns = []*regexp.Regexp{
	regexp.MustCompile(`\bS[A-Z2-7]{55}\b`),                                     // Stellar seed
	regexp.MustCompile(`\bG[A-Z2-7]{55}\b`),                                     // Stellar public key
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`), // JWT
	regexp.MustCompile(`\b1_[A-Za-z0-9_-]{16,}\b`),                              // Zeam API key
}

// Redact returns a shallow copy of attrs with sensitive values replaced by
// a constant placeholder. Nested maps are redacted recursively. The input
// map is not mutated.
func Redact(attrs map[string]any) map[string]any {
	if attrs == nil {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		out[k] = redactValue(k, v)
	}
	return out
}

// redactValue applies redaction rules to a single key/value pair.
func redactValue(key string, v any) any {
	lk := strings.ToLower(key)
	if _, sensitive := sensitiveBodyKeys[lk]; sensitive {
		return "<redacted>"
	}
	if _, sensitive := sensitiveHeaderKeys[lk]; sensitive {
		return "<redacted>"
	}
	switch vv := v.(type) {
	case string:
		return redactString(vv)
	case []byte:
		return "<redacted-bytes>"
	case map[string]any:
		return Redact(vv)
	case []any:
		out := make([]any, len(vv))
		for i, item := range vv {
			out[i] = redactValue(key, item)
		}
		return out
	default:
		return v
	}
}

// redactString applies pattern-based redaction to a bare string.
func redactString(s string) string {
	for _, p := range patterns {
		s = p.ReplaceAllString(s, "<redacted>")
	}
	return s
}

// RedactHeaders returns a copy of a header-like map with sensitive headers
// replaced by a placeholder string.
func RedactHeaders(h map[string][]string) map[string][]string {
	out := make(map[string][]string, len(h))
	for k, vs := range h {
		if _, sensitive := sensitiveHeaderKeys[strings.ToLower(k)]; sensitive {
			out[k] = []string{"<redacted>"}
			continue
		}
		clean := make([]string, len(vs))
		for i, v := range vs {
			clean[i] = redactString(v)
		}
		out[k] = clean
	}
	return out
}

// Fingerprint returns the first eight characters of a bearer-like string
// so logs can correlate a single session without ever exposing the full
// value. Safe for audit logs (no reversal possible at eight chars).
func Fingerprint(token string) string {
	if len(token) == 0 {
		return ""
	}
	if len(token) <= 8 {
		return fmt.Sprintf("len=%d", len(token))
	}
	return token[:8] + "…"
}
