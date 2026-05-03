package mcp

import (
	"strings"
	"testing"
	"time"
)

func TestRateLimiter_AllowsBurst(t *testing.T) {
	l := newRateLimiter()
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)

	for i := 0; i < prCommentBurst; i++ {
		if err := l.allow("k", now.Add(time.Duration(i)*time.Second)); err != nil {
			t.Errorf("call %d should be allowed within burst, got %v", i+1, err)
		}
	}
}

func TestRateLimiter_RejectsBeyondBurst(t *testing.T) {
	l := newRateLimiter()
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)

	for i := 0; i < prCommentBurst; i++ {
		_ = l.allow("k", now.Add(time.Duration(i)*time.Second))
	}
	err := l.allow("k", now.Add(time.Duration(prCommentBurst)*time.Second))
	if err == nil {
		t.Fatal("expected rate limit error after burst")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("expected 'rate limit' in error, got %v", err)
	}
}

func TestRateLimiter_AllowsAfterWindow(t *testing.T) {
	l := newRateLimiter()
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)

	for i := 0; i < prCommentBurst; i++ {
		_ = l.allow("k", now)
	}
	// Move past the window
	if err := l.allow("k", now.Add(prCommentWindow+time.Second)); err != nil {
		t.Errorf("expected call to succeed after window expired, got %v", err)
	}
}

func TestRateLimiter_KeysAreIndependent(t *testing.T) {
	l := newRateLimiter()
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)

	for i := 0; i < prCommentBurst; i++ {
		_ = l.allow("a", now)
	}
	if err := l.allow("b", now); err != nil {
		t.Errorf("key 'b' must be independent of 'a', got %v", err)
	}
}
