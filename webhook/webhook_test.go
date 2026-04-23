package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

const secret = "super-secret-do-not-log"

func signBody(ts int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.", ts)
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func buildRequest(t *testing.T, body []byte, ts int64, includeSig, includeTS, includeEvt bool) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	if includeSig {
		req.Header.Set(HeaderSignature, signBody(ts, body))
	}
	if includeTS {
		req.Header.Set(HeaderTimestamp, strconv.FormatInt(ts, 10))
	}
	if includeEvt {
		req.Header.Set(HeaderEventID, "evt-1")
	}
	return req
}

func TestVerifyHappyPath(t *testing.T) {
	body := []byte(`{"event":"payment.succeeded"}`)
	ts := time.Now().Unix()
	req := buildRequest(t, body, ts, true, true, false)

	got, err := Verify(req, []byte(secret))
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("body mismatch: got %q want %q", got, body)
	}
}

func TestVerifyRejectsStale(t *testing.T) {
	body := []byte(`{"event":"payment.succeeded"}`)
	ts := time.Now().Add(-1 * time.Hour).Unix()
	req := buildRequest(t, body, ts, true, true, false)

	_, err := Verify(req, []byte(secret))
	if !errors.Is(err, ErrStaleTimestamp) {
		t.Fatalf("expected ErrStaleTimestamp, got %v", err)
	}
}

func TestVerifyRejectsBadSig(t *testing.T) {
	body := []byte(`{"event":"payment.succeeded"}`)
	ts := time.Now().Unix()
	req := buildRequest(t, body, ts, true, true, false)
	req.Header.Set(HeaderSignature, "00deadbeef")

	_, err := Verify(req, []byte(secret))
	if !errors.Is(err, ErrBadSignature) {
		t.Fatalf("expected ErrBadSignature, got %v", err)
	}
}

func TestVerifyMissingHeaders(t *testing.T) {
	body := []byte(`{}`)
	ts := time.Now().Unix()

	if _, err := Verify(buildRequest(t, body, ts, false, true, false), []byte(secret)); !errors.Is(err, ErrMissingSignature) {
		t.Errorf("missing signature: got %v", err)
	}
	if _, err := Verify(buildRequest(t, body, ts, true, false, false), []byte(secret)); !errors.Is(err, ErrMissingTimestamp) {
		t.Errorf("missing timestamp: got %v", err)
	}
}

func TestReplayCacheRejectsDuplicate(t *testing.T) {
	body := []byte(`{"event":"payment.succeeded"}`)
	ts := time.Now().Unix()
	cache := NewLRU(16)

	v := NewVerifier([]byte(secret), WithReplayCache(cache))

	_, err := v.Verify(buildRequest(t, body, ts, true, true, true))
	if err != nil {
		t.Fatalf("first verify: %v", err)
	}
	_, err = v.Verify(buildRequest(t, body, ts, true, true, true))
	if !errors.Is(err, ErrReplay) {
		t.Fatalf("second verify: expected ErrReplay, got %v", err)
	}
}

func TestHandlerWiresStatus(t *testing.T) {
	body := []byte(`{"ok":true}`)
	ts := time.Now().Unix()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ := io.ReadAll(r.Body)
		if !bytes.Equal(got, body) {
			t.Errorf("next received %q, want %q", got, body)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	h := Handler(next, []byte(secret))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, buildRequest(t, body, ts, true, true, false))
	if rec.Code != http.StatusNoContent {
		t.Errorf("happy path status: got %d want 204", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, buildRequest(t, body, ts, false, true, false))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("missing sig status: got %d want 401", rec.Code)
	}
}

func TestLRUBoundsCapacity(t *testing.T) {
	c := NewLRU(2)
	if c.Seen("a", time.Minute) {
		t.Fatal("a should be new")
	}
	if c.Seen("b", time.Minute) {
		t.Fatal("b should be new")
	}
	if c.Seen("c", time.Minute) {
		t.Fatal("c should be new (eviction of a)")
	}
	if c.Seen("a", time.Minute) {
		t.Fatal("a should have been evicted and now be new again")
	}
}
