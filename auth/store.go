package auth

import (
	"context"
	"errors"
	"sync"
)

// TokenStore persists Sessions by [Track]. Implementations must be safe
// for concurrent use; the SDK reads and writes from multiple goroutines
// during auto-refresh.
//
// Key design rule: a store MUST isolate sessions by Track. Saving a
// TrackBusiness session under TrackConnect (or vice versa) is a contract
// violation.
type TokenStore interface {
	Put(ctx context.Context, sess *Session) error
	Get(ctx context.Context, track Track) (*Session, error)
	Delete(ctx context.Context, track Track) error
	Close() error
}

// ErrNoSession is returned when no session exists for the requested track.
var ErrNoSession = errors.New("auth: no session for track")

// MemoryStore is an in-memory TokenStore. It is the default; the SDK
// never writes tokens to disk without an explicit opt-in.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[Track]*Session
}

// NewMemoryStore constructs an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sessions: make(map[Track]*Session)}
}

// Put saves sess under sess.Track().
func (m *MemoryStore) Put(_ context.Context, sess *Session) error {
	if sess == nil {
		return errors.New("auth: nil session")
	}
	if sess.Track() == TrackUnknown {
		return errors.New("auth: session has unknown track")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.sessions[sess.Track()]; ok && existing != sess {
		existing.Erase()
	}
	m.sessions[sess.Track()] = sess
	return nil
}

// Get returns the current session for the given track.
func (m *MemoryStore) Get(_ context.Context, track Track) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[track]
	if !ok {
		return nil, ErrNoSession
	}
	return sess, nil
}

// Delete removes the session for the given track and erases its tokens.
func (m *MemoryStore) Delete(_ context.Context, track Track) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sess, ok := m.sessions[track]; ok {
		sess.Erase()
		delete(m.sessions, track)
	}
	return nil
}

// Close erases every session and empties the store.
func (m *MemoryStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, sess := range m.sessions {
		sess.Erase()
	}
	m.sessions = map[Track]*Session{}
	return nil
}
