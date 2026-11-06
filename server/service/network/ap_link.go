package network

import (
	"bufio"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// APPortalHeader must accompany AP link authorization. A custom header turns
// a cross-site request into a CORS preflight, which the production server
// never approves, so another page cannot forge a provisioning request.
const APPortalHeader = "X-NanoKVM-AP-Portal"

const procNetARP = "/proc/net/arp"

var (
	apGatewayAddr = netip.MustParseAddr("192.168.12.1")
	apSubnet      = netip.MustParsePrefix("192.168.12.0/24")

	apLinkNeighborDevice = readARPDevice
	isWirelessInterface  = func(name string) bool {
		_, err := os.Stat(filepath.Join("/sys/class/net", name, "wireless"))
		return err == nil
	}
)

// hasAPPortalAccess accepts either the AP password or proof that the client is
// associated with the WPA2 AP. Associating already requires the AP password,
// so both carry the same authority.
func hasAPPortalAccess(c *gin.Context) bool {
	return hasValidAPKey(c) || isAPLinkRequest(c.Request)
}

// isAPLinkRequest reports whether the request came from a station on the AP.
// The destination and source addresses alone are not enough because Linux
// accepts packets for any local address on any interface, so the peer must
// also be a neighbor on a wireless interface. The Host must name the gateway
// literally: a DNS-rebound page on a phone joined to the AP would otherwise be
// same-origin with the device and could send the portal header freely.
func isAPLinkRequest(req *http.Request) bool {
	if req.Header.Get(APPortalHeader) != "1" || !isAPGatewayHost(req.Host) {
		return false
	}

	localAddr, ok := req.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok || localAddr == nil {
		return false
	}
	local, err := netip.ParseAddrPort(localAddr.String())
	if err != nil || local.Addr().Unmap() != apGatewayAddr {
		return false
	}

	remote, err := netip.ParseAddrPort(req.RemoteAddr)
	if err != nil {
		return false
	}
	remoteAddr := remote.Addr().Unmap()
	if !apSubnet.Contains(remoteAddr) || remoteAddr == apGatewayAddr {
		return false
	}

	device, ok := apLinkNeighborDevice(remoteAddr)
	return ok && isWirelessInterface(device)
}

func isAPGatewayHost(host string) bool {
	if h, port, err := net.SplitHostPort(host); err == nil {
		if port != "80" && port != "443" {
			return false
		}
		host = h
	}
	addr, err := netip.ParseAddr(host)
	return err == nil && addr == apGatewayAddr
}

func readARPDevice(addr netip.Addr) (string, bool) {
	file, err := os.Open(procNetARP)
	if err != nil {
		return "", false
	}
	defer file.Close()

	return findARPDevice(bufio.NewScanner(file), addr)
}

// findARPDevice parses /proc/net/arp:
// IP address  HW type  Flags  HW address  Mask  Device
func findARPDevice(scanner *bufio.Scanner, addr netip.Addr) (string, bool) {
	const flagComplete = 0x2

	scanner.Scan() // header
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 6 {
			continue
		}
		ip, err := netip.ParseAddr(fields[0])
		if err != nil || ip != addr {
			continue
		}
		flags, err := strconv.ParseUint(strings.TrimPrefix(fields[2], "0x"), 16, 32)
		if err != nil || flags&flagComplete == 0 {
			return "", false
		}
		return fields[5], true
	}
	return "", false
}
