// Package zeam is the official Go SDK for the Zeam Platform API Gateway.
//
// The SDK exposes:
//
//   - [Client], the entry-point constructed via [New].
//   - Typed 1:1 clients for gateway endpoints under the client/<domain> sub-packages.
//   - High-level recipes for the most common workflows under recipes/.
//   - OTP and SEP-10 authentication flows with a pluggable token store under auth/.
//   - A narrow wrapper over the Stellar SDK under stellar/.
//   - Inbound webhook HMAC verification under webhook/.
//
// Design principles:
//
//   - Secure by default: no secret reaches disk, logs, or telemetry.
//   - Contract parity: SDK vA.B.C targets gateway vA.B.≥0. [MinGatewayVersion]
//     encodes the lowest gateway the current SDK is known-good against, and a
//     runtime handshake against /healthz refuses mismatched gateways.
//   - Explicit context: every public call takes ctx context.Context.
//   - No globals: all configuration flows through Options passed to [New].
//
// The packages under internal/ and transport/ are private implementation
// details and changes to them are explicitly not considered breaking.
package zeam
