// Package redact enumerates the field names and regex patterns the SDK
// considers sensitive. Kept in internal/ so partners cannot depend on
// the specific list; the SDK's public redaction surface lives in
// transport/redaction.go.
package redact

import "regexp"

// Keys is the authoritative list of case-insensitive keys that should
// never appear in logs, traces, or error messages.
var Keys = []string{
	"idtoken",
	"refreshtoken",
	"accesstoken",
	"customtoken",
	"otp",
	"code",
	"secret",
	"stellarsecret",
	"connectsecret",
	"apikey",
	"apikeysecret",
	"webhooksecret",
	"password",
	"pin",
	"otp_session_id",
	"otpsessionid",
	"requestid",
	"authorization",
	"bearer",
	"seed",
	"privatekey",
}

// Patterns is the list of value-shape regexes redacted regardless of
// the key they appear under.
var Patterns = []*regexp.Regexp{
	regexp.MustCompile(`\bS[A-Z2-7]{55}\b`),
	regexp.MustCompile(`\bG[A-Z2-7]{55}\b`),
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`),
	regexp.MustCompile(`\b1_[A-Za-z0-9_-]{16,}\b`),
}
