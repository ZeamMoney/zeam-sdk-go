# Security Policy

The Zeam Go SDK is a public, security-critical library. The team treats
security as a first-class design principle. This document captures:

1. The threat model the SDK is built against.
2. How to report a vulnerability.
3. What is and is not part of the stability contract.
4. Baseline expectations for integrators.

## Reporting a vulnerability

Email **security@zeam.app** with a description, reproduction, and impact. Do
**not** open public issues for security problems. The team will acknowledge
within one business day and target a fix within 30 days for high-severity
issues. We honour a 90-day coordinated-disclosure window by default.

Please include:

- SDK version (`zeam.Version`).
- Minimum reproduction (Go code + relevant gateway endpoint).
- Whether you need credentials redacted from the report.

We publish security advisories at <https://github.com/ZeamMoney/zeam-sdk-go/security/advisories>
once fixes ship.

## Threat model

Assets the SDK protects:

- **One-time application secrets** returned by `POST /v1/application`:
  `stellar.secret`, `connectSecret`, `apiKey.secret`, `webhookSecret.secret`.
- **Bearer tokens** issued by OTP or SEP-10 (`idToken`, `refreshToken`,
  `access_token`).
- **OTP codes** delivered out-of-band to end users.
- The integrity of every inbound webhook payload.

Adversaries considered:

- Passive on-path attackers.
- Partners with write access to logs, tracing, or crash-reporting pipelines who
  might inadvertently capture secrets.
- Malicious partners attempting SSRF, replay, or cross-track reuse via public
  SDK entry points.
- Supply-chain attackers targeting transitive dependencies or release artefacts.

## Defensive posture

| Control | Mechanism |
|---|---|
| Secret-in-memory only | Default `TokenStore` is `MemoryStore`; disk persistence requires explicit `WithInsecureFileStore()` which prints a stderr warning. |
| Redaction | All request/response payloads pass through `transport.Redactor` before any user-supplied logger / tracer sees them. The redactor strips `Authorization`, `x-zeam-auth`, `Set-Cookie`, JWT-shaped strings, and Stellar seed (`S…`) / public key (`G…`) patterns. |
| Transport | TLS 1.3 minimum on the default HTTP client. Plain `http://` environments require `WithInsecureTransport()` and the `ZEAM_SDK_ALLOW_INSECURE=1` environment variable. |
| Cert pinning | Opt-in via `zeam.WithPinnedRootCAs`. |
| Replay protection | `Idempotency-Key` is generated for every mutation and reused across retries. Webhook verification includes a replay cache and clock-skew bound. |
| Cross-track guard | `auth.Session` carries a `Track` enum; mixing a Business token into a Connect call is refused at runtime and via distinct typed session wrappers. |
| Race-safe refresh | Token refresh is `singleflight`-gated to avoid racing requests corrupting the stored tokens. |
| SSRF guard | `client/connect.Exec` validates the path against a restrictive regex and refuses absolute URIs. |
| Secure error messages | `errors.Error.Error()` surfaces only the canonical `code` + `request_id`; upstream `message` is only exposed when `WithVerboseErrors()` is set. |

## Stability contract

The **public API** of the SDK is:

- `github.com/ZeamMoney/zeam-sdk-go` (top-level types, `New`, options, errors,
  environment).
- `.../auth`, `.../stellar`, `.../recipes`, `.../webhook`.
- `.../client/*` sub-packages.

Everything under `internal/` and `transport/` is **private**. Depending on
private packages is unsupported; they may change without notice and changes
are explicitly **not** considered breaking under semver.

## Expectations of integrators

- Store the one-time credentials issued at registration in a cloud secret
  manager or HSM — never in a file committed to source control.
- Rotate credentials on suspicion of compromise via
  `recipes.RotateCredential`.
- Run with an accurate system clock (NTP / chrony) — SEP-10 signatures are
  time-bound.
- Allow-list gateway egress only; the SDK does not talk to any other host
  under the `EnvironmentProduction` setting.

See [docs/operational.md](docs/operational.md) for the full operational
recommendations.
