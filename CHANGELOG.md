# Changelog

All notable changes to the Zeam Go SDK are documented in this file. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

SDK version `vA.B.C` targets gateway version `vA.B.≥0`. See
[docs/versioning.md](docs/versioning.md) for the full parity policy.

## [Unreleased]

### Added

- Initial scaffold: `zeam.Client`, options, canonical error kinds, environments.
- `transport/` package with envelope unwrap, redaction, retry, and
  observability hooks.
- `auth/` package with Business OTP and SEP-10 flows, `TokenStore` interface,
  `MemoryStore`, and single-flight autorefresh.
- `stellar/` narrow wrapper exposing `Keypair`, `Signer`, and `Asset`.
- `client/` sub-packages for health, business, application, connect,
  payments, and reports.
- `recipes/` package: `LoginOTP`, `RegisterApplication`, `ConnectLogin`,
  `ConnectPayment` (9-step off-ramp), `RotateCredential`, and
  `QuoteThenExecute`.
- `webhook/` package with HMAC verification, constant-time comparison,
  clock-skew bounds, and replay cache.
- Example programs under `examples/`.
- MkDocs documentation scaffold under `docs/`.

### Security

- Default `TokenStore` is memory-only; persistence requires an explicit
  insecure opt-in.
- Redactor strips bearer tokens, `x-zeam-auth`, `Set-Cookie`, JWTs, and
  Stellar seed / public key patterns before any user-supplied logger sees
  them.
- `client/connect.Exec` rejects absolute URIs and restricts the path
  regex to guard against SSRF abuse.

[Unreleased]: https://github.com/ZeamMoney/zeam-sdk-go/compare/v0.0.0...HEAD
