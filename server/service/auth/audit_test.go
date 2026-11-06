package auth

import (
	"net/http"
	"strings"
	"testing"

	"NanoKVM-Server/proto"

	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func TestAuthAuditFiltersSensitiveFields(t *testing.T) {
	hook := logtest.NewGlobal()
	t.Cleanup(func() { hook.Reset() })

	auditAuthEvent("auth_login_failed", log.Fields{
		"user":               "admin",
		"ip":                 "192.0.2.1",
		"op":                 "login",
		"reason":             "credential_invalid",
		"code_id":            "code-1",
		"password":           "secret",
		"code":               "123456",
		"recovery_code":      "nkv-8H7K-2QPD-9M4X",
		"recovery_code_hash": "$2a$04$hash",
		"challenge":          "opaque-challenge",
	})

	entries := hook.AllEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.Message != "auth_login_failed" {
		t.Fatalf("unexpected audit event %q", entry.Message)
	}
	for _, key := range []string{"user", "ip", "op", "reason", "code_id"} {
		if _, ok := entry.Data[key]; !ok {
			t.Fatalf("expected allowed audit field %q", key)
		}
	}
	for _, key := range []string{"password", "code", "recovery_code", "recovery_code_hash", "challenge"} {
		if _, ok := entry.Data[key]; ok {
			t.Fatalf("sensitive audit field %q must not be logged", key)
		}
	}
}

func TestAuthAuditLoginFailureDoesNotLogSensitiveRequest(t *testing.T) {
	setupAuthHandlerTest(t)
	hook := logtest.NewGlobal()
	t.Cleanup(func() { hook.Reset() })
	service := NewService()

	_, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "wrong-secret"})
	assertAuthCode(t, rsp, -2)

	entry := findAuditEntry(t, hook.AllEntries(), "auth_login_failed")
	assertAuditEntryOmits(t, entry, "wrong-secret", "123456", "nkv-8H7K-2QPD-9M4X", "opaque-challenge")
	if entry.Data["user"] != "admin" || entry.Data["reason"] != "credential_invalid" {
		t.Fatalf("unexpected login failure audit fields: %+v", entry.Data)
	}
}

func TestAuthAuditWebTerminalFailureDoesNotLogStepUpSecrets(t *testing.T) {
	setupAuthHandlerTest(t)
	hook := logtest.NewGlobal()
	t.Cleanup(func() { hook.Reset() })
	service := NewService()

	_, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   SensitiveActionOpenWebTerminal,
		Password: "wrong-terminal-password",
		Code:     "terminal-recovery-code",
	}, "admin")
	assertAuthCode(t, rsp, -2)

	entry := findAuditEntry(t, hook.AllEntries(), "auth_sensitive_action_failed")
	assertAuditEntryOmits(t, entry, "wrong-terminal-password", "terminal-recovery-code")
	if entry.Data["op"] != SensitiveActionOpenWebTerminal || entry.Data["reason"] != "credential_invalid" {
		t.Fatalf("unexpected Web terminal failure audit fields: %+v", entry.Data)
	}
}

func TestAuthAuditRecoveryCodeUsedDoesNotLogPlaintextCode(t *testing.T) {
	setupAuthHandlerTest(t)
	hook := logtest.NewGlobal()
	t.Cleanup(func() { hook.Reset() })
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	recoveryFile, err := buildRecoveryCodesFile([]string{"nkv-8H7K-2QPD-9M4X"})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	if err := saveRecoveryCodes(recoveryFile); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}

	_, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	challenge := decodeAuthData[proto.LoginRsp](t, rsp).Challenge
	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Challenge: challenge, Code: "nkv-8H7K-2QPD-9M4X"})
	assertAuthCode(t, rsp, 0)

	entry := findAuditEntry(t, hook.AllEntries(), "auth_recovery_code_used")
	assertAuditEntryOmits(t, entry, "nkv-8H7K-2QPD-9M4X", challenge, recoveryFile.Codes[0].Hash)
	if entry.Data["code_id"] != recoveryFile.Codes[0].ID {
		t.Fatalf("expected recovery code id %q, got fields %+v", recoveryFile.Codes[0].ID, entry.Data)
	}
}

func TestAuthAuditTwoFactorOperationEvents(t *testing.T) {
	setupAuthHandlerTest(t)
	hook := logtest.NewGlobal()
	t.Cleanup(func() { hook.Reset() })
	service := NewService()

	_, rsp := performAuthJSONRequest(t, service.TwoFactorSetup, http.MethodPost, "/api/auth/2fa/setup", proto.TwoFactorSetupReq{Password: "secret"})
	assertAuthCode(t, rsp, 0)
	setup := decodeAuthData[proto.TwoFactorSetupRsp](t, rsp)
	_, rsp = performAuthJSONRequest(t, service.TwoFactorEnable, http.MethodPost, "/api/auth/2fa/enable", proto.TwoFactorEnableReq{Code: "000000"})
	assertAuthCode(t, rsp, -2)

	failed := findAuditEntry(t, hook.AllEntries(), "auth_2fa_operation_failed")
	assertAuditEntryOmits(t, failed, "secret", "000000", setup.Secret)
	if failed.Data["op"] != "enable" || failed.Data["reason"] != "second_factor_invalid" {
		t.Fatalf("unexpected 2FA failure audit fields: %+v", failed.Data)
	}

	validCode, err := generateTOTPCode(setup.Secret, totpNow())
	if err != nil {
		t.Fatalf("generateTOTPCode returned error: %v", err)
	}
	_, rsp = performAuthJSONRequest(t, service.TwoFactorEnable, http.MethodPost, "/api/auth/2fa/enable", proto.TwoFactorEnableReq{Code: validCode})
	assertAuthCode(t, rsp, 0)
	findAuditEntry(t, hook.AllEntries(), "auth_2fa_enabled")
}

func TestAuthAuditSensitiveRequestsDoNotUseDebugRequestLogging(t *testing.T) {
	hook := logtest.NewGlobal()
	t.Cleanup(func() { hook.Reset() })
	originalLevel := log.GetLevel()
	log.SetLevel(log.DebugLevel)
	t.Cleanup(func() { log.SetLevel(originalLevel) })

	if err := bindSensitiveRequest(authTestContext(t, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"}), &proto.LoginReq{}); err != nil {
		t.Fatalf("bindSensitiveRequest login returned error: %v", err)
	}
	if err := bindSensitiveRequest(authTestContext(t, http.MethodPost, "/api/auth/password", proto.ChangePasswordReq{Username: "admin", Password: "new-secret"}), &proto.ChangePasswordReq{}); err != nil {
		t.Fatalf("bindSensitiveRequest change password returned error: %v", err)
	}
	for _, entry := range hook.AllEntries() {
		if strings.Contains(entry.Message, "request:") {
			t.Fatalf("sensitive request binder should not debug-log request bodies, got %q", entry.Message)
		}
		assertAuditEntryOmits(t, entry, "secret", "new-secret")
	}
}

func findAuditEntry(t *testing.T, entries []*log.Entry, event string) *log.Entry {
	t.Helper()
	for _, entry := range entries {
		if entry.Message == event {
			return entry
		}
	}
	t.Fatalf("audit event %q not found in %d entries", event, len(entries))
	return nil
}

func assertAuditEntryOmits(t *testing.T, entry *log.Entry, values ...string) {
	t.Helper()
	for _, forbidden := range values {
		if forbidden == "" {
			continue
		}
		for key, value := range entry.Data {
			if value == forbidden {
				t.Fatalf("audit field %q leaked forbidden value %q", key, forbidden)
			}
		}
	}
}
