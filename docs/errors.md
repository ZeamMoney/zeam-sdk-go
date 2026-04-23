# Error handling

Every SDK call that talks to the gateway returns either `nil` or a
`*zeam.Error`. The error is mapped from the gateway's canonical error
envelope (ADR 0003 amendment 2026-04-22) and carries:

- `Code` — the stable, machine-readable code (e.g. `invalid_token`,
  `missing_amount`, `upstream_timeout`).
- `Kind` — the classified `zeam.Kind` (validation/auth/authz/not_found/
  conflict/transient/remote).
- `Status` — the HTTP status code.
- `RequestID` — the gateway's `X-Request-Id` value; surface it to
  end users for support triage.
- `Message` — the gateway's human-readable message; ONLY included in
  `err.Error()` when `zeam.WithVerboseErrors()` is set.
- `Details` — structured upstream context (e.g. `retry_after_ms`,
  `upstream_status`).

## Matching patterns

```go
if errors.Is(err, zeam.KindAuth) {
    // session expired or revoked; call ConnectLogin / LoginOTP again.
}
if errors.Is(err, zeam.KindTransient) {
    // retry-after is already applied by the SDK transport.
    return
}
var zerr *zeam.Error
if errors.As(err, &zerr) && zerr.Code == "invalid_stellar_secret" {
    // the seed the partner supplied was malformed; rotate.
}
```

## Surfacing to end users

Never surface `zerr.Message` directly — it may be upstream copy tuned
for internal operators. Surface:

- The **canonical code** if you have UX that maps it.
- A short category-appropriate message, plus **`zerr.RequestID`** so
  support can find the request in the gateway logs.
```go
fmt.Sprintf("Sorry — %s failed (ref %s).", action, zerr.RequestID)
```

## HTTP-status mapping

| Status | `zeam.Kind` |
|---|---|
| 400, 422 | `KindValidation` |
| 401 | `KindAuth` |
| 403 | `KindAuthz` |
| 404 | `KindNotFound` |
| 409 | `KindConflict` |
| 408, 429, 503, 504 | `KindTransient` |
| 5xx (other) | `KindRemote` |
