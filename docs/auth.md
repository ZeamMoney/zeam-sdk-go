# Authentication

The gateway has two independent authentication tracks:

- **Business** — end-user OTP / Firebase. Used by `/v1/business/*`,
  `/v1/application/*`, `/v1/payments`, `/v1/reports/*`.
- **Connect** — per-application SEP-10. Used by `/v1/connect-*`.

**Tokens are not interchangeable.** The SDK refuses to attach a Business
token to a Connect call (and vice versa) via the `auth.Track` type.

## Business OTP

```go
sess, err := recipes.LoginOTP(ctx, client, recipes.LoginOTPInput{
    MobileNumber: "+27821234567",
    AskCode: func(ctx context.Context, hint recipes.OTPHint) (string, error) {
        // show the masked destination to the user, read the code.
        return askUser(hint.MaskedDestination), nil
    },
})
```

Under the hood the recipe:

1. `POST /v1/public/auth/request-otp` → `{ requestId, maskedDestination, expiresAt }`
2. The partner-provided `AskCode` callback collects the code.
3. `POST /v1/public/auth/verify-otp` → `{ idToken, refreshToken, expiresIn, customToken }`
4. The session is stored in the client's `auth.TokenStore`
   (default: `auth.MemoryStore`).

## Connect SEP-10

```go
sess, err := recipes.ConnectLogin(ctx, client, recipes.ConnectLoginInput{
    StellarSeed: vaultStellarSeed,
    PublicKey:   appPublicKey,
})
```

Under the hood:

1. `GET /auth-connect?account=<publicKey>` → challenge XDR.
2. Sign the challenge locally with the seed (the seed never leaves the
   caller's process).
3. `POST /auth-connect` with the signed XDR → `{ idToken, refreshToken, expiresIn }`.
4. Persist the session.

## Refresh lifecycle

Refresh tokens are **single-use**. The SDK's `auth.AutoRefresher` uses
`singleflight` so concurrent callers coalesce into one refresh. Rehydrate
the refresher on application start if you persist sessions across
restarts.

## Track isolation — enforced by the SDK

```go
// This panics at type level long before any request is built:
// client/connect expects an auth.TrackConnect session; passing a
// TrackBusiness session returns auth.ErrWrongTrack.
connects := client.Connect()
_, err := connects.QueryConnectors(ctx, businessSess, input)
// err == auth.ErrWrongTrack
```
