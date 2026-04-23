# SDK-0001 — Repo, versioning, and parity
- Status: Accepted
- Date: 2026-04-23
- Phase: SDK v1 scaffold

## Context

Partners and first-party clients (mobile, web, `zeam-cli.go`)
re-implement the gateway contract by hand today. Every integrator
repeats the same mistakes (leaking seeds, persisting bearer tokens in
env files, mixing tracks, skipping `Idempotency-Key`, ignoring
`requiresQuote`). We need a single, opinionated, secure-by-default Go
SDK that is distributed publicly and moves in lockstep with the
gateway.

## Decision

### Repo strategy
- Gateway: `github.com/BeamMoney/zeam-api-gateway.go` (private).
- SDK: `github.com/ZeamMoney/zeam-sdk-go` (public, Apache-2.0, new
  `ZeamMoney` GitHub org).
- Separate repos preferred over a monorepo because (a) the gateway is
  private and carries internal-only infra, (b) Go module paths pin
  to repo paths, (c) public CI should carry zero internal credentials.

### Versioning
- SemVer. SDK `vA.B.C` targets gateway `vA.B.≥0`.
- `zeam.MinGatewayVersion` encodes the lowest gateway known-good at
  release time and is verified at compile time against the SHA of the
  bundled OpenAPI spec.
- Runtime handshake: `Client.Ping` inspects the gateway's `/healthz`
  `version` and returns `zeam.ErrIncompatibleGateway` on mismatch.

### Parity mechanisms
1. Shared version axis (above).
2. Compile-time pin on `MinGatewayVersion` + `SpecHash`.
3. Runtime handshake via `/healthz`.
4. OpenAPI single source of truth owned by the gateway.
5. Automated cross-repo `repository_dispatch` that opens an
   `api-sync/vX.Y.Z` PR in the SDK for every new gateway tag; gateway
   CI runs the SDK's contract suite on every gateway PR and fails on
   drift.

### Release management
- Cut via `goreleaser` on tag push.
- Each release publishes: source archive, signed SLSA provenance,
  cosign-signed tags, static docs rebuild to `sdk.zeam.app`.
- `CHANGELOG.md` is authoritative; GitHub Releases copy its body.

### Backward compatibility
- Additions → minor. Removals / signature changes → major.
- `internal/` and `transport/` are NEVER breaking.
- `apidiff` runs on every PR; incompatible changes fail CI without a
  `!breaking` label and an ADR reference.

### Distribution
- Apache-2.0.
- Released via Go module proxy. No binary distribution — this is a
  library.

## Consequences
- Partners have one canonical, versioned entry point.
- Contract drift fails loudly in both repos' CI.
- Public surface is deliberately narrow: `zeam`, `auth`, `stellar`,
  `recipes`, `webhook`, `client/*`.
- Private packages (`internal/*`, `transport/*`) are free to evolve
  between minor releases without breaking partners.

## Amendments

*Append dated entries when the pattern evolves.*
