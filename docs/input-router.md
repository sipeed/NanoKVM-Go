# Input Router Architecture

## Goals

The input pipeline separates browser capture from USB HID output:

```text
Browser input sources
  -> capture adapters
  -> Viewport / TouchPort mapping
  -> normalized input protocol
  -> WebSocket transport
  -> KVM-GO input router
  -> validation and policy
  -> HID report encoders
  -> gadget endpoint registry
  -> /dev/hidgN
```

The router must not infer a device from payload length. Every input frame carries a protocol
version, source, semantic event kind, sequence, capture timestamp, and typed payload.

## Browser Sources

Sources describe where an event was captured, independently of the requested remote device:

- browser keyboard
- browser pointer
- browser touchscreen
- virtual keyboard
- virtual pointer
- virtual touchscreen
- WebHID device

Legacy WebSocket message types `1` and `2` are migration transport sources rather than browser
capture APIs. The server labels them `legacy-websocket-keyboard` and
`legacy-websocket-pointer`, converts their HID-shaped reports into semantic events, and sends
them through the same validation, session, trace, recovery, and route pipeline as version 1
frames. They never write Gadget endpoints directly.

Capture adapters own browser APIs and browser-side state only. They normalize coordinates,
buttons, contacts, and key state, but do not know Linux gadget paths or USB report descriptors.

## Browser Coordinate Model

The browser coordinate pipeline follows the same separation used by
[Sunshine's touch-port model](https://github.com/LizardByte/Sunshine/blob/master/src/input.h):

- `InputViewport` describes the complete source frame and the visible content rectangle inside it.
- `TouchPort` combines that viewport with the browser input region's client rectangle.
- Client coordinates first become normalized content coordinates, then map into normalized source
  coordinates before entering the input-router protocol.

Absolute mouse, Stylus, and direct touchscreen routes use normalized coordinates from the same
`Viewport`/`TouchPort` model. Coordinates from a cropped or black-border-removed content rectangle are
first inverse-mapped into the original source frame. The browser does not apply an aspect-ratio
compensation to Stylus; device-specific behavior is handled by the selected HID Profile. Relative
pointer input does not use the viewport mapping.

The active `TouchPort` is copied when a mouse drag or touchscreen gesture starts and remains fixed
until every button or contact is released. Video layout changes and automatic crop updates therefore
cannot apply different transforms to contacts within one gesture.

Four-point calibration profiles apply after `TouchPort` mapping and before the semantic event enters
the input router. Profiles are stored on the KVM-GO at `/etc/kvm/input-profiles.json`, shared by all
browsers through `/api/vm/input-profiles`, and cached in browser storage for offline fallback. The first
browser opening an upgraded device migrates its existing browser-local profiles when no device config
exists. Profiles are user-named and store independent mappings for absolute mouse, Android stylus, and
direct touchscreen modes. Each mapping pairs four normalized source points with four normalized HID
target points. Guided calibration first lets the user drag and resize a
four-point frame inside the visible remote display. During point capture the selected profile's old
calibration is completely bypassed. Absolute mouse and Stylus both use the frozen `Viewport`/`TouchPort`
baseline, including inverse mapping from the visible crop into the original source frame. The user then
moves the uncalibrated remote pointer onto that point and confirms the last raw HID coordinate. The
resulting transform maps the visible control point to the sampled HID value, rather than compounding a
previous calibration.

The default frame uses the visible 10%–90% safe area, and the projective mapping extrapolates to the
full frame before clamping the final HID coordinate. For Stylus, an active calibration adjusts the same
full-range coordinate directly. A completed profile and its mapping are frozen for the duration of an
active drag or multi-touch gesture. Relative pointer input does not use calibration
profiles. Built-in profiles are read-only and cannot start guided calibration; the user must first
create a custom profile. Built-in platform profiles must be based on physical-device results, so the
browser never guesses a generic iOS, Android, or Windows transform. Guided calibration also requires
visible pointer or touch feedback from the remote system.

WebHID extensions register an explicit device profile containing permission filters, device
matching, report decoding, and neutral release events. The profile manager can request a device
from a user gesture or reconnect devices whose permission was previously granted. Profiles emit
the same semantic events as native browser capture; raw reports never bypass server validation.
No generic “input adapter” menu is required or restored by this extension point.

The desktop mounts one `WebHIDInputRuntime`, while device-specific modules call
`registerWebHIDProfile(profile)` and expose their own contextual user-gesture button that calls
`requestWebHIDProfile(profile.id)`. A profile includes a stable ID, optional display label,
permission filters, a strict device matcher, a report decoder into existing semantic kinds, and
neutral release events. Registering a profile reconnects only matching devices for which the
browser already retains permission. Unregistering first removes the profile from matching, then
releases and closes every active device using it. Page hide and component teardown also release
and close active devices; page restore reconnects granted matching devices. Manual disconnects do
not immediately reconnect because automatic reconnect reacts only to profile-list changes.
Decoder exceptions remain isolated to that profile's report callback, and a release exception
cannot prevent the browser device from being closed. This keeps future device support modular
without restoring a global input-device selector or allowing arbitrary raw HID forwarding.

The baseline capture APIs remain independent:

- [Pointer Events](https://www.w3.org/TR/pointerevents/) supplies mouse, pen, and concurrent
  touchscreen pointers with stable IDs for each active gesture.
- [WebHID](https://wicg.github.io/webhid/) is an optional profile-based extension point. It is a
  WICG draft rather than a baseline dependency, so browsers without WebHID retain keyboard,
  pointer, touchscreen, and virtual-input routes.

The visible video/input region is the only pointer and touchscreen capture target. Events that
start outside it remain browser events. A gesture that starts inside remains owned until release
so remote buttons and contacts cannot become stuck.

The `#input-region` node is rendered through a portal whose host switches with the video pipeline
state (normal screen vs. no-signal screen), so the node is destroyed and recreated across host
switches. Pointer, touchscreen, and calibration listeners must bind through the shared element
registry in `web/src/lib/input-region.ts` (`subscribeInputRegion`), which always resolves to the
currently attached node; capturing the element once at mount leaves the handlers on a detached
node after a host switch and silently drops every input event.

Touchscreen capture sends changed contact state on the next 16 ms sampling tick and repeats an
unchanged active state every 250 ms. This avoids continuously flooding `hidg2` while retaining a
low-rate state heartbeat that can recover from a transient Gadget write timeout.

## Wire Protocol Version 1

WebSocket message type `3` carries one input-router frame. All multi-byte integers are little
endian.

| Offset |     Size | Field                                                  |
| ------ | -------: | ------------------------------------------------------ |
| 0      |        1 | protocol version                                       |
| 1      |        1 | source                                                 |
| 2      |        1 | semantic event kind                                    |
| 3      |        1 | flags                                                  |
| 4      |        4 | source sequence                                        |
| 8      |        4 | browser capture timestamp in milliseconds, modulo 2^32 |
| 12     | variable | kind-specific payload                                  |

Version 1 requires flags and all reserved bytes to be zero. Pointer button state is limited to
five semantic buttons. Stylus and touchscreen state use Tip and In Range bits only, and touchscreen
Contact IDs are in the semantic range `1..10`. Invalid values are decode rejects and never reach a
Gadget writer.

Event kinds:

- Keyboard state: the normalized boot-keyboard state, 8 bytes.
- Relative pointer: buttons, signed 16-bit X/Y, signed 16-bit wheel.
- Absolute pointer: buttons, normalized 16-bit X/Y, signed 16-bit wheel.
- Stylus: state, normalized 16-bit X/Y, pressure.
- Touchscreen frame: zero to ten contacts with stable IDs, state, normalized X/Y, and pressure.
- Pointer release: one byte of route mask (`1<<0` relative pointer, `1<<1` absolute pointer,
  `1<<2` stylus, `1<<3` touchscreen) declaring that this session no longer asserts those pointer
  routes. The mask must be non-zero and contain no undefined bits. The kind is an additive version
  1 extension: it does not raise the protocol version, change existing payloads, or change HID
  report shapes, and a device without support rejects it as an unknown kind.
- Raw HID: reserved for explicitly authorized WebHID routing profiles.

Normalized absolute axes use the full unsigned 16-bit range. HID encoders scale them to the
logical range declared by the selected gadget descriptor.

## KVM-GO Router

The server pipeline has explicit stages:

1. Decode and version-check the transport frame.
2. Validate source, event kind, payload size, contact count, and state.
3. Apply route policy and device compatibility profile.
4. Encode the selected USB HID report.
5. Apply the route's backpressure/recovery policy and dispatch through the Gadget registry.
6. Publish a trace record without blocking HID output.

Legacy keyboard and mouse WebSocket messages remain temporarily supported during migration. A
compatibility adapter converts keyboard, relative pointer, absolute mouse, stylus, and
touchscreen reports into version 1 events before routing. New browser capture code must use the
input-router protocol directly.

Server-side producers use the same router through virtual source sessions: paste and AI actions
use virtual keyboard or virtual pointer, the mouse jiggler uses virtual pointer, and legacy HID
queues use virtual keyboard, virtual pointer, or virtual touchscreen. These producers therefore
share validation, ownership, release, recovery, and trace behavior with browser input. Gadget
open/close, descriptor setup, and full-device reset remain direct device-management operations;
they are not input events and do not enter this route.

Each control WebSocket receives a unique input session ID. The router keeps the latest report
from every session for each logical route. When a session disconnects, its state is removed; the
route is restored to the latest still-connected session, or is neutralized from the released
session's own last report when no session remains. Neutralizing from the last report matters for
the absolute-pointer and stylus routes: they have no position-free report form, and a fixed
center-coordinate neutral report would move the host pointer to the middle of the frame whenever a
one-shot producer such as an AI action finished.

The absolute-pointer and stylus routes also keep the released session's own coordinates when other
sessions still hold state, because restoring another session's last report would move the host
pointer back to stale coordinates and silently undo a one-shot move. Only the buttons are
neutralized; the remaining session keeps ownership of the route, and the next report from that
session re-establishes its position. Other routes still restore the previous session, so a
disconnected browser cannot release another browser's keys, buttons, contacts, or pointer state.

Neutral absolute-pointer and stylus reports reuse the disconnecting session's last observed
position with buttons and wheel cleared. When the session never reported a position, the neutral
HID2 write is omitted entirely: the router must never fabricate a coordinate the host has not
observed (earlier versions fell back to the descriptor center, which visibly teleported the host
pointer). The same rule applies to device-management releases (HID mode changes, Reset HID,
pointer-mode switches): with no known absolute position, the absolute release report is skipped.

A HID2 absolute release is also skipped when the last successfully written absolute report
carried no state bits. Every absolute report also carries X/Y, so re-sending a stateless report
during a device rebuild (virtual network or virtual media rebuild, HID mode change, Reset HID,
routing quiesce) or a session release would move the host cursor back to a position the session no
longer owns. Only a report fully accepted by the HID write boundary becomes release evidence, and
a release that still has to clear a mouse button or a stylus tip/in-range state keeps writing at
that last observed position.

The browser also sends the pointer-release kind when it stops owning the pointer: when preview
mode turns on, when the input region unmounts, and when the pointer leaves the input region with
no button held. The frame is the last pointer frame of that ownership episode. The server applies
it per route as a release scoped to that session: a route still held by another session is
restored to that session (with the touchscreen owner-change barrier when the route declares one),
otherwise the route's neutral report is written: for the absolute and stylus routes only when the
released report still carried state bits, so a stateless absolute or stylus report drops the route
without a device write, while the relative and touchscreen routes always keep their zero or flush
report. The
session's state and ownership for the listed routes are removed before that cleanup write, matching
session-disconnect behavior: a failed cleanup write is reported as a dispatch failure instead of
preserving a stale owner. A route rejected by the active pointer-mode gate has its state and
ownership dropped without any device write, because the mode switch already released it.
Pointer-release frames are published with the `client-release` trace stage, while each route write
keeps its own `release`, `restore`, or `restore-clear` stage.

Touchscreen disconnect and Reset HID use a ten-slot inactive flush report with Contact IDs
`1..10`. A count-zero report alone cannot identify stale Linux HID multi-touch slots after an
abrupt disconnect. Explicitly naming every slot also clears evdev `BTN_TOUCH`, rather than only
leaving Android's higher-level pointer state neutral.

The touchscreen route also declares the same flush as an owner-change barrier. Before a frame
from another session becomes authoritative, or before an older session is restored after its
successor disconnects, the router clears all host slots and publishes an `owner-switch-clear` or
`restore-clear` trace. Other routes do not pay this extra report cost because their HID reports
replace the complete keyboard, button, or axis state directly.

## Gadget Registry

Initial logical outputs:

| Route            | Endpoint       | Purpose                         |
| ---------------- | -------------- | ------------------------------- |
| keyboard         | hidg0          | boot keyboard                   |
| relative-pointer | hidg1          | relative mouse                  |
| absolute-pointer | hidg2 report 1 | generic desktop absolute mouse  |
| stylus           | hidg2 report 2 | digitizer stylus (Android-compatible) |
| touchscreen      | hidg2 report 3 | up to ten touch contacts        |

New devices must add a semantic decoder, registered route, encoder, gadget descriptor, endpoint
validation, recovery policy, and compatibility tests without changing capture adapters for
unrelated devices.
WebHID raw reports are rejected until an explicit device profile registers their meaning and
destination; arbitrary browser reports are never forwarded directly to a gadget endpoint.

Every route declares how a transient Gadget timeout is recovered. Keyboard state can be repeated
safely. Absolute mouse, stylus, and touchscreen reports are repeated only for button,
tip, or contact-topology transitions. Relative-pointer button recovery clears X/Y/wheel so motion
cannot be duplicated. Replaceable coordinate-only frames are recorded as `backpressure-drop` and
left for the next frame or touchscreen heartbeat to supersede; they are not misclassified as
protocol rejects.

## Debugging

Each routed frame produces one structured trace containing:

- source sequence and capture timestamp
- source and semantic event kind
- captured wire payload
- normalized meaning and points
- structured semantic state for keys, buttons, relative motion, wheel, and stylus
- selected route and gadget path
- encoded HID report
- dispatch result or error

The production manager does not retain trace history. The standalone input debug dashboard opens
an authenticated newline-delimited JSON stream and stores received records only in that browser
page's memory. Closing or refreshing the page releases the subscription and discards its local
records; when no page is subscribed, new trace records are consumed without being cached. Debug
delivery is best effort and must never delay or fail HID dispatch.

Authenticated management endpoints expose the current registry and trace ring:

- `GET /api/hid/input-router/status`
- `GET /api/hid/input-router/trace-stream`

The status response is the `input_router_mgmt` view. It includes aggregate source counters,
active WebSocket sessions, each session's source mix and selected routes, route ownership and
successful and failed dispatch counters, separately counted protocol decode rejects and HID
dispatch failures, transient write retries, replaceable backpressure drops, plus Debug APP
connection/delivery health. Session data is removed when its WebSocket closes; aggregate counters
remain available for diagnostics until the server restarts.

## Compatibility Validation

The currently tested client-browser matrix is:

| Client platform                         | Tested input modes                                            | Result                                                                               |
| --------------------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| Windows                                 | Follow-pointer mouse, Multi-touch screen                      | Supported in testing                                                                 |
| macOS                                   | Follow-pointer mouse, Android-compatible follow-pointer mouse | Supported in testing                                                                 |
| Android                                 | Android-compatible follow-pointer mouse, Multi-touch screen   | Supported in testing                                                                 |
| iPhone/iPad with Assistive mode enabled | Follow-pointer mouse                                          | Usable, but tapping in the browser can cause unexpected vertical scrolling           |
| iPhone without Assistive mode           | None                                                          | Not supported in testing                                                             |
| iPad without Assistive mode             | Follow-pointer mouse, Relative mouse                          | Follow-pointer mode supports movement only; Relative mouse supports clicks and wheel |

This matrix describes the browser/client side. It does not imply that the remote operating system
accepts every HID collection or that iOS/iPadOS exposes the same browser capture APIs in every mode.
The iPhone/iPad Assistive mode behavior and the iPad split between follow-pointer and relative mouse
are current device observations, not HID descriptor guarantees.

Automated tests cover protocol decoding, validation, encoding, routing, and release frames; they
do not replace testing against a real remote host.

Remote hosts tested so far:

- Android 16 over physical USB: keyboard, relative mouse, stylus, and ten-contact multi-touch.
  Android enumerates `hidg0` as a keyboard, `hidg1` as a mouse, and `hidg2` as a stylus plus a
  direct multi-touch screen.
- Linux: descriptors parsed by the kernel HID stack through UHID (see the kernel probe below),
  without physical USB transport.

On the browser side, Firefox on Linux was also checked for keyboard, pointer, and two-contact
touch capture; this does not cover Firefox on other platforms.

Windows applies
[additional mandatory usage and coordinate-metadata rules](https://learn.microsoft.com/en-us/windows-hardware/design/component-guidelines/touchscreen-required-hid-top-level-collections)
to HID touchscreen collections. The current matrix records a successful Windows multi-touch test,
but descriptor-level compatibility and application-specific touch behavior remain separate concerns.

## Known Limitations

### Android mirrored non-resizable apps

Some Android OEM builds (observed on ColorOS) move a non-resizable full-screen app from the
external display to the internal display and mirror it back. The KVM-GO touchscreen stays bound
to the external display, and the OS forwards its coordinates to the internal app without
converting the viewport or orientation, so part of the internal screen becomes unreachable.

This is host-side Android input routing, not a KVM-GO bug. A USB HID report cannot select an
Android display, and browser-side calibration cannot extend Android's post-HID axis limits.
Display-to-input association is configured on the Android host (input port association or an
input device configuration file). The single-contact stylus route remains a fallback.

References: [AOSP input routing](https://source.android.com/docs/core/display/multi_display/input-routing)
and [AOSP InputReader viewport selection](https://android.googlesource.com/platform/frameworks/native/+/65a1504/services/inputflinger/InputReader.cpp).

## Repeatable Device Probe

`web/scripts/input-router-device-probe.mjs` records pre-test, active-session, and post-test
management state around two marked WebSocket sessions. It first reads the device's authoritative
pointer profile, then sends neutral keyboard and relative-pointer reports plus only the active HID2
kind. It stresses that active HID2 route, verifies decode/dispatch deltas, and proves that its own
sessions and route owners disappear after disconnect. The output uses the
`nanokvm-input-router-probe/v2` JSON schema and can be saved as evidence for each target:

```sh
cd web
NANOKVM_URL=https://192.168.1.100 \
NANOKVM_TOKEN="$TOKEN" \
npm run probe:input-router -- \
  --insecure \
  --target-os Android \
  --target-version 16 \
  --output ../build/input-router-android.json
```

The default probe is neutral-only: it proves transport, profile-aware routing, Gadget writes,
session cleanup, and target availability, but not complete visible input behavior. Use
`--hid2-frames N` to set the active-profile stress count; the old `--touch-frames` spelling remains
an alias. Confirming visible input on a target OS still requires an attended active-input pass.

The probe's own stress frames deliberately target the center of the input region, so an active
probe pass can visibly move the host pointer to center. This is probe-authored traffic, not a
server-side release: server neutral/release reports reuse the last observed position or are
omitted when none exists (see above).

For an attended active multi-touch pass, first select the device `touch-screen` profile and add
`--active-touch`. This opt-in mode appends one
two-contact down frame, one lift frame, and one empty frame; it must be paired with a target-side
observer such as Android `getevent` and can affect the target UI. It is never enabled by default.

For a keyboard transport pass, add `--active-keyboard`. It sends only a left-Shift modifier
press/release (no printable key) and can be observed on the target keyboard event node.

For a relative-pointer transport pass, add `--active-pointer`. It sends two no-button movements
of `+1` and `-1` X units, returning the pointer to its prior position without clicking.

For a browser-layer multi-touch pass, use Chromium's CDP probe. It sets the selected mouse mode,
dispatches two touch contacts through a real browser page, and verifies that the browser source
counter and `hidg2/touchscreen` route both advance without rejects:

```sh
cd web
NANOKVM_URL=https://192.168.1.100 \
NANOKVM_TOKEN="$TOKEN" \
npm run probe:input-router-browser -- \
  --insecure \
  --browser /usr/bin/chromium \
  --port 9334 \
  --contacts 2 \
  --output ../build/input-router-browser-android.json
```

The same probe accepts `--input keyboard` for a modifier-only browser keyboard event, or
`--input mouse` with `--mode absolute`, `--mode digitizer-stylus`, or `--mode relative` to
exercise the corresponding browser pointer path and Gadget route.

The Firefox keyboard and pointer probe drives the same routes through WebDriver BiDi:

```sh
cd web
NANOKVM_URL=https://192.168.1.100 \
NANOKVM_TOKEN="$TOKEN" \
npm run probe:input-router-firefox -- --insecure
```

It defaults to both keyboard and relative-pointer input. `--input keyboard|mouse|touch|all` and
`--mode relative|absolute|digitizer-stylus|touch-screen` select a narrower source/route
combination. Use `--input touch --mode touch-screen` for the two-contact Pointer Events check.

The canonical internal name for the report-ID-2 digitizer collection is
`digitizer-stylus`. The legacy `absolute-android` probe argument is accepted as
an alias and normalized before the request is run.

On a Linux host with writable `/dev/uhid` and the Python `evdev` package, the kernel probe checks
that the keyboard, relative-mouse, and touchscreen descriptors match `usbdev.sh` and that the host
HID stack parses them, without physical USB transport. Run it from the repository root:

```sh
python3 support/tools/input-router-linux-uhid-probe.py \
  --output build/input-router-linux-uhid.json
```
