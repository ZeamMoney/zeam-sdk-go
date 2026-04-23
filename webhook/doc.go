// Package webhook verifies inbound HMAC-signed webhooks issued by the
// Zeam platform. Partners capture the webhook signing secret at
// application registration (see recipes.RegisterApplication) and pass
// it to [Verify] or [Handler].
//
// The verifier:
//
//   - Compares signatures in constant time (crypto/subtle).
//   - Enforces a configurable clock-skew window (default 5 minutes) so
//     a leaked signature can't be replayed indefinitely.
//   - Optionally deduplicates by the platform's event ID via a [ReplayCache].
//
// Per ADR 0008 R8 the signing secret is never logged; the verifier only
// emits the event ID, latency, and outcome to the configured observer.
package webhook
