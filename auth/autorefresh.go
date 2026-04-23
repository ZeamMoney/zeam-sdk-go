package auth

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"
)

// Refresher knows how to refresh a single track's session using its
// existing refresh token. Implementations are supplied by [OTPFlow] and
// [SEP10Flow].
type Refresher interface {
	// Refresh exchanges the session's current refresh token for a new
	// id-token / refresh-token pair. Returns an updated Session (the
	// same pointer with mutated fields) or an error.
	Refresh(ctx context.Context, sess *Session) (*Session, error)
}

// AutoRefresher wraps a Refresher with single-flight semantics so
// concurrent callers coalesce into a single refresh round-trip.
// Refresh is triggered when the id-token is within Threshold of expiry.
type AutoRefresher struct {
	Store     TokenStore
	Refresher Refresher
	Threshold time.Duration // default: 5m

	sf singleflight.Group
}

// NewAutoRefresher constructs an AutoRefresher. If threshold is zero, a
// 5-minute default is applied.
func NewAutoRefresher(store TokenStore, r Refresher, threshold time.Duration) *AutoRefresher {
	if threshold <= 0 {
		threshold = 5 * time.Minute
	}
	return &AutoRefresher{Store: store, Refresher: r, Threshold: threshold}
}

// Ensure returns a valid session for the given track, refreshing if
// required. Concurrent callers share the refresh work via singleflight.
func (a *AutoRefresher) Ensure(ctx context.Context, track Track) (*Session, error) {
	sess, err := a.Store.Get(ctx, track)
	if err != nil {
		return nil, err
	}
	if !sess.NeedsRefresh(time.Now(), a.Threshold) {
		return sess, nil
	}
	key := track.String()
	v, err, _ := a.sf.Do(key, func() (any, error) {
		// Re-check after acquiring the single-flight slot — a peer may
		// have refreshed while we were waiting.
		current, err := a.Store.Get(ctx, track)
		if err != nil {
			return nil, err
		}
		if !current.NeedsRefresh(time.Now(), a.Threshold) {
			return current, nil
		}
		refreshed, err := a.Refresher.Refresh(ctx, current)
		if err != nil {
			return nil, fmt.Errorf("auth: refresh %s: %w", track, err)
		}
		if err := a.Store.Put(ctx, refreshed); err != nil {
			return nil, fmt.Errorf("auth: persist refreshed session: %w", err)
		}
		return refreshed, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*Session), nil
}
