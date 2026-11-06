# memfab

`memfab` is the NanoKVM-Go memory fabric runtime for OCR-backed local memory.

## Runtime defaults

- Binary: `memfab`
- Runtime root: `MEMFAB_ROOT=/var/lib/memfab`
- Systemd service: `memfab.service`
- Internal tool socket: `/run/memfab/tools.sock` (local root-only backend)
- Public MCP endpoint: `https://<device>/api/mcp` (served by NanoKVM-Go)
- Ingest endpoint: `http://127.0.0.1:8743/v1/observations`
- Memory/OCR tool names are unchanged: `list_memories`, `read_memory`, `search`,
  `recent_activity`, `read_recent_capture`, `search_captures`, `current_context`, and
  `get_schema`

## MCP migration

The unauthenticated public `http://<device>:8742/mcp` listener has been retired. Remote
clients must migrate to `https://<device>/api/mcp` and send
`Authorization: Bearer <existing MCP API key>`. Enable both **MCP Service** and
**Memory Fabric** in `Settings > AI`; Memory Fabric alone continues local processing but
does not expose remote tools.

The existing MCP key now authorizes the combined control/media and memory/OCR tool set.
Treat every current key holder as having those additional read capabilities whenever both
switches are enabled. The key is not sent to the memfab process.

## Build and test

```bash
go test -tags sqlite_fts5 ./...
cmake -S . -B build-go
cmake --build build-go --target memfab-native
cmake --build build-go --target memfab-linux-armv7-with-toolchain
```

The armv7 build uses the NanoKVM-Go toolchain at:

```text
../support/toolchains/armv7hf-toolchains/bin/arm-none-linux-gnueabihf-gcc
```

## Local smoke

```bash
MEMFAB_ROOT=/tmp/memfab-status ./build-go/dist/memfab status
file build-go/dist/memfab-linux-armv7
```

The standalone `memfab mcp` command is a foreground diagnostic for the local Unix-socket
backend. It does not open a TCP MCP listener.

## Deploy

```bash
scripts/deploy-nanokvm-go.sh root@<device-ip>
```

The authenticated MCP smoke requires the existing key without printing it:

```bash
MCP_API_KEY='<existing-key>' scripts/deploy-nanokvm-go.sh root@<device-ip>
```

The deploy script keeps its NanoKVM-Go-oriented filename, but installs:

- `/usr/bin/memfab`
- `/etc/systemd/system/memfab.service`
- `/var/lib/memfab`
