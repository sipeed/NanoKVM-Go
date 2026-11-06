package auth

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

const unknownAuthRateLimitValue = "unknown"

var (
	defaultAuthRateLimiter = newAuthRateLimiter(time.Now)

	loginUsernameIPRateLimitRule = authRateLimitRule{MaxFailures: 5, Window: 5 * time.Minute, Cooldown: 5 * time.Minute}
	loginIPRateLimitRule         = authRateLimitRule{MaxFailures: 20, Window: 5 * time.Minute, Cooldown: 10 * time.Minute}
	loginUsernameRateLimitRule   = authRateLimitRule{MaxFailures: 10, Window: 10 * time.Minute, Cooldown: 10 * time.Minute}

	sensitiveOperationSessionRateLimitRule  = authRateLimitRule{MaxFailures: 5, Window: 10 * time.Minute, Cooldown: 10 * time.Minute}
	sensitiveOperationUsernameRateLimitRule = authRateLimitRule{MaxFailures: 5, Window: 10 * time.Minute, Cooldown: 10 * time.Minute}
)

type authRateLimitRule struct {
	MaxFailures int
	Window      time.Duration
	Cooldown    time.Duration
}

type authRateLimitRecord struct {
	failures       int
	firstFailureAt time.Time
	lastFailureAt  time.Time
	blockedUntil   time.Time
}

type authRateLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	records map[string]authRateLimitRecord
}

type authRateLimitKey struct {
	key  string
	rule authRateLimitRule
}

func newAuthRateLimiter(now func() time.Time) *authRateLimiter {
	if now == nil {
		now = time.Now
	}

	return &authRateLimiter{
		now:     now,
		records: make(map[string]authRateLimitRecord),
	}
}

func (l *authRateLimiter) blocked(key string, rule authRateLimitRule) bool {
	_, ok := l.blockedUntil(key, rule)
	return ok
}

func (l *authRateLimiter) blockedUntil(key string, rule authRateLimitRule) (time.Time, bool) {
	if l == nil || key == "" || !validAuthRateLimitRule(rule) {
		return time.Time{}, false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	record, ok := l.records[key]
	if !ok {
		return time.Time{}, false
	}

	now := l.now()
	if record.blockedUntil.After(now) {
		return record.blockedUntil, true
	}
	if !record.blockedUntil.IsZero() || authRateLimitRecordExpired(record, rule, now) {
		delete(l.records, key)
	}

	return time.Time{}, false
}

func (l *authRateLimiter) recordFailure(key string, rule authRateLimitRule) {
	if l == nil || key == "" || !validAuthRateLimitRule(rule) {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	record := l.records[key]
	if !record.blockedUntil.IsZero() && !record.blockedUntil.After(now) {
		record = authRateLimitRecord{}
	}
	if record.firstFailureAt.IsZero() || now.Sub(record.firstFailureAt) > rule.Window {
		record = authRateLimitRecord{
			firstFailureAt: now,
		}
	}

	record.failures++
	record.lastFailureAt = now
	if record.failures >= rule.MaxFailures {
		record.blockedUntil = now.Add(rule.Cooldown)
	}
	l.records[key] = record
}

func (l *authRateLimiter) clear(key string) {
	if l == nil || key == "" {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.records, key)
}

func (l *authRateLimiter) cleanup() {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	for key, record := range l.records {
		if record.firstFailureAt.IsZero() || (!record.blockedUntil.IsZero() && !record.blockedUntil.After(now)) {
			delete(l.records, key)
		}
	}
}

func validAuthRateLimitRule(rule authRateLimitRule) bool {
	return rule.MaxFailures > 0 && rule.Window > 0 && rule.Cooldown > 0
}

func authRateLimitRecordExpired(record authRateLimitRecord, rule authRateLimitRule, now time.Time) bool {
	if record.firstFailureAt.IsZero() {
		return true
	}
	windowExpired := now.Sub(record.firstFailureAt) > rule.Window
	cooldownExpired := record.blockedUntil.IsZero() || !record.blockedUntil.After(now)
	return windowExpired && cooldownExpired
}

func loginRateLimitKeys(username, ip string) []authRateLimitKey {
	username = authRateLimitValue(username)
	ip = authRateLimitValue(ip)

	return []authRateLimitKey{
		{
			key:  fmt.Sprintf("login:username_ip:%q:%q", username, ip),
			rule: loginUsernameIPRateLimitRule,
		},
		{
			key:  fmt.Sprintf("login:ip:%q", ip),
			rule: loginIPRateLimitRule,
		},
		{
			key:  fmt.Sprintf("login:username:%q", username),
			rule: loginUsernameRateLimitRule,
		},
	}
}

func isLoginRateLimited(username, ip string) bool {
	_, ok := loginRateLimitedUntil(username, ip)
	return ok
}

func loginRateLimitedUntil(username, ip string) (time.Time, bool) {
	return rateLimitedUntil(loginRateLimitKeys(username, ip))
}

func recordLoginFailure(username, ip string) {
	for _, key := range loginRateLimitKeys(username, ip) {
		defaultAuthRateLimiter.recordFailure(key.key, key.rule)
	}
}

func clearLoginFailures(username, ip string) {
	for _, key := range loginRateLimitKeys(username, ip) {
		defaultAuthRateLimiter.clear(key.key)
	}
}

func sensitiveOperationRateLimitKeys(username, sessionKey string) []authRateLimitKey {
	username = authRateLimitValue(username)
	sessionKey = authRateLimitValue(sessionKey)

	return []authRateLimitKey{
		{
			key:  fmt.Sprintf("sensitive:session:%q", sessionKey),
			rule: sensitiveOperationSessionRateLimitRule,
		},
		{
			key:  fmt.Sprintf("sensitive:username:%q", username),
			rule: sensitiveOperationUsernameRateLimitRule,
		},
	}
}

func isSensitiveOperationRateLimited(username, sessionKey string) bool {
	_, ok := sensitiveOperationRateLimitedUntil(username, sessionKey)
	return ok
}

func sensitiveOperationRateLimitedUntil(username, sessionKey string) (time.Time, bool) {
	return rateLimitedUntil(sensitiveOperationRateLimitKeys(username, sessionKey))
}

func rateLimitedUntil(keys []authRateLimitKey) (time.Time, bool) {
	var latest time.Time
	for _, key := range keys {
		blockedUntil, ok := defaultAuthRateLimiter.blockedUntil(key.key, key.rule)
		if ok && blockedUntil.After(latest) {
			latest = blockedUntil
		}
	}
	return latest, !latest.IsZero()
}

func recordSensitiveOperationFailure(username, sessionKey string) {
	for _, key := range sensitiveOperationRateLimitKeys(username, sessionKey) {
		defaultAuthRateLimiter.recordFailure(key.key, key.rule)
	}
}

func clearSensitiveOperationFailures(username, sessionKey string) {
	for _, key := range sensitiveOperationRateLimitKeys(username, sessionKey) {
		defaultAuthRateLimiter.clear(key.key)
	}
}

func authRateLimitValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return unknownAuthRateLimitValue
	}
	return value
}

func authRateLimitResponse(until time.Time) proto.AuthRateLimitRsp {
	now := time.Now()
	if defaultAuthRateLimiter != nil && defaultAuthRateLimiter.now != nil {
		now = defaultAuthRateLimiter.now()
	}

	retryAfter := int(math.Ceil(until.Sub(now).Seconds()))
	if retryAfter < 1 {
		retryAfter = 1
	}

	return proto.AuthRateLimitRsp{
		RateLimited: true,
		RetryAfter:  retryAfter,
		RetryAt:     until.Unix(),
	}
}

func writeAuthRateLimitRsp(c *gin.Context, rsp *proto.Response, until time.Time) {
	time.Sleep(invalidCredentialDelay)
	rsp.ErrRspWithData(c, -2, invalidAuthMessage, authRateLimitResponse(until))
}
