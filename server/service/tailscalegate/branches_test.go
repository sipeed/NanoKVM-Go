// This file exercises Tailscale gate interaction branches through a scripted
// Driver. Each case sets a report and asserts the HTTP status, complete body,
// action message, code shape, and driver call counts.
//
// Entire-refresh failures are expressed as a Snapshot error, matching the
// boundary reached by a malformed or dropped component response.
//
// Host verification (from the repository root):
//
//	cd server && CGO_ENABLED=0 go test -count=1 -run TestDocumentedBranches ./service/tailscalegate
//	cd server && CGO_ENABLED=0 go test -count=1 ./...

package tailscalegate

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"NanoKVM-Server/proto"
)

// readyOnlineReport is the report of the reference ready-online gate
// (start=true, match=true, tunnel online with a peer identity).
func readyOnlineReport() Report {
	return Report{
		Status: ptr("online"),
		Start:  ptr(true),
		Match:  ptr(true),
		Name:   ptr("MockKVM"),
		IP:     ptr("100.64.0.1"),
	}
}

// readyOfflineReport is the report of the reference ready-offline gate: both
// gates are confirmed and the peer is known, but the tunnel is down.
func readyOfflineReport() Report {
	return Report{
		Status: ptr("offline"),
		Start:  ptr(true),
		Match:  ptr(true),
		Name:   ptr("MockKVM"),
		IP:     ptr("100.64.0.1"),
	}
}

// readyOnlineSnapshot is the resolved heartbeat for readyOnlineReport.
func readyOnlineSnapshot() snapshotResponse {
	return snapshotResponse{
		Status: StatusOnline,
		Start:  true,
		Match:  true,
		Name:   "MockKVM",
		IP:     "100.64.0.1",
	}
}

// readyOfflineSnapshot is the resolved heartbeat for readyOfflineReport: the
// offline tunnel hides the peer identity even though the driver still knows it.
func readyOfflineSnapshot() snapshotResponse {
	return snapshotResponse{Status: StatusOffline, Start: true, Match: true}
}

// wantSnapshot asserts a successful heartbeat: HTTP 200, a bounded response,
// and every one of the five snapshot fields resolved exactly as expected.
func wantSnapshot(t *testing.T, recorder *httptest.ResponseRecorder, want snapshotResponse) {
	t.Helper()
	assertStatus(t, recorder, http.StatusOK)
	assertBoundedResponse(t, recorder)
	if got := decodeJSON[snapshotResponse](t, recorder); got != want {
		t.Fatalf("snapshot = %+v, want %+v (body %s)", got, want, recorder.Body.String())
	}
}

// wantAction asserts a successful action response; action endpoints answer
// HTTP 200 even when the action itself failed.
func wantAction(t *testing.T, recorder *httptest.ResponseRecorder, want ActionResponse) {
	t.Helper()
	assertStatus(t, recorder, http.StatusOK)
	assertBoundedResponse(t, recorder)
	if got := decodeJSON[ActionResponse](t, recorder); got != want {
		t.Fatalf("action = %+v, want %+v (body %s)", got, want, recorder.Body.String())
	}
}

// wantCode asserts a code response and its exact single-field body.
func wantCode(t *testing.T, recorder *httptest.ResponseRecorder, want string) {
	t.Helper()
	assertStatus(t, recorder, http.StatusOK)
	assertBoundedResponse(t, recorder)
	wantBody, err := json.Marshal(codeResponse{Code: want})
	if err != nil {
		t.Fatalf("marshal expected code: %v", err)
	}
	if got := recorder.Body.String(); got != string(wantBody) {
		t.Fatalf("body = %s, want %s", got, wantBody)
	}
}

// wantError asserts a failure response and its exact single-field body, so a
// driver detail or path cannot hide in the payload.
func wantError(t *testing.T, recorder *httptest.ResponseRecorder, status int, text string) {
	t.Helper()
	assertStatus(t, recorder, status)
	assertBoundedResponse(t, recorder)
	wantBody, err := json.Marshal(errorResponse{Error: text})
	if err != nil {
		t.Fatalf("marshal expected error: %v", err)
	}
	if got := recorder.Body.String(); got != string(wantBody) {
		t.Fatalf("body = %s, want %s", got, wantBody)
	}
}

// wantCounts asserts the exact per-method driver call counts.
func wantCounts(t *testing.T, driver *fakeDriver, want fakeCallCounts) {
	t.Helper()
	if got := driver.callCounts(); got != want {
		t.Fatalf("driver calls = %+v, want %+v", got, want)
	}
}

func TestDocumentedBranches(t *testing.T) {
	cases := []struct {
		key     string
		name    string
		comment string
		// actionBudget overrides the injected action budget when non-zero, so
		// the timeout branch runs on a millisecond clock without sleeping.
		actionBudget time.Duration
		run          func(t *testing.T, driver *fakeDriver, handler *Handler)
	}{
		{
			key:  "1",
			name: "not initialized",
			comment: "Branch 1 (README 8.1): start=false renders Start, so a code request is the %off% " +
				"compatibility value and an exit is NOT_READY without reaching the driver.",
			run: func(t *testing.T, driver *fakeDriver, handler *Handler) {
				driver.setReport(Report{
					Status: ptr("offline"),
					Start:  ptr(false),
					Match:  ptr(false),
					Name:   ptr(""),
					IP:     ptr(""),
				})

				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					snapshotResponse{Status: StatusOffline})
				wantCode(t, serve(t, handler, http.MethodGet, LoginCodePath), OffCode())
				wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true"),
					ActionResponse{Message: MessageNotReady})
				wantCounts(t, driver, fakeCallCounts{snapshot: 2})
			},
		},
		{
			key:  "2",
			name: "initialized, not paired",
			comment: "Branch 2 (README 8.2): start=true, match=false keeps the buttons hidden while the " +
				"plain verification code is fetched from the driver on every code request. The driver " +
				"value is filtered to printable ASCII.",
			run: func(t *testing.T, driver *fakeDriver, handler *Handler) {
				driver.setReport(Report{
					Status: ptr("offline"),
					Start:  ptr(true),
					Match:  ptr(false),
					Name:   ptr("MockKVM"),
					IP:     ptr("100.64.0.1"),
				})
				driver.setLoginCode("12\x003456")

				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					snapshotResponse{Status: StatusOffline, Start: true})
				wantCode(t, serve(t, handler, http.MethodGet, LoginCodePath), "123456")
				wantCode(t, serve(t, handler, http.MethodGet, LoginCodePath), "123456")
				wantCounts(t, driver, fakeCallCounts{snapshot: 3, loginCode: 2})
			},
		},
		{
			key:  "3",
			name: "paired, online, exit succeeds",
			comment: "Branch 3 (README 8.3): the snapshot carries the peer name and ip, a code request " +
				"answers the %online% pairing directive without asking the driver for a code, and a " +
				"long-press Exit reaches Deinitialize once.",
			run: func(t *testing.T, driver *fakeDriver, handler *Handler) {
				driver.setReport(readyOnlineReport())

				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					readyOnlineSnapshot())
				wantCode(t, serve(t, handler, http.MethodGet, LoginCodePath), "%online%MockKVM%100.64.0.1%")
				wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true"), SuccessAction())
				wantCounts(t, driver, fakeCallCounts{snapshot: 3, deinitialize: 1})
			},
		},
		{
			key:  "4",
			name: "paired, offline",
			comment: "Branch 4 (README 8.4): ready but offline is a legal transition state, so the " +
				"snapshot hides name and ip, while the directive is still built from the raw report.",
			run: func(t *testing.T, driver *fakeDriver, handler *Handler) {
				driver.setReport(readyOfflineReport())

				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					readyOfflineSnapshot())
				wantCode(t, serve(t, handler, http.MethodGet, LoginCodePath), "%online%MockKVM%100.64.0.1%")
				wantCounts(t, driver, fakeCallCounts{snapshot: 2})
			},
		},
		{
			key:  "5",
			name: "next ?start fails",
			comment: "Branch 5 (README 8.5): the driver message NO_DAEMON is returned and, because a " +
				"failed action performs no re-confirm, the retained gate stays un-started and a " +
				"following exit is NOT_READY.",
			run: func(t *testing.T, driver *fakeDriver, handler *Handler) {
				driver.setReport(Report{Start: ptr(false)})
				driver.setInitializeErr(testMessageError("NO_DAEMON"))

				wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?start=true"),
					ActionResponse{Status: false, Message: "NO_DAEMON"})
				wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true"),
					ActionResponse{Message: MessageNotReady})
				wantCounts(t, driver, fakeCallCounts{initialize: 1})
			},
		},
		{
			key:          "6",
			name:         "next ?start times out",
			comment:      "Branch 6 (README 8.6): an action that outlives its budget answers TIMEOUT, and the single action slot is released so the next action is not BUSY.",
			actionBudget: 25 * time.Millisecond,
			run: func(t *testing.T, driver *fakeDriver, handler *Handler) {
				driver.setReport(Report{Start: ptr(true)})
				driver.setInitializeBlock(blockDeadline)

				wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?start=true"),
					ActionResponse{Message: MessageTimeout})

				// The slot is free: a second action runs instead of answering BUSY.
				driver.setInitializeBlock(blockNone)
				wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?start=true"), SuccessAction())
				wantCounts(t, driver, fakeCallCounts{snapshot: 1, initialize: 2})
			},
		},
		{
			key:  "7",
			name: "next ?exit fails",
			comment: "Branch 7 (README 8.7): the failing exit returns NO_DAEMON and leaves the confirmed " +
				"ready gate untouched, so the next heartbeat still reports start=true, match=true.",
			run: func(t *testing.T, driver *fakeDriver, handler *Handler) {
				driver.setReport(readyOfflineReport())
				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					readyOfflineSnapshot())

				driver.setDeinitializeErr(testMessageError("NO_DAEMON"))
				wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true"),
					ActionResponse{Status: false, Message: "NO_DAEMON"})
				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					readyOfflineSnapshot())
				wantCounts(t, driver, fakeCallCounts{snapshot: 2, deinitialize: 1})
			},
		},
		{
			key:          "8",
			name:         "next ?exit times out",
			comment:      "Branch 8 (README 8.8): a deinitialize that outlives its budget answers TIMEOUT, and the single action slot is released so the next exit is not BUSY.",
			actionBudget: 25 * time.Millisecond,
			run: func(t *testing.T, driver *fakeDriver, handler *Handler) {
				driver.setReport(readyOnlineReport())
				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					readyOnlineSnapshot())

				driver.setDeinitializeBlock(blockDeadline)
				wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true"),
					ActionResponse{Message: MessageTimeout})

				// The slot is free: a second exit runs instead of answering BUSY.
				driver.setDeinitializeBlock(blockNone)
				wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true"), SuccessAction())

				// Once the component's own report converges, the heartbeat
				// returns to the uninitialized stage.
				driver.setReport(Report{Start: ptr(false), Match: ptr(false)})
				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					snapshotResponse{Status: StatusOffline})
				wantCounts(t, driver, fakeCallCounts{snapshot: 3, deinitialize: 2})
			},
		},
		{
			key:  "9",
			name: "exit succeeds but heartbeat lags",
			comment: "Branch 9 (README 8.9): the server writes no optimistic state, so a successful exit " +
				"is followed by the component's own report. While that report still says ready the " +
				"heartbeat keeps reporting start=true, match=true; once it converges the next heartbeat " +
				"reports start=false.",
			run: func(t *testing.T, driver *fakeDriver, handler *Handler) {
				driver.setReport(readyOnlineReport())
				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					readyOnlineSnapshot())

				// The exit succeeds but the component report still says ready
				// during the lag window; the follow-up re-confirm therefore keeps
				// the ready gate.
				wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true"), SuccessAction())
				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					readyOnlineSnapshot())

				// The converged report drops the gate.
				driver.setReport(Report{Start: ptr(false), Match: ptr(false)})
				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					snapshotResponse{Status: StatusOffline})
				wantCounts(t, driver, fakeCallCounts{snapshot: 4, deinitialize: 1})
			},
		},
		{
			key:  "a",
			name: "heartbeat fails entirely (sticky)",
			comment: "Branch a (README 8.10): a whole-refresh failure answers 503 and retains the ready " +
				"gate, so an exit still reaches the driver; a later successful snapshot is reflected " +
				"again.",
			run: func(t *testing.T, driver *fakeDriver, handler *Handler) {
				driver.setReport(readyOnlineReport())
				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					readyOnlineSnapshot())

				driver.setSnapshotErr(errors.New("heartbeat transport dropped"))
				wantError(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					http.StatusServiceUnavailable, errorStateUnavailable)

				// The retained ready gate still lets the exit reach the driver.
				wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true"), SuccessAction())

				// Once the component recovers, the next snapshot replaces the gate.
				driver.setSnapshotErr(nil)
				driver.setReport(Report{Start: ptr(false)})
				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					snapshotResponse{Status: StatusOffline})
				wantCounts(t, driver, fakeCallCounts{snapshot: 4, deinitialize: 1})
			},
		},
		{
			key:  "b",
			name: "exit before ready is NOT_READY",
			comment: "Branch b (README 8.11): while start=true, match=false the exit is rejected with " +
				"NOT_READY and the server keeps the client-init gate.",
			run: func(t *testing.T, driver *fakeDriver, handler *Handler) {
				driver.setReport(Report{Start: ptr(true), Match: ptr(false)})
				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					snapshotResponse{Status: StatusOffline, Start: true})

				wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true"),
					ActionResponse{Status: false, Message: MessageNotReady})
				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					snapshotResponse{Status: StatusOffline, Start: true})
				wantCounts(t, driver, fakeCallCounts{snapshot: 2})
			},
		},
		{
			key:  "c",
			name: "pairing completes",
			comment: "Branch c (README 8.12): the first code request is a plain driver code; once the " +
				"report flips to match=true the next request answers the directive, and a retained " +
				"match never falls back to a plain code even when the refresh fails.",
			run: func(t *testing.T, driver *fakeDriver, handler *Handler) {
				driver.setReport(Report{
					Start: ptr(true),
					Match: ptr(false),
					Name:  ptr("MockKVM"),
					IP:    ptr("100.64.0.1"),
				})
				driver.setLoginCode("111222")
				wantCode(t, serve(t, handler, http.MethodGet, LoginCodePath), "111222")

				driver.setReport(readyOnlineReport())
				wantCode(t, serve(t, handler, http.MethodGet, LoginCodePath), "%online%MockKVM%100.64.0.1%")

				// A retained match with a failed refresh must not degrade to the
				// compatibility/plain code path.
				driver.setSnapshotErr(errors.New("snapshot dropped"))
				wantError(t, serve(t, handler, http.MethodGet, LoginCodePath),
					http.StatusServiceUnavailable, errorStateUnavailable)
				wantCounts(t, driver, fakeCallCounts{snapshot: 3, loginCode: 1})
			},
		},
		{
			key:  "d",
			name: "heartbeat returns only part of the fields",
			comment: "Branch d (README 8.13): a report carrying only status resolves the missing fields " +
				"to their defaults, so the observation replaces the ready gate and a following exit " +
				"is NOT_READY.",
			run: func(t *testing.T, driver *fakeDriver, handler *Handler) {
				driver.setReport(readyOnlineReport())
				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					readyOnlineSnapshot())

				driver.setReport(Report{Status: ptr("online")})
				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					snapshotResponse{Status: StatusOnline})
				wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true"),
					ActionResponse{Message: MessageNotReady})
				wantCounts(t, driver, fakeCallCounts{snapshot: 2})
			},
		},
		{
			key:  "e",
			name: "heartbeat returns malformed content",
			comment: "Branch e (README 8.14): malformed content fails the whole refresh (a Snapshot " +
				"error at this boundary), answers 503, and retains the previously confirmed ready " +
				"gate, so a following exit still reaches the driver.",
			run: func(t *testing.T, driver *fakeDriver, handler *Handler) {
				driver.setReport(readyOnlineReport())
				wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					readyOnlineSnapshot())

				driver.setSnapshotErr(errors.New("malformed heartbeat body"))
				wantError(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
					http.StatusServiceUnavailable, errorStateUnavailable)
				wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true"), SuccessAction())
				wantCounts(t, driver, fakeCallCounts{snapshot: 3, deinitialize: 1})
			},
		},
	}

	for _, tc := range cases {
		t.Run("branch "+tc.key+" - "+tc.name, func(t *testing.T) {
			// The quoted documented expectation for this reference key.
			t.Log(tc.comment)

			driver := newFakeDriver()
			deps := testDeps(driver, proto.ApplicationRegionCN)
			if tc.actionBudget > 0 {
				deps.ActionBudget = tc.actionBudget
			}
			handler := NewHandler(deps)
			tc.run(t, driver, handler)
		})
	}
}

// TestDocumentedBranchesPartialVersusFailedHeartbeat pins the two distinct
// heartbeat failure modes next to each other: a partial report (branch d) is a
// successful refresh that resets the gate, while a whole-refresh failure
// (branch e) is a 503 that retains the confirmed gate. Running both against the
// same confirmed ready gate makes it impossible for a future change to collapse
// one behavior into the other.
func TestDocumentedBranchesPartialVersusFailedHeartbeat(t *testing.T) {
	t.Run("branch d - partial report resets the gate", func(t *testing.T) {
		driver := newFakeDriver()
		handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))
		driver.setReport(readyOnlineReport())
		wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
			readyOnlineSnapshot())

		driver.setReport(Report{Status: ptr("online")})
		wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
			snapshotResponse{Status: StatusOnline})
		wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true"),
			ActionResponse{Message: MessageNotReady})
		wantCounts(t, driver, fakeCallCounts{snapshot: 2})
	})

	t.Run("branch e - whole refresh failure keeps the gate", func(t *testing.T) {
		driver := newFakeDriver()
		handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))
		driver.setReport(readyOnlineReport())
		wantSnapshot(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
			readyOnlineSnapshot())

		driver.setSnapshotErr(errors.New("malformed heartbeat body"))
		wantError(t, serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true"),
			http.StatusServiceUnavailable, errorStateUnavailable)
		wantAction(t, serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true"), SuccessAction())
		wantCounts(t, driver, fakeCallCounts{snapshot: 3, deinitialize: 1})
	})
}
