package transport

import (
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// retryRoundTripper wraps another round-tripper to apply the bounded,
// idempotent retry policy described in ADR 0008 R6:
//
//   - GET: retry on 408/429/502/503/504, max 2 attempts, full-jitter
//     exponential backoff (base 100ms, cap 1s), honouring Retry-After.
//   - POST/PUT/DELETE/PATCH: never retry on a received non-2xx response.
//     Network-level errors (no response at all) are retried at most once,
//     relying on Idempotency-Key propagation to make this safe.
type retryRoundTripper struct {
	base     http.RoundTripper
	maxGET   int
	maxWrite int
	sleep    func(time.Duration)
	rng      *rand.Rand
}

func withRetry(base http.RoundTripper) http.RoundTripper {
	return &retryRoundTripper{
		base:     base,
		maxGET:   2,
		maxWrite: 1,
		sleep:    func(d time.Duration) { time.Sleep(d) },
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *retryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	budget := r.maxGET
	writeVerb := false
	if req.Method != http.MethodGet {
		budget = r.maxWrite
		writeVerb = true
	}

	var lastResp *http.Response
	var lastErr error
	for attempt := 0; attempt <= budget; attempt++ {
		if attempt > 0 {
			// Before retrying, drain and close the previous body so the
			// connection can be reused.
			if lastResp != nil {
				_, _ = io.Copy(io.Discard, lastResp.Body)
				_ = lastResp.Body.Close()
				lastResp = nil
			}
			r.sleep(r.backoff(attempt, lastResp))
		}

		// context cancellation short-circuits immediately.
		if err := req.Context().Err(); err != nil {
			return nil, err
		}

		resp, err := r.base.RoundTrip(req)
		lastErr = err
		lastResp = resp

		if err != nil {
			// Retry at most once for write verbs on pure network errors.
			if writeVerb && attempt < r.maxWrite {
				continue
			}
			// For GETs, retry.
			if !writeVerb && attempt < r.maxGET {
				continue
			}
			return nil, err
		}

		if writeVerb {
			// Never retry a write verb on a received non-2xx response.
			return resp, nil
		}

		if !retryableStatus(resp.StatusCode) || attempt == budget {
			return resp, nil
		}
	}
	return lastResp, lastErr
}

func retryableStatus(code int) bool {
	switch code {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// backoff returns the next sleep. If the previous response carried a
// Retry-After header and it is shorter than the cap, it wins; otherwise
// full-jitter exponential with base 100ms and cap 1s.
func (r *retryRoundTripper) backoff(attempt int, prev *http.Response) time.Duration {
	if prev != nil {
		if ra := prev.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil && secs >= 0 {
				d := time.Duration(secs) * time.Second
				if d > time.Second {
					d = time.Second
				}
				return d
			}
		}
	}
	// Exponential base 100ms, cap 1s, full jitter.
	max := time.Duration(1) << uint(attempt-1) * 100 * time.Millisecond
	if max > time.Second {
		max = time.Second
	}
	return time.Duration(r.rng.Int63n(int64(max) + 1))
}
