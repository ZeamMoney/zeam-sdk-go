# Versioning policy

SDK `vA.B.C` targets gateway `vA.B.≥0`. The SDK contract and the
gateway's `/v1/*` contract move together.

## Parity mechanisms

1. **Shared version axis** — cutting a gateway `A.B` cuts an SDK `A.B`
   in the same week.
2. **Compile-time pin** — `zeam.MinGatewayVersion` is set at release
   and verified against the SHA of the bundled `api/openapi.yaml`.
3. **Runtime handshake** — `Client.Ping` compares the gateway's
   `/healthz` `version` against `MinGatewayVersion` on first use and
   returns `zeam.ErrIncompatibleGateway` on mismatch.
4. **OpenAPI single source of truth** — the gateway owns
   `docs/openapi.yaml`; the SDK syncs it at release time.
5. **Automated sync + reciprocal contract gate** — a cross-repo
   `repository_dispatch` workflow regenerates `internal/wire/` and
   opens a PR in the SDK for every new gateway tag. The gateway repo
   runs the SDK's contract suite on every gateway PR.

## Compatibility rules

- Public API additions → minor bump.
- Public API removals or signature changes → major bump; a `release/vN`
  branch retains the previous major for ≥6 months.
- Deprecations remain available for at least one full minor cycle and
  carry a `// Deprecated:` godoc marker pointing at the replacement.
- `internal/` and `transport/` package changes are **never** considered
  breaking.

## Upgrading

- Minor upgrades: `go get github.com/ZeamMoney/zeam-sdk-go@latest`; run
  `go mod tidy` and your existing tests. Breaking changes at minor
  versions are a release-engineering bug — open an issue.
- Major upgrades: check the CHANGELOG's **Removed** / **Changed**
  sections and the matching migration guide linked from the release
  notes.
