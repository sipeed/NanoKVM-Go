package auth

import (
	"testing"
	"time"
)

func resetLoginChallengesForTest() {
	loginChallenges.Lock()
	defer loginChallenges.Unlock()
	loginChallenges.items = make(map[string]loginChallenge)
}

func TestLoginChallenge(t *testing.T) {
	useTempTOTPSecretFile(t)
	resetLoginChallengesForTest()
	t.Cleanup(resetLoginChallengesForTest)
	now := time.Unix(1700000000, 0)
	totpNow = func() time.Time { return now }

	token, err := createLoginChallenge("admin", "192.0.2.1", "agent")
	if err != nil {
		t.Fatalf("createLoginChallenge returned error: %v", err)
	}
	if token == "" {
		t.Fatal("createLoginChallenge returned empty token")
	}

	username, ok := consumeLoginChallenge(token, "192.0.2.1", "agent")
	if !ok || username != "admin" {
		t.Fatalf("expected successful consume for admin, got username=%q ok=%v", username, ok)
	}
	if _, ok := consumeLoginChallenge(token, "192.0.2.1", "agent"); ok {
		t.Fatal("login challenge was consumed twice")
	}
}

func TestLoginChallengeRejectsExpiredAndMismatchedClient(t *testing.T) {
	useTempTOTPSecretFile(t)
	resetLoginChallengesForTest()
	t.Cleanup(resetLoginChallengesForTest)
	now := time.Unix(1700000000, 0)
	totpNow = func() time.Time { return now }

	token, err := createLoginChallenge("admin", "192.0.2.1", "agent")
	if err != nil {
		t.Fatalf("createLoginChallenge returned error: %v", err)
	}
	if _, ok := consumeLoginChallenge(token, "192.0.2.2", "agent"); ok {
		t.Fatal("login challenge accepted mismatched IP")
	}
	if _, ok := consumeLoginChallenge(token, "192.0.2.1", "other-agent"); ok {
		t.Fatal("login challenge accepted mismatched user agent")
	}
	if _, ok := consumeLoginChallenge(token, "192.0.2.1", "agent"); !ok {
		t.Fatal("mismatch attempts should not consume a valid challenge")
	}

	token, err = createLoginChallenge("admin", "192.0.2.1", "agent")
	if err != nil {
		t.Fatalf("createLoginChallenge returned error: %v", err)
	}
	expiredAt := now.Add(loginChallengeTTL + time.Second)
	cleanupExpiredLoginChallenges(expiredAt)
	totpNow = func() time.Time { return expiredAt }
	if _, ok := consumeLoginChallenge(token, "192.0.2.1", "agent"); ok {
		t.Fatal("expired login challenge was accepted")
	}
}
