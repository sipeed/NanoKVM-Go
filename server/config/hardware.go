package config

import (
	"os/exec"
	"regexp"
	"strings"
)

type HWVersion int

const (
	HWVersionUnknown HWVersion = iota
)

var defaultHardware = Hardware{
	Version: HWVersionUnknown,
}

const unknownDeviceNumber = "unknown"

// Device-number data may use either "[A]number" or
// "NanoKVM-Go-Plus[A]number". Normalize those known formats without treating
// brackets inside any other raw device number as metadata.
var deviceNumberPrefix = regexp.MustCompile(`^(?:NanoKVM-Go(?:-Plus)?)?\[[[:alpha:]]+\][[:space:]]*`)

func (h HWVersion) String() string {
	return "Unknown"
}

func DeviceNumber() string {
	output, err := exec.Command("ax_efuse_tool", "--str-get").Output()
	if err != nil {
		return unknownDeviceNumber
	}

	return parseDeviceNumber(string(output))
}

func parseDeviceNumber(output string) string {
	content := strings.TrimSpace(output)
	if content == "" {
		return unknownDeviceNumber
	}

	if prefix := deviceNumberPrefix.FindStringIndex(content); prefix != nil {
		content = strings.TrimSpace(content[prefix[1]:])
	}

	if content == "" {
		return unknownDeviceNumber
	}

	return content
}

func GetHwVersion() HWVersion {
	return HWVersionUnknown
}

func getHardware() (h Hardware) {
	return defaultHardware
}
