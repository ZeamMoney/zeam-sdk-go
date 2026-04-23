# Local development & testing

## Running against a local gateway

```bash
ZEAM_SDK_ALLOW_INSECURE=1 go run ./examples/login_otp
```

```go
client, err := zeam.New(
    zeam.WithEnvironment(zeam.EnvironmentCustom("http://localhost:8080")),
    zeam.WithInsecureTransport(),
    zeam.WithSkipVersionCheck(),
)
```

The combination of `WithInsecureTransport()` and
`ZEAM_SDK_ALLOW_INSECURE=1` is the only way to use a plain-HTTP URL;
the SDK prints a stderr warning on every client constructed this way.

## Unit tests

```bash
make test-unit
```

Every package under `/auth`, `/transport`, `/stellar`, `/webhook`, and
the top-level has httptest-based unit tests. The fake gateway helpers
live in [`test/fake`](../test/fake/fake.go).

## Integration tests

```bash
make test-integration
```

Build tag: `integration`. Uses httptest fakes but exercises more of the
transport stack end-to-end.

## Contract tests against staging

```bash
ZEAM_API_URL=https://api.staging.zeam.app \
ZEAM_CONTRACT_TESTS=1 \
ZEAM_CONTRACT_TOKEN=<bearer> \
make test-contract
```

Never run in default CI without explicit intent — these tests exercise
the gateway directly. The GitHub Actions workflow
`.github/workflows/contract-staging.yml` provides OIDC-federated
credentials for the staging sandbox.

## Writing a test with the fake gateway

```go
srv := fake.NewServer([]fake.Route{
    {Method: http.MethodGet, Path: "/healthz", Handler: func(w http.ResponseWriter, r *http.Request) {
        _ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "version": "0.1.0"})
    }},
})
defer srv.Close()

t.Setenv("ZEAM_SDK_ALLOW_INSECURE", "1")
client, _ := zeam.New(
    zeam.WithEnvironment(zeam.EnvironmentCustom(srv.URL())),
    zeam.WithInsecureTransport(),
)
```
