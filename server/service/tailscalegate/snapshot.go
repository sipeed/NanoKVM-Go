// Package tailscalegate implements the server side of the kvm_ui Tailscale
// initialization gate. It answers the single Unix-socket path
// LoginCodePath, whose query parameters select the snapshot, action, or
// login-code behavior.
//
// Heartbeat snapshots always carry five fields, a missing field falls back to
// its documented default, and an entire refresh failure is reported as an
// error instead of a synthesized snapshot. The Driver interface is the only
// boundary to the real Tailscale components, so this package stays host-testable.
package tailscalegate

import "strings"

// Status is the tunnel health reported in a heartbeat snapshot.
type Status string

const (
	// StatusOnline means the Tailscale tunnel is currently connected.
	StatusOnline Status = "online"
	// StatusOffline is the default when the tunnel is not connected or the
	// driver cannot determine the status.
	StatusOffline Status = "offline"
)

const (
	snapshotNameMaxBytes = 64
	snapshotIPMaxBytes   = 64
)

// Report is the raw heartbeat data a driver returns.
//
// A nil pointer means the driver cannot currently determine that field. The
// interactive layer resolves each unknown field independently to its
// documented default rather than treating the whole report as failed; only a
// driver error fails the whole snapshot.
type Report struct {
	Status *string
	Start  *bool
	Match  *bool
	Name   *string
	IP     *string
}

// Snapshot is a resolved heartbeat payload. Every field is always present: no
// field is a pointer and none is optional, so a successful refresh can never
// produce a partial object on the wire.
type Snapshot struct {
	Status Status
	Start  bool
	Match  bool
	Name   string
	IP     string
}

// Resolve converts a driver report into the wire snapshot.
//
// Each field is resolved independently so a single missing or malformed value
// cannot invalidate the rest of the snapshot:
//
//   - status is online only when the trimmed driver value is exactly "online";
//     anything else, including a missing value, becomes offline;
//   - start and match default to false;
//   - name and ip are filtered to printable ASCII and then truncated to 64
//     bytes; both are dropped unless the tunnel is online.
//
// The name/ip convergence mirrors the reference mock server, which emits the
// peer identity only while the tunnel reports online. The raw driver report
// keeps its values even while offline, because the pairing directive is built
// from the report rather than this resolved snapshot.
func Resolve(report Report) Snapshot {
	snap := Snapshot{Status: StatusOffline}
	if report.Status != nil && strings.TrimSpace(*report.Status) == string(StatusOnline) {
		snap.Status = StatusOnline
	}
	if report.Start != nil {
		snap.Start = *report.Start
	}
	if report.Match != nil {
		snap.Match = *report.Match
	}
	if report.Name != nil {
		snap.Name = printableASCII(*report.Name, snapshotNameMaxBytes)
	}
	if report.IP != nil {
		snap.IP = printableASCII(*report.IP, snapshotIPMaxBytes)
	}
	if snap.Status != StatusOnline {
		snap.Name = ""
		snap.IP = ""
	}
	return snap
}

// GateState is the subset of a snapshot retained across a failed refresh.
//
// Only the two gate booleans are kept: status, name, and ip are connectivity
// facts that must not be served from an outdated snapshot.
type GateState struct {
	Start bool
	Match bool
}

// Gate projects the retained gate fields out of a resolved snapshot.
func (s Snapshot) Gate() GateState {
	return GateState{Start: s.Start, Match: s.Match}
}

// Confirm replaces the retained gate state with the snapshot from a successful
// refresh. It is a whole-value replacement, not a merge.
//
// A failed refresh must never call Confirm. Keeping the previous value is the
// server-side counterpart of the client sticky rule: one bad poll does not
// push the page back to Start, and the next successful snapshot can restore
// the live gate.
func (g GateState) Confirm(next Snapshot) GateState {
	return GateState{Start: next.Start, Match: next.Match}
}

// printableASCII drops every byte outside 0x20..0x7E and, when limit is
// positive, keeps at most limit bytes. It is the single convergence helper for
// values that leave this package, so no non-printable byte can reach the wire.
func printableASCII(value string, limit int) string {
	if len(value) == 0 {
		return ""
	}
	out := make([]byte, 0, len(value))
	for i := 0; i < len(value); i++ {
		b := value[i]
		if b < 0x20 || b > 0x7E {
			continue
		}
		out = append(out, b)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return string(out)
}
