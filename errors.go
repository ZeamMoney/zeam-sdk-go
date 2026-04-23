package zeam

import (
	"errors"
	"fmt"
	"net/http"
)

// Kind is the canonical error kind emitted by the gateway. It mirrors
// ADR 0003 amendment 2026-04-22 and is stable across SDK versions.
type Kind int

// Enumerated Kinds. New kinds are only added on a gateway ADR amendment.
const (
	// KindUnknown represents an unclassified error.
	KindUnknown Kind = iota
	// KindValidation — HTTP 400 / 422.
	KindValidation
	// KindAuth — HTTP 401.
	KindAuth
	// KindAuthz — HTTP 403.
	KindAuthz
	// KindNotFound — HTTP 404.
	KindNotFound
	// KindConflict — HTTP 409.
	KindConflict
	// KindTransient — HTTP 408 / 429 / 503 / 504.
	KindTransient
	// KindRemote — HTTP 5xx not covered by KindTransient.
	KindRemote
)

// Error satisfies the error interface so Kind values can be used as
// targets for errors.Is (e.g. errors.Is(err, zeam.KindAuth)).
func (k Kind) Error() string { return "zeam: " + k.String() }

// String returns the canonical name of the kind.
func (k Kind) String() string {
	switch k {
	case KindValidation:
		return "validation"
	case KindAuth:
		return "auth"
	case KindAuthz:
		return "authz"
	case KindNotFound:
		return "not_found"
	case KindConflict:
		return "conflict"
	case KindTransient:
		return "transient"
	case KindRemote:
		return "remote"
	default:
		return "unknown"
	}
}

// KindFromStatus maps an HTTP status to the canonical [Kind] per ADR 0003.
func KindFromStatus(status int) Kind {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return KindValidation
	case http.StatusUnauthorized:
		return KindAuth
	case http.StatusForbidden:
		return KindAuthz
	case http.StatusNotFound:
		return KindNotFound
	case http.StatusConflict:
		return KindConflict
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return KindTransient
	}
	if status >= 500 && status < 600 {
		return KindRemote
	}
	return KindUnknown
}

// Error is the gateway-shaped error surfaced by every SDK call. The Code
// field matches the canonical enum in ADR 0003 amendment 2026-04-22.
//
// Error satisfies the standard error interface and supports errors.Is /
// errors.As against both [*Error] and [Kind] targets.
type Error struct {
	// Code is the canonical, stable reason code (e.g. "invalid_token",
	// "missing_amount", "upstream_timeout").
	Code string
	// Kind is the classified error kind derived from the HTTP status.
	Kind Kind
	// Status is the HTTP status code from the gateway.
	Status int
	// RequestID is the gateway's X-Request-Id header. Always populated on
	// errors; callers should surface it to users for support triage.
	RequestID string
	// Message is the human-readable gateway message. Only surfaced to
	// callers when WithVerboseErrors is set on the client. Never surface
	// this to end users without auditing it first.
	Message string
	// Details carries structured upstream context (keys per ADR 0008 R5).
	Details map[string]any
}

// Error returns the safe string form of the error: canonical code plus
// request id. The verbose gateway message is intentionally omitted unless
// [WithVerboseErrors] is enabled.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Code == "" {
		return fmt.Sprintf("zeam: error (kind=%s status=%d request_id=%s)", e.Kind, e.Status, e.RequestID)
	}
	return fmt.Sprintf("zeam: %s (kind=%s status=%d request_id=%s)", e.Code, e.Kind, e.Status, e.RequestID)
}

// Is reports whether target matches this error. It compares against other
// [*Error] values (by Code) and against [Kind] values. This lets callers
// write errors.Is(err, zeam.KindTransient).
func (e *Error) Is(target error) bool {
	if e == nil {
		return target == nil
	}
	if other, ok := target.(*Error); ok && other != nil {
		if other.Code != "" {
			return e.Code == other.Code
		}
		if other.Kind != KindUnknown {
			return e.Kind == other.Kind
		}
		return false
	}
	if k, ok := target.(Kind); ok {
		return e.Kind == k
	}
	// Kind values passed via errors.Is are treated as error targets; the
	// preceding case matches them because Kind satisfies the error
	// interface via [Kind.Error].
	return false
}

// ErrIncompatibleGateway is returned by the version handshake when the
// gateway version is older than [MinGatewayVersion].
var ErrIncompatibleGateway = errors.New("zeam: incompatible gateway version")

// Sentinel convenience errors. Prefer matching via errors.Is against a
// [Kind] (e.g. errors.Is(err, zeam.KindAuth)) rather than these, so the
// full context of the Error is preserved.
var (
	// ErrAuth is the generic auth sentinel (KindAuth).
	ErrAuth = &Error{Kind: KindAuth}
	// ErrValidation is the generic validation sentinel (KindValidation).
	ErrValidation = &Error{Kind: KindValidation}
	// ErrNotFound is the generic not-found sentinel (KindNotFound).
	ErrNotFound = &Error{Kind: KindNotFound}
	// ErrConflict is the generic conflict sentinel (KindConflict).
	ErrConflict = &Error{Kind: KindConflict}
	// ErrTransient is the generic transient sentinel (KindTransient).
	ErrTransient = &Error{Kind: KindTransient}
	// ErrRemote is the generic remote-failure sentinel (KindRemote).
	ErrRemote = &Error{Kind: KindRemote}
)
