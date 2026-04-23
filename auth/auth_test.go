package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryStoreRoundTrip(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	if _, err := store.Get(ctx, TrackBusiness); !errors.Is(err, ErrNoSession) {
		t.Fatalf("expected ErrNoSession, got %v", err)
	}

	sess := NewSession(TrackBusiness, "id-1", "rt-1", time.Now().Add(time.Hour), "user@example.com")
	if err := store.Put(ctx, sess); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, err := store.Get(ctx, TrackBusiness)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.IDToken() != "id-1" {
		t.Errorf("id token: got %q want id-1", got.IDToken())
	}

	// Putting an unknown-track session is a contract violation.
	bad := NewSession(TrackUnknown, "x", "y", time.Now(), "")
	if err := store.Put(ctx, bad); err == nil {
		t.Error("expected unknown-track put to fail")
	}

	// Replacing erases the previous session's tokens.
	replacement := NewSession(TrackBusiness, "id-2", "rt-2", time.Now().Add(time.Hour), "user@example.com")
	if err := store.Put(ctx, replacement); err != nil {
		t.Fatalf("replace: %v", err)
	}
	if sess.IDToken() != "" {
		t.Errorf("previous session should have been erased, got idToken %q", sess.IDToken())
	}

	if err := store.Delete(ctx, TrackBusiness); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Get(ctx, TrackBusiness); !errors.Is(err, ErrNoSession) {
		t.Fatalf("expected ErrNoSession after delete, got %v", err)
	}

	_ = store.Close()
}

func TestSessionNeedsRefresh(t *testing.T) {
	now := time.Now()
	sess := NewSession(TrackBusiness, "id", "rt", now.Add(time.Minute), "")
	if !sess.NeedsRefresh(now, 5*time.Minute) {
		t.Error("session within threshold should need refresh")
	}
	sess2 := NewSession(TrackBusiness, "id", "rt", now.Add(time.Hour), "")
	if sess2.NeedsRefresh(now, 5*time.Minute) {
		t.Error("session outside threshold should NOT need refresh")
	}
}

type stubRefresher struct {
	calls int
	delay time.Duration
}

func (s *stubRefresher) Refresh(_ context.Context, sess *Session) (*Session, error) {
	s.calls++
	time.Sleep(s.delay)
	sess.Update("id-new", "rt-new", time.Now().Add(time.Hour))
	return sess, nil
}

func TestAutoRefresherSingleFlight(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	old := NewSession(TrackBusiness, "id-old", "rt-old", time.Now().Add(-time.Minute), "")
	_ = store.Put(ctx, old)

	r := &stubRefresher{delay: 10 * time.Millisecond}
	ar := NewAutoRefresher(store, r, 5*time.Minute)

	done := make(chan error, 5)
	for i := 0; i < 5; i++ {
		go func() {
			_, err := ar.Ensure(ctx, TrackBusiness)
			done <- err
		}()
	}
	for i := 0; i < 5; i++ {
		if err := <-done; err != nil {
			t.Fatalf("ensure: %v", err)
		}
	}
	if r.calls != 1 {
		t.Errorf("expected 1 refresh call, got %d", r.calls)
	}

	sess, _ := store.Get(ctx, TrackBusiness)
	if sess.IDToken() != "id-new" {
		t.Errorf("expected refreshed token, got %q", sess.IDToken())
	}
}

func TestExpiryFromExpiresIn(t *testing.T) {
	ts, err := expiryFromExpiresIn("3600")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if ts.Before(time.Now().Add(30 * time.Minute)) {
		t.Errorf("expiry too soon: %v", ts)
	}
	if _, err := expiryFromExpiresIn(""); err == nil {
		t.Error("empty expiresIn should error")
	}
	if _, err := expiryFromExpiresIn("abc"); err == nil {
		t.Error("non-numeric expiresIn should error")
	}
	if _, err := expiryFromExpiresIn("0"); err == nil {
		t.Error("zero expiresIn should error")
	}
}

func TestTrackString(t *testing.T) {
	if TrackBusiness.String() != "business" {
		t.Errorf("TrackBusiness.String() = %q", TrackBusiness.String())
	}
	if TrackConnect.String() != "connect" {
		t.Errorf("TrackConnect.String() = %q", TrackConnect.String())
	}
	if TrackUnknown.String() != "unknown" {
		t.Errorf("TrackUnknown.String() = %q", TrackUnknown.String())
	}
}
