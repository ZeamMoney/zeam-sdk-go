package transport

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestUnwrapSuccess(t *testing.T) {
	body := []byte(`{"ok":true,"request_id":"rid-1","resource":"foo","verb":"get","data":{"hello":"world"}}`)
	data, errEnv := Unwrap(http.StatusOK, body)
	if errEnv != nil {
		t.Fatalf("unexpected error envelope: %+v", errEnv)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if got["hello"] != "world" {
		t.Fatalf("got %q want %q", got["hello"], "world")
	}
}

func TestUnwrapBareJSONSuccess(t *testing.T) {
	body := []byte(`{"idToken":"abc","refreshToken":"def","expiresIn":"3600"}`)
	data, errEnv := Unwrap(http.StatusOK, body)
	if errEnv != nil {
		t.Fatalf("unexpected error envelope")
	}
	if !strings.Contains(string(data), `"idToken"`) {
		t.Fatalf("bare JSON passthrough lost data")
	}
}

func TestUnwrapError(t *testing.T) {
	body := []byte(`{"ok":false,"request_id":"rid-2","errors":[{"code":"invalid_token","message":"nope","details":{"k":"v"}}]}`)
	_, errEnv := Unwrap(http.StatusUnauthorized, body)
	if errEnv == nil {
		t.Fatal("expected error envelope, got nil")
	}
	if errEnv.Code != "invalid_token" {
		t.Errorf("code: got %q want invalid_token", errEnv.Code)
	}
	if errEnv.RequestID != "rid-2" {
		t.Errorf("request id: got %q want rid-2", errEnv.RequestID)
	}
	if errEnv.Status != http.StatusUnauthorized {
		t.Errorf("status: got %d want 401", errEnv.Status)
	}
	if errEnv.Details["k"] != "v" {
		t.Errorf("details lost: %+v", errEnv.Details)
	}
}

func TestRedactStripsSeedAndJWT(t *testing.T) {
	seed := "SABCDEFGHIJKLMNOPQRSTUVWXYZ234567ABCDEFGHIJKLMNOPQRSTUVW"
	jwt := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.abc"
	attrs := map[string]any{
		"note":          "please keep the " + seed + " around",
		"jwt":           jwt,
		"Authorization": "Bearer " + jwt,
		"nested": map[string]any{
			"stellarSecret": seed,
			"safe":          "hello",
		},
	}
	out := Redact(attrs)
	if strings.Contains(out["note"].(string), seed) {
		t.Fatalf("seed leaked through note: %q", out["note"])
	}
	if out["Authorization"] != "<redacted>" {
		t.Fatalf("authorization header not redacted: %q", out["Authorization"])
	}
	nested := out["nested"].(map[string]any)
	if nested["stellarSecret"] != "<redacted>" {
		t.Fatalf("stellar secret key not redacted: %q", nested["stellarSecret"])
	}
	if nested["safe"] != "hello" {
		t.Fatalf("non-sensitive key got redacted: %q", nested["safe"])
	}
}

func TestFingerprint(t *testing.T) {
	got := Fingerprint("eyJhbGciOiJIUzI1NiJ9.short")
	if !strings.HasPrefix(got, "eyJhbGci") {
		t.Fatalf("fingerprint should preserve the first 8 chars, got %q", got)
	}
	if strings.Contains(got, ".") {
		t.Fatalf("fingerprint must not leak beyond the first 8 chars: %q", got)
	}
	short := Fingerprint("abc")
	if !strings.HasPrefix(short, "len=") {
		t.Fatalf("short token fingerprint should indicate length only: %q", short)
	}
}

func TestIsMutating(t *testing.T) {
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		if !isMutating(m) {
			t.Errorf("%s should be mutating", m)
		}
	}
	for _, m := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		if isMutating(m) {
			t.Errorf("%s should NOT be mutating", m)
		}
	}
}

func TestRetryableStatus(t *testing.T) {
	for _, c := range []int{408, 429, 502, 503, 504} {
		if !retryableStatus(c) {
			t.Errorf("status %d should be retryable", c)
		}
	}
	for _, c := range []int{200, 400, 401, 403, 404, 500} {
		if retryableStatus(c) {
			t.Errorf("status %d should NOT be retryable", c)
		}
	}
}
