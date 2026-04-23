// Package recipes bundles opinionated, end-to-end workflows over the
// low-level client/ sub-packages. A recipe is a small piece of state
// plus a sequence of named steps the caller can drive individually (to
// interleave partner-specific logic) or via a single .Do(ctx) call.
//
// Ship-day recipes:
//
//   - [LoginOTP]            — Business OTP flow (request + verify)
//   - [RegisterApplication] — Register an integrator application and
//     capture its one-time credentials via a partner-provided callback.
//   - [ConnectLogin]        — Full SEP-10 login using a stored Stellar seed.
//   - [ConnectPayment]      — 9-step off-ramp payment orchestration.
//   - [RotateCredential]    — API key / webhook secret rotation.
//   - [QuoteThenExecute]    — Generic "quote → execute" helper.
//
// All recipes are safe for concurrent use across distinct Input values.
// Each recipe documents the specific gateway ADR it mirrors.
package recipes
