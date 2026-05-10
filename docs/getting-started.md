# Getting Started

## Prerequisites

- Go 1.26.2 or later.
- A Zeam platform account with issued credentials.

## Install

```bash
go get github.com/ZeamMoney/zeam-sdk-go
```

## Your first call

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ZeamMoney/zeam-sdk-go"
)

func main() {
    ctx := context.Background()

    client, err := zeam.New(
        zeam.WithEnvironment(zeam.EnvironmentProduction),
    )
    if err != nil {
        log.Fatal(err)
    }

    h, err := client.Health().Get(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("gateway version: %s\n", h.Version)
}
```

## Configuration

| Option | Default | Description |
|---|---|---|
| `zeam.EnvironmentProduction` | `https://api-gateway.zeam.app` | Canonical API gateway (production and sandbox) |
| `zeam.EnvironmentCustom(url)` | — | Local development (e.g. `http://localhost:8080`) |

### Sandbox mode

Zeam does not provide a separate sandbox URL. Sandbox mode runs against the
same `https://api-gateway.zeam.app` endpoint. Your credentials and account
configuration determine your access mode — you do not change URLs to switch
between sandbox and production.

## Version compatibility

The SDK declares `zeam.MinGatewayVersion`. On the first call, `Client.Ping`
compares it against the gateway's `/healthz` `version` field and returns
`zeam.ErrIncompatibleGateway` if the gateway is older. Opt out during early
development with `zeam.WithSkipVersionCheck()`.

## Next steps

- [Authentication](auth.md) — get a Business or Connect session.
- [Recipes](recipes.md) — full workflows in one call.
- [Error handling](errors.md) — canonical codes and `errors.Is` patterns.
