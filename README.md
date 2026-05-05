# zeam-sdk-go

[![Go Version](https://img.shields.io/github/go-mod/go-version/ZeamMoney/zeam-sdk-go)](https://go.dev/)
[![License](https://img.shields.io/github/license/ZeamMoney/zeam-sdk-go)](./LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/ZeamMoney/zeam-sdk-go.svg)](https://pkg.go.dev/github.com/ZeamMoney/zeam-sdk-go)

Official Go SDK for the **Zeam Platform API Gateway**. Secure-by-default, opinionated,
and kept in lockstep with the gateway contract.

The SDK gives partner and first-party integrators:

- Typed, 1:1 clients for every `/v1/*` gateway endpoint.
- High-level **recipes** for common end-to-end flows (OTP login, SEP-10 login,
  application registration, connect payment orchestration, credential rotation).
- Automatic authentication lifecycle management (acquisition, single-flight
  refresh, cross-track isolation, secure storage).
- A narrow wrapper over the Stellar SDK so partners never import `stellar/go`
  directly.
- Inbound webhook HMAC verification with replay protection.

```go
import (
    "context"

    "github.com/ZeamMoney/zeam-sdk-go"
    "github.com/ZeamMoney/zeam-sdk-go/auth"
    "github.com/ZeamMoney/zeam-sdk-go/recipes"
)

func main() {
    ctx := context.Background()

    client, err := zeam.New(
        zeam.WithEnvironment(zeam.EnvironmentProduction),
        zeam.WithTokenStore(auth.NewMemoryStore()),
    )
    if err != nil {
        panic(err)
    }

    sess, err := recipes.LoginOTP(ctx, client, recipes.LoginOTPInput{
        MobileNumber: "+27821234567",
        AskCode: func(ctx context.Context, hint recipes.OTPHint) (string, error) {
            // partner-provided UX returns the code the end user typed
            return "123456", nil
        },
    })
    _ = sess
}
```

## Getting Started

```bash
go get github.com/ZeamMoney/zeam-sdk-go
```

See [docs/getting-started.md](docs/getting-started.md) for a full walkthrough.

## Features

- **Two authentication tracks** — Business (OTP/Firebase) and Connect (SEP-10)
  are isolated at the type level; the SDK refuses to send a Business token to a
  Connect endpoint or vice versa.
- **Recipes** — one-call workflows for the most common orchestrations:
  - `recipes.LoginOTP` — Business OTP login.
  - `recipes.RegisterApplication` — one-time-secret capture at registration.
  - `recipes.ConnectLogin` — full SEP-10 flow using a stored Stellar seed.
  - `recipes.ConnectPayment` — 9-step off-ramp payment orchestration.
  - `recipes.RotateCredential` — API key / webhook secret rotation.
- **Secure by default** — memory-only token store, redaction before any
  user-supplied logger sees payloads, TLS 1.3, constant-time webhook
  signature verification, SSRF guards on `connect.Exec`.
- **Observable** — OpenTelemetry tracing hooks, structured events, and
  `X-Request-Id` propagation. No secrets ever reach spans or logs.
- **Versioned parity** — the SDK declares `zeam.MinGatewayVersion` and performs
  a runtime handshake against `/healthz` so a mismatched gateway fails fast
  instead of corrupting a request.

## Security

This SDK is distributed publicly. Read [SECURITY.md](SECURITY.md) for the
threat model, disclosure process, and the list of operational patterns
integrators are expected to follow.

Never commit any of the one-time credentials returned by `POST /v1/application`
(`stellar.secret`, `connectSecret`, `apiKey.secret`, `webhookSecret.secret`)
to source control. Use a cloud secret manager or the SDK's optional keyring
backend.

## Versioning

SDK `vA.B.C` targets gateway `vA.B.≥0`. Breaking contract changes bump
`A` (gateway) and `A` (SDK) together. See [docs/versioning.md](docs/versioning.md)
for the full policy.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). The goal is to get a new contributor
from clone to first passing test in under ten minutes.

## License

Copyright © 2026 Paytec Technologies B.V. All rights reserved.

This repository is publicly visible for transparency and reference purposes only.

No part of this codebase may be copied, modified, distributed, sublicensed, or used in commercial or production systems without prior written permission from the copyright holder.

External contributions are not accepted at this stage.
