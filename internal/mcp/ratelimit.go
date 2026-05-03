package mcp

import (
	"fmt"
	"sync"
	"time"
)

// prCommentRateLimit guards against an agent in a loop hammering the
// pr-comment tool. The PR-comment workflow is idempotent at the API level
// (find existing comment + update vs create), but rapid repeat calls still
// burn API quota and trip GitHub abuse detection — getting the user's
// token rate-limited or temporarily banned.
//
// Per (owner, repo, prNumber) key: max prCommentBurst calls within
// prCommentWindow. Calls beyond the budget are rejected with a clear error
// rather than a confused-deputy GitHub 403/abuse failure.
const (
	prCommentWindow = 5 * time.Minute
	prCommentBurst  = 5
)

type rateLimiter struct {
	mu      sync.Mutex
	history map[string][]time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{history: make(map[string][]time.Time)}
}

// allow returns nil if a call to key is within budget, or a descriptive
// error if not. Records the call timestamp on success. Garbage-collects
// timestamps older than the window on every invocation.
func (l *rateLimiter) allow(key string, now time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-prCommentWindow)
	kept := l.history[key][:0]
	for _, t := range l.history[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= prCommentBurst {
		oldest := kept[0]
		retryIn := prCommentWindow - now.Sub(oldest)
		return fmt.Errorf("pr-comment rate limit: %d calls in the last %s for %s; retry in %s",
			len(kept), prCommentWindow, key, retryIn.Round(time.Second))
	}
	kept = append(kept, now)
	l.history[key] = kept
	return nil
}
