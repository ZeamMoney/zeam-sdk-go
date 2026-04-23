# Operational patterns

The SDK is safe by default but cannot by itself guarantee a safe
deployment. This page captures the infrastructure expectations the
team recommends for partners running the SDK in production.

## Secret storage

| Class | Recommended | Acceptable | Discouraged |
|---|---|---|---|
| **One-time secrets** (`stellar.secret`, `connectSecret`, `apiKey.secret`, `webhookSecret.secret`) | Cloud secret manager (Azure Key Vault, AWS Secrets Manager, GCP Secret Manager, HashiCorp Vault) | OS keychain via `auth.KeyringStore` | `.env`, plaintext files, git history |
| **Issued bearer / refresh tokens** | `auth.MemoryStore` (default) | Durable secret manager for long-lived daemons | Disk without encryption |
| **Gateway URLs, integrator IDs** | Non-secret config file / env var | Build-time constants | — |

## Environment variables

The SDK reads exactly one env var: `ZEAM_SDK_ALLOW_INSECURE`. Every
other configuration flows through `zeam.Option` so it's explicit in
code. Partners should keep that policy: **the fewer environment
variables the SDK reads, the smaller the blast radius when an operator
misconfigures a container**.

## Rotation

- Use `recipes.RotateCredential` for API-key rotation. The old key
  enters `rotating` state upstream; migrate before the window closes.
- Use `application.Client.RotateWebhookSecret` for webhook HMAC
  rotation. The old secret stays valid while `rotating`; swap your
  verifier once the new secret is stored.
- Treat any suspected compromise as a rotation event — the SDK does
  not detect compromise for you.

## Logging & observability

- Structured logs only. The SDK's `transport.Redact` runs before any
  user-supplied logger, but your app may still emit secrets if you
  log request bodies directly.
- OpenTelemetry tracing is opt-in. Provide a `zeam.WithObservability`
  hook and the SDK emits `http.request` events with request-id,
  latency, and outcome — never bodies or headers.
- For Sentry integration, wire the SDK's observer hook to a Sentry
  breadcrumb emitter and make sure bearer tokens are scrubbed at the
  breadcrumb layer as well.

## Deployment hardening

- Run as a **non-root** user.
- Use a read-only filesystem (Kubernetes
  `securityContext.readOnlyRootFilesystem: true`).
- Set memory limits — the process shouldn't swap; swapped pages
  containing seed material are recoverable.
- Drop Linux capabilities (`CAP_DROP=ALL`).
- Keep NTP / chrony running — SEP-10 signatures are time-bound.

## Incident response

1. Revoke the compromised credential via the platform portal.
2. Call `recipes.RotateCredential` / the webhook-secret rotation.
3. Invalidate any active Firebase sessions server-side.
4. Email `security@zeam.app` with the partner integrator ID and
   suspected blast radius.

## Backup & disaster recovery

Treat SDK runtime state as ephemeral. On recovery, partners should
re-authenticate (OTP / SEP-10) rather than restore cached refresh
tokens. Persistent token caches multiply the blast radius of any
backup compromise.
