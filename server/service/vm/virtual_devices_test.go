package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/storageguard"

	"github.com/gin-gonic/gin"
)

func TestStorageBusyRestoresAudioEdidBeforeReturning(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	virtualMicMarkerExists = func() bool { return true }
	virtualSpeakerMarkerExists = func() bool { return false }
	virtualMicRuntimeMounted = func() bool { return true }
	virtualSpeakerRuntimeMounted = func() bool { return false }
	runVirtualDeviceCommands = func([]string) error { return storageguard.ErrBusy }
	resetVirtualAudioCapture = func() {}
	var edidUpdates []bool
	applyVirtualAudioEdid = func(enabled bool) error {
		edidUpdates = append(edidUpdates, enabled)
		return nil
	}
	resumed := false
	pauseVirtualAudioPlayback = func() func() { return func() { resumed = true } }
	rsp := performVirtualDeviceUpdate(t, proto.UpdateVirtualDeviceReq{Device: "audio", Mode: "duplex"})
	if rsp.Code != -7 {
		t.Fatalf("response = %+v", rsp)
	}
	if !reflect.DeepEqual(edidUpdates, []bool{false, true}) || !resumed {
		t.Fatalf("audio state was not restored on busy storage: EDID=%v resumed=%t", edidUpdates, resumed)
	}
}

func TestGetAudioCommandsSetsExplicitMode(t *testing.T) {
	tests := []struct {
		mode string
		want []string
	}{
		{
			mode: virtualAudioModeOff,
			want: []string{
				scriptUsbDev + " stop",
				"rm -f " + virtualMic,
				"rm -f " + virtualSpeaker,
				scriptUsbDev + " start",
			},
		},
		{
			mode: virtualAudioModeMic,
			want: []string{
				scriptUsbDev + " stop",
				"touch " + virtualMic,
				"rm -f " + virtualSpeaker,
				scriptUsbDev + " start",
			},
		},
		{
			mode: virtualAudioModeDuplex,
			want: []string{
				scriptUsbDev + " stop",
				"touch " + virtualMic,
				"touch " + virtualSpeaker,
				scriptUsbDev + " start",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.mode, func(t *testing.T) {
			got, err := getAudioCommands(test.mode)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("commands = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestGetAudioCommandsRejectsInvalidMode(t *testing.T) {
	if commands, err := getAudioCommands(""); err == nil || commands != nil {
		t.Fatalf("getAudioCommands(\"\") = %#v, %v; want nil, error", commands, err)
	}
	if commands, err := getAudioCommands("speaker"); err == nil || commands != nil {
		t.Fatalf("getAudioCommands(\"speaker\") = %#v, %v; want nil, error", commands, err)
	}
}

func TestResolveVirtualAudioModeUsesBootMarkerCombinations(t *testing.T) {
	tests := []struct {
		name           string
		micEnabled     bool
		speakerEnabled bool
		want           string
	}{
		{name: "off", want: virtualAudioModeOff},
		{name: "speaker without mic is invalid and treated as off", speakerEnabled: true, want: virtualAudioModeOff},
		{name: "microphone only", micEnabled: true, want: virtualAudioModeMic},
		{name: "duplex", micEnabled: true, speakerEnabled: true, want: virtualAudioModeDuplex},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := resolveVirtualAudioMode(test.micEnabled, test.speakerEnabled); got != test.want {
				t.Fatalf("resolveVirtualAudioMode(%t, %t) = %q, want %q", test.micEnabled, test.speakerEnabled, got, test.want)
			}
		})
	}
}

func TestResolveDeviceCommandsRejectsLegacyAudioDevices(t *testing.T) {
	for _, device := range []string{"mic", "speaker"} {
		t.Run(device, func(t *testing.T) {
			commands, err := resolveDeviceCommands(device, "", 0)
			if err == nil || commands != nil {
				t.Fatalf("resolveDeviceCommands(%q, \"\", 0) = %#v, %v; want nil, error", device, commands, err)
			}
		})
	}
}

func TestUpdateVirtualDeviceRequestKeepsNonAudioModeOptional(t *testing.T) {
	for _, device := range []string{"network", "disk"} {
		t.Run(device, func(t *testing.T) {
			if err := proto.ValidateRequest(&proto.UpdateVirtualDeviceReq{Device: device}); err != nil {
				t.Fatalf("validate %s request: %v", device, err)
			}
		})
	}
}

func TestVirtualDiskSizeLimitsReserveSystemStorage(t *testing.T) {
	originalAvailableMiB := virtualDiskAvailableMiB
	t.Cleanup(func() { virtualDiskAvailableMiB = originalAvailableMiB })

	tests := []struct {
		name      string
		available int64
		wantMax   int64
		wantErr   bool
	}{
		{name: "minimum valid capacity", available: 10 * 1024, wantMax: 5 * 1024},
		{name: "larger capacity", available: 32 * 1024, wantMax: 27 * 1024},
		{name: "insufficient capacity", available: 9 * 1024, wantMax: 4 * 1024, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			virtualDiskAvailableMiB = func() (int64, error) { return test.available, nil }
			minSizeMiB, maxSizeMiB, err := getVirtualDiskSizeLimits()
			if minSizeMiB != virtualDiskMinSizeMiB || maxSizeMiB != test.wantMax {
				t.Fatalf("limits = (%d, %d), want (%d, %d)", minSizeMiB, maxSizeMiB, virtualDiskMinSizeMiB, test.wantMax)
			}
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}

func TestGetDiskCommandsInitializesSelectedCapacity(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	virtualDiskRuntimeMounted = func() bool { return false }
	virtualDiskImagePresent = func() bool { return false }
	virtualDiskAvailableMiB = func() (int64, error) { return 32 * 1024, nil }

	var managerArgs []string
	runVirtualDiskImageManager = func(args ...string) error {
		managerArgs = append([]string(nil), args...)
		return nil
	}

	commands, err := getDiskCommands(16 * 1024)
	if err != nil {
		t.Fatal(err)
	}
	wantCommands := []string{
		"rm -f " + virtualDisk,
		"touch " + virtualDisk,
		scriptUsbDev + " restart",
	}
	if !reflect.DeepEqual(commands, wantCommands) {
		t.Fatalf("commands = %#v, want %#v", commands, wantCommands)
	}
	if !reflect.DeepEqual(managerArgs, []string{"start", "--size", "16384"}) {
		t.Fatalf("image manager args = %#v", managerArgs)
	}
}

func TestGetDiskCommandsRequiresCapacityForFirstInitialization(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	virtualDiskRuntimeMounted = func() bool { return false }
	virtualDiskImagePresent = func() bool { return false }
	runVirtualDiskImageManager = func(args ...string) error {
		t.Fatalf("image manager unexpectedly called with %#v", args)
		return nil
	}

	if commands, err := getDiskCommands(0); err == nil || commands != nil {
		t.Fatalf("getDiskCommands(0) = %#v, %v; want nil, error", commands, err)
	}
}

func TestGetDiskCommandsReusesInitializedStorageWithoutCapacity(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	virtualDiskRuntimeMounted = func() bool { return false }
	virtualDiskImagePresent = func() bool { return true }
	runVirtualDiskImageManager = func(args ...string) error {
		t.Fatalf("image manager unexpectedly called with %#v", args)
		return nil
	}

	commands, err := getDiskCommands(0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(commands, []string{
		"rm -f " + virtualDisk,
		"touch " + virtualDisk,
		scriptUsbDev + " restart",
	}) {
		t.Fatalf("commands = %#v", commands)
	}
}

func TestUpdateVirtualDeviceAudioModeLifecycle(t *testing.T) {
	tests := []struct {
		name              string
		from              string
		to                string
		wantDpAudioUpdate []bool
		wantRunCalls      int
		wantResumeCalls   int
		wantResetCalls    int
	}{
		{name: "off to off", from: virtualAudioModeOff, to: virtualAudioModeOff, wantRunCalls: 0, wantResumeCalls: 0, wantResetCalls: 0},
		{name: "off to mic", from: virtualAudioModeOff, to: virtualAudioModeMic, wantRunCalls: 1, wantResumeCalls: 1, wantResetCalls: 1},
		{name: "off to duplex", from: virtualAudioModeOff, to: virtualAudioModeDuplex, wantDpAudioUpdate: []bool{false}, wantRunCalls: 1, wantResumeCalls: 1, wantResetCalls: 1},
		{name: "mic to off", from: virtualAudioModeMic, to: virtualAudioModeOff, wantRunCalls: 1, wantResumeCalls: 0, wantResetCalls: 1},
		{name: "mic to mic", from: virtualAudioModeMic, to: virtualAudioModeMic, wantRunCalls: 0, wantResumeCalls: 0, wantResetCalls: 0},
		{name: "mic to duplex", from: virtualAudioModeMic, to: virtualAudioModeDuplex, wantDpAudioUpdate: []bool{false}, wantRunCalls: 1, wantResumeCalls: 1, wantResetCalls: 1},
		{name: "duplex to off", from: virtualAudioModeDuplex, to: virtualAudioModeOff, wantDpAudioUpdate: []bool{true}, wantRunCalls: 1, wantResumeCalls: 0, wantResetCalls: 1},
		{name: "duplex to mic", from: virtualAudioModeDuplex, to: virtualAudioModeMic, wantDpAudioUpdate: []bool{true}, wantRunCalls: 1, wantResumeCalls: 1, wantResetCalls: 1},
		{name: "duplex to duplex", from: virtualAudioModeDuplex, to: virtualAudioModeDuplex, wantRunCalls: 0, wantResumeCalls: 0, wantResetCalls: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupVirtualDeviceHandlerTest(t)
			pauseCalls := 0
			resumeCalls := 0
			pauseVirtualAudioPlayback = func() func() {
				pauseCalls++
				return func() { resumeCalls++ }
			}

			micEnabled := test.from != virtualAudioModeOff
			speakerEnabled := test.from == virtualAudioModeDuplex
			micMounted := micEnabled
			speakerMounted := speakerEnabled
			virtualMicMarkerExists = func() bool { return micEnabled }
			virtualSpeakerMarkerExists = func() bool { return speakerEnabled }
			virtualMicRuntimeMounted = func() bool { return micMounted }
			virtualSpeakerRuntimeMounted = func() bool { return speakerMounted }
			runCalls := 0
			var commands []string
			runVirtualDeviceCommands = func(got []string) error {
				runCalls++
				commands = append([]string(nil), got...)
				micEnabled = test.to != virtualAudioModeOff
				speakerEnabled = test.to == virtualAudioModeDuplex
				micMounted = micEnabled
				speakerMounted = speakerEnabled
				return nil
			}
			resetCalls := 0
			resetVirtualAudioCapture = func() { resetCalls++ }
			var dpAudioUpdates []bool
			applyVirtualAudioEdid = func(enabled bool) error {
				dpAudioUpdates = append(dpAudioUpdates, enabled)
				return nil
			}

			rsp := performVirtualDeviceUpdate(t, proto.UpdateVirtualDeviceReq{
				Device: "audio",
				Mode:   test.to,
			})
			if rsp.Code != 0 {
				t.Fatalf("response = code %d msg %q, want success", rsp.Code, rsp.Msg)
			}
			if runCalls != test.wantRunCalls {
				t.Fatalf("command executions = %d, want %d", runCalls, test.wantRunCalls)
			}
			if pauseCalls != test.wantRunCalls || resumeCalls != test.wantResumeCalls {
				t.Fatalf(
					"playback pause/resume calls = %d/%d, want %d/%d",
					pauseCalls,
					resumeCalls,
					test.wantRunCalls,
					test.wantResumeCalls,
				)
			}
			if test.wantRunCalls == 1 {
				assertSingleUsbDevStopStart(t, commands)
			}
			if resetCalls != test.wantResetCalls {
				t.Fatalf("audio capture resets = %d, want %d", resetCalls, test.wantResetCalls)
			}
			if !reflect.DeepEqual(dpAudioUpdates, test.wantDpAudioUpdate) {
				t.Fatalf("DP audio updates = %v, want %v", dpAudioUpdates, test.wantDpAudioUpdate)
			}
		})
	}
}

func TestUpdateVirtualDeviceSameAudioModeRepairsRuntimeWithoutEdid(t *testing.T) {
	tests := []struct {
		name           string
		mode           string
		micMarker      bool
		speakerMarker  bool
		micRuntime     bool
		speakerRuntime bool
	}{
		{name: "off with stale microphone runtime", mode: virtualAudioModeOff, micRuntime: true},
		{name: "mic with missing microphone runtime", mode: virtualAudioModeMic, micMarker: true, micRuntime: false},
		{name: "duplex with missing speaker runtime", mode: virtualAudioModeDuplex, micMarker: true, speakerMarker: true, micRuntime: true, speakerRuntime: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupVirtualDeviceHandlerTest(t)

			micEnabled := test.micMarker
			speakerEnabled := test.speakerMarker
			micMounted := test.micRuntime
			speakerMounted := test.speakerRuntime
			virtualMicMarkerExists = func() bool { return micEnabled }
			virtualSpeakerMarkerExists = func() bool { return speakerEnabled }
			virtualMicRuntimeMounted = func() bool { return micMounted }
			virtualSpeakerRuntimeMounted = func() bool { return speakerMounted }
			runCalls := 0
			var commands []string
			runVirtualDeviceCommands = func(got []string) error {
				runCalls++
				commands = append([]string(nil), got...)
				micEnabled = test.mode != virtualAudioModeOff
				speakerEnabled = test.mode == virtualAudioModeDuplex
				micMounted = micEnabled
				speakerMounted = speakerEnabled
				return nil
			}
			resetCalls := 0
			resetVirtualAudioCapture = func() { resetCalls++ }
			applyVirtualAudioEdid = func(bool) error {
				t.Fatal("runtime repair must not update EDID")
				return nil
			}

			rsp := performVirtualDeviceUpdate(t, proto.UpdateVirtualDeviceReq{Device: "audio", Mode: test.mode})
			if rsp.Code != 0 {
				t.Fatalf("response = code %d msg %q, want success", rsp.Code, rsp.Msg)
			}
			if runCalls != 1 {
				t.Fatalf("command executions = %d, want 1", runCalls)
			}
			assertSingleUsbDevStopStart(t, commands)
			if resetCalls != 1 {
				t.Fatalf("audio capture resets = %d, want 1", resetCalls)
			}
		})
	}
}

func TestUpdateVirtualDeviceSkipsAudioRebuildWhenStateMatches(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	virtualMicMarkerExists = func() bool { return true }
	virtualSpeakerMarkerExists = func() bool { return false }
	virtualMicRuntimeMounted = func() bool { return true }
	virtualSpeakerRuntimeMounted = func() bool { return false }
	runVirtualDeviceCommands = func([]string) error {
		t.Fatal("matching audio mode must not rebuild the USB gadget")
		return nil
	}
	resetVirtualAudioCapture = func() {
		t.Fatal("matching audio mode must not reset audio capture")
	}
	applyVirtualAudioEdid = func(bool) error {
		t.Fatal("matching audio mode must not update EDID")
		return nil
	}

	rsp := performVirtualDeviceUpdate(t, proto.UpdateVirtualDeviceReq{
		Device: "audio",
		Mode:   virtualAudioModeMic,
	})
	if rsp.Code != 0 {
		t.Fatalf("response = code %d msg %q, want success", rsp.Code, rsp.Msg)
	}
}

func TestVirtualAudioStateUsesRawSpeakerRuntime(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	virtualMicMarkerExists = func() bool { return true }
	virtualSpeakerMarkerExists = func() bool { return false }
	virtualMicRuntimeMounted = func() bool { return true }
	virtualSpeakerRuntimeMounted = func() bool { return true }

	if isVirtualAudioStateCurrent(virtualAudioModeMic) {
		t.Fatal("mic mode must not be current while raw speaker capture remains enabled")
	}
}

func TestUpdateVirtualDeviceEdidFailureRestoresPreviousCapability(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	pauseVirtualAudioPlayback = func() func() {
		t.Fatal("playback must not pause when EDID fails before gadget rebuild")
		return nil
	}
	virtualMicMarkerExists = func() bool { return false }
	virtualSpeakerMarkerExists = func() bool { return false }
	virtualMicRuntimeMounted = func() bool { return false }
	virtualSpeakerRuntimeMounted = func() bool { return false }
	runVirtualDeviceCommands = func([]string) error {
		t.Fatal("USB gadget must not rebuild after EDID failure")
		return nil
	}
	resetVirtualAudioCapture = func() {
		t.Fatal("audio capture must not reset after EDID failure")
	}
	var dpAudioUpdates []bool
	applyVirtualAudioEdid = func(enabled bool) error {
		dpAudioUpdates = append(dpAudioUpdates, enabled)
		if len(dpAudioUpdates) == 1 {
			return errors.New("injected failure")
		}
		return nil
	}

	rsp := performVirtualDeviceUpdate(t, proto.UpdateVirtualDeviceReq{
		Device: "audio",
		Mode:   virtualAudioModeDuplex,
	})
	if rsp.Code != -4 {
		t.Fatalf("response code = %d, want -4", rsp.Code)
	}
	if !reflect.DeepEqual(dpAudioUpdates, []bool{false, true}) {
		t.Fatalf("DP audio updates = %v, want failed target followed by restore", dpAudioUpdates)
	}
}

func TestUpdateVirtualDeviceEdidRestoreFailureReturnsError(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	virtualMicMarkerExists = func() bool { return false }
	virtualSpeakerMarkerExists = func() bool { return false }
	virtualMicRuntimeMounted = func() bool { return false }
	virtualSpeakerRuntimeMounted = func() bool { return false }
	runVirtualDeviceCommands = func([]string) error {
		t.Fatal("USB gadget must not rebuild after EDID failure")
		return nil
	}
	resetVirtualAudioCapture = func() {
		t.Fatal("audio capture must not reset after EDID failure")
	}
	var dpAudioUpdates []bool
	applyVirtualAudioEdid = func(enabled bool) error {
		dpAudioUpdates = append(dpAudioUpdates, enabled)
		return errors.New("injected failure")
	}

	rsp := performVirtualDeviceUpdate(t, proto.UpdateVirtualDeviceReq{
		Device: "audio",
		Mode:   virtualAudioModeDuplex,
	})
	if rsp.Code != -4 {
		t.Fatalf("response code = %d, want -4", rsp.Code)
	}
	if !reflect.DeepEqual(dpAudioUpdates, []bool{false, true}) {
		t.Fatalf("DP audio updates = %v, want failed target and restore", dpAudioUpdates)
	}
}

func TestUpdateVirtualDeviceRejectsInvalidAudioRequests(t *testing.T) {
	tests := []struct {
		req      proto.UpdateVirtualDeviceReq
		wantCode int
	}{
		{req: proto.UpdateVirtualDeviceReq{Device: "audio"}, wantCode: -1},
		{req: proto.UpdateVirtualDeviceReq{Device: "audio", Mode: "speaker"}, wantCode: -1},
		{req: proto.UpdateVirtualDeviceReq{Device: "mic"}, wantCode: -2},
		{req: proto.UpdateVirtualDeviceReq{Device: "speaker"}, wantCode: -2},
	}

	for _, test := range tests {
		t.Run(test.req.Device+"_"+test.req.Mode, func(t *testing.T) {
			setupVirtualDeviceHandlerTest(t)
			runVirtualDeviceCommands = func([]string) error {
				t.Fatal("commands must not run for an invalid audio request")
				return nil
			}
			resetVirtualAudioCapture = func() {
				t.Fatal("audio capture must not reset for an invalid audio request")
			}
			applyVirtualAudioEdid = func(bool) error {
				t.Fatal("EDID must not update for an invalid audio request")
				return nil
			}

			rsp := performVirtualDeviceUpdate(t, test.req)
			if rsp.Code != test.wantCode {
				t.Fatalf("response = code %d msg %q, want code %d", rsp.Code, rsp.Msg, test.wantCode)
			}
		})
	}
}

func TestUpdateVirtualDeviceRejectsAudioInHidOnlyMode(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	virtualAudioHidOnly = func() bool { return true }
	runVirtualDeviceCommands = func([]string) error {
		t.Fatal("HID-only mode must not rebuild the normal USB gadget")
		return nil
	}
	resetVirtualAudioCapture = func() {
		t.Fatal("HID-only rejection must not reset audio capture")
	}
	applyVirtualAudioEdid = func(bool) error {
		t.Fatal("HID-only rejection must not update EDID")
		return nil
	}

	rsp := performVirtualDeviceUpdate(t, proto.UpdateVirtualDeviceReq{
		Device: "audio",
		Mode:   virtualAudioModeMic,
	})
	if rsp.Code != -5 {
		t.Fatalf("response code = %d, want -5", rsp.Code)
	}
}

func TestUpdateVirtualDeviceNetworkSkipsAudioFollowUp(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	virtualSpeakerMarkerExists = func() bool {
		t.Fatal("network update must not inspect the virtual speaker marker")
		return false
	}
	runCalls := 0
	var commands []string
	runVirtualDeviceCommands = func(got []string) error {
		runCalls++
		commands = append([]string(nil), got...)
		return nil
	}
	resetVirtualAudioCapture = func() {
		t.Fatal("network update must not reset audio capture")
	}
	applyVirtualAudioEdid = func(bool) error {
		t.Fatal("network update must not update EDID")
		return nil
	}

	rsp := performVirtualDeviceUpdate(t, proto.UpdateVirtualDeviceReq{Device: "network"})
	if rsp.Code != 0 {
		t.Fatalf("response = code %d msg %q, want success", rsp.Code, rsp.Msg)
	}
	if runCalls != 1 {
		t.Fatalf("command executions = %d, want 1", runCalls)
	}
	assertSingleUsbDevStopStart(t, commands)
}

func TestUpdateVirtualDeviceCommandFailureRollsBack(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	micEnabled := false
	speakerEnabled := false
	micMounted := false
	speakerMounted := false
	virtualMicMarkerExists = func() bool { return micEnabled }
	virtualSpeakerMarkerExists = func() bool { return speakerEnabled }
	virtualMicRuntimeMounted = func() bool { return micMounted }
	virtualSpeakerRuntimeMounted = func() bool { return speakerMounted }
	runCalls := 0
	var events []string
	pauseVirtualAudioPlayback = func() func() {
		events = append(events, "playback pause")
		return func() { events = append(events, "playback resume") }
	}
	runVirtualDeviceCommands = func(commands []string) error {
		runCalls++
		if runCalls == 1 {
			events = append(events, "usb update")
			micEnabled = true
			speakerEnabled = true
			return errors.New("injected failure")
		}
		events = append(events, "usb rollback")
		if !reflect.DeepEqual(commands, getAudioStateCommands(false, false)) {
			t.Fatalf("rollback commands = %#v, want off state", commands)
		}
		micEnabled = false
		// Simulate a failed USB rollback that leaves a stale speaker marker.
		// EDID recovery must still use the state captured before the update.
		speakerEnabled = true
		micMounted = false
		speakerMounted = false
		return errors.New("injected rollback failure")
	}
	resetCalls := 0
	resetVirtualAudioCapture = func() { resetCalls++ }
	var dpAudioUpdates []bool
	applyVirtualAudioEdid = func(enabled bool) error {
		dpAudioUpdates = append(dpAudioUpdates, enabled)
		if enabled {
			events = append(events, "EDID with DP audio")
		} else {
			events = append(events, "EDID without DP audio")
		}
		return nil
	}

	rsp := performVirtualDeviceUpdate(t, proto.UpdateVirtualDeviceReq{
		Device: "audio",
		Mode:   virtualAudioModeDuplex,
	})
	if rsp.Code != -3 {
		t.Fatalf("response code = %d, want -3", rsp.Code)
	}
	if runCalls != 2 {
		t.Fatalf("command executions = %d, want failed update plus rollback", runCalls)
	}
	if resetCalls != 1 {
		t.Fatalf("audio capture resets = %d, want 1 after rollback", resetCalls)
	}
	if !reflect.DeepEqual(dpAudioUpdates, []bool{false, true}) {
		t.Fatalf("DP audio updates = %v, want target update followed by rollback", dpAudioUpdates)
	}
	wantEvents := []string{
		"EDID without DP audio",
		"playback pause",
		"usb update",
		"usb rollback",
		"EDID with DP audio",
	}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
}

func TestUpdateVirtualDevicePostconditionFailureRollsBack(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	micEnabled := false
	speakerEnabled := false
	micMounted := false
	speakerMounted := false
	virtualMicMarkerExists = func() bool { return micEnabled }
	virtualSpeakerMarkerExists = func() bool { return speakerEnabled }
	virtualMicRuntimeMounted = func() bool { return micMounted }
	virtualSpeakerRuntimeMounted = func() bool { return speakerMounted }
	runCalls := 0
	var events []string
	pauseVirtualAudioPlayback = func() func() {
		events = append(events, "playback pause")
		return func() { events = append(events, "playback resume") }
	}
	runVirtualDeviceCommands = func(commands []string) error {
		runCalls++
		if runCalls == 1 {
			events = append(events, "usb update")
			micEnabled = true
			speakerEnabled = true
			// Simulate usbdev.sh returning success without creating the configfs runtime.
			return nil
		}
		events = append(events, "usb rollback")
		if !reflect.DeepEqual(commands, getAudioStateCommands(false, false)) {
			t.Fatalf("rollback commands = %#v, want off state", commands)
		}
		micEnabled = false
		// Leave the speaker marker stale to prove rollback does not inspect it.
		speakerEnabled = true
		micMounted = false
		speakerMounted = false
		return nil
	}
	resetCalls := 0
	resetVirtualAudioCapture = func() { resetCalls++ }
	var dpAudioUpdates []bool
	applyVirtualAudioEdid = func(enabled bool) error {
		dpAudioUpdates = append(dpAudioUpdates, enabled)
		if enabled {
			events = append(events, "EDID with DP audio")
		} else {
			events = append(events, "EDID without DP audio")
		}
		return nil
	}

	rsp := performVirtualDeviceUpdate(t, proto.UpdateVirtualDeviceReq{
		Device: "audio",
		Mode:   virtualAudioModeDuplex,
	})
	if rsp.Code != -3 {
		t.Fatalf("response code = %d, want -3", rsp.Code)
	}
	if runCalls != 2 {
		t.Fatalf("command executions = %d, want update plus rollback", runCalls)
	}
	if resetCalls != 1 {
		t.Fatalf("audio capture resets = %d, want 1 after rollback", resetCalls)
	}
	if !reflect.DeepEqual(dpAudioUpdates, []bool{false, true}) {
		t.Fatalf("DP audio updates = %v, want target update followed by rollback", dpAudioUpdates)
	}
	wantEvents := []string{
		"EDID without DP audio",
		"playback pause",
		"usb update",
		"usb rollback",
		"EDID with DP audio",
	}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
}

func TestUpdateVirtualDeviceOffFailureResumesPreviousPlayback(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	micEnabled := true
	speakerEnabled := false
	micMounted := true
	speakerMounted := false
	virtualMicMarkerExists = func() bool { return micEnabled }
	virtualSpeakerMarkerExists = func() bool { return speakerEnabled }
	virtualMicRuntimeMounted = func() bool { return micMounted }
	virtualSpeakerRuntimeMounted = func() bool { return speakerMounted }
	var events []string
	pauseVirtualAudioPlayback = func() func() {
		events = append(events, "playback pause")
		return func() { events = append(events, "playback resume") }
	}
	runCalls := 0
	runVirtualDeviceCommands = func(commands []string) error {
		runCalls++
		if runCalls == 1 {
			events = append(events, "usb update")
			micEnabled = false
			micMounted = false
			return errors.New("injected failure")
		}
		events = append(events, "usb rollback")
		if !reflect.DeepEqual(commands, getAudioStateCommands(true, false)) {
			t.Fatalf("rollback commands = %#v, want mic state", commands)
		}
		micEnabled = true
		micMounted = true
		return nil
	}
	resetCalls := 0
	resetVirtualAudioCapture = func() { resetCalls++ }
	applyVirtualAudioEdid = func(bool) error {
		t.Fatal("off from mic must not update EDID")
		return nil
	}

	rsp := performVirtualDeviceUpdate(t, proto.UpdateVirtualDeviceReq{
		Device: "audio",
		Mode:   virtualAudioModeOff,
	})
	if rsp.Code != -3 {
		t.Fatalf("response code = %d, want -3", rsp.Code)
	}
	if runCalls != 2 {
		t.Fatalf("command executions = %d, want update plus rollback", runCalls)
	}
	if resetCalls != 1 {
		t.Fatalf("audio capture resets = %d, want 1 after rollback", resetCalls)
	}
	wantEvents := []string{"playback pause", "usb update", "usb rollback", "playback resume"}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
}

func TestDisableVirtualDiskRemovesDisconnectedStorage(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	virtualDiskBackingState = func() (bool, error) { return false, nil }
	virtualDiskMountedImageState = func() (bool, error) { return false, nil }
	virtualDiskRuntimePresent = func() bool { return true }

	markerRemovals := 0
	removeVirtualDiskMarker = func() error {
		markerRemovals++
		return nil
	}
	var gadgetCommands []string
	runVirtualDeviceCommands = func(commands []string) error {
		gadgetCommands = append([]string(nil), commands...)
		return nil
	}
	imagePresent := true
	virtualDiskImagePresent = func() bool { return imagePresent }
	var managerArgs []string
	runVirtualDiskImageManager = func(args ...string) error {
		managerArgs = append([]string(nil), args...)
		imagePresent = false
		return nil
	}

	rsp := performVirtualDiskDisable(t)
	if rsp.Code != 0 {
		t.Fatalf("response = code %d msg %q, want success", rsp.Code, rsp.Msg)
	}
	if markerRemovals != 1 {
		t.Fatalf("marker removals = %d, want 1", markerRemovals)
	}
	if !reflect.DeepEqual(gadgetCommands, []string{scriptUsbDev + " restart"}) {
		t.Fatalf("gadget commands = %#v", gadgetCommands)
	}
	if !reflect.DeepEqual(managerArgs, []string{"remove"}) {
		t.Fatalf("image manager args = %#v", managerArgs)
	}
}

func TestDisableVirtualDiskRejectsConnectedBackingFile(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	virtualDiskBackingState = func() (bool, error) { return true, nil }
	removeVirtualDiskMarker = func() error {
		t.Fatal("connected virtual disk marker must not be removed")
		return nil
	}

	rsp := performVirtualDiskDisable(t)
	if rsp.Code != -2 {
		t.Fatalf("response = code %d msg %q, want code -2", rsp.Code, rsp.Msg)
	}
}

func TestDisableVirtualDiskRejectsMountedImage(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	virtualDiskBackingState = func() (bool, error) { return false, nil }
	virtualDiskMountedImageState = func() (bool, error) { return true, nil }
	removeVirtualDiskMarker = func() error {
		t.Fatal("virtual disk marker must not be removed while an image is mounted")
		return nil
	}

	rsp := performVirtualDiskDisable(t)
	if rsp.Code != -4 {
		t.Fatalf("response = code %d msg %q, want code -4", rsp.Code, rsp.Msg)
	}
}

func TestDisableVirtualDiskReportsBusyStorage(t *testing.T) {
	setupVirtualDeviceHandlerTest(t)
	virtualDiskBackingState = func() (bool, error) { return false, nil }
	virtualDiskMountedImageState = func() (bool, error) { return false, nil }
	virtualDiskRuntimePresent = func() bool { return false }
	removeVirtualDiskMarker = func() error { return nil }
	virtualDiskImagePresent = func() bool { return true }
	runVirtualDiskImageManager = func(args ...string) error {
		return errors.New("data.mount is busy")
	}

	rsp := performVirtualDiskDisable(t)
	if rsp.Code != -6 {
		t.Fatalf("response = code %d msg %q, want code -6", rsp.Code, rsp.Msg)
	}
}

type virtualDeviceTestResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func setupVirtualDeviceHandlerTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	originalRun := runVirtualDeviceCommands
	originalReset := resetVirtualAudioCapture
	originalPausePlayback := pauseVirtualAudioPlayback
	originalApplyEdid := applyVirtualAudioEdid
	originalHidOnly := virtualAudioHidOnly
	originalMicMarkerExists := virtualMicMarkerExists
	originalSpeakerMarkerExists := virtualSpeakerMarkerExists
	originalMicRuntimeMounted := virtualMicRuntimeMounted
	originalSpeakerRuntimeMounted := virtualSpeakerRuntimeMounted
	originalDiskRuntimeMounted := virtualDiskRuntimeMounted
	originalDiskImagePresent := virtualDiskImagePresent
	originalDiskRuntimePresent := virtualDiskRuntimePresent
	originalDiskBackingState := virtualDiskBackingState
	originalDiskMountedImageState := virtualDiskMountedImageState
	originalDiskAvailableMiB := virtualDiskAvailableMiB
	originalRemoveDiskMarker := removeVirtualDiskMarker
	originalRunDiskImageManager := runVirtualDiskImageManager
	virtualAudioHidOnly = func() bool { return false }
	pauseVirtualAudioPlayback = func() func() { return func() {} }
	t.Cleanup(func() {
		runVirtualDeviceCommands = originalRun
		resetVirtualAudioCapture = originalReset
		pauseVirtualAudioPlayback = originalPausePlayback
		applyVirtualAudioEdid = originalApplyEdid
		virtualAudioHidOnly = originalHidOnly
		virtualMicMarkerExists = originalMicMarkerExists
		virtualSpeakerMarkerExists = originalSpeakerMarkerExists
		virtualMicRuntimeMounted = originalMicRuntimeMounted
		virtualSpeakerRuntimeMounted = originalSpeakerRuntimeMounted
		virtualDiskRuntimeMounted = originalDiskRuntimeMounted
		virtualDiskImagePresent = originalDiskImagePresent
		virtualDiskRuntimePresent = originalDiskRuntimePresent
		virtualDiskBackingState = originalDiskBackingState
		virtualDiskMountedImageState = originalDiskMountedImageState
		virtualDiskAvailableMiB = originalDiskAvailableMiB
		removeVirtualDiskMarker = originalRemoveDiskMarker
		runVirtualDiskImageManager = originalRunDiskImageManager
	})
}

func performVirtualDeviceUpdate(t *testing.T, req proto.UpdateVirtualDeviceReq) virtualDeviceTestResponse {
	t.Helper()

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/vm/device/virtual", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	NewService().UpdateVirtualDevice(ctx)

	var rsp virtualDeviceTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	return rsp
}

func performVirtualDiskDisable(t *testing.T) virtualDeviceTestResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/vm/device/virtual/disk/disable", nil)

	NewService().DisableVirtualDisk(ctx)

	var rsp virtualDeviceTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	return rsp
}

func assertSingleUsbDevStopStart(t *testing.T, commands []string) {
	t.Helper()

	stop := scriptUsbDev + " stop"
	start := scriptUsbDev + " start"
	stopCount := 0
	startCount := 0
	for _, command := range commands {
		switch command {
		case stop:
			stopCount++
		case start:
			startCount++
		}
	}
	if stopCount != 1 || startCount != 1 {
		t.Fatalf("commands = %#v, want exactly one stop and one start", commands)
	}
	if len(commands) < 2 || commands[0] != stop || commands[len(commands)-1] != start {
		t.Fatalf("commands = %#v, want stop first and start last", commands)
	}
}

func TestWebcamEndpointBudget(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"hid.keyboard", "hid.mouse", "hid.touch", "uvc.usb0", "mass_storage.disk0", "mass_storage.disk1"} {
		if err := os.Symlink("../functions/"+name, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	if !webcamEndpointsAvailable(dir, "mass_storage.disk1") {
		t.Fatal("reusing the already-linked disk should fit all seven IN endpoints")
	}
	if webcamEndpointsAvailable(dir, "uac2.usb0") || webcamEndpointsAvailable(dir, "ncm.usb0") {
		t.Fatal("an additional USB function exceeds the seven IN endpoints")
	}
	if err := os.Remove(filepath.Join(dir, "mass_storage.disk1")); err != nil {
		t.Fatal(err)
	}
	if !webcamEndpointsAvailable(dir, "uac2.usb0") || webcamEndpointsAvailable(dir, "ncm.usb0") {
		t.Fatal("one free endpoint should fit audio but not networking")
	}
}
