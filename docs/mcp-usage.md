# NanoKVM-Go MCP Usage

NanoKVM-Go exposes one optional MCP service for control/media tools and Memory Fabric's
read-only memory/OCR tools.

The service is disabled by default. Enable it only on trusted networks and
treat the MCP API key as a credential for the union of all enabled tools.

> **Migration from the old memfab endpoint:** unauthenticated
> `http://<device>:8742/mcp` has been retired. Change the client URL to
> `https://<device>/api/mcp`, add `Authorization: Bearer <existing MCP API key>`, and
> enable both **MCP Service** and **Memory Fabric**. Do not reopen port 8742 as a
> fallback. Existing `/api/mcp` control clients keep their URL and key, but the same key
> now also authorizes memory/OCR reads whenever Memory Fabric is enabled.

## Requirements

- A NanoKVM-Go device running a build that includes the AI MCP service.
- Network access from the MCP client to the device HTTPS endpoint.
- An MCP client that supports Streamable HTTP transport.
- The MCP service enabled in `Settings > AI`.
- The current MCP API key copied from `Settings > AI`.
- For memory/OCR tools, Memory Fabric also enabled in `Settings > AI`.

## Enable the Service

1. Open the NanoKVM-Go web UI.
2. Go to `Settings > AI`.
3. Enable `MCP Service`.
4. Copy the generated API key.
5. If memory/OCR tools are needed, separately enable `Memory Fabric`.

The endpoint shown by the UI has this form:

```text
https://<device-ip-or-host>/api/mcp
```

Example:

```text
https://192.168.1.100/api/mcp
```

The device may redirect plain HTTP to HTTPS with `307 Temporary Redirect`.
Use the HTTPS URL directly.

## Authentication

MCP requests use a dedicated API key. Do not use the normal web login token.
That single key authorizes control/media tools and, when Memory Fabric is enabled,
read-only memory/OCR tools. Review existing key holders before enabling Memory Fabric.

Every MCP request must include:

```http
Authorization: Bearer <mcp_api_key>
```

The `Bearer ` prefix is required.

Expected authentication responses:

| Case | Response |
| --- | --- |
| MCP service disabled | `403 {"error":"MCP service is disabled"}` |
| Missing, malformed, or wrong key | `401 {"error":"invalid MCP API key"}` |
| Valid key, raw `GET` without session | MCP protocol error such as `GET requires an Mcp-Session-Id header` |

The raw `GET` protocol error means authentication passed, but the request is
not a complete MCP session request.

## TLS Notes

Many devices use a self-signed HTTPS certificate. For `curl`, use `-k` to skip certificate verification:

```bash
curl -k -i https://<device-ip>/api/mcp
```

## MCP Client Configuration

Use Streamable HTTP transport:

```text
URL: https://<device-ip-or-host>/api/mcp
Authorization: Bearer <mcp_api_key>
```

If the client lets you configure custom headers, add:

```json
{
  "Authorization": "Bearer <mcp_api_key>"
}
```

This first version exposes an HTTP MCP endpoint only. It does not provide a
stdio MCP server.

## Switches and Tool Availability

| MCP Service | Memory Fabric | Remote result |
| --- | --- | --- |
| Off | Off | `/api/mcp` returns `403`; no remote tools |
| Off | On | `/api/mcp` returns `403`; local memory processing may continue |
| On | Off | Authenticated control/media tools only |
| On | On and ready | Authenticated control/media plus memory/OCR tools |
| On | On but unavailable | Control/media tools remain available; memory/OCR tools are omitted or fail closed |

Memory Fabric never creates a second remote endpoint. Its eight existing read-only tools
join the same authenticated `tools/list` response without changing their names, input
schemas, or result shape:

| Memory/OCR tool | Purpose |
| --- | --- |
| `list_memories` | List memory files |
| `read_memory` | Read a selected memory file |
| `search` | Search stored memories |
| `recent_activity` | Return recent memory activity |
| `read_recent_capture` | Read a recent OCR-backed capture |
| `search_captures` | Search OCR-backed captures |
| `current_context` | Return current OCR-backed screen context |
| `get_schema` | Return the memory schema |

Use `tools/list` as the authoritative schema source for the connected version. Successful
memory/OCR calls keep their existing JSON text content and structured-result contract.

## Control and Media Tools

### `type_text`

Types literal text using the target keyboard layout saved on the NanoKVM. Select
**Keyboard → Keyboards → language → layout** to match the target's active PC
layout. English (US) is the default. Web Paste uses the same setting. Keep Caps Lock off.
A newline sends Enter and a tab sends Tab.

Russian uses the standard Windows ЙЦУКЕН keyboard. Persian and Persian (Standard,
ISIRI 9147) are separate Windows layouts. These profiles reject characters unavailable
on that layout, including Latin letters.

For Mandarin, select **Chinese → Simplified Chinese · Microsoft Pinyin**. On the
target, activate Microsoft Pinyin in Chinese mode with Full Pinyin and V-mode
enabled. The KVM enters each printable Basic Multilingual Plane character by its
Unicode code point, including Latin letters and punctuation in mixed text.
This takes eight key strokes per character. Supplementary characters are rejected
before typing. [Microsoft Pinyin V-mode](https://support.microsoft.com/en-us/windows/hardware/input-devices/microsoft-simplified-chinese-ime)

Arguments:

```json
{
  "text": "hello",
  "delayMs": 30
}
```

Fields:

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `text` | string | yes | Up to 1,024 Unicode code points. An unsupported character returns an error before any keys are sent. |
| `delayMs` | integer | no | Delay between HID reports. Defaults to `30`. Clamped to `0..1000`. |

Result:

```json
{
  "typed": 5,
  "skippedCount": 0
}
```

### `press_keys`

Presses one keyboard shortcut on the remote host.

Arguments:

```json
{
  "keys": ["ControlLeft", "AltLeft", "Delete"],
  "holdMs": 50
}
```

Fields:

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `keys` | string array | yes | KeyboardEvent.code-style key names. |
| `holdMs` | integer | no | Hold time in milliseconds. Defaults to `50`. Clamped to `0..1000`. |

Result:

```json
{
  "ok": true
}
```

Use destructive shortcuts only in a safe test environment.

### `move_mouse`

Moves the remote mouse pointer.

Absolute movement:

```json
{
  "mode": "absolute",
  "x": 0.5,
  "y": 0.5
}
```

Relative movement:

```json
{
  "mode": "relative",
  "deltaX": 40,
  "deltaY": 20
}
```

Fields:

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `mode` | string | yes | `absolute` or `relative`. |
| `x` | number | for absolute | Normalized horizontal position inside the screenshot content area, `0.0..1.0`. |
| `y` | number | for absolute | Normalized vertical position inside the screenshot content area, `0.0..1.0`. |
| `deltaX` | integer | for relative | Relative horizontal movement. |
| `deltaY` | integer | for relative | Relative vertical movement. |

Result:

```json
{
  "ok": true
}
```

### `click_mouse`

Clicks a remote mouse button. It can optionally move to an absolute position
before clicking.

If the target is an iPhone, right-click once before the first click.

Arguments:

```json
{
  "button": "left",
  "clicks": 1,
  "mode": "absolute",
  "x": 0.5,
  "y": 0.5
}
```

Fields:

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `button` | string | yes | `left`, `right`, `middle`, `back`, or `forward`. |
| `clicks` | integer | no | Number of clicks. Defaults to `1`. Clamped to `1..5`. |
| `mode` | string | no | `absolute` or `relative`. Defaults to `relative`. |
| `x` | number | for absolute | Normalized horizontal position, `0.0..1.0`. |
| `y` | number | for absolute | Normalized vertical position, `0.0..1.0`. |

Result:

```json
{
  "ok": true
}
```

### `scroll_mouse`

Scrolls the remote mouse wheel vertically.

Arguments:

```json
{
  "deltaY": -3
}
```

Fields:

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `deltaY` | integer | yes | Vertical wheel movement. Sign depends on the target OS and UI. |

Result:

```json
{
  "ok": true
}
```

### `screenshot`

Captures a JPEG screenshot from the remote display.

Arguments:

```json
{
  "quality": 75,
  "timeoutMs": 5000,
  "x": 0,
  "y": 0,
  "w": 0,
  "h": 0
}
```

Fields:

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `quality` | integer | no | JPEG quality. |
| `timeoutMs` | integer | no | Screenshot timeout in milliseconds. |
| `x` | integer | no | Crop origin X inside the content area. Needs `w` and `h`. |
| `y` | integer | no | Crop origin Y inside the content area. Needs `w` and `h`. |
| `w` | integer | no | Crop width inside the content area. `0` keeps the whole content area. A crop does not change the mouse coordinate space. |
| `h` | integer | no | Crop height inside the content area. `0` keeps the whole content area. A crop does not change the mouse coordinate space. |

Result metadata:

```json
{
  "ok": true,
  "retCode": 0,
  "message": "",
  "width": 1596,
  "height": 924,
  "size": 178096,
  "sourceWidth": 3840,
  "sourceHeight": 2160,
  "coordinateSpace": "content",
  "areaSource": "auto",
  "areaUpdatedAt": 1757588298605,
  "content": { "left": 1073, "top": 638, "width": 1596, "height": 924 },
  "capture": { "left": 1073, "top": 638, "width": 1596, "height": 924 }
}
```

The MCP content includes an image item:

```json
{
  "type": "image",
  "mimeType": "image/jpeg",
  "data": "<base64 jpeg>"
}
```

### Coordinate space and portal cropping

`screenshot`, `move_mouse`, and absolute `click_mouse` share one coordinate space: the
**content area** of the remote frame. Portal cropping ("black border removal", configured in the
web UI) is applied on the server, so the returned JPEG is already de-letterboxed and normalized
absolute coordinates are relative to the full content area, never to a cropped image. `x`/`y` are
fractions of the image (`0.0..1.0`), never pixels; a value outside that range is rejected instead
of being clamped to a corner.

- Portal mode `off`, or no learned/detected area: the content area is the whole frame and behavior
  is unchanged.
- Portal mode `manual`: the content area is the selected region or resolution preset.
- Portal mode `auto`: the content area is the last stable area detected by the web UI, stored on
  the device (`/etc/kvm/portal.json`), so MCP keeps cropping while no browser is open.
- A region recorded for a different frame aspect is ignored instead of being rescaled.

`screenshot` metadata reports the applied region:
`sourceWidth`/`sourceHeight` describe the whole captured frame, `width`/`height` the returned
image, `content` the full content area inside the source frame, and `capture` the part of the
source frame the returned image actually shows. The two are equal until a caller passes `x`/`y`/`w`/`h`
to `screenshot`. `coordinateSpace` is always `content`; `areaSource` is `auto`, `manual`, or `none`;
`areaUpdatedAt` is the browser report timestamp in milliseconds (absent for manual regions).

Absolute coordinates follow the device input profile exactly like the browser does: the content
point is expressed in the captured frame and the selected profile calibration is applied on top.
With the built-in `automatic-crop-calibration` profile (the device default) the detected content
rectangle is mapped onto the complete HID range, so `x`/`y` is the normalized position inside the
content area. Custom profiles apply their four-point calibration unchanged.

A cropped screenshot does not move that space. To click a point seen in a cropped image, first map
it back to the content area, where image coordinates are normalized `0.0..1.0` inside the captured
image:

```text
contentX = (capture.left - content.left + x * capture.width) / content.width
contentY = (capture.top - content.top + y * capture.height) / content.height
```

The capture hardware requires an even crop origin and sides of at least 32 pixels, so the applied
crop may be aligned by up to one pixel and a smaller `w`/`h` request is enlarged to 32.

Callers that must keep the raw whole-frame meaning of `x`/`y` can pass
`"coordinateSpace": "source"` to `move_mouse` or absolute `click_mouse`.

Relative mouse movement never uses the content area.

### Media sessions

MCP controls real-time media; it does not carry Opus packets. Create a
short-lived session, then establish WebRTC media tracks using the returned
token and signaling path.

New agents should call the token-free `media_session_help` tool first. It
returns the versioned integration contract: capability request, WebSocket
offer/answer/ICE envelope, speaker and microphone track directions, lifecycle,
and security constraints. This lets an agent build a WebRTC client without
depending on repository source or a debugging utility.

`media_session_create` accepts `speakerReceive`, `micSend`, and optional
`videoReceive`. It returns an opaque token, expiry, and an audio-only signaling
path such as `/api/media/audio/webrtc?media_session=<token>` (or the legacy
video signaling endpoint when video is requested). The agent establishes WebRTC and
uses its audio media track for speaker receive and microphone send.

Use `media_session_status` to inspect a token and `media_session_close` to
revoke it. Tokens are scoped to their requested capabilities and expire within
ten minutes. Do not use the MCP API key as a WebRTC credential.

For transcription or conversation, the agent receives speaker audio through
its WebRTC media session and handles segmentation, ASR, response generation,
and TTS `micSend` playback. Configure any ASR service in the agent runtime.

The audio-only signaling endpoint is a WebSocket. Send JSON messages in this
form:

```json
{"event":"offer","data":"<JSON-encoded RTCSessionDescription>"}
{"event":"candidate","data":"<JSON-encoded RTCIceCandidateInit>"}
```

The server replies with `answer` and `candidate` messages in the same envelope.
`speakerReceive` adds one server-to-client Opus track; `micSend` permits one
client-to-server Opus microphone track delivered to the UAC2 gadget.

## Curl Smoke Test

The commands below validate the endpoint without issuing keyboard or mouse
actions. Replace the host and key first.

```bash
HOST='https://<device-ip-or-host>'
KEY='<mcp_api_key>'
OUT='/tmp/NanoKVM-Go-mcp-test'
mkdir -p "$OUT"

curl -k -sS -D "$OUT/init.headers" -o "$OUT/init.sse" -X POST \
  -H "Authorization: Bearer $KEY" \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  --data '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"curl-smoke","version":"0.1"}}}' \
  "$HOST/api/mcp"

SESSION_ID="$(awk 'tolower($1)=="mcp-session-id:" {gsub("\r","",$2); print $2}' "$OUT/init.headers" | tail -n1)"
echo "SESSION_ID=$SESSION_ID"
sed -n '1,40p' "$OUT/init.sse"
```

Expected:

- HTTP status `200`.
- Response header `mcp-session-id`.
- SSE body containing `nanokvm-remote-control`.

Send the initialized notification:

```bash
curl -k -i -X POST \
  -H "Authorization: Bearer $KEY" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -H 'Mcp-Protocol-Version: 2025-03-26' \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  --data '{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}' \
  "$HOST/api/mcp"
```

Expected: HTTP status `202`.

List tools:

```bash
curl -k -sS -D "$OUT/tools-list.headers" -o "$OUT/tools-list.sse" -X POST \
  -H "Authorization: Bearer $KEY" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -H 'Mcp-Protocol-Version: 2025-03-26' \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  --data '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  "$HOST/api/mcp"

grep -Eo '"name":"[^"]+"' "$OUT/tools-list.sse"
```

Expected tool names:

```text
click_mouse
move_mouse
press_keys
screenshot
scroll_mouse
type_text
```

Capture a screenshot:

```bash
curl -k -sS -D "$OUT/screenshot.headers" -o "$OUT/screenshot.sse" -X POST \
  -H "Authorization: Bearer $KEY" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -H 'Mcp-Protocol-Version: 2025-03-26' \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  --data '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"screenshot","arguments":{"quality":75,"timeoutMs":5000}}}' \
  "$HOST/api/mcp"

grep -Eo '"mimeType":"image/jpeg"|"ok":true|"width":[0-9]+|"height":[0-9]+|"size":[0-9]+' "$OUT/screenshot.sse"
```

Expected:

- HTTP status `200`.
- `mimeType` is `image/jpeg`.
- Metadata includes `ok:true`, `width`, `height`, and `size`.

Close the MCP session when finished:

```bash
curl -k -i -X DELETE \
  -H "Authorization: Bearer $KEY" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -H 'Accept: application/json, text/event-stream' \
  "$HOST/api/mcp"
```

Expected: HTTP status `204`.

## Control Tool Examples

These commands send real input to the remote host. Use them only when the
remote machine is in a safe test state.

Type text:

```bash
curl -k -i -X POST \
  -H "Authorization: Bearer $KEY" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -H 'Mcp-Protocol-Version: 2025-03-26' \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  --data '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"type_text","arguments":{"text":"NanoKVM-Go MCP test 123","delayMs":30}}}' \
  "$HOST/api/mcp"
```

Move mouse to the center:

```bash
curl -k -i -X POST \
  -H "Authorization: Bearer $KEY" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -H 'Mcp-Protocol-Version: 2025-03-26' \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  --data '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"move_mouse","arguments":{"mode":"absolute","x":0.5,"y":0.5}}}' \
  "$HOST/api/mcp"
```

Left click:

```bash
curl -k -i -X POST \
  -H "Authorization: Bearer $KEY" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -H 'Mcp-Protocol-Version: 2025-03-26' \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  --data '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"click_mouse","arguments":{"button":"left","clicks":1}}}' \
  "$HOST/api/mcp"
```

Scroll:

```bash
curl -k -i -X POST \
  -H "Authorization: Bearer $KEY" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -H 'Mcp-Protocol-Version: 2025-03-26' \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  --data '{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"scroll_mouse","arguments":{"deltaY":-3}}}' \
  "$HOST/api/mcp"
```

Safe key press example:

```bash
curl -k -i -X POST \
  -H "Authorization: Bearer $KEY" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -H 'Mcp-Protocol-Version: 2025-03-26' \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  --data '{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"press_keys","arguments":{"keys":["ShiftLeft"],"holdMs":50}}}' \
  "$HOST/api/mcp"
```

Use a short delay between rapid tool calls. The server has a small per-tool
rate limit to protect the HID and screenshot paths.

## Troubleshooting

| Symptom | Meaning | Fix |
| --- | --- | --- |
| `307 Temporary Redirect` | HTTP was redirected to HTTPS. | Use `https://<device>/api/mcp`. |
| `curl: (60) SSL certificate problem` | The device certificate is self-signed. | Use `curl -k` for tests or trust the certificate. |
| `401 invalid MCP API key` | Missing key, wrong key, or missing `Bearer ` prefix. | Use `Authorization: Bearer <mcp_api_key>`. |
| `403 MCP service is disabled` | The AI MCP switch is off. | Enable `Settings > AI > MCP Service`. |
| A memory/OCR tool is absent | Memory Fabric is off, starting, unavailable, or its local contract failed validation. | Enable both switches and check the local Memory Fabric service; do not expose port 8742. |
| `GET requires an Mcp-Session-Id header` | Raw GET is not a complete MCP request. | Run `initialize` first or use a real MCP client. |
| `session not found` | The session id is missing, expired, wrong, or from another server restart. | Re-run `initialize` and use the new `mcp-session-id`. |
| Tool returns a rate limit error | Calls are too close together. | Wait briefly and retry. |

## Security Notes

- Anyone with the MCP API key can use control/media tools and, when Memory Fabric is
  enabled, read memory and OCR-derived screen context.
- Keep the service disabled when not in use.
- Regenerate the key if it may have been exposed. Existing clients must be
  updated after regeneration.
- Restrict network access to the device. Do not expose `/api/mcp` directly to
  untrusted networks.
