# Getting Started

## Prerequisites

- Go 1.26.2 or later.
- A Zeam platform account (sandbox is fine for first integration).

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
        zeam.WithEnvironment(zeam.EnvironmentSandbox),
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

## Environments

| Environment | Base URL |
|---|---|
| `zeam.EnvironmentProduction` | `https://api.zeam.app` |
| `zeam.EnvironmentStaging` | `https://api.staging.zeam.app` |
| `zeam.EnvironmentSandbox` | `https://api.sandbox.zeam.app` |
| `zeam.EnvironmentCustom(url)` | partner-supplied |

## Version compatibility

The SDK declares `zeam.MinGatewayVersion`. On the first call, `Client.Ping`
compares it against the gateway's `/healthz` `version` field and returns
`zeam.ErrIncompatibleGateway` if the gateway is older. Opt out during
sandbox development with `zeam.WithSkipVersionCheck()`.

## Next steps

- [Authentication](auth.md) — get a Business or Connect session.
- [Recipes](recipes.md) — full workflows in one call.
- [Error handling](errors.md) — canonical codes and `errors.Is` patterns.
