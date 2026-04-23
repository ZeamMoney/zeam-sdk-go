# Token lifecycle

The SDK isolates token storage behind `auth.TokenStore`, which has two
concrete implementations:

- `auth.MemoryStore` (default) — in-memory, never persisted.
- `auth.KeyringStore` (optional, `keyring` build tag) — wraps the OS
  keychain (macOS Keychain, Windows DPAPI, Linux libsecret).

## Choosing a store

- **Short-lived workloads** (containers, serverless, CLI one-shots): use
  `MemoryStore`. Let the user re-authenticate on each cold start.
- **Long-lived desktop tools**: build with `-tags keyring` and use
  `NewKeyringStore`.
- **Server-side integrations with persistent sessions**: back your own
  `TokenStore` onto your secret manager. Implement the four methods in
  `auth.TokenStore` and pass it via `zeam.WithTokenStore`.

!!! warning "Never persist tokens to disk in plaintext"
    The default refuses. A file-backed store is only available via
    `zeam.WithInsecureFileStore()`, which prints a startup warning and is
    intended for local development only.

## Refresh semantics

- Every auth response is a new pair `{idToken, refreshToken}`. The old
  refresh token is **single-use** and invalidated upstream.
- `auth.AutoRefresher` schedules refresh when the current id-token is
  within five minutes of expiry (configurable).
- Concurrent callers coalesce through `singleflight`; a thundering herd
  results in exactly one refresh round-trip.
- If refresh fails with `zeam.KindAuth`, call the appropriate login
  recipe again (`LoginOTP` / `ConnectLogin`).

## Erasure

- `Session.Erase()` zeros the id-token and refresh-token byte slices.
- Replacing a session in `auth.MemoryStore` erases the previous value.
- Session fingerprints (`Session.Fingerprint()`) keep only the first 8
  characters of the id-token — safe for log correlation, never
  reversible.

## Cross-track guard

`Session` carries a `Track` enum. `client/*` sub-packages refuse
sessions of the wrong track with `auth.ErrWrongTrack` before any
request is built.
