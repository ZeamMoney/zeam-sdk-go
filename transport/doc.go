// Package transport holds the HTTP plumbing the SDK uses to talk to the
// gateway. It is explicitly NOT part of the SDK's public stability
// contract: package paths, types, and signatures here may change without
// notice. Do not depend on this package directly.
//
// The transport stack, composed in order, looks like:
//
//	http.Client
//	  └── Decorate (UA + X-Request-Id capture + Idempotency-Key + redaction)
//	        └── retryRoundTripper (GET-only bounded retries, ADR 0008 R6)
//	              └── Base (*http.Transport with TLS 1.3 min)
//
// Unwrapping the SPEC §18 envelope is done at the call site via [Unwrap].
package transport
