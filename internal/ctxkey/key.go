// Package ctxkey holds the typed context.Context keys used across the
// SDK. Keeping them in internal/ ensures partner code cannot collide
// with the SDK's key set.
package ctxkey

// Key is a typed, unexported-payload context key.
type Key int

const (
	// RequestID is the gateway's X-Request-Id value.
	RequestID Key = iota
	// SessionFingerprint is the 8-char prefix of the current id-token.
	SessionFingerprint
	// IdempotencyKey is the value attached to the outgoing request.
	IdempotencyKey
)
