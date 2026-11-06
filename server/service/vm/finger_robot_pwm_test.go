package vm

import (
	"errors"
	"reflect"
	"testing"
)

func TestFingerRobotPWMInitialDutyUsesSafeReleasePoint(t *testing.T) {
	if fingerRobotPWMSafeReleaseNS != "2050000" {
		t.Fatalf("initial PWM duty = %s, want 2050000", fingerRobotPWMSafeReleaseNS)
	}
}

func TestFingerRobotPWMUsesOneOutputAndGPIOCapturePinMux(t *testing.T) {
	if len(fingerRobotPWMChannels) != 1 || fingerRobotPWMChannels[0] != fingerRobotPWMChannel0 {
		t.Fatalf("finger robot PWM channels = %v, want only PWM0", fingerRobotPWMChannels)
	}

	var got []fingerRobotPWMPinMuxSetting
	err := configureFingerRobotPWMMuxWith(
		fingerRobotPWMPinMuxSettings,
		func(addr, value string) error {
			got = append(got, fingerRobotPWMPinMuxSetting{addr: addr, value: value})
			return nil
		},
	)
	if err != nil {
		t.Fatalf("configureFingerRobotPWMMuxWith: %v", err)
	}
	want := []fingerRobotPWMPinMuxSetting{
		{addr: "0x02304090", value: "0x00060013"},
		{addr: "0x02304084", value: "0x00030003"},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("pin mux sequence = %+v, want %+v", got, want)
	}
}

func TestFingerRobotPWMPinMuxStopsAfterFirstFailure(t *testing.T) {
	wantErr := errors.New("capture pin mux failed")
	var calls []fingerRobotPWMPinMuxSetting
	err := configureFingerRobotPWMMuxWith(
		fingerRobotPWMPinMuxSettings,
		func(addr, value string) error {
			calls = append(calls, fingerRobotPWMPinMuxSetting{addr: addr, value: value})
			return wantErr
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("pin mux error = %v, want %v", err, wantErr)
	}
	if len(calls) != 1 || calls[0] != fingerRobotPWMPinMuxSettings[0] {
		t.Fatalf("pin mux continued after failure: %+v", calls)
	}
}

func TestFingerRobotPWMShutdownPinsPrepareGPIOAndUsePulldownMux(t *testing.T) {
	var calls []string
	var gpioWrites []struct {
		addr  string
		value uint32
	}
	readValues := map[string]uint32{
		fingerRobotPWMOutputGPIOAddr:  0xA5A5A5A7,
		fingerRobotPWMCaptureGPIOAddr: 0x5A5A5A5B,
	}

	err := restoreFingerRobotPWMShutdownPinsWith(
		fingerRobotPWMShutdownGPIOAddrs,
		func(addr string) (uint32, error) {
			calls = append(calls, "read:"+addr)
			return readValues[addr], nil
		},
		func(addr string, value uint32) error {
			calls = append(calls, "gpio:"+addr)
			gpioWrites = append(gpioWrites, struct {
				addr  string
				value uint32
			}{addr: addr, value: value})
			return nil
		},
		fingerRobotPWMShutdownMuxSettings,
		func(addr, value string) error {
			calls = append(calls, "mux:"+addr+"="+value)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("restoreFingerRobotPWMShutdownPinsWith: %v", err)
	}

	wantCalls := []string{
		"read:" + fingerRobotPWMOutputGPIOAddr,
		"read:" + fingerRobotPWMCaptureGPIOAddr,
		"gpio:" + fingerRobotPWMOutputGPIOAddr,
		"gpio:" + fingerRobotPWMCaptureGPIOAddr,
		"mux:" + fingerRobotPWMOutputMuxAddr + "=0x00060043",
		"mux:" + fingerRobotPWMCaptureMuxAddr + "=0x00060043",
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("shutdown pin calls = %v, want %v", calls, wantCalls)
	}
	wantGPIOWrites := []struct {
		addr  string
		value uint32
	}{
		{addr: fingerRobotPWMOutputGPIOAddr, value: 0xA5A5A5A4},
		{addr: fingerRobotPWMCaptureGPIOAddr, value: 0x5A5A5A58},
	}
	if !reflect.DeepEqual(gpioWrites, wantGPIOWrites) {
		t.Fatalf("GPIO writes = %#v, want %#v", gpioWrites, wantGPIOWrites)
	}
}

func TestFingerRobotPWMShutdownPinsWaitForFreshLowBeforeMux(t *testing.T) {
	var calls []string
	err := restoreFingerRobotPWMShutdownPinsWithWait(
		fingerRobotPWMShutdownGPIOAddrs,
		func(addr string) (uint32, error) {
			calls = append(calls, "read:"+addr)
			return 0xFFFFFFFF, nil
		},
		func(addr string, _ uint32) error {
			calls = append(calls, "gpio:"+addr)
			return nil
		},
		fingerRobotPWMShutdownMuxSettings,
		func(addr, _ string) error {
			calls = append(calls, "mux:"+addr)
			return nil
		},
		func() error {
			calls = append(calls, "wait-for-falling-edge")
			return nil
		},
	)
	if err != nil {
		t.Fatalf("restoreFingerRobotPWMShutdownPinsWithWait: %v", err)
	}

	want := []string{
		"read:" + fingerRobotPWMOutputGPIOAddr,
		"read:" + fingerRobotPWMCaptureGPIOAddr,
		"gpio:" + fingerRobotPWMOutputGPIOAddr,
		"gpio:" + fingerRobotPWMCaptureGPIOAddr,
		"wait-for-falling-edge",
		"mux:" + fingerRobotPWMOutputMuxAddr,
		"mux:" + fingerRobotPWMCaptureMuxAddr,
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("shutdown boundary calls = %v, want %v", calls, want)
	}
}

func TestFingerRobotPWMShutdownPinsMuxStillRunsWhenBoundaryWaitFails(t *testing.T) {
	waitErr := errors.New("safe pulse falling edge timed out")
	var muxCalls []string
	err := restoreFingerRobotPWMShutdownPinsWithWait(
		fingerRobotPWMShutdownGPIOAddrs,
		func(string) (uint32, error) { return 0, nil },
		func(string, uint32) error { return nil },
		fingerRobotPWMShutdownMuxSettings,
		func(addr, _ string) error {
			muxCalls = append(muxCalls, addr)
			return nil
		},
		func() error { return waitErr },
	)
	if !errors.Is(err, waitErr) {
		t.Fatalf("boundary wait error = %v, want %v", err, waitErr)
	}
	if !reflect.DeepEqual(muxCalls, []string{fingerRobotPWMOutputMuxAddr, fingerRobotPWMCaptureMuxAddr}) {
		t.Fatalf("mux writes after boundary wait failure = %v", muxCalls)
	}
}

func TestFingerRobotPWMShutdownPinsContinueAfterIndividualFailures(t *testing.T) {
	readErr := errors.New("GPIO read failed")
	writeErr := errors.New("GPIO write failed")
	muxErr := errors.New("pin mux restore failed")
	var muxCalls []string

	err := restoreFingerRobotPWMShutdownPinsWith(
		fingerRobotPWMShutdownGPIOAddrs,
		func(addr string) (uint32, error) {
			if addr == fingerRobotPWMOutputGPIOAddr {
				return 0, readErr
			}
			return 0x3, nil
		},
		func(addr string, _ uint32) error {
			if addr == fingerRobotPWMCaptureGPIOAddr {
				return writeErr
			}
			return nil
		},
		fingerRobotPWMShutdownMuxSettings,
		func(addr, _ string) error {
			muxCalls = append(muxCalls, addr)
			return muxErr
		},
	)

	if !errors.Is(err, readErr) || !errors.Is(err, writeErr) || !errors.Is(err, muxErr) {
		t.Fatalf("shutdown pin restore errors = %v, want all injected errors", err)
	}
	if !reflect.DeepEqual(muxCalls, []string{fingerRobotPWMOutputMuxAddr, fingerRobotPWMCaptureMuxAddr}) {
		t.Fatalf("pin mux restore calls = %v", muxCalls)
	}
}

func TestParseFingerRobotGPIORegisterValue(t *testing.T) {
	for _, test := range []struct {
		name   string
		output []byte
		want   uint32
	}{
		{name: "hex", output: []byte("0xA5A5A5A7\n"), want: 0xA5A5A5A7},
		{name: "decimal", output: []byte("305419896\n"), want: 0x12345678},
		{name: "zero-padded hex without prefix", output: []byte("00000010\n"), want: 0x10},
		{name: "trailing text", output: []byte("devmem: 0x12345678\n"), want: 0x12345678},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseFingerRobotGPIORegisterValue(test.output)
			if err != nil || got != test.want {
				t.Fatalf("parsed value = %#x, err=%v, want %#x", got, err, test.want)
			}
		})
	}

	if _, err := parseFingerRobotGPIORegisterValue(nil); err == nil {
		t.Fatal("empty devmem output was accepted")
	}
	if _, err := parseFingerRobotGPIORegisterValue([]byte("not-a-register")); err == nil {
		t.Fatal("invalid devmem output was accepted")
	}
}

func TestWriteFingerRobotPWMAllAttemptsEverySynchronizedChannel(t *testing.T) {
	firstErr := errors.New("pwm0 failed")
	secondErr := errors.New("pwm1 failed")
	var calls []string

	err := writeFingerRobotPWMAllWith(
		[]string{"pwm0", "pwm1"},
		"duty_cycle",
		fingerRobotPWMSafeReleaseNS,
		func(path, name, value string) error {
			calls = append(calls, path)
			if name != "duty_cycle" || value != fingerRobotPWMSafeReleaseNS {
				t.Fatalf("unexpected PWM write: path=%s name=%s value=%s", path, name, value)
			}
			if path == "pwm0" {
				return firstErr
			}
			return secondErr
		},
	)

	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("expected joined channel errors, got %v", err)
	}
	if len(calls) != 2 || calls[0] != "pwm0" || calls[1] != "pwm1" {
		t.Fatalf("expected both PWM channels to be attempted, got %v", calls)
	}
}

func TestInitFingerRobotPWMsRetainsPartialPathsForSafetyRelease(t *testing.T) {
	partialInitErr := errors.New("pwm0 enable failed")
	paths, err := initFingerRobotPWMsWith(
		[]string{"0", "1"},
		func() error { return nil },
		func(channel string) (string, error) {
			path := "/sys/class/pwm/pwmchip0/pwm" + channel
			if channel == "0" {
				return path, partialInitErr
			}
			return path, nil
		},
	)
	if !errors.Is(err, partialInitErr) {
		t.Fatalf("expected partial initialization error, got %v", err)
	}
	wantPaths := []string{
		"/sys/class/pwm/pwmchip0/pwm0",
		"/sys/class/pwm/pwmchip0/pwm1",
	}
	if len(paths) != len(wantPaths) || paths[0] != wantPaths[0] || paths[1] != wantPaths[1] {
		t.Fatalf("partial initialization lost a known PWM path: got %v want %v", paths, wantPaths)
	}

	releaseErr := errors.New("pwm0 safe release write failed")
	var releaseCalls []string
	err = writeFingerRobotPWMAllWith(
		paths,
		"duty_cycle",
		fingerRobotPWMSafeReleaseNS,
		func(path, name, value string) error {
			releaseCalls = append(releaseCalls, path)
			if name != "duty_cycle" || value != fingerRobotPWMSafeReleaseNS {
				t.Fatalf("unexpected safety release write: path=%s name=%s value=%s", path, name, value)
			}
			if path == wantPaths[0] {
				return releaseErr
			}
			return nil
		},
	)
	if !errors.Is(err, releaseErr) {
		t.Fatalf("expected safety release write error, got %v", err)
	}
	if len(releaseCalls) != 2 || releaseCalls[0] != wantPaths[0] || releaseCalls[1] != wantPaths[1] {
		t.Fatalf("safety release did not attempt both known paths: %v", releaseCalls)
	}
}

func TestInitFingerRobotPWMsDoesNotInitializeOutputAfterPinMuxFailure(t *testing.T) {
	wantErr := errors.New("pin mux failed")
	initializeCalls := 0
	paths, err := initFingerRobotPWMsWith(
		[]string{fingerRobotPWMChannel0},
		func() error { return wantErr },
		func(string) (string, error) {
			initializeCalls++
			return "pwm0", nil
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("initialization error = %v, want %v", err, wantErr)
	}
	if initializeCalls != 0 || len(paths) != 0 {
		t.Fatalf("PWM output initialized after pin mux failure: calls=%d paths=%v", initializeCalls, paths)
	}
}
