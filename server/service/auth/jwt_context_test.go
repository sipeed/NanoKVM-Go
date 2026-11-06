package auth

import "testing"

func TestCurrentJWTUsernameFromContext(t *testing.T) {
	ctx := authTestContext(t, "GET", "/api/auth/account", nil)
	ctx.Set("username", " admin ")

	if got := currentJWTUsername(ctx); got != "admin" {
		t.Fatalf("expected username %q, got %q", "admin", got)
	}
}

func TestCurrentJWTUsernameEmptyWithoutClaims(t *testing.T) {
	ctx := authTestContext(t, "GET", "/api/auth/account", nil)

	if got := currentJWTUsername(ctx); got != "" {
		t.Fatalf("expected empty username, got %q", got)
	}
}
