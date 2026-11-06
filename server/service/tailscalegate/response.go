package tailscalegate

import (
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// maxResponseBytes is the client-side response ceiling. The LCD client treats
// a body larger than 64 KiB as a failed request, so a payload above this size
// is never written as-is.
const maxResponseBytes = 64 << 10

// Fixed, safe error texts. Every one is printable ASCII and names no path,
// host, driver detail, or system state, so an error body can never leak
// internal information.
const (
	// errorNotFound is the body for an unknown path or a region that does not
	// enable Tailscale behavior (404).
	errorNotFound = "not found"
	// errorMethodNotSupported is the body for a non-GET request (400).
	errorMethodNotSupported = "method not supported"
	// errorStateUnavailable is the body for a failed snapshot refresh (503).
	errorStateUnavailable = "tailscale state unavailable"
	// errorLoginCodeUnavailable is the body for a failed code fetch (503).
	errorLoginCodeUnavailable = "tailscale login code unavailable"
	// errorPeerIdentityUnavailable is the body when a valid pairing directive
	// cannot be built (503).
	errorPeerIdentityUnavailable = "tailscale peer identity unavailable"
	// errorResponseTooLarge is the body for a payload above maxResponseBytes
	// (503).
	errorResponseTooLarge = "response too large"
	// errorInternal is the body for an encoding failure (500).
	errorInternal = "internal error"
)

// snapshotResponse is the heartbeat wire payload. All five fields are always
// serialized: no field is optional, so a successful refresh never produces a
// partial object. Values must already be resolved by Resolve; this type only
// controls the wire shape.
type snapshotResponse struct {
	Status Status `json:"status"`
	Start  bool   `json:"start"`
	Match  bool   `json:"match"`
	Name   string `json:"name"`
	IP     string `json:"ip"`
}

// newSnapshotResponse projects a resolved snapshot onto the wire payload.
func newSnapshotResponse(s Snapshot) snapshotResponse {
	return snapshotResponse{
		Status: s.Status,
		Start:  s.Start,
		Match:  s.Match,
		Name:   s.Name,
		IP:     s.IP,
	}
}

// codeResponse is the login-code wire payload.
type codeResponse struct {
	Code string `json:"code"`
}

// errorResponse is the uniform error wire payload.
type errorResponse struct {
	Error string `json:"error"`
}

// writeJSON serializes payload and writes it with an explicit Content-Length.
//
// The body is marshalled to memory before any header is written, so the exact
// encoded length is known and the status can still be changed when encoding
// fails or the body exceeds maxResponseBytes. The over-limit path writes a
// small fixed body directly and does not re-check its size, which keeps the
// guard from recursing on itself.
//
// No Connection header is set by hand: HTTP/1.1 keep-alive is the default and
// the contract's delimiting requirement is Content-Length, not an explicit
// keep-alive token. Connection reuse is verified by a socket-level test in the
// commit that mounts this handler.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		// The payload is built from validated values, so this is a programming
		// error; log only the operation, never the payload values.
		log.Warn("failed to encode tailscale gate response")
		writeFixedJSON(w, http.StatusInternalServerError, errorInternal)
		return
	}
	if len(body) > maxResponseBytes {
		writeFixedJSON(w, http.StatusServiceUnavailable, errorResponseTooLarge)
		return
	}
	writeBody(w, status, body)
}

// writeError writes a uniform {"error": text} payload. It goes through the
// normal size guard, so an unexpectedly long text cannot bypass the ceiling.
func writeError(w http.ResponseWriter, status int, text string) {
	writeJSON(w, status, errorResponse{Error: text})
}

// writeFixedJSON writes a short, already-safe error payload without the
// maxResponseBytes guard. It is only called with the fixed texts above.
func writeFixedJSON(w http.ResponseWriter, status int, text string) {
	body, err := json.Marshal(errorResponse{Error: text})
	if err != nil {
		// errorResponse holds a single string, so this cannot happen; fall back
		// to a hardcoded safe body rather than panicking mid-response.
		body = []byte(`{"error":"internal error"}`)
	}
	writeBody(w, status, body)
}

// writeBody writes one complete response with an exact Content-Length.
func writeBody(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		log.Warnf("failed to write tailscale gate response: %v", err)
	}
}
