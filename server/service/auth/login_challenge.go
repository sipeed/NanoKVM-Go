package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

const loginChallengeTTL = 5 * time.Minute

type loginChallenge struct {
	username        string
	createdAt       time.Time
	expiresAt       time.Time
	ip              string
	userAgentDigest string
	used            bool
}

var loginChallenges = struct {
	sync.Mutex
	items map[string]loginChallenge
}{items: make(map[string]loginChallenge)}

func createLoginChallenge(username, ip, userAgent string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	token := base64.RawURLEncoding.EncodeToString(buf)
	now := totpNow()

	loginChallenges.Lock()
	defer loginChallenges.Unlock()
	cleanupExpiredLoginChallengesLocked(now)
	loginChallenges.items[loginChallengeKey(token)] = loginChallenge{
		username:        strings.TrimSpace(username),
		createdAt:       now,
		expiresAt:       now.Add(loginChallengeTTL),
		ip:              strings.TrimSpace(ip),
		userAgentDigest: userAgentHash(userAgent),
	}

	return token, nil
}

func consumeLoginChallenge(token, ip, userAgent string) (string, bool) {
	now := totpNow()
	key := loginChallengeKey(token)

	loginChallenges.Lock()
	defer loginChallenges.Unlock()
	cleanupExpiredLoginChallengesLocked(now)

	challenge, ok := loginChallenges.items[key]
	if !ok || challenge.used || !now.Before(challenge.expiresAt) {
		delete(loginChallenges.items, key)
		return "", false
	}
	if challenge.ip != strings.TrimSpace(ip) || challenge.userAgentDigest != userAgentHash(userAgent) {
		return "", false
	}

	challenge.used = true
	loginChallenges.items[key] = challenge
	delete(loginChallenges.items, key)
	return challenge.username, true
}

func validateLoginChallenge(token, ip, userAgent string) (string, bool) {
	now := totpNow()
	key := loginChallengeKey(token)

	loginChallenges.Lock()
	defer loginChallenges.Unlock()
	cleanupExpiredLoginChallengesLocked(now)

	challenge, ok := loginChallenges.items[key]
	if !ok || challenge.used || !now.Before(challenge.expiresAt) {
		delete(loginChallenges.items, key)
		return "", false
	}
	if challenge.ip != strings.TrimSpace(ip) || challenge.userAgentDigest != userAgentHash(userAgent) {
		return "", false
	}

	return challenge.username, true
}

func cleanupExpiredLoginChallenges(now time.Time) {
	loginChallenges.Lock()
	defer loginChallenges.Unlock()
	cleanupExpiredLoginChallengesLocked(now)
}

func cleanupExpiredLoginChallengesLocked(now time.Time) {
	for key, challenge := range loginChallenges.items {
		if !now.Before(challenge.expiresAt) || challenge.used {
			delete(loginChallenges.items, key)
		}
	}
}

func loginChallengeKey(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func userAgentHash(userAgent string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(userAgent)))
	return hex.EncodeToString(sum[:])
}
