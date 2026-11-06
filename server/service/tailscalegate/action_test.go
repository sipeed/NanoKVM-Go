package tailscalegate

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type stubMessageError struct {
	message string
}

func (e stubMessageError) Error() string { return "stub failure" }

func (e stubMessageError) GateMessage() string { return e.message }

// stubTimeoutMessageError implements MessageError and also wraps a budget
// deadline. It pins the deliberate precedence in FailureAction: a deadline
// reports TIMEOUT instead of the driver's own code.
type stubTimeoutMessageError struct{}

func (stubTimeoutMessageError) Error() string       { return "stub timeout" }
func (stubTimeoutMessageError) GateMessage() string { return "NO_DAEMON" }
func (stubTimeoutMessageError) Unwrap() error       { return context.DeadlineExceeded }

func TestSuccessAction(t *testing.T) {
	if got := SuccessAction(); got != (ActionResponse{Status: true, Message: ""}) {
		t.Fatalf("SuccessAction() = %+v, want {true }", got)
	}
}

func TestFailureActionMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want ActionResponse
	}{
		{name: "nil is success", err: nil, want: ActionResponse{Status: true, Message: ""}},
		{
			name: "deadline exceeded is timeout",
			err:  context.DeadlineExceeded,
			want: ActionResponse{Message: MessageTimeout},
		},
		{
			name: "canceled is timeout",
			err:  context.Canceled,
			want: ActionResponse{Message: MessageTimeout},
		},
		{
			name: "wrapped deadline is timeout",
			err:  errors.Join(context.DeadlineExceeded),
			want: ActionResponse{Message: MessageTimeout},
		},
		{
			name: "plain error is fail",
			err:  errors.New("boom"),
			want: ActionResponse{Message: MessageFailure},
		},
		{
			name: "message error wrapping a deadline is timeout",
			err:  stubTimeoutMessageError{},
			want: ActionResponse{Message: MessageTimeout},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FailureAction(tc.err); got != tc.want {
				t.Fatalf("FailureAction(%v) = %+v, want %+v", tc.err, got, tc.want)
			}
		})
	}
}

func TestFailureActionBoundsMessageError(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "short message passes through", raw: "NO_DAEMON", want: "NO_DAEMON"},
		{name: "long message truncates", raw: "ABCDEFGHIJKL", want: "ABCDEFGHIJ"},
		{name: "non printable filtered then truncated", raw: "\x00ABCDEFGHIJKL\x7f", want: "ABCDEFGHIJ"},
		{name: "empty message stays empty", raw: "", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FailureAction(stubMessageError{message: tc.raw})
			want := ActionResponse{Message: tc.want}
			if got != want {
				t.Fatalf("FailureAction(MessageError{%q}) = %+v, want %+v", tc.raw, got, want)
			}
		})
	}
}

func TestActionResponseJSONFieldNames(t *testing.T) {
	cases := []struct {
		name string
		in   ActionResponse
		want string
	}{
		{name: "success", in: SuccessAction(), want: `{"status":true,"message":""}`},
		{name: "failure", in: FailureAction(errors.New("boom")), want: `{"status":false,"message":"FAIL"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatalf("json.Marshal returned error: %v", err)
			}
			if string(encoded) != tc.want {
				t.Fatalf("json.Marshal = %s, want %s", encoded, tc.want)
			}
		})
	}
}

func TestFailureMessageConstantsAreBounded(t *testing.T) {
	for _, message := range []string{
		MessageNotReady,
		MessageBusy,
		MessageTimeout,
		MessageNotImplemented,
		MessageFailure,
	} {
		if len(message) > maxMessageBytes {
			t.Fatalf("message %q is %d bytes, want <= %d", message, len(message), maxMessageBytes)
		}
		if boundedMessage(message) != message {
			t.Fatalf("message %q is not printable ASCII", message)
		}
	}
}
