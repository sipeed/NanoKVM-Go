package tailscalegate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// mustMarshal encodes payload for the size-boundary test so the expected
// lengths come from the real encoder instead of hardcoded constants. That
// keeps the test stable if a field is ever added to a payload struct.
func mustMarshal(t *testing.T, payload any) []byte {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal(%T) returned error: %v", payload, err)
	}
	return body
}

// assertExactKeys asserts that got contains exactly the wanted keys: one
// missing key and one extra key both fail. It is how the snapshot test proves
// that no field carries omitempty.
func assertExactKeys(t *testing.T, got map[string]any, want ...string) {
	t.Helper()
	for _, key := range want {
		if _, ok := got[key]; !ok {
			t.Fatalf("response is missing field %q: %v", key, got)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("response has %d fields (%v), want exactly %v", len(got), got, want)
	}
}

func TestSnapshotResponseJSONShape(t *testing.T) {
	cases := []struct {
		name string
		in   Snapshot
		want map[string]any
	}{
		{
			name: "online and ready",
			in: Snapshot{
				Status: StatusOnline,
				Start:  true,
				Match:  true,
				Name:   "MockKVM",
				IP:     "100.64.0.1",
			},
			want: map[string]any{
				"status": "online",
				"start":  true,
				"match":  true,
				"name":   "MockKVM",
				"ip":     "100.64.0.1",
			},
		},
		{
			// Resolve(Report{}) is the documented default snapshot: false
			// booleans and empty strings. Every one of those keys must still be
			// present, which is what detects an accidental omitempty.
			name: "all defaults",
			in:   Resolve(Report{}),
			want: map[string]any{
				"status": "offline",
				"start":  false,
				"match":  false,
				"name":   "",
				"ip":     "",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeJSON(rec, http.StatusOK, newSnapshotResponse(tc.in))

			var got map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal snapshot response: %v", err)
			}
			assertExactKeys(t, got, "status", "start", "match", "name", "ip")
			for key, want := range tc.want {
				if got[key] != want {
					t.Fatalf("field %q = %#v, want %#v", key, got[key], want)
				}
			}
		})
	}
}

func TestCodeResponseJSON(t *testing.T) {
	cases := []struct {
		name string
		code string
		want string
	}{
		{name: "plain code", code: "123456", want: `{"code":"123456"}`},
		{name: "off compatibility", code: OffCode(), want: `{"code":"%off%"}`},
		{
			name: "online directive",
			code: "%online%MockKVM%100.64.0.1%",
			want: `{"code":"%online%MockKVM%100.64.0.1%"}`,
		},
		{name: "empty code", code: "", want: `{"code":""}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeJSON(rec, http.StatusOK, codeResponse{Code: tc.code})
			if got := rec.Body.String(); got != tc.want {
				t.Fatalf("body = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestErrorResponseJSON(t *testing.T) {
	cases := []struct {
		name   string
		status int
		text   string
	}{
		{name: "not found", status: http.StatusNotFound, text: errorNotFound},
		{name: "state unavailable", status: http.StatusServiceUnavailable, text: errorStateUnavailable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeError(rec, tc.status, tc.text)

			var got errorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal error response: %v", err)
			}
			if got.Error != tc.text {
				t.Fatalf("error = %q, want %q", got.Error, tc.text)
			}
			var keys map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &keys); err != nil {
				t.Fatalf("unmarshal error response keys: %v", err)
			}
			assertExactKeys(t, keys, "error")
		})
	}
}

// TestErrorTextConstantsAreSafePayloads pins the invariant that every fixed
// error body is a short printable-ASCII phrase with no path separator, so an
// error can never leak filesystem or system detail.
func TestErrorTextConstantsAreSafePayloads(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{name: "not found", text: errorNotFound},
		{name: "method not supported", text: errorMethodNotSupported},
		{name: "state unavailable", text: errorStateUnavailable},
		{name: "login code unavailable", text: errorLoginCodeUnavailable},
		{name: "peer identity unavailable", text: errorPeerIdentityUnavailable},
		{name: "response too large", text: errorResponseTooLarge},
		{name: "internal", text: errorInternal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.text == "" {
				t.Fatalf("error text must not be empty")
			}
			if printableASCII(tc.text, 0) != tc.text {
				t.Fatalf("error text %q is not printable ASCII", tc.text)
			}
			if strings.ContainsAny(tc.text, `/\`) {
				t.Fatalf("error text %q contains a path separator", tc.text)
			}
		})
	}
}

func TestWriteJSONStatusAndHeaders(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		payload any
	}{
		{
			name:    "ok snapshot",
			status:  http.StatusOK,
			payload: newSnapshotResponse(Resolve(Report{})),
		},
		{
			name:    "bad request error",
			status:  http.StatusBadRequest,
			payload: errorResponse{Error: errorMethodNotSupported},
		},
		{
			name:    "service unavailable error",
			status:  http.StatusServiceUnavailable,
			payload: errorResponse{Error: errorStateUnavailable},
		},
		{
			name:    "action response",
			status:  http.StatusOK,
			payload: FailureAction(nil),
		},
		{
			name:   "multi byte code keeps a byte Content-Length",
			status: http.StatusOK,
			payload: codeResponse{
				Code: "\u9a8c\u8bc1\u7801\"\\\\\t",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeJSON(rec, tc.status, tc.payload)

			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d", rec.Code, tc.status)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Fatalf("Content-Type = %q, want an application/json prefix", ct)
			}
			if cl := rec.Header().Get("Content-Length"); cl != strconv.Itoa(rec.Body.Len()) {
				t.Fatalf("Content-Length = %q, want %d", cl, rec.Body.Len())
			}
		})
	}
}

// TestWriteJSONSizeBoundary drives the exact 64 KiB ceiling from real encoded
// lengths. A codeResponse with a long printable string has no JSON escaping
// overhead, so its marshalled size is exactly the string length plus the fixed
// object overhead.
func TestWriteJSONSizeBoundary(t *testing.T) {
	overhead := len(mustMarshal(t, codeResponse{}))
	if overhead >= maxResponseBytes {
		t.Fatalf("codeResponse overhead %d >= maxResponseBytes %d", overhead, maxResponseBytes)
	}

	exact := codeResponse{Code: strings.Repeat("A", maxResponseBytes-overhead)}
	exactBody := mustMarshal(t, exact)
	if len(exactBody) != maxResponseBytes {
		t.Fatalf("exact payload encoded to %d bytes, want %d", len(exactBody), maxResponseBytes)
	}

	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, exact)
	if rec.Code != http.StatusOK {
		t.Fatalf("exact-size status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.Len() != maxResponseBytes {
		t.Fatalf("exact-size body = %d bytes, want %d", rec.Body.Len(), maxResponseBytes)
	}
	if cl := rec.Header().Get("Content-Length"); cl != strconv.Itoa(maxResponseBytes) {
		t.Fatalf("exact-size Content-Length = %q, want %d", cl, maxResponseBytes)
	}

	over := codeResponse{Code: strings.Repeat("A", maxResponseBytes-overhead+1)}
	overBody := mustMarshal(t, over)
	if len(overBody) != maxResponseBytes+1 {
		t.Fatalf("over payload encoded to %d bytes, want %d", len(overBody), maxResponseBytes+1)
	}

	rec = httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, over)
	wantBody := `{"error":"response too large"}`
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("over-size status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got := rec.Body.String(); got != wantBody {
		t.Fatalf("over-size body = %q, want %q", got, wantBody)
	}
	if cl := rec.Header().Get("Content-Length"); cl != strconv.Itoa(len(wantBody)) {
		t.Fatalf("over-size Content-Length = %q, want %d", cl, len(wantBody))
	}
}

func TestWriteJSONMarshalFailure(t *testing.T) {
	rec := httptest.NewRecorder()
	// A channel value cannot be represented in JSON, so encoding fails.
	writeJSON(rec, http.StatusOK, struct {
		Unsupported chan int `json:"unsupported"`
	}{Unsupported: make(chan int)})

	wantBody := `{"error":"internal error"}`
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if got := rec.Body.String(); got != wantBody {
		t.Fatalf("body = %q, want %q", got, wantBody)
	}
	if cl := rec.Header().Get("Content-Length"); cl != strconv.Itoa(len(wantBody)) {
		t.Fatalf("Content-Length = %q, want %d", cl, len(wantBody))
	}
}

// TestWriteErrorAppliesSizeGuard proves writeError is not a bypass: a long
// caller-supplied text still hits the same ceiling and is replaced by the
// fixed small body.
func TestWriteErrorAppliesSizeGuard(t *testing.T) {
	rec := httptest.NewRecorder()
	long := strings.Repeat("x", maxResponseBytes+1)
	writeError(rec, http.StatusBadRequest, long)

	wantBody := `{"error":"response too large"}`
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got := rec.Body.String(); got != wantBody {
		t.Fatalf("body = %q, want %q", got, wantBody)
	}
	if strings.Contains(rec.Body.String(), long) {
		t.Fatalf("long caller text leaked into the response body")
	}
	if cl := rec.Header().Get("Content-Length"); cl != strconv.Itoa(len(wantBody)) {
		t.Fatalf("Content-Length = %q, want %d", cl, len(wantBody))
	}
}
