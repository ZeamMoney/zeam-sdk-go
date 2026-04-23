# Zeam Go SDK

The official Go SDK for the Zeam Platform API Gateway. Secure-by-default,
opinionated, and kept in lockstep with the gateway contract.

- **Install**: `go get github.com/ZeamMoney/zeam-sdk-go`
- **Quickstart**: [Getting Started](getting-started.md)
- **Common flows**: [Recipes](recipes.md)
- **Security posture**: [Security](security.md)
- **Source**: [ZeamMoney/zeam-sdk-go](https://github.com/ZeamMoney/zeam-sdk-go)

## What the SDK gives you

- Typed, 1:1 clients under `client/*` for every gateway endpoint.
- **Recipes** — opinionated end-to-end workflows for the 80% happy path.
- Automatic authentication lifecycle management (acquisition, single-flight
  refresh, cross-track isolation, secure storage).
- A narrow wrapper over the Stellar SDK (partners don't import `stellar/go`
  directly).
- Inbound webhook HMAC verification with replay protection.

## Why use it

- **Fewer footguns.** Cross-track token reuse, unsigned connect execs, and
  mid-flight secret leaks are rejected at the type level.
- **Faster integration.** One function per workflow (`recipes.LoginOTP`,
  `recipes.ConnectPayment`) instead of ~10 handwritten HTTP calls.
- **Versioned parity.** `zeam.MinGatewayVersion` + runtime `/healthz`
  handshake catches mismatched gateways before the first business call.
- **Secure defaults.** Memory-only token store, redaction before any
  user-supplied logger, TLS 1.3 min, constant-time webhook HMAC.

## Non-goals

- Disk-backed token persistence by default. Opt-in via
  `WithInsecureFileStore()` or the keyring build tag.
- Full access to every upstream GraphQL field. The SDK exposes the narrow
  `ConnectQueryConnectors` wrapper; use the `Raw()` escape hatch if you
  need fields we haven't typed yet.
