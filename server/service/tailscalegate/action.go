package tailscalegate

import (
	"context"
	"errors"
)

// maxMessageBytes bounds every action failure message. The client truncates
// both the remote message and its own local text to the same length.
const maxMessageBytes = 10

// Stable action failure messages. Each is printable ASCII and at most
// maxMessageBytes bytes.
const (
	// MessageNotReady means an exit was requested before start and match were
	// both confirmed; the driver is not called.
	MessageNotReady = "NOT_READY"
	// MessageBusy means another action is already in flight; requests are not
	// queued behind it.
	MessageBusy = "BUSY"
	// MessageTimeout means the driver call exceeded its action budget.
	MessageTimeout = "TIMEOUT"
	// MessageNotImplemented means the configured driver has no implementation
	// for this action.
	MessageNotImplemented = "NOTIMPL"
	// MessageFailure is the fallback for any other driver error.
	MessageFailure = "FAIL"
	// MessageConsent means the start action requires the privacy-terms
	// acceptance to be persisted (through the consent flag or an earlier
	// explicit agreement) before the component may initialize.
	MessageConsent = "CONSENT"
	// MessageDisabled means the feature manifest forbids the managed mode, so
	// the start action is refused before any consent write, sync, or driver
	// call. It is a terminal policy answer; a 404 would be indistinguishable
	// from a transport failure on the client.
	MessageDisabled = "DISABLED"
)

// ActionResponse is the shared wire response for ?start and ?exit.
type ActionResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

// SuccessAction returns the successful action response.
func SuccessAction() ActionResponse {
	return ActionResponse{Status: true, Message: ""}
}

// MessageError lets a driver publish a short, stable failure code instead of
// the generic fallback. The value is filtered to printable ASCII and bounded to
// maxMessageBytes before it reaches the wire.
type MessageError interface {
	error
	GateMessage() string
}

// FailureAction maps a driver error to the action response.
//
// A nil error is success. A budget deadline or cancellation becomes TIMEOUT.
// An error implementing MessageError contributes its GateMessage, bounded and
// filtered. Everything else becomes FAIL. The returned Status is true only for
// a nil error.
func FailureAction(err error) ActionResponse {
	if err == nil {
		return SuccessAction()
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return ActionResponse{Message: MessageTimeout}
	}
	var messageErr MessageError
	if errors.As(err, &messageErr) {
		return ActionResponse{Message: boundedMessage(messageErr.GateMessage())}
	}
	return ActionResponse{Message: MessageFailure}
}

// boundedMessage filters value to printable ASCII and truncates it to the
// action message ceiling. Every failure message path goes through it so a
// driver cannot place a long or non-printable string on the wire.
func boundedMessage(value string) string {
	return printableASCII(value, maxMessageBytes)
}
