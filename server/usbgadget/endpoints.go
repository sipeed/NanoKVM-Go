// Package usbgadget describes the Go+ USB device controller's IN endpoint budget.
package usbgadget

import (
	"fmt"
	"os"
	"strings"
)

const INEndpointLimit = 7
const ConfigDir = "/sys/kernel/config/usb_gadget/g0/configs/c.1"

// INEndpoints counts linked functions, including any additional proposed links.
func INEndpoints(configDir string, additional ...string) (int, error) {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return 0, err
	}
	links := make(map[string]bool)
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			links[entry.Name()] = true
		}
	}
	for _, link := range additional {
		links[link] = true
	}
	used := 0
	for link := range links {
		kind, _, _ := strings.Cut(link, ".")
		switch kind {
		case "hid", "uac2", "mass_storage", "Loopback":
			used++
		case "uvc", "rndis", "ncm", "acm":
			used += 2
		default:
			return 0, fmt.Errorf("unknown USB function %s", link)
		}
	}
	return used, nil
}
