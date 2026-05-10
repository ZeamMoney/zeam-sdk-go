# zeam-sdk-go

[![Go Version](https://img.shields.io/github/go-mod/go-version/ZeamMoney/zeam-sdk-go)](https://go.dev/)
[![License](https://img.shields.io/github/license/ZeamMoney/zeam-sdk-go)](./LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/ZeamMoney/zeam-sdk-go.svg)](https://pkg.go.dev/github.com/ZeamMoney/zeam-sdk-go)

Official Go SDK for the **Zeam API Gateway**. Typed clients, high-level recipes,
and automatic auth lifecycle management — kept in lockstep with the gateway contract.

## Features

- **Typed clients** — 1:1 wrappers for every `/v1/*` gateway endpoint
- **Recipes** — one-call workflows for OTP login, SEP-10 auth, connect payments, credential rotation
- **Two auth tracks** — Business (OTP/Firebase) and Connect (SEP-10), isolated at the type level
- **Secure by default** — memory-only token store, TLS 1.3, payload redaction, constant-time webhook verification
- **Observable** — OpenTelemetry hooks, structured events, `X-Request-Id` propagation
- **Versioned parity** — runtime handshake against `/healthz` fails fast on gateway mismatch

## Installation

```bash
go get github.com/ZeamMoney/zeam-sdk-go
```

## Quick Start

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
            return "123456", nil // partner-provided UX
        },
    })
    _ = sess
}
```

See [docs/getting-started.md](docs/getting-started.md) for a full walkthrough.

## Configuration

| Option | Default | Description |
|---|---|---|
| `zeam.EnvironmentProduction` | `https://api-gateway.zeam.app` | Canonical API gateway |
| `zeam.EnvironmentCustom(url)` | — | Local dev (e.g. `http://localhost:8080`) |
| `zeam.WithTimeout(d)` | 30s | Per-call deadline |
| `zeam.WithVerboseErrors()` | off | Include upstream gateway messages in errors |
| `zeam.WithSkipVersionCheck()` | off | Disable `/healthz` handshake |

### Environment variables

```bash
ZEAM_API_BASE_URL=https://api-gateway.zeam.app
ZEAM_CLIENT_ID=your_stellar_public_key
ZEAM_CLIENT_SECRET=your_stellar_seed     # from your secret manager
ZEAM_API_KEY=your_api_key                # required for Connect endpoints
```

### Sandbox mode

Zeam does not provide a separate sandbox URL. All integrations — sandbox and
production — call the same `https://api-gateway.zeam.app` endpoint. Your
credentials and Zeam-side account configuration determine your access mode.
You do not change URLs to switch between sandbox and production.

## Error handling

```go
result, err := client.Raw().GET(ctx, "/v1/business/association/all", nil)
if err != nil {
    var e *zeam.Error
    if errors.As(err, &e) {
        fmt.Printf("code=%s kind=%s status=%d request_id=%s\n",
            e.Code, e.Kind, e.Status, e.RequestID)
    }
    if errors.Is(err, zeam.KindTransient) {
        // safe to retry
    }
}
```

## Security

This SDK is distributed publicly. Read [SECURITY.md](SECURITY.md) for the
threat model, disclosure process, and operational patterns integrators must follow.

Never commit one-time credentials returned by `POST /v1/application`
(`stellar.secret`, `connectSecret`, `apiKey.secret`, `webhookSecret.secret`)
to source control. Use a cloud secret manager or the SDK's optional keyring backend.

## Versioning

SDK `vA.B.C` targets gateway `vA.B.≥0`. Breaking contract changes bump
`A` (gateway) and `A` (SDK) together. See [docs/versioning.md](docs/versioning.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Clone to first passing test in under ten minutes.

## License

Copyright © 2026 Paytec Technologies B.V. All rights reserved.

This repository is publicly visible for transparency and reference purposes only.

No part of this codebase may be copied, modified, distributed, sublicensed, or used in commercial or production systems without prior written permission from the copyright holder.

External contributions are not accepted at this stage.
