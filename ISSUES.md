# Tracer Bullet — Issues Log

Open issues discovered while validating the Go SDK against the API Gateway tracer bullet flow.

## #1 — Register application: 502 from origin

- **Step**: 3 (Register application)
- **Endpoint**: `POST /v1/application`
- **Symptom**: Cloudflare returns 502 Bad Gateway. The `association-applications.zeam.go` origin service is down or misconfigured. Other gateway endpoints (auth, business) work fine.
- **Impact**: Cannot register applications via SDK or any client.
- **Owner**: Backend team.

## #2 — Beneficiary payment destination missing `country_iso`

- **Step**: 7 (Connectors for beneficiary)
- **Endpoint**: `GET /v1/business/beneficiaries/{associationId}`
- **Symptom**: Payment destination has `country_iso: ""` — the beneficiary was created without a country on the destination. The SDK's connector query rejects empty country.
- **Impact**: Step 7 cannot use beneficiary destination country for connector lookup.
- **Workaround**: Tracer bullet falls back to `ZA`.
- **Owner**: Data / onboarding team — destinations should require country.

## #3 — No dedicated "connectors for beneficiary" endpoint

- **Step**: 7 (Connectors for beneficiary)
- **Symptom**: The SDK reuses the general `QueryConnectors(countryISO, method)` endpoint. There's no gateway endpoint that takes a beneficiary ID and returns matching connectors.
- **Impact**: The caller must manually extract country + method from the beneficiary's destination and re-query.
- **Owner**: Gateway team — discuss at check-in whether a dedicated endpoint is needed.
