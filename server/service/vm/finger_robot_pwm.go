package vm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	fingerRobotPWMOutputMuxAddr    = "0x02304084"
	fingerRobotPWMOutputMuxValue   = "0x00030003"
	fingerRobotPWMCaptureMuxAddr   = "0x02304090"
	fingerRobotPWMCaptureMuxValue  = "0x00060013"
	fingerRobotPWMOutputGPIOAddr   = "0x0480100C"
	fingerRobotPWMCaptureGPIOAddr  = "0x04801010"
	fingerRobotPWMShutdownMuxValue = "0x00060043"
	fingerRobotGPIOConfigMask      = uint32(0xFFFFFFFC)

	fingerRobotPWMChipPath      = "/sys/class/pwm/pwmchip0"
	fingerRobotPWMChannel0      = "0"
	fingerRobotPWMPeriodNS      = "20000000"
	fingerRobotPWMSafeReleaseNS = "2050000"

	fingerRobotPWMMuxCommandTimeout = 2 * time.Second
)

type fingerRobotPWMPinMuxSetting struct {
	addr  string
	value string
}

var (
	fingerRobotPWMChannels       = []string{fingerRobotPWMChannel0}
	fingerRobotPWMPinMuxSettings = []fingerRobotPWMPinMuxSetting{
		// Configure the second, electrically connected pad as a high-impedance
		// GPIO input before configuring the only remaining PWM output.
		{addr: fingerRobotPWMCaptureMuxAddr, value: fingerRobotPWMCaptureMuxValue},
		{addr: fingerRobotPWMOutputMuxAddr, value: fingerRobotPWMOutputMuxValue},
	}
	fingerRobotPWMShutdownGPIOAddrs = []string{
		fingerRobotPWMOutputGPIOAddr,
		fingerRobotPWMCaptureGPIOAddr,
	}
	fingerRobotPWMShutdownMuxSettings = []fingerRobotPWMPinMuxSetting{
		{addr: fingerRobotPWMOutputMuxAddr, value: fingerRobotPWMShutdownMuxValue},
		{addr: fingerRobotPWMCaptureMuxAddr, value: fingerRobotPWMShutdownMuxValue},
	}
)

func initFingerRobotPWMs() ([]string, error) {
	return initFingerRobotPWMsWith(fingerRobotPWMChannels, configureFingerRobotPWMMux, initFingerRobotPWMChannel)
}

func initFingerRobotPWMsWith(
	channels []string,
	configureMux func() error,
	initializeChannel func(string) (string, error),
) ([]string, error) {
	if err := configureMux(); err != nil {
		return nil, err
	}

	var errs []error
	pwmPaths := make([]string, 0, len(channels))
	for _, channel := range channels {
		pwmPath, err := initializeChannel(channel)
		if pwmPath != "" {
			// Keep every determinable path even when channel setup only partly
			// succeeded. The controller's following safety release must still
			// try to move every known output to the safe release point.
			pwmPaths = append(pwmPaths, pwmPath)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("initialize pwm%s: %w", channel, err))
		}
	}

	return pwmPaths, errors.Join(errs...)
}

func initFingerRobotPWMChannel(channel string) (string, error) {
	pwmPath := filepath.Join(fingerRobotPWMChipPath, "pwm"+channel)
	if _, err := os.Stat(pwmPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return pwmPath, err
		}

		if err := os.WriteFile(filepath.Join(fingerRobotPWMChipPath, "export"), []byte(channel), 0o666); err != nil {
			return pwmPath, fmt.Errorf("export pwm%s: %w", channel, err)
		}

		if err := waitForFingerRobotPWM(pwmPath); err != nil {
			return pwmPath, err
		}
	}

	// A freshly exported AX630C PWM has period=0. Its driver may reject an
	// initial disable with EINVAL until period/duty have been configured, so
	// this best-effort write must not abort first-time initialization.
	_ = writeFingerRobotPWM(pwmPath, "enable", "0")
	if err := writeFingerRobotPWM(pwmPath, "period", fingerRobotPWMPeriodNS); err != nil {
		return pwmPath, err
	}
	if err := writeFingerRobotPWM(pwmPath, "duty_cycle", fingerRobotPWMSafeReleaseNS); err != nil {
		return pwmPath, err
	}
	if err := writeFingerRobotPWM(pwmPath, "enable", "1"); err != nil {
		return pwmPath, err
	}

	return pwmPath, nil
}

func configureFingerRobotPWMMux() error {
	return configureFingerRobotPWMMuxWith(fingerRobotPWMPinMuxSettings, writeFingerRobotPWMPinMux)
}

func configureFingerRobotPWMMuxWith(settings []fingerRobotPWMPinMuxSetting, writer func(string, string) error) error {
	for _, setting := range settings {
		if err := writer(setting.addr, setting.value); err != nil {
			return err
		}
	}
	return nil
}

// restoreFingerRobotPWMShutdownPins prepares both connected pads as
// low-latched GPIO inputs with their internal pull-down enabled. The GPIO
// registers are read-modify-written so unrelated bits are preserved.
func restoreFingerRobotPWMShutdownPins() error {
	return restoreFingerRobotPWMShutdownPinsWithWait(
		fingerRobotPWMShutdownGPIOAddrs,
		readFingerRobotGPIORegister,
		writeFingerRobotGPIORegister,
		fingerRobotPWMShutdownMuxSettings,
		writeFingerRobotPWMPinMux,
		nil,
	)
}

func restoreFingerRobotPWMShutdownPinsWith(
	gpioAddrs []string,
	readGPIO func(string) (uint32, error),
	writeGPIO func(string, uint32) error,
	muxSettings []fingerRobotPWMPinMuxSetting,
	writeMux func(string, string) error,
) error {
	return restoreFingerRobotPWMShutdownPinsWithWait(
		gpioAddrs,
		readGPIO,
		writeGPIO,
		muxSettings,
		writeMux,
		nil,
	)
}

func restoreFingerRobotPWMShutdownPinsWithWait(
	gpioAddrs []string,
	readGPIO func(string) (uint32, error),
	writeGPIO func(string, uint32) error,
	muxSettings []fingerRobotPWMPinMuxSetting,
	writeMux func(string, string) error,
	waitForLow func() error,
) error {
	var errs []error
	type gpioWrite struct {
		addr  string
		value uint32
	}
	gpioWrites := make([]gpioWrite, 0, len(gpioAddrs))

	// Read both registers before changing either one, matching the board-level
	// read/modify/write sequence used during manual recovery.
	for _, addr := range gpioAddrs {
		value, err := readGPIO(addr)
		if err != nil {
			errs = append(errs, fmt.Errorf("read finger robot GPIO register %s: %w", addr, err))
			continue
		}
		gpioWrites = append(gpioWrites, gpioWrite{addr: addr, value: value & fingerRobotGPIOConfigMask})
	}

	for _, gpio := range gpioWrites {
		if err := writeGPIO(gpio.addr, gpio.value); err != nil {
			errs = append(errs, fmt.Errorf("set finger robot GPIO register %s to input-low: %w", gpio.addr, err))
		}
	}

	// The caller has already written the safe duty cycle. Wait for a fresh
	// falling edge after that write so the following mux transition cannot cut
	// the safe pulse short.
	if waitForLow != nil {
		if err := waitForLow(); err != nil {
			errs = append(errs, fmt.Errorf("wait for final safe PWM falling edge: %w", err))
		}
	}

	// Configure the pinmux only after the GPIO direction/latch state is ready.
	// Continue through all settings so one failed devmem operation does not
	// leave the other pad in its active PWM function.
	for _, setting := range muxSettings {
		if err := writeMux(setting.addr, setting.value); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func readFingerRobotGPIORegister(addr string) (uint32, error) {
	output, err := runFingerRobotDevMem(addr, "32")
	if err != nil {
		return 0, fmt.Errorf("read GPIO register %s: %w: %s", addr, err, output)
	}

	value, err := parseFingerRobotGPIORegisterValue(output)
	if err != nil {
		return 0, fmt.Errorf("read GPIO register %s: %w", addr, err)
	}
	return value, nil
}

func parseFingerRobotGPIORegisterValue(output []byte) (uint32, error) {
	fields := strings.Fields(string(output))
	if len(fields) == 0 {
		return 0, errors.New("returned empty output")
	}
	token := fields[len(fields)-1]
	digits := token
	base := 10
	if strings.HasPrefix(token, "0x") || strings.HasPrefix(token, "0X") {
		digits = token[2:]
		base = 16
	} else if (len(token) > 1 && token[0] == '0') || strings.ContainsAny(token, "abcdefABCDEF") {
		// Some devmem builds omit the 0x prefix but keep zero-padded hex.
		base = 16
	}
	value, parseErr := strconv.ParseUint(digits, base, 32)
	if parseErr != nil {
		return 0, fmt.Errorf("parse value %q: %w", token, parseErr)
	}
	return uint32(value), nil
}

func writeFingerRobotGPIORegister(addr string, value uint32) error {
	output, err := runFingerRobotDevMem(addr, "32", fmt.Sprintf("0x%08x", value))
	if err != nil {
		return fmt.Errorf("write GPIO register %s: %w: %s", addr, err, output)
	}
	return nil
}

func runFingerRobotDevMem(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fingerRobotPWMMuxCommandTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, "devmem", args...).CombinedOutput()
	if err == nil {
		return output, nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = fmt.Errorf("command timed out after %s: %w", fingerRobotPWMMuxCommandTimeout, err)
	}
	return output, err
}

func writeFingerRobotPWMPinMux(addr, value string) error {
	output, err := runFingerRobotDevMem(addr, "32", value)
	if err != nil {
		return fmt.Errorf("set pin mux %s to %s: %w: %s", addr, value, err, output)
	}
	return nil
}

func waitForFingerRobotPWM(pwmPath string) error {
	for i := 0; i < 20; i++ {
		if _, err := os.Stat(pwmPath); err == nil {
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}

	return fmt.Errorf("pwm path %s was not created", pwmPath)
}

func writeFingerRobotPWM(pwmPath string, name string, value string) error {
	path := filepath.Join(pwmPath, name)
	if err := os.WriteFile(path, []byte(value), 0o666); err != nil {
		log.Errorf("write pwm %s failed: %s", path, err)
		return err
	}

	return nil
}

func writeFingerRobotPWMAll(pwmPaths []string, name string, value string) error {
	return writeFingerRobotPWMAllWith(pwmPaths, name, value, writeFingerRobotPWM)
}

func writeFingerRobotPWMAllWith(pwmPaths []string, name string, value string, writer func(string, string, string) error) error {
	var errs []error
	for _, pwmPath := range pwmPaths {
		if err := writer(pwmPath, name, value); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// forceExistingFingerRobotSafeRelease is the only hardware action permitted
// after realtime thread setup fails. It never exports or initializes PWM and
// never changes pinmux; it only clears a duty retained by the kernel from an
// earlier process if that channel already exists.
func forceExistingFingerRobotSafeRelease() error {
	pwmPath := filepath.Join(fingerRobotPWMChipPath, "pwm"+fingerRobotPWMChannel0)
	if _, err := os.Stat(pwmPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect existing finger robot PWM for emergency release: %w", err)
	}
	if err := writeFingerRobotPWM(pwmPath, "duty_cycle", fingerRobotPWMSafeReleaseNS); err != nil {
		return fmt.Errorf("force existing finger robot PWM to safe release: %w", err)
	}
	return nil
}
