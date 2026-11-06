package vm

import (
	"fmt"
	"net/netip"
	"os"
	"strings"

	"NanoKVM-Server/proto"
)

const (
	usbNetworkModeServer              = "server"
	usbNetworkModeClient              = "client"
	usbNetworkDefaultForwardInterface = "wlan0"
)

type usbNetworkPaths struct {
	dhcp    string
	ipv4    string
	forward string
}

type usbNetworkFileSnapshot struct {
	path    string
	exists  bool
	content []byte
	mode    os.FileMode
}

var (
	usbNetworkConfigPaths = usbNetworkPaths{
		dhcp:    "/boot/ncm.dhcp",
		ipv4:    "/boot/ncm.ipv4",
		forward: "/boot/ncm.forward",
	}
)

func normalizeUsbNetworkConfig(req proto.SetUsbNetworkReq) (*proto.GetUsbNetworkRsp, error) {
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode != usbNetworkModeServer && mode != usbNetworkModeClient {
		return nil, fmt.Errorf("invalid USB network mode")
	}

	ipv4 := strings.TrimSpace(req.IPv4)
	if mode == usbNetworkModeClient {
		if ipv4 != "" || req.Forwarding {
			return nil, fmt.Errorf("client mode does not support server configuration")
		}
		return &proto.GetUsbNetworkRsp{Mode: mode}, nil
	}

	if ipv4 != "" {
		prefix, err := netip.ParsePrefix(ipv4)
		if err != nil || !prefix.Addr().Is4() || !prefix.Addr().IsGlobalUnicast() || prefix.Addr().IsLoopback() || prefix.Addr().IsLinkLocalUnicast() || prefix.Bits() < 1 || prefix.Bits() > 30 {
			return nil, fmt.Errorf("invalid USB network IPv4 address")
		}
		ipv4 = prefix.Addr().String() + "/" + fmt.Sprint(prefix.Bits())
	}

	return &proto.GetUsbNetworkRsp{
		Mode:       mode,
		IPv4:       ipv4,
		Forwarding: req.Forwarding,
	}, nil
}

func readUsbNetworkConfig(paths usbNetworkPaths) (*proto.GetUsbNetworkRsp, error) {
	mode := usbNetworkModeServer
	if exists, err := usbNetworkFileExists(paths.dhcp); err != nil {
		return nil, err
	} else if exists {
		mode = usbNetworkModeClient
	}

	ipv4, err := readOptionalUsbNetworkFile(paths.ipv4)
	if err != nil {
		return nil, err
	}
	uplink, err := readOptionalUsbNetworkFile(paths.forward)
	if err != nil {
		return nil, err
	}
	return &proto.GetUsbNetworkRsp{
		Mode:       mode,
		IPv4:       ipv4,
		Forwarding: mode == usbNetworkModeServer && uplink != "",
	}, nil
}

func writeUsbNetworkConfig(paths usbNetworkPaths, config *proto.GetUsbNetworkRsp) error {
	if config.Mode == usbNetworkModeClient {
		if err := os.WriteFile(paths.dhcp, nil, 0644); err != nil {
			return err
		}
	} else if err := removeUsbNetworkFile(paths.dhcp); err != nil {
		return err
	}

	if err := writeOptionalUsbNetworkFile(paths.ipv4, config.IPv4); err != nil {
		return err
	}

	uplink := ""
	if config.Mode == usbNetworkModeServer && config.Forwarding {
		uplink = usbNetworkDefaultForwardInterface
	}
	return writeOptionalUsbNetworkFile(paths.forward, uplink)
}

func snapshotUsbNetworkFiles(paths usbNetworkPaths) ([]usbNetworkFileSnapshot, error) {
	result := make([]usbNetworkFileSnapshot, 0, 3)
	for _, path := range []string{paths.dhcp, paths.ipv4, paths.forward} {
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			result = append(result, usbNetworkFileSnapshot{path: path})
			continue
		}
		if err != nil {
			return nil, err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		result = append(result, usbNetworkFileSnapshot{
			path:    path,
			exists:  true,
			content: content,
			mode:    info.Mode().Perm(),
		})
	}
	return result, nil
}

func isUsbNetworkConfigCurrent(paths usbNetworkPaths, config *proto.GetUsbNetworkRsp) (bool, error) {
	current, err := readUsbNetworkConfig(paths)
	if err != nil {
		return false, err
	}
	if current.Mode != config.Mode {
		return false, nil
	}

	ipv4Matches, err := usbNetworkOptionalFileMatches(paths.ipv4, config.IPv4)
	if err != nil {
		return false, err
	}
	if !ipv4Matches {
		return false, nil
	}

	expectedForwardInterface := ""
	if config.Mode == usbNetworkModeServer && config.Forwarding {
		expectedForwardInterface = usbNetworkDefaultForwardInterface
	}
	return usbNetworkOptionalFileMatches(paths.forward, expectedForwardInterface)
}

func restoreUsbNetworkFiles(snapshot []usbNetworkFileSnapshot) error {
	var firstErr error
	for _, file := range snapshot {
		var err error
		if file.exists {
			err = os.WriteFile(file.path, file.content, file.mode)
		} else {
			err = removeUsbNetworkFile(file.path)
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func readOptionalUsbNetworkFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func writeOptionalUsbNetworkFile(path, value string) error {
	if value == "" {
		return removeUsbNetworkFile(path)
	}
	return os.WriteFile(path, []byte(value+"\n"), 0644)
}

func usbNetworkOptionalFileMatches(path, expected string) (bool, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return expected == "", nil
	}
	if err != nil {
		return false, err
	}
	if expected == "" {
		return false, nil
	}
	return strings.TrimSpace(string(content)) == expected, nil
}

func removeUsbNetworkFile(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func usbNetworkFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
