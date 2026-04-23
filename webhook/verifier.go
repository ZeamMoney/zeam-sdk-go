package webhook

import (
	"container/list"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// HeaderSignature carries the HMAC hex digest. Matches the gateway's
	// outbound webhook signature header.
	HeaderSignature = "X-Zeam-Signature"
	// HeaderTimestamp carries the Unix seconds timestamp the signature
	// covers.
	HeaderTimestamp = "X-Zeam-Timestamp"
	// HeaderEventID carries the platform event id (used for replay
	// detection).
	HeaderEventID = "X-Zeam-Event-Id"

	defaultMaxSkew = 5 * time.Minute
	maxBodySize    = 2 << 20 // 2 MiB
)

// Public sentinel errors returned by [Verify].
var (
	ErrMissingSignature = errors.New("webhook: missing X-Zeam-Signature")
	ErrMissingTimestamp = errors.New("webhook: missing X-Zeam-Timestamp")
	ErrInvalidTimestamp = errors.New("webhook: invalid X-Zeam-Timestamp")
	ErrStaleTimestamp   = errors.New("webhook: timestamp outside clock-skew window")
	ErrBadSignature     = errors.New("webhook: signature mismatch")
	ErrReplay           = errors.New("webhook: replayed event")
	ErrBodyTooLarge     = errors.New("webhook: request body exceeds 2 MiB cap")
)

// ReplayCache is the minimal contract for a replay deduplication cache.
// Implementations must be safe for concurrent use. [NewLRU] returns a
// bounded in-memory cache.
type ReplayCache interface {
	// Seen reports whether this event id has been observed already, and
	// marks it as seen if not. The supplied TTL bounds how long the
	// entry is retained; implementations MAY drop entries earlier if the
	// cache is full.
	Seen(eventID string, ttl time.Duration) bool
}

// Verifier holds the parsed configuration used by [Verify] and [Handler].
type Verifier struct {
	Secret  []byte
	MaxSkew time.Duration
	Replay  ReplayCache
	Now     func() time.Time
}

// Option configures a [Verifier].
type Option func(*Verifier)

// WithMaxSkew overrides the clock-skew tolerance. Default: 5 minutes.
func WithMaxSkew(d time.Duration) Option {
	return func(v *Verifier) { v.MaxSkew = d }
}

// WithReplayCache attaches a cache so duplicate events are rejected with
// [ErrReplay].
func WithReplayCache(r ReplayCache) Option {
	return func(v *Verifier) { v.Replay = r }
}

// WithClock overrides the Now function; intended for tests.
func WithClock(now func() time.Time) Option {
	return func(v *Verifier) { v.Now = now }
}

// NewVerifier constructs a verifier bound to the given HMAC secret.
func NewVerifier(secret []byte, opts ...Option) *Verifier {
	v := &Verifier{
		Secret:  append([]byte(nil), secret...),
		MaxSkew: defaultMaxSkew,
		Now:     time.Now,
	}
	for _, o := range opts {
		if o != nil {
			o(v)
		}
	}
	return v
}

// Verify validates req against the configured HMAC secret. On success it
// returns the raw request body (the caller should json.Unmarshal it or
// consume it in another way). On failure it returns an error matching one
// of the exported sentinels.
func (v *Verifier) Verify(req *http.Request) ([]byte, error) {
	sig := req.Header.Get(HeaderSignature)
	if sig == "" {
		return nil, ErrMissingSignature
	}
	tsHeader := req.Header.Get(HeaderTimestamp)
	if tsHeader == "" {
		return nil, ErrMissingTimestamp
	}
	tsInt, err := strconv.ParseInt(strings.TrimSpace(tsHeader), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTimestamp, err)
	}
	ts := time.Unix(tsInt, 0)

	now := v.Now()
	skew := v.MaxSkew
	if skew <= 0 {
		skew = defaultMaxSkew
	}
	if absDuration(now.Sub(ts)) > skew {
		return nil, ErrStaleTimestamp
	}

	// Read the body up to the cap; a webhook larger than 2 MiB is
	// treated as malicious / buggy and rejected.
	if req.ContentLength > maxBodySize {
		return nil, ErrBodyTooLarge
	}
	body, err := io.ReadAll(io.LimitReader(req.Body, maxBodySize+1))
	if err != nil {
		return nil, fmt.Errorf("webhook: read body: %w", err)
	}
	if len(body) > maxBodySize {
		return nil, ErrBodyTooLarge
	}

	if !v.verifySignature(body, tsInt, sig) {
		return nil, ErrBadSignature
	}

	if v.Replay != nil {
		if eid := req.Header.Get(HeaderEventID); eid != "" {
			if v.Replay.Seen(eid, skew*2) {
				return nil, ErrReplay
			}
		}
	}
	return body, nil
}

// verifySignature computes HMAC-SHA256(secret, ts + "." + body) and
// compares it in constant time against the hex-encoded signature.
func (v *Verifier) verifySignature(body []byte, ts int64, hexSig string) bool {
	sig, err := hex.DecodeString(hexSig)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, v.Secret)
	fmt.Fprintf(mac, "%d.", ts)
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	return hmac.Equal(sig, expected)
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// Verify is the package-level helper for callers without a Verifier. It
// is shorthand for NewVerifier(secret).Verify(req).
func Verify(req *http.Request, secret []byte, opts ...Option) ([]byte, error) {
	return NewVerifier(secret, opts...).Verify(req)
}

// LRU is a bounded in-memory replay cache. The eviction policy is pure
// LRU; entries past their TTL are lazily expired on access.
type LRU struct {
	mu   sync.Mutex
	cap  int
	list *list.List
	m    map[string]*list.Element
}

type lruEntry struct {
	key       string
	expiresAt time.Time
}

// NewLRU returns an LRU replay cache with the given capacity.
func NewLRU(capacity int) *LRU {
	if capacity <= 0 {
		capacity = 1024
	}
	return &LRU{
		cap:  capacity,
		list: list.New(),
		m:    make(map[string]*list.Element, capacity),
	}
}

// Seen reports whether eventID has been observed, and records it with
// the supplied TTL.
func (c *LRU) Seen(eventID string, ttl time.Duration) bool {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.m[eventID]; ok {
		entry := el.Value.(*lruEntry)
		if entry.expiresAt.After(now) {
			c.list.MoveToFront(el)
			return true
		}
		c.list.Remove(el)
		delete(c.m, eventID)
	}

	for c.list.Len() >= c.cap {
		oldest := c.list.Back()
		if oldest == nil {
			break
		}
		c.list.Remove(oldest)
		delete(c.m, oldest.Value.(*lruEntry).key)
	}

	el := c.list.PushFront(&lruEntry{key: eventID, expiresAt: now.Add(ttl)})
	c.m[eventID] = el
	return false
}
