package webhook

import (
	"bytes"
	"errors"
	"io"
	"net/http"
)

// Handler returns a net/http middleware that verifies every incoming
// request against the supplied secret before delegating to next. On
// verification failure it writes a 401, 400, or 429 status as
// appropriate and does NOT call next. On success it hands next a
// request whose Body has been rewound so the handler can read it
// directly.
func Handler(next http.Handler, secret []byte, opts ...Option) http.Handler {
	v := NewVerifier(secret, opts...)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := v.Verify(r)
		if err != nil {
			statusFromError(w, err)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(w, r)
	})
}

// statusFromError writes a status appropriate to the verification error.
// The gateway's canonical error-kind table in ADR 0003 amendment
// 2026-04-22 maps stale / bad signatures to 401 and replay to 409.
func statusFromError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrMissingSignature),
		errors.Is(err, ErrMissingTimestamp),
		errors.Is(err, ErrInvalidTimestamp),
		errors.Is(err, ErrStaleTimestamp),
		errors.Is(err, ErrBadSignature):
		http.Error(w, "unauthorised", http.StatusUnauthorized)
	case errors.Is(err, ErrReplay):
		http.Error(w, "replayed event", http.StatusConflict)
	case errors.Is(err, ErrBodyTooLarge):
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
	default:
		http.Error(w, "bad request", http.StatusBadRequest)
	}
}
