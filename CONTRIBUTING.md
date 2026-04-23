# Contributing to zeam-sdk-go

Thanks for contributing. The goal of this document is to get you from
clone to first passing test in **under ten minutes**.

## Prerequisites

- Go 1.26.2 or later.
- `make` (optional but recommended).
- A local `zeam-api-gateway.go` on `http://localhost:8080` if you want to
  run contract tests. Unit and integration tests do not need the gateway.

## Quick start

```bash
git clone https://github.com/ZeamMoney/zeam-sdk-go.git
cd zeam-sdk-go

make lint       # golangci-lint, gofumpt
make test-unit  # fast unit tests, no build tags
make test-race  # -race enabled

# optional — requires a running local gateway
ZEAM_API_URL=http://localhost:8080 \
ZEAM_CONTRACT_TESTS=1 \
make test-contract
```

## Project layout

- `zeam.go`, `options.go`, `errors.go`, `environment.go` — top-level API.
- `auth/` — OTP + SEP-10 authentication flows, token store, refresh.
- `transport/` — HTTP plumbing (retry, redaction, envelope unwrap, otel).
- `stellar/` — the **only** package allowed to import
  `github.com/stellar/go-stellar-sdk`.
- `client/<domain>/` — 1:1 typed wrappers for gateway endpoints.
- `recipes/` — opinionated multi-step workflows.
- `webhook/` — inbound webhook HMAC verification.
- `examples/` — runnable examples. Each subdirectory is a `main` package.
- `internal/` — implementation details, not part of the public API.
- `test/contract/` — build-tagged contract tests exercising a live gateway.
- `test/fake/` — httptest-based fakes used by unit tests.
- `api/openapi.yaml` — gateway-owned contract, synced via `make sync-spec`.

## Code guidelines

- Every exported symbol has a godoc comment that starts with its name.
- No package may import `internal/` packages across module boundaries.
- `stellar/go-stellar-sdk` is only imported from `stellar/`. Everything
  else uses the wrapper types.
- `transport/redaction.go` runs **before** any user-supplied logger. When
  you add a new sensitive field, update `internal/redact/denylist.go`.
- Every public call takes `ctx context.Context` as its first argument.
- Every mutating call generates or propagates an `Idempotency-Key`.
- Tests use table-driven form. Integration tests (`//go:build integration`)
  and contract tests (`//go:build contract`) are kept out of the default
  `go test ./...` matrix.

## Pull request checklist

- [ ] `make lint` is clean.
- [ ] `make test-race` passes.
- [ ] New public symbols have godoc.
- [ ] CHANGELOG entry added under `[Unreleased]`.
- [ ] If the change touches wire shape: the accompanying gateway PR is
      linked and the `api/CHANGELOG.md` entry matches.

## Signing commits

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/).
All commits are DCO-signed (`git commit -s`). Release tags are cosign-signed
by the maintainers.

## Code of conduct

See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
