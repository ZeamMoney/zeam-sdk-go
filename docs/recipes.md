# Recipes

Recipes are opinionated, one-function workflows over the low-level
`client/*` sub-packages. Each is safe to use directly or to drive step-
by-step when you need to interleave partner logic.

## Overview

| Recipe | Purpose |
|---|---|
| `recipes.LoginOTP` | Business OTP login (request + verify + persist). |
| `recipes.RegisterApplication` | `POST /v1/application` with secure one-time-secret capture. |
| `recipes.ConnectLogin` | Full SEP-10 login using a stored Stellar seed. |
| `recipes.ConnectPayment` | 9-step Connect off-ramp orchestration. |
| `recipes.RotateCredential` | Rotate an application API key. |
| `recipes.QuoteThenExecute` | Generic `quote → execute` pattern (generics). |

## `ConnectPayment` — the flagship

```go
flow := recipes.NewConnectPayment(client, recipes.ConnectPaymentInput{
    BusinessSession:      businessSess,
    ApplicationSeed:      vaultStellarSeed,
    ApplicationPublicKey: appPublicKey,
    AssociationID:        "6f…e2",
    WalletID:             "wallet-id",
    FundingAsset:         stellar.MustAsset("USDC:GA…"),
    BeneficiaryID:        "bb…12",
    Method:               "MOBILE_MONEY",
    CountryISO:           "ZW",
    SendAmount:           "100.00",
})

result, err := flow.Do(ctx)
```

Steps (all exported for manual driving):

1. `ListAssociations`
2. `ListWallets`
3. `GetBeneficiary`
4. `SignInConnect` (SEP-10)
5. `DiscoverConnectors`
6. `SelectConnector` + `GetConnectQuote`
7. `RequiresStellarQuote()` + `GetStellarQuote` (conditional)
8. `ExecuteStellarTransaction`
9. `ExecuteConnectPayment`

The SDK applies the strict-receive decision rule from
`docs/CONNECT_PAYMENT_FLOW.md §3` upstream: if `FundingAsset !=
connector.acceptedAsset` after normalisation, Step 7 runs and Step 8
uses `STRICT_PATH_RECEIVE` with the returned `sendMax`.

## `RegisterApplication` — secure capture callback

```go
result, err := recipes.RegisterApplication(ctx, client, recipes.RegisterAppInput{
    Session: businessSess,
    Payload: registrationPayload,
    CaptureOneTimeSecrets: func(ctx context.Context, s recipes.OneTimeSecrets) error {
        return vault.Put(ctx, map[string]string{
            "stellar.secret":  s.StellarSeed,
            "connect.secret":  s.ConnectSecret,
            "api.key":         s.APIKey,
            "webhook.secret":  s.WebhookSecret,
        })
    },
})
```

The SDK zeros the secrets immediately after your callback returns. If it
returns an error, the SDK returns it — your caller knows the credentials
were minted but the partner couldn't store them, and must rotate.
