package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/middleware"
	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
)

func TestTerminalAuthGrantIsBoundToCurrentJWTAndConsumedOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenA, err := middleware.GenerateJWT("admin")
	if err != nil {
		t.Fatal(err)
	}
	tokenB, err := middleware.GenerateJWT("operator")
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "terminal_auth")
	if err := createTerminalAuthGrant(terminalAuthTestContext(tokenA), path, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	if consumeTerminalAuthGrant(terminalAuthTestContext(tokenB), path, time.Now()) {
		t.Fatal("expected a different JWT session to be rejected")
	}
	if !consumeTerminalAuthGrant(terminalAuthTestContext(tokenA), path, time.Now()) {
		t.Fatal("expected the matching JWT session to consume the grant")
	}
	if consumeTerminalAuthGrant(terminalAuthTestContext(tokenA), path, time.Now()) {
		t.Fatal("expected grant to be single use")
	}
}

func TestTerminalAuthGrantAllowsConcurrentJWTGrants(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenA, err := middleware.GenerateJWT("admin")
	if err != nil {
		t.Fatal(err)
	}
	tokenB, err := middleware.GenerateJWT("operator")
	if err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(t.TempDir(), "terminal_auth.d")
	if err := createTerminalAuthGrant(terminalAuthTestContext(tokenA), dir, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := createTerminalAuthGrant(terminalAuthTestContext(tokenB), dir, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	if !consumeTerminalAuthGrant(terminalAuthTestContext(tokenA), dir, time.Now()) {
		t.Fatal("expected first JWT session to consume its grant")
	}
	if !consumeTerminalAuthGrant(terminalAuthTestContext(tokenB), dir, time.Now()) {
		t.Fatal("expected second JWT session to consume its grant")
	}
}

func TestTerminalAuthGrantRejectsExpiredGrant(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token, err := middleware.GenerateJWT("admin")
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "terminal_auth")
	if err := createTerminalAuthGrant(terminalAuthTestContext(token), path, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	if consumeTerminalAuthGrant(terminalAuthTestContext(token), path, time.Now()) {
		t.Fatal("expected expired terminal grant to be rejected")
	}
}

func TestTerminalShellPrefersBash(t *testing.T) {
	if _, err := os.Stat(terminalBash); err != nil {
		t.Skipf("%s is not available: %v", terminalBash, err)
	}

	if got := terminalShell(); got != terminalBash {
		t.Fatalf("expected terminal shell %q, got %q", terminalBash, got)
	}
}

func TestTerminalEnvironmentSetsMissingHome(t *testing.T) {
	home := t.TempDir()

	for _, initialEnv := range [][]string{
		{"PATH=/usr/bin"},
		{"PATH=/usr/bin", "HOME="},
	} {
		env := terminalEnvironment(initialEnv, home)

		if got := environmentValue(env, "HOME"); got != home {
			t.Fatalf("expected HOME %q, got %q", home, got)
		}
		if got := environmentValue(env, "TERM"); got != "xterm-256color" {
			t.Fatalf("expected TERM xterm-256color, got %q", got)
		}
		if got := environmentValue(env, "COLORTERM"); got != "truecolor" {
			t.Fatalf("expected COLORTERM truecolor, got %q", got)
		}

		cmd := exec.Command(terminalShell(), "-c", "cd / && cd && pwd")
		cmd.Env = env
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("expected cd without arguments to succeed: %v: %s", err, output)
		}
		if got := strings.TrimSpace(string(output)); got != home {
			t.Fatalf("expected cd to enter %q, got %q", home, got)
		}
	}
}

func TestTerminalEnvironmentPreservesExistingHome(t *testing.T) {
	const home = "/tmp/custom-home"
	env := terminalEnvironment([]string{"HOME=" + home}, terminalDefaultHome)

	if got := environmentValue(env, "HOME"); got != home {
		t.Fatalf("expected HOME %q, got %q", home, got)
	}
}

func terminalAuthTestContext(token string) *gin.Context {
	_, ctx := terminalAuthTestContextWithRecorder(token, http.MethodGet, "/api/vm/terminal", nil)
	return ctx
}

func terminalAuthTestContextWithRecorder(token, method, target string, body []byte) (*httptest.ResponseRecorder, *gin.Context) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, bytes.NewReader(body))
	ctx.Request.AddCookie(&http.Cookie{Name: "nano-kvm-token", Value: token})
	claims, err := middleware.ParseJWT(token)
	if err == nil {
		ctx.Set("username", claims.Username)
		ctx.Set("jwtToken", token)
	}
	return recorder, ctx
}

func TestTerminalAuthChallengeCreatesShortLivedGrant(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalVerify := verifyTerminalChallenge
	originalEnabled := terminalAuthenticationEnabled
	originalDir := terminalAuthGrantDir
	verifyTerminalChallenge = func(_ *gin.Context, challenge, action string) error {
		if challenge != "terminal-challenge" {
			return errors.New("unexpected challenge")
		}
		if action != authservice.SensitiveActionOpenWebTerminal {
			return errors.New("unexpected action")
		}
		return nil
	}
	terminalAuthenticationEnabled = func() bool { return true }
	terminalAuthGrantDir = filepath.Join(t.TempDir(), "terminal_auth.d")
	t.Cleanup(func() {
		verifyTerminalChallenge = originalVerify
		terminalAuthenticationEnabled = originalEnabled
		terminalAuthGrantDir = originalDir
	})

	token, err := middleware.GenerateJWT("admin")
	if err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(proto.SensitiveChallengeReq{Challenge: "terminal-challenge"})
	service := NewService()
	recorder, ctx := terminalAuthTestContextWithRecorder(token, http.MethodPost, "/api/vm/terminal/auth", body)
	ctx.Request.Header.Set("Content-Type", "application/json")

	service.TerminalAuth(ctx)

	var rsp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	if rsp.Code != 0 {
		t.Fatalf("expected success, got code=%d body=%s", rsp.Code, recorder.Body.String())
	}

	subject, ok := terminalAuthSubject(ctx)
	if !ok {
		t.Fatal("expected authenticated terminal subject")
	}
	content, err := os.ReadFile(terminalAuthGrantPath(terminalAuthGrantDir, subject))
	if err != nil {
		t.Fatal(err)
	}
	var grant terminalAuthGrant
	if err := json.Unmarshal(content, &grant); err != nil {
		t.Fatal(err)
	}
	remaining := time.Until(time.Unix(grant.ExpiresAt, 0))
	if remaining <= 0 || remaining > terminalAuthTTL {
		t.Fatalf("expected a grant lasting at most %s, got %s", terminalAuthTTL, remaining)
	}
	if !consumeTerminalAuthGrant(terminalAuthTestContext(token), terminalAuthGrantDir, time.Now()) {
		t.Fatal("expected challenge path to mint a consumable grant")
	}
}

func TestTerminalAuthRejectsInvalidChallenge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalVerify := verifyTerminalChallenge
	originalEnabled := terminalAuthenticationEnabled
	originalDir := terminalAuthGrantDir
	verifyTerminalChallenge = func(_ *gin.Context, _, _ string) error {
		return errors.New("invalid challenge")
	}
	terminalAuthenticationEnabled = func() bool { return true }
	terminalAuthGrantDir = filepath.Join(t.TempDir(), "terminal_auth.d")
	t.Cleanup(func() {
		verifyTerminalChallenge = originalVerify
		terminalAuthenticationEnabled = originalEnabled
		terminalAuthGrantDir = originalDir
	})

	token, err := middleware.GenerateJWT("admin")
	if err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(proto.SensitiveChallengeReq{Challenge: "wrong"})
	service := NewService()
	recorder, ctx := terminalAuthTestContextWithRecorder(token, http.MethodPost, "/api/vm/terminal/auth", body)
	ctx.Request.Header.Set("Content-Type", "application/json")

	service.TerminalAuth(ctx)

	var rsp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	if rsp.Code == 0 {
		t.Fatalf("expected failure, got body=%s", recorder.Body.String())
	}
	if consumeTerminalAuthGrant(terminalAuthTestContext(token), terminalAuthGrantDir, time.Now()) {
		t.Fatal("invalid challenge must not mint a terminal grant")
	}
}

func TestTerminalAuthRejectsDisabledAuthenticationAndMissingIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalVerify := verifyTerminalChallenge
	originalEnabled := terminalAuthenticationEnabled
	originalDir := terminalAuthGrantDir
	verifyCalled := false
	verifyTerminalChallenge = func(_ *gin.Context, _, _ string) error {
		verifyCalled = true
		return nil
	}
	terminalAuthGrantDir = filepath.Join(t.TempDir(), "terminal_auth.d")
	t.Cleanup(func() {
		verifyTerminalChallenge = originalVerify
		terminalAuthenticationEnabled = originalEnabled
		terminalAuthGrantDir = originalDir
	})

	token, err := middleware.GenerateJWT("admin")
	if err != nil {
		t.Fatal(err)
	}

	service := NewService()
	body, _ := json.Marshal(proto.SensitiveChallengeReq{Challenge: "terminal-challenge"})

	tests := []struct {
		name         string
		enabled      bool
		withIdentity bool
	}{
		{name: "authentication disabled", enabled: false, withIdentity: true},
		{name: "identity missing", enabled: true, withIdentity: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terminalAuthenticationEnabled = func() bool { return tt.enabled }
			recorder, ctx := terminalAuthTestContextWithRecorder(token, http.MethodPost, "/api/vm/terminal/auth", body)
			if !tt.withIdentity {
				ctx.Set("username", "")
				ctx.Set("jwtToken", "")
			}
			ctx.Request.Header.Set("Content-Type", "application/json")

			service.TerminalAuth(ctx)

			var rsp struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
				t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
			}
			if rsp.Code == 0 {
				t.Fatalf("expected failure, got body=%s", recorder.Body.String())
			}
		})
	}
	if verifyCalled {
		t.Fatal("challenge verifier must not run without an enabled authenticated session")
	}
}

func TestTerminalGrantUsesOwnerOnlyModes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "terminal_auth.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	token, err := middleware.GenerateJWT("admin")
	if err != nil {
		t.Fatal(err)
	}
	ctx := terminalAuthTestContext(token)
	if err := createTerminalAuthGrant(ctx, dir, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("expected grant directory mode 0700, got %04o", got)
	}
	subject, ok := terminalAuthSubject(ctx)
	if !ok {
		t.Fatal("expected authenticated terminal subject")
	}
	fileInfo, err := os.Stat(terminalAuthGrantPath(dir, subject))
	if err != nil {
		t.Fatal(err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected grant file mode 0600, got %04o", got)
	}
}

func TestTerminalAuthSubjectRequiresMiddlewareJWTIdentity(t *testing.T) {
	token, err := middleware.GenerateJWT("admin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/vm/terminal", nil)
	ctx.Request.RemoteAddr = "192.0.2.1:1234"
	ctx.Request.AddCookie(&http.Cookie{Name: "nano-kvm-token", Value: token})
	if _, ok := terminalAuthSubject(ctx); ok {
		t.Fatal("client IP or an unprocessed cookie must not identify a terminal session")
	}

	ctx.Set("username", "operator")
	ctx.Set("jwtToken", token)
	if _, ok := terminalAuthSubject(ctx); ok {
		t.Fatal("JWT claims must match the middleware username")
	}
}

func TestTerminalWebSocketRejectsDisabledAuthenticationBeforeConsumingGrant(t *testing.T) {
	originalEnabled := terminalAuthenticationEnabled
	originalDir := terminalAuthGrantDir
	terminalAuthenticationEnabled = func() bool { return false }
	terminalAuthGrantDir = filepath.Join(t.TempDir(), "terminal_auth.d")
	t.Cleanup(func() {
		terminalAuthenticationEnabled = originalEnabled
		terminalAuthGrantDir = originalDir
	})

	token, err := middleware.GenerateJWT("admin")
	if err != nil {
		t.Fatal(err)
	}
	recorder, ctx := terminalAuthTestContextWithRecorder(token, http.MethodGet, "/api/vm/terminal", nil)
	if err := createTerminalAuthGrant(ctx, terminalAuthGrantDir, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	NewService().Terminal(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}

	terminalAuthenticationEnabled = func() bool { return true }
	if !consumeTerminalAuthGrant(terminalAuthTestContext(token), terminalAuthGrantDir, time.Now()) {
		t.Fatal("disabled-auth rejection must happen before grant consumption")
	}
}
