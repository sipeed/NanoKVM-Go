package auth

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestAuthRateLimiterBlocksAtThreshold(t *testing.T) {
	now := time.Unix(1700000000, 0)
	limiter := newAuthRateLimiter(func() time.Time { return now })
	rule := authRateLimitRule{MaxFailures: 3, Window: time.Minute, Cooldown: 5 * time.Minute}

	limiter.recordFailure("login:admin", rule)
	limiter.recordFailure("login:admin", rule)
	if limiter.blocked("login:admin", rule) {
		t.Fatal("threshold should not block before max failures")
	}

	limiter.recordFailure("login:admin", rule)
	if !limiter.blocked("login:admin", rule) {
		t.Fatal("threshold should block at max failures")
	}
}

func TestAuthRateLimiterResetsAfterWindow(t *testing.T) {
	now := time.Unix(1700000000, 0)
	limiter := newAuthRateLimiter(func() time.Time { return now })
	rule := authRateLimitRule{MaxFailures: 3, Window: time.Minute, Cooldown: 5 * time.Minute}

	limiter.recordFailure("login:admin", rule)
	limiter.recordFailure("login:admin", rule)
	now = now.Add(rule.Window + time.Second)
	limiter.recordFailure("login:admin", rule)

	if limiter.blocked("login:admin", rule) {
		t.Fatal("failures outside the window should not block")
	}
	limiter.recordFailure("login:admin", rule)
	if limiter.blocked("login:admin", rule) {
		t.Fatal("window reset should start a fresh failure count")
	}
	limiter.recordFailure("login:admin", rule)
	if !limiter.blocked("login:admin", rule) {
		t.Fatal("new window should block after max failures")
	}
}

func TestAuthRateLimiterUnblocksAfterCooldown(t *testing.T) {
	now := time.Unix(1700000000, 0)
	limiter := newAuthRateLimiter(func() time.Time { return now })
	rule := authRateLimitRule{MaxFailures: 2, Window: time.Minute, Cooldown: 5 * time.Minute}

	limiter.recordFailure("login:admin", rule)
	limiter.recordFailure("login:admin", rule)
	if !limiter.blocked("login:admin", rule) {
		t.Fatal("expected key to be blocked")
	}

	now = now.Add(rule.Cooldown + time.Second)
	if limiter.blocked("login:admin", rule) {
		t.Fatal("cooldown expiry should unblock key")
	}

	limiter.recordFailure("login:admin", rule)
	if limiter.blocked("login:admin", rule) {
		t.Fatal("cooldown expiry should clear prior failure count")
	}
}

func TestAuthRateLimiterBlockedUntilReportsRetryMetadata(t *testing.T) {
	now := time.Unix(1700000000, 0)
	limiter := newAuthRateLimiter(func() time.Time { return now })
	rule := authRateLimitRule{MaxFailures: 2, Window: time.Minute, Cooldown: 5 * time.Minute}

	limiter.recordFailure("login:admin", rule)
	limiter.recordFailure("login:admin", rule)

	blockedUntil, ok := limiter.blockedUntil("login:admin", rule)
	if !ok {
		t.Fatal("expected blocked key to report retry metadata")
	}
	if !blockedUntil.Equal(now.Add(rule.Cooldown)) {
		t.Fatalf("unexpected blockedUntil %s", blockedUntil)
	}
}

func TestAuthRateLimiterBlockedUntilClearsExpiredBlock(t *testing.T) {
	now := time.Unix(1700000000, 0)
	limiter := newAuthRateLimiter(func() time.Time { return now })
	rule := authRateLimitRule{MaxFailures: 2, Window: time.Minute, Cooldown: 5 * time.Minute}

	limiter.recordFailure("login:admin", rule)
	limiter.recordFailure("login:admin", rule)
	now = now.Add(rule.Cooldown + time.Second)

	if blockedUntil, ok := limiter.blockedUntil("login:admin", rule); ok {
		t.Fatalf("expired block should be cleared, got %s", blockedUntil)
	}
	if limiter.blocked("login:admin", rule) {
		t.Fatal("expired block should not remain blocked")
	}
}

func TestAuthRateLimiterClear(t *testing.T) {
	now := time.Unix(1700000000, 0)
	limiter := newAuthRateLimiter(func() time.Time { return now })
	rule := authRateLimitRule{MaxFailures: 2, Window: time.Minute, Cooldown: 5 * time.Minute}

	limiter.recordFailure("login:admin", rule)
	limiter.recordFailure("login:admin", rule)
	if !limiter.blocked("login:admin", rule) {
		t.Fatal("expected key to be blocked")
	}

	limiter.clear("login:admin")
	if limiter.blocked("login:admin", rule) {
		t.Fatal("clear should remove blocked state")
	}

	limiter.recordFailure("login:admin", rule)
	if limiter.blocked("login:admin", rule) {
		t.Fatal("clear should remove prior failure count")
	}
}

func TestAuthRateLimiterCleanupDeletesExpiredRecords(t *testing.T) {
	now := time.Unix(1700000000, 0)
	limiter := newAuthRateLimiter(func() time.Time { return now })
	rule := authRateLimitRule{MaxFailures: 2, Window: time.Minute, Cooldown: 5 * time.Minute}

	limiter.recordFailure("active", rule)
	limiter.recordFailure("expired", rule)
	limiter.recordFailure("expired", rule)

	now = now.Add(rule.Cooldown + time.Second)
	limiter.cleanup()

	if _, ok := limiter.records["expired"]; ok {
		t.Fatal("cleanup should delete expired cooldown records")
	}
	if _, ok := limiter.records["active"]; !ok {
		t.Fatal("cleanup should keep records that never entered cooldown")
	}
}

func TestLoginRateLimitUsernameIPThreshold(t *testing.T) {
	setupAuthRateLimitTest(t)

	for i := 0; i < loginUsernameIPRateLimitRule.MaxFailures-1; i++ {
		recordLoginFailure("admin", "192.0.2.1")
	}
	if isLoginRateLimited("admin", "192.0.2.1") {
		t.Fatal("username+ip key should not block before threshold")
	}

	recordLoginFailure("admin", "192.0.2.1")
	if !isLoginRateLimited("admin", "192.0.2.1") {
		t.Fatal("username+ip key should block at threshold")
	}
	if isLoginRateLimited("admin", "192.0.2.2") {
		t.Fatal("different ip should not share username+ip limit")
	}
}

func TestLoginRateLimitIPThreshold(t *testing.T) {
	setupAuthRateLimitTest(t)

	for i := 0; i < loginIPRateLimitRule.MaxFailures; i++ {
		recordLoginFailure(fmt.Sprintf("user-%d", i), "192.0.2.1")
	}

	if !isLoginRateLimited("another-user", "192.0.2.1") {
		t.Fatal("ip key should block after failures across usernames")
	}
	if isLoginRateLimited("another-user", "192.0.2.2") {
		t.Fatal("different ip should not share ip limit")
	}
}

func TestLoginRateLimitUsernameThreshold(t *testing.T) {
	setupAuthRateLimitTest(t)

	for i := 0; i < loginUsernameRateLimitRule.MaxFailures; i++ {
		recordLoginFailure("admin", fmt.Sprintf("192.0.2.%d", i+1))
	}

	if !isLoginRateLimited("admin", "198.51.100.1") {
		t.Fatal("username key should block after failures across ips")
	}
	if isLoginRateLimited("operator", "198.51.100.1") {
		t.Fatal("different username should not share username limit")
	}
}

func TestLoginRateLimitClearLoginFailures(t *testing.T) {
	setupAuthRateLimitTest(t)

	for i := 0; i < loginUsernameIPRateLimitRule.MaxFailures; i++ {
		recordLoginFailure("admin", "192.0.2.1")
	}
	if !isLoginRateLimited("admin", "192.0.2.1") {
		t.Fatal("expected login to be rate limited before clear")
	}

	clearLoginFailures("admin", "192.0.2.1")
	if isLoginRateLimited("admin", "192.0.2.1") {
		t.Fatal("successful login should clear user/ip failure count")
	}

	recordLoginFailure("admin", "192.0.2.1")
	if isLoginRateLimited("admin", "192.0.2.1") {
		t.Fatal("clear should start fresh counters")
	}
}

func TestLoginRateLimitKeysUseUnknownPlaceholder(t *testing.T) {
	keys := loginRateLimitKeys(" ", "")
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	for _, key := range keys {
		if key.key == "" {
			t.Fatal("rate limit key should not be empty")
		}
		if !strings.Contains(key.key, unknownAuthRateLimitValue) {
			t.Fatalf("rate limit key should use unknown placeholder, got %q", key.key)
		}
	}
}

func TestSensitiveOperationRateLimitSessionThreshold(t *testing.T) {
	setupAuthRateLimitTest(t)

	for i := 0; i < sensitiveOperationSessionRateLimitRule.MaxFailures; i++ {
		recordSensitiveOperationFailure(fmt.Sprintf("user-%d", i), "session-hash")
	}

	if !isSensitiveOperationRateLimited("another-user", "session-hash") {
		t.Fatal("session key should block after failures across usernames")
	}
	if isSensitiveOperationRateLimited("another-user", "other-session-hash") {
		t.Fatal("different session key should not share session limit")
	}
}

func TestSensitiveOperationRateLimitUsernameThreshold(t *testing.T) {
	setupAuthRateLimitTest(t)

	for i := 0; i < sensitiveOperationUsernameRateLimitRule.MaxFailures; i++ {
		recordSensitiveOperationFailure("admin", fmt.Sprintf("session-hash-%d", i))
	}

	if !isSensitiveOperationRateLimited("admin", "other-session-hash") {
		t.Fatal("username key should block after failures across sessions")
	}
	if isSensitiveOperationRateLimited("operator", "other-session-hash") {
		t.Fatal("different username should not share username limit")
	}
}

func TestSensitiveOperationRateLimitClearFailures(t *testing.T) {
	setupAuthRateLimitTest(t)

	for i := 0; i < sensitiveOperationSessionRateLimitRule.MaxFailures; i++ {
		recordSensitiveOperationFailure("admin", "session-hash")
	}
	if !isSensitiveOperationRateLimited("admin", "session-hash") {
		t.Fatal("expected sensitive operation to be rate limited before clear")
	}

	clearSensitiveOperationFailures("admin", "session-hash")
	if isSensitiveOperationRateLimited("admin", "session-hash") {
		t.Fatal("successful operation should clear user/session failure count")
	}

	recordSensitiveOperationFailure("admin", "session-hash")
	if isSensitiveOperationRateLimited("admin", "session-hash") {
		t.Fatal("clear should start fresh counters")
	}
}

func setupAuthRateLimitTest(t *testing.T) {
	t.Helper()
	original := defaultAuthRateLimiter
	now := time.Unix(1700000000, 0)
	defaultAuthRateLimiter = newAuthRateLimiter(func() time.Time { return now })
	t.Cleanup(func() {
		defaultAuthRateLimiter = original
	})
}
