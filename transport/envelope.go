package transport

import (
	"encoding/json"
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

// successEnvelope mirrors SPEC §18 `{ok, request_id, resource, verb, data,
// state?, substate?}`.
type successEnvelope struct {
	OK        bool            `json:"ok"`
	RequestID string          `json:"request_id"`
	Data      json.RawMessage `json:"data"`
}

// errorEnvelope mirrors the error-side SPEC §18 shape.
type errorEnvelope struct {
	OK        bool   `json:"ok"`
	RequestID string `json:"request_id"`
	Errors    []struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	} `json:"errors"`
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
		var env successEnvelope
		if err := json.Unmarshal(body, &env); err == nil && env.OK {
			return env.Data, nil
		}
		// Surface as bare JSON when the response didn't match the envelope
		// (e.g. /auth-connect which is explicitly bare JSON).
		return body, nil
	}

	out := &EnvelopeError{Status: status}
	var env errorEnvelope
	if err := json.Unmarshal(body, &env); err == nil {
		out.RequestID = env.RequestID
		if len(env.Errors) > 0 {
			out.Code = env.Errors[0].Code
			out.Message = env.Errors[0].Message
			out.Details = env.Errors[0].Details
		}
	}
	return nil, out
}
