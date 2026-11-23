package ai

import (
	"sync"
	"time"
)

type toolRateLimiter struct {
	mu          sync.Mutex
	minInterval time.Duration
	nextAllowed map[string]time.Time
}

func newToolRateLimiter(minInterval time.Duration) *toolRateLimiter {
	return &toolRateLimiter{
		minInterval: minInterval,
		nextAllowed: make(map[string]time.Time),
	}
}

func (l *toolRateLimiter) allowN(tool string, count int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if now.Before(l.nextAllowed[tool]) {
		return false
	}

	l.nextAllowed[tool] = now.Add(time.Duration(count) * l.minInterval)
	return true
}
