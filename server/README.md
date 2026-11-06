# NanoKVM Server

This is the backend server implementation for NanoKVM.

For detailed documentation, please visit our [Wiki](https://wiki.sipeed.com/nanokvmpro).

## Structure

```shell
server
├── common       # Common utility components
├── config       # Server configuration
├── dl_lib       # Shared object libraries
├── include      # Header files for shared objects
├── logger       # Logging system
├── middleware   # Server middleware components
├── proto        # API request/response definitions
├── router       # API route handlers
├── service      # Core service implementations
├── utils        # Utility functions
└── main.go
```

## Configuration

The configuration file path is `/etc/kvm/server.yaml`.

The legacy `cn`/`global` region marker no longer restricts Tailscale modes.
Both regions offer official Tailscale and Sipeed Remote Network when
`/etc/kvm/server-feature.json` has `features.tailscale_cn.enabled: true`.
The existing flag name and `/config/cn/features.json` download URL are retained
for compatibility. Opening Settings → Remote Network sends an authenticated
`POST /api/application/features/sync` and waits for the manifest refresh before
loading network status. Each visit refreshes once; status polling does not fetch
the CDN. A failed refresh keeps the existing local manifest. Server startup
does not fetch it; the legacy region-change API still refreshes it when switching.
The region API remains available and no longer disconnects a Sipeed network.
The device screen hides both the Tailscale and Region settings entries
regardless of the region or feature manifest.

Sipeed Remote Network saves explicit consent in
`sipeedRemoteNetwork.disclaimerAccepted` (default: `false`). The first web enable
request must include `acceptDisclaimer: true`; saved consent is reused on later
starts, including requests through the retained local gate driver.
Ordinary logout preserves consent.
`POST /api/extensions/tailscale/sipeed/disclaimer/revoke` clears consent, logs out,
and uses the existing best-effort cloud network/user deletion. Local cleanup
failures remain retryable; cloud deletion failures do not block local cleanup.

```yaml
proto: http
port:
    http: 80
    https: 443
cert:
    crt: server.crt
    key: server.key

# Log level (debug/info/warn/error)
# Note: Use 'info' or 'error' in production, 'debug' only for development
logger:
    level: info
    file: stdout

# Authentication setting (enable/disable)
# Note: Only disable authentication in development environment
authentication: enable

# Finger robot runtime mode (normal/debug). Only the exact value "debug"
# enables the protected engineering test functions. Missing, empty, or invalid
# values run in normal mode. Changes take effect after restarting the server.
fingerRobot: normal

jwt:
   # JWT secret key. If left empty, a random 64-byte key will be generated automatically.
   secretKey: ""
   # JWT token expiration time in seconds. Default: 2678400 (31 days)
   refreshTokenDuration: 2678400
   # Invalidate all JWT tokens when the user logs out. Default: true
   revokeTokensOnLogout: true

twoFactorAuth:
   # Active TOTP secret file. The file is created with 0600 permissions.
   secretFile: /etc/kvm/totp.secret
   # TOTP issuer shown by authenticator apps.
   issuer: NanoKVM
   # Accepted TOTP time-step drift. Valid range: 0-2. Default: 1.
   window: 1

security:
    # CORS is consulted only when authentication is disabled. The built-in
    # development policy permits same-origin/no-Origin clients and the Vite
    # origins http://localhost:3001, http://127.0.0.1:3001, and
    # http://[::1]:3001. Add other development frontends as exact origins;
    # paths, wildcards, and malformed values are ignored.
    cors:
        allowedOrigins:
            - "http://localhost:4173"
    csrf:
        # Reject cookie-authenticated POST/PUT/PATCH/DELETE requests when both
        # Origin and Referer are missing. Keep this false unless you must support
        # legacy scripts that cannot send either header.
        allowMissingOrigin: false
        # Extra origins allowed to make cookie-authenticated state-changing
        # requests, for example a trusted HTTPS reverse proxy.
        trustedOrigins:
            - "https://kvm.example.com"
    redirect:
        # Extra Host values allowed when redirecting HTTP to HTTPS. Add the
        # public reverse-proxy host here when it is not a local/LAN host name.
        trustedHosts:
            - "kvm.example.com"
    download:
        # Allow image downloads from private/LAN addresses. Prefer allowedHosts
        # or allowedCIDRs over enabling this globally.
        allowPrivate: false
        # Private hosts that may be used as image download sources.
        allowedHosts:
            - "nas.local"
        # Private CIDR ranges that may be used as image download sources.
        allowedCIDRs:
            - "192.168.1.0/24"
        # Abort image downloads when no bytes are received for this many seconds.
        noProgressTimeout: 60

# Address for custom STUN server
# Note: You can disable the STUN service by setting it to 'disable' (e.g., in a LAN environment)
stun: stun.l.google.com:19302

# Address and authentication for custom TURN server
turn:
    turnAddr: example_addr
    turnUser: example_user
    turnCred: example_cred
```

Two-factor authentication uses TOTP. See [docs/2fa.md](../docs/2fa.md) for
runtime files, recovery-code behavior, password-change verification, and the
device validation checklist.

Finger robot Debug tests run asynchronously. While a test is running, the
client must poll `GET /api/vm/finger-robot/debug` more frequently than the
returned `leaseTimeoutMs`. If that lease expires (for example, after a browser
is closed or disconnected), the server cancels the sequence and restores the
servo to the safe `2050 us` release point. The fixed `1500 us` factory zero is
used only by the explicit Debug reset action.

PWM initialization follows the Finger Robot switch stored in
`/etc/kvm/menubar`. If the file is missing, malformed, or contains `"power"` in
`disabledItems`, the server starts without touching PWM. Enabling the switch
initializes both `normal` and `debug` modes directly at the `2050 us` release
point. Disabling it restores `2050 us` and keeps PWM enabled at that value.

## Build & Deploy

1. Install the ARMv7 toolchain

```bash
cd ROOT/support/scripts
./toolchain_setup.sh
```

This installs the ARMv7 hard-float toolchain and armhf libopus packages, then
writes `ROOT/support/toolchains/toolchain.ini`.

If you already have a compatible toolchain, you can skip `toolchain_setup.sh`
and create `toolchain.ini` yourself:

```ini
[toolchain]
cc = /absolute/path/to/arm-none-linux-gnueabihf-gcc
cxx = /absolute/path/to/arm-none-linux-gnueabihf-g++
ld = /absolute/path/to/arm-none-linux-gnueabihf-ld
```

You can also override the toolchain at build time:

```bash
export CC=/absolute/path/to/arm-none-linux-gnueabihf-gcc
export CXX=/absolute/path/to/arm-none-linux-gnueabihf-g++
export SYSROOT=/path/to/armv7/sysroot
```

2. Build `NanoKVM-Server` for ARMv7

```bash
cd ROOT
python3 build.py server
```

The root driver calls the component build implementation and copies the result
to `ROOT/build/NanoKVM-Server`.

3. Deploy

```bash
cd ROOT
export NANOKVM_IP=your-nanokvm-ip

# Stop nanokvm-go service
ssh root@${NANOKVM_IP} 'systemctl stop nanokvm-go'

# Push server binary
scp build/NanoKVM-Server root@${NANOKVM_IP}:/kvmapp/server/

# Start nanokvm-go service
ssh root@${NANOKVM_IP} 'systemctl start nanokvm-go'
```
