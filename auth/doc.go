// Package auth manages the two authentication tracks exposed by the
// Zeam gateway: Business (OTP / Firebase) and Connect (SEP-10).
//
// Key types:
//
//   - [Session] — an authenticated bearer tied to a specific [Track].
//   - [TokenStore] — pluggable persistence; [MemoryStore] is the default.
//   - [OTPFlow] — orchestrates the Business OTP flow.
//   - [SEP10Flow] — orchestrates the Connect SEP-10 flow.
//   - [AutoRefresher] — background refresh with single-flight semantics.
//
// Design rules enforced at compile time:
//
//   - A session carries its [Track]; the client/ sub-packages reject a
//     session of the wrong track at the type level.
//   - Refresh tokens rotate on every use (single-use semantics). The old
//     value is zeroed after a successful refresh.
//   - No session is ever written to disk unless the caller explicitly
//     passes a disk-backed [TokenStore].
package auth
