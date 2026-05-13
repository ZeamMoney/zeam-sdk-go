# Tracer Bullet — Issues Log

Issues and improvement points identified while validating the Go SDK against the API Gateway tracer bullet flow.

## SDK Improvements Required

### #1 — Improve documentation around capabilities

- The SDK lacks clear documentation on what each client can do and which gateway endpoints it covers.
- External developers cannot tell from the SDK alone which endpoints exist, what the request/response shapes are, or what the expected flow is.
- The tracer bullet exposed multiple cases where JSON field names, envelope shapes, and HTTP status semantics were undocumented or wrong.
- **Action**: Document every endpoint the SDK covers, with request/response examples and expected HTTP status codes (including non-obvious ones like 402 for quotes).

### #2 — Strongly define the clients — decide on clients vs recipes

- The SDK has two layers: low-level typed clients (`client/business`, `client/connect`, `client/payments`) and high-level recipes (`recipes/`). Both try to do the same thing and it's confusing which one to use.
- The recipes assume a specific flow (e.g. `ConnectPayment` bundles 9 steps) but the real integration flow discovered in the tracer bullet doesn't match the recipe's assumptions (e.g. transaction init/load isn't in the recipe, the quote response shape was wrong, track enforcement was wrong).
- **Action**: Pick one approach. Either make clients the primary surface with clear typed methods, or make recipes the primary surface with correct flows. Don't ship both half-baked.

### #3 — Bring request and response objects into the SDK for reference

- Multiple endpoints have no schema in the gateway's OpenAPI spec (`connect-execute`, `connect-quote`, `transaction/load`). The SDK was guessing field names and types.
- The tracer bullet found wrong field names (`connector_id` vs `connectorId`, `association_name` vs `associationName`, `ApplicationName` vs `applicationName`), wrong types (`string` vs `float64` for amounts), wrong envelope shapes (`ok` vs `status`), and missing fields (`Reference`, `RefundAccount` on execute).
- **Action**: Define typed request/response structs for every endpoint, derived from the actual gateway + upstream service contracts. No `any` payloads.

### #4 — Strongly type all payloads and parameters

- Several SDK methods accepted `any` as the request body (`Register`, `CreatePayment`, `CreateQuote`, `CreateOrder`). This forces developers to guess the payload shape and discover errors only at runtime via 400s from the gateway.
- The tracer bullet hit this on registration (`name` vs `ApplicationName` vs `applicationName`), quote execution (missing `Reference`, `RefundAccount`, `TransactionHash`), and connector query (`BANK_TRANSFER` is not a valid method enum).
- **Action**: Every SDK method that sends a body must accept a typed struct. Every response must deserialize into a typed struct. No `map[string]any`, no `json.RawMessage` for primary fields, no `any`.

## Backend / Gateway Issues

### #5 — Register application: 502 from origin

- **Endpoint**: `POST /v1/application`
- **Symptom**: Cloudflare returns 502 Bad Gateway. The `association-applications.zeam.go` origin is down.
- **Owner**: Backend team.

### #6 — Beneficiary payment destination missing `country_iso`

- **Endpoint**: `GET /v1/business/beneficiaries/{associationId}`
- **Symptom**: Payment destinations have empty `country_iso`. Connector queries require a country.
- **Owner**: Data / onboarding team.
