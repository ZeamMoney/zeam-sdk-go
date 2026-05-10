package transport

import (
	"encoding/json"
	"strings"
)

// EnvelopeError is the transport-layer representation of a SPEC §18 error
// envelope. The top-level package converts it to *zeam.Error before
// returning to callers.
type EnvelopeError struct {
	Status    int
	RequestID string
	Code      string
	Message   string
	Details   map[string]any
}

// gatewayEnvelope mirrors the standard gateway response shape
// `{data, status, message}`.
type gatewayEnvelope struct {
	Data    json.RawMessage `json:"data"`
	Status  int             `json:"status"`
	Message string          `json:"message"`
}

// Unwrap decodes a gateway response. On success it returns the inner
// `data` raw JSON. On a structured error it returns a populated
// [*EnvelopeError]. A network-level read error is signalled by returning
// nil, nil; callers should treat any non-2xx status as an error regardless
// of decode success.
//
// status is the HTTP status code from the response line; body is the raw
// response bytes.
func Unwrap(status int, body []byte) (json.RawMessage, *EnvelopeError) {
	if status >= 200 && status < 300 {
		var env gatewayEnvelope
		if err := json.Unmarshal(body, &env); err == nil && env.Status >= 200 && env.Data != nil {
			return env.Data, nil
		}
		// Bare-JSON passthrough for endpoints that don't use the
		// standard envelope (e.g. /auth-connect returns bare SEP-10 JSON).
		return body, nil
	}

	out := &EnvelopeError{Status: status}
	var env gatewayEnvelope
	if err := json.Unmarshal(body, &env); err == nil {
		out.Message = env.Message
		// The gateway encodes error codes as the prefix before ":" in the
		// message field (e.g. "missing_field: amount is required").
		if code, _, ok := strings.Cut(env.Message, ":"); ok {
			out.Code = strings.TrimSpace(code)
			out.Message = strings.TrimSpace(env.Message)
		}
	}
	return nil, out
}
