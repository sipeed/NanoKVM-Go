# Two-Factor Authentication

NanoKVM-Go supports two-factor authentication with TOTP codes from a standard authenticator app. The first implementation is a single device-level TOTP model: one active TOTP secret protects the Web UI login flow and 2FA-sensitive account operations.

It does not support multiple enrolled TOTP devices, WebAuthn/passkeys, U2F hardware keys, SMS, email codes, or external MFA providers.

## Runtime Files

The active TOTP secret is stored at:

```text
/etc/kvm/totp.secret
```

The file is created with `0600` permissions. A missing or empty file means TOTP is disabled. The secret must not be logged, copied into bug reports, or committed to this repository.

Recovery code state is stored at:

```text
/etc/kvm/recovery_codes.json
```

This file is also created with `0600` permissions. It stores recovery code hashes and usage metadata, not plaintext recovery codes.

## Behavior

When 2FA is disabled, password login remains compatible with the existing username and password flow.

When 2FA is enabled, login is a two-step flow:

1. The user submits username and password.
2. If the password is valid, the server returns a short-lived in-memory login challenge without issuing a JWT.
3. The user submits the challenge with either a valid TOTP code or an unused recovery code.
4. The server consumes the challenge and issues the JWT.

Login challenges are single-use and kept only in server memory. They are cleared by process restart.

TOTP uses 30-second time steps and accepts the configured time drift window. Device time must be accurate; NTP or another reliable time source is required for dependable login.

## Setup And Recovery Codes

The setup flow creates a pending TOTP secret in server memory and returns a provisioning URI for QR-code rendering. The active secret is written only after the user confirms setup with a valid code. Pending setup expires after 10 minutes and can be cancelled. After 2FA is enabled, the setup and enable endpoints reject direct TOTP replacement; disable 2FA first, then start setup again.

Recovery codes are disabled by default. After TOTP is enabled, the user can enable recovery codes. Existing recovery codes cannot be separately disabled while 2FA remains enabled; they can only be regenerated. Disabling 2FA invalidates all recovery codes.

Recovery codes are accepted anywhere the Web UI asks for a second factor after 2FA is enabled: login, password-change verification, recovery-code regeneration, and disabling 2FA. This is intentional so a user who loses the authenticator device can still disable 2FA if they still know the account password and have an unused recovery code.

Recovery codes are shown to the user once when generated or regenerated. They should be stored outside the NanoKVM device and must not be saved in repository files, screenshots, logs, or test artifacts.

## Sensitive Account Actions

When 2FA is enabled, changing the account password requires a sensitive-action verification step before `POST /api/auth/password` can succeed. The verification step checks the current password and a valid TOTP or recovery code, then returns a short-lived single-use challenge for the `change_password` action.

Direct password API calls without that challenge are rejected while 2FA is enabled. Refreshing or directly opening the password page must not bypass this requirement.
