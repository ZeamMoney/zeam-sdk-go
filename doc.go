// Package zeam is the official Go SDK for the Zeam Platform API Gateway.
//
// The SDK exposes:
//
//   - [Client], the entry-point constructed via [New].
//   - Typed clients for gateway endpoints: client/business, client/application,
//     client/connect, client/payments.
//   - OTP and SEP-10 authentication flows under auth/.
//   - A narrow wrapper over the Stellar SDK under stellar/.
//   - Inbound webhook HMAC verification under webhook/.
//
// Design principles:
//
//   - Strongly typed: every request and response is a Go struct.
//   - Explicit context: every public call takes ctx context.Context.
//   - No globals: all configuration flows through Options passed to [New].
//
// The packages under internal/ and transport/ are private implementation
// details and changes to them are explicitly not considered breaking.
package zeam
