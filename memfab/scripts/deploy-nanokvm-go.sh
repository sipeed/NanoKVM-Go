#!/usr/bin/env bash
set -euo pipefail

TARGET="${1:-${TARGET:-}}"
if [[ -z "$TARGET" ]]; then
  echo "usage: $0 <user@host>" >&2
  echo "  or set TARGET=user@host" >&2
  exit 1
fi
REMOTE_BIN="${REMOTE_BIN:-/usr/bin/memfab}"
REMOTE_ROOT="${REMOTE_ROOT:-/var/lib/memfab}"
SERVICE_NAME="${SERVICE_NAME:-memfab.service}"
LOCAL_BINARY="${LOCAL_BINARY:-build-go/dist/memfab-linux-armv7}"
LOCAL_SERVICE="${LOCAL_SERVICE:-packaging/systemd/memfab.service}"
LOCAL_FIXTURE="${LOCAL_FIXTURE:-tests/ocr_example.json}"
GO_TAGS="sqlite_fts5"
MCP_API_KEY="${MCP_API_KEY:-}"

log() { printf '[memfab-deploy] %s\n' "$*"; }
remote() { ssh -o BatchMode=yes "$TARGET" "$@"; }
upload() {
  local src="$1"
  local dst="$2"
  remote "cat > '$dst'" < "$src"
}

if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
  log "configuring CMake"
  cmake -S . -B build-go \
    -DMEMFAB_GO_TAGS="sqlite_fts5" \
    -DMEMFAB_ARM_GO_TAGS="$GO_TAGS"
  log "cross-building armv7 binary"
  cmake --build build-go --target memfab-linux-armv7-with-toolchain
fi

if [[ "${SKIP_SMOKE:-0}" != "1" && -z "$MCP_API_KEY" ]]; then
  echo "MCP_API_KEY must contain the existing NanoKVM MCP key for the authenticated smoke test" >&2
  exit 1
fi

if [[ ! -x "$LOCAL_BINARY" ]]; then
  echo "missing executable binary: $LOCAL_BINARY" >&2
  exit 1
fi
if [[ ! -f "$LOCAL_SERVICE" ]]; then
  echo "missing systemd unit: $LOCAL_SERVICE" >&2
  exit 1
fi
if command -v file >/dev/null 2>&1; then
  if ! file "$LOCAL_BINARY" | grep -Eq 'ELF 32-bit.*ARM.*EABI5'; then
    echo "binary is not an armv7 EABI5 executable: $(file "$LOCAL_BINARY")" >&2
    exit 1
  fi
fi
log "uploading binary to $TARGET:/tmp/memfab-linux-armv7"
upload "$LOCAL_BINARY" /tmp/memfab-linux-armv7
remote "chmod +x /tmp/memfab-linux-armv7"

log "uploading systemd unit to $TARGET:/tmp/$SERVICE_NAME"
upload "$LOCAL_SERVICE" "/tmp/$SERVICE_NAME"

if [[ -f "$LOCAL_FIXTURE" ]]; then
  log "uploading OCR fixture to $TARGET:/tmp/memfab-ocr-example.json"
  upload "$LOCAL_FIXTURE" /tmp/memfab-ocr-example.json
else
  log "OCR fixture not found at $LOCAL_FIXTURE; fixture smoke test will be skipped"
fi

log "installing binary, root directory, and systemd service"
remote "set -e; mkdir -p '$REMOTE_ROOT'; install -m 0755 /tmp/memfab-linux-armv7 '$REMOTE_BIN'; install -m 0644 '/tmp/$SERVICE_NAME' '/etc/systemd/system/$SERVICE_NAME'; systemctl daemon-reload; systemctl enable '$SERVICE_NAME'; systemctl restart '$SERVICE_NAME'"

if [[ "${SKIP_SMOKE:-0}" == "1" ]]; then
  log "SKIP_SMOKE=1; deployment complete without smoke tests"
  exit 0
fi

log "checking service is active"
remote "systemctl is-active '$SERVICE_NAME'"

log "checking ingest health"
remote "for i in 1 2 3 4 5 6 7 8 9 10; do curl -fsS http://127.0.0.1:8743/health && exit 0; sleep 1; done; exit 1"

if [[ -f "$LOCAL_FIXTURE" ]]; then
  log "posting OCR fixture to /v1/observations"
  remote "python3 - <<'PYFIXTURE'
import json
from datetime import datetime, timedelta, timezone
path = '/tmp/memfab-ocr-example.json'
with open(path, 'r', encoding='utf-8') as f:
    data = json.load(f)
data['captured_at'] = (datetime.now(timezone.utc) - timedelta(seconds=90)).isoformat().replace('+00:00', 'Z')
data['capture_id'] = 'smoke:' + data['captured_at']
with open('/tmp/memfab-ocr-smoke-now.json', 'w', encoding='utf-8') as f:
    json.dump(data, f, ensure_ascii=False)
PYFIXTURE"
  remote "curl -fsS -X POST http://127.0.0.1:8743/v1/observations -H 'Content-Type: application/json' --data-binary @/tmp/memfab-ocr-smoke-now.json >/tmp/memfab-ocr-ingest.out; cat /tmp/memfab-ocr-ingest.out"
  log "building fallback timeline block from OCR fixture"
  remote "MEMFAB_ROOT='$REMOTE_ROOT' OPENAI_API_KEY= '$REMOTE_BIN' timeline tick; MEMFAB_ROOT='$REMOTE_ROOT' '$REMOTE_BIN' timeline list -n 5"
fi

log "checking that retired public MCP port 8742 is not listening"
remote "if command -v ss >/dev/null 2>&1; then
  if ss -ltn | grep -Eq '[:.]8742[[:space:]]'; then
    echo 'retired public MCP port 8742 is still listening' >&2
    exit 1
  fi
elif command -v netstat >/dev/null 2>&1; then
  if netstat -ltn | grep -Eq '[:.]8742[[:space:]]'; then
    echo 'retired public MCP port 8742 is still listening' >&2
    exit 1
  fi
else
  echo 'port 8742 check skipped: neither ss nor netstat is available' >&2
fi"

log "checking authenticated unified MCP tools/list and search_captures"
printf '%s' "$MCP_API_KEY" | remote "set -e
umask 077
key_file=\$(mktemp /tmp/memfab-mcp-key.XXXXXX)
trap 'rm -f \"\$key_file\"' EXIT
cat >\"\$key_file\"
MCP_KEY_FILE=\"\$key_file\" python3 - <<'PYREMOTE'
import json
import os
import ssl
import urllib.request

URL = 'https://127.0.0.1/api/mcp'
with open(os.environ['MCP_KEY_FILE'], 'r', encoding='utf-8') as key_file:
    api_key = key_file.read()
if not api_key:
    raise SystemExit('empty MCP API key')
BASE_HEADERS = {
    'Authorization': 'Bearer ' + api_key,
    'Content-Type': 'application/json',
    'Accept': 'application/json, text/event-stream',
}
TLS_CONTEXT = ssl._create_unverified_context()

def response_json(body):
    if body.lstrip().startswith('{'):
        return json.loads(body)
    for line in body.splitlines():
        if line.startswith('data: '):
            return json.loads(line[6:])
    raise RuntimeError('missing MCP JSON response')

def post(payload, session_id=None):
    headers = dict(BASE_HEADERS)
    if session_id:
        headers['Mcp-Session-Id'] = session_id
        headers['Mcp-Protocol-Version'] = '2025-06-18'
    req = urllib.request.Request(URL, data=json.dumps(payload).encode(), headers=headers, method='POST')
    with urllib.request.urlopen(req, timeout=10, context=TLS_CONTEXT) as resp:
        return resp.headers, resp.read().decode()

headers, body = post({
    'jsonrpc': '2.0',
    'id': 1,
    'method': 'initialize',
    'params': {
        'protocolVersion': '2025-06-18',
        'capabilities': {},
        'clientInfo': {'name': 'memfab-deploy', 'version': '0.1'},
    },
})
session_id = headers.get('mcp-session-id')
post({'jsonrpc': '2.0', 'method': 'notifications/initialized', 'params': {}}, session_id)
_, body = post({'jsonrpc': '2.0', 'id': 2, 'method': 'tools/list', 'params': {}}, session_id)
tools = [tool['name'] for tool in response_json(body)['result']['tools']]
for name in (
    'type_text',
    'list_memories',
    'read_memory',
    'search',
    'recent_activity',
    'read_recent_capture',
    'search_captures',
    'current_context',
    'get_schema',
):
    if name not in tools:
        raise SystemExit(f'missing MCP tool: {name}')
_, body = post({
    'jsonrpc': '2.0',
    'id': 3,
    'method': 'tools/call',
    'params': {'name': 'search_captures', 'arguments': {'query': 'Sipeed', 'limit': 3}},
}, session_id)
call_result = response_json(body)
if 'error' in call_result:
    raise SystemExit('search_captures returned an MCP error')
if 'Sipeed' not in body:
    raise SystemExit('search_captures smoke test did not find Sipeed')
print('Authenticated unified MCP smoke ok')
PYREMOTE"

log "target status"
remote "MEMFAB_ROOT='$REMOTE_ROOT' '$REMOTE_BIN' status"

log "resource snapshot"
remote "ps -o pid,rss,vsz,comm,args | grep memfab | grep -v grep || true; free -m; du -sh '$REMOTE_ROOT' || true"

log "deployment complete"
