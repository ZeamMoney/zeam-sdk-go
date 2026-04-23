package auth

import (
	"sync"
	"time"
)

// Session represents a live authenticated session. It is safe for
// concurrent use; all access is gated by a mutex. The raw bearer token
// and refresh token are never exposed via logging helpers, and the
// [Erase] method wipes their underlying byte slices.
type Session struct {
	mu sync.RWMutex

	track        Track
	idToken      []byte
	refreshToken []byte
	expiresAt    time.Time
	subject      string // optional identifier (zeamid, publicKey, etc.)
}

// NewSession constructs a Session from freshly issued tokens. Callers
// receive the Session unlocked; concurrent callers should route through
// the shared TokenStore rather than share Session values directly.
func NewSession(track Track, idToken, refreshToken string, expiresAt time.Time, subject string) *Session {
	return &Session{
		track:        track,
		idToken:      []byte(idToken),
		refreshToken: []byte(refreshToken),
		expiresAt:    expiresAt,
		subject:      subject,
	}
}

// Track returns the authentication track that issued this session.
func (s *Session) Track() Track {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.track
}

// IDToken returns the bearer token. Do NOT log this value; use
// [Session.Fingerprint] for correlation.
func (s *Session) IDToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return string(s.idToken)
}

// RefreshToken returns the current refresh token.
func (s *Session) RefreshToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return string(s.refreshToken)
}

// ExpiresAt returns the current id-token expiry.
func (s *Session) ExpiresAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.expiresAt
}

// Subject returns the human-meaningful owner of the session (zeamid,
// application publicKey, etc.). Free of secret material.
func (s *Session) Subject() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.subject
}

// Fingerprint returns a short, non-reversible identifier for log
// correlation. Safe to include in structured logs.
func (s *Session) Fingerprint() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.idToken) < 8 {
		return ""
	}
	return string(s.idToken[:8]) + "…"
}

// Update atomically replaces the id-token, refresh-token, and expiry
// values. Called by [AutoRefresher] on a successful refresh; zeros the
// previous buffers before swapping.
func (s *Session) Update(idToken, refreshToken string, expiresAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	zero(s.idToken)
	zero(s.refreshToken)
	s.idToken = []byte(idToken)
	s.refreshToken = []byte(refreshToken)
	s.expiresAt = expiresAt
}

// NeedsRefresh reports whether the session's id-token is within the
// supplied threshold of expiry. A threshold of zero means "only refresh
// when already expired".
func (s *Session) NeedsRefresh(now time.Time, threshold time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return now.Add(threshold).After(s.expiresAt)
}

// Erase zeroes the token buffers and resets the track to TrackUnknown.
// After Erase, the session is no longer usable.
func (s *Session) Erase() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	zero(s.idToken)
	zero(s.refreshToken)
	s.idToken = nil
	s.refreshToken = nil
	s.track = TrackUnknown
	s.expiresAt = time.Time{}
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
