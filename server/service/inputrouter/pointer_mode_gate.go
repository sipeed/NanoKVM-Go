package inputrouter

import (
	"fmt"

	"NanoKVM-Server/service/hid"
)

type pointerModeStatusProvider func() hid.PointerModeStatus

func pointerModeEventGate(status pointerModeStatusProvider) EventGate {
	return func(kind Kind) error {
		reportID, ok := hid2ReportIDForKind(kind)
		if !ok {
			return nil
		}

		active := status()
		if err := hid.ValidatePointerModeReportID(active.Mode, reportID); err != nil {
			return fmt.Errorf("reject %s input: %w", kindName(kind), err)
		}
		return nil
	}
}

func hid2ReportIDForKind(kind Kind) (byte, bool) {
	switch kind {
	case KindAbsolutePointer:
		return hid.AbsoluteReportIDMouse, true
	case KindStylus:
		return hid.AbsoluteReportIDStylus, true
	case KindTouchscreen:
		return hid.TouchScreenReportID, true
	default:
		return 0, false
	}
}
