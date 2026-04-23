# Security

This page summarises the SDK's security posture. The authoritative
source is [`SECURITY.md`](https://github.com/ZeamMoney/zeam-sdk-go/blob/main/SECURITY.md)
at the repo root.

## Reporting a vulnerability

Email **security@zeam.app**. Do not open public issues.

## Defensive defaults

- `auth.MemoryStore` is the default token store; disk persistence
  requires `zeam.WithInsecureFileStore()` and prints a warning.
- TLS 1.3 minimum on the default HTTP client.
- `transport.Redact` removes bearer tokens, `x-zeam-auth`, `Set-Cookie`,
  JWTs, and Stellar seed / public-key patterns from every event
  **before** it reaches a user-supplied logger.
- `client/connect.Exec` validates the path against a strict regex and
  rejects absolute URIs.
- Webhook verification is constant-time, clock-skew-bounded, and
  replay-protected via the LRU cache.
- Every mutation carries a generated `Idempotency-Key` that's reused on
  retries.
- Cross-track guards refuse Business↔Connect token reuse with
  `auth.ErrWrongTrack`.

## Stability contract

- **Public** packages: top-level `zeam`, `auth`, `stellar`, `recipes`,
  `webhook`, `client/*`.
- **Private** packages: `internal/*`, `transport/*`. Changes here are
  **never** considered breaking under semver.

## Expectations of integrators

See [Operational patterns](operational.md):

- Store one-time credentials in a cloud secret manager or HSM.
- Keep clocks synced.
- Run as non-root with a read-only FS.
- Rotate credentials via `recipes.RotateCredential` on any suspected
  compromise.
