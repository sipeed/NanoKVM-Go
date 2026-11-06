package tailscale

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	routingStateFile        = "/etc/kvm/tailscale-routing.json"
	routingSysctlConfigFile = "/etc/sysctl.d/99-nanokvm-tailscale-routing.conf"
	procNetRouteFile        = "/proc/net/route"
	procNetIPv6RouteFile    = "/proc/net/ipv6_route"

	procIPv4Forward       = "/proc/sys/net/ipv4/ip_forward"
	procIPv6Forward       = "/proc/sys/net/ipv6/conf/all/forwarding"
	procIPv6DefaultAccept = "/proc/sys/net/ipv6/conf/default/accept_ra"
)

var routingInterfaceAddrs = func(name string) ([]net.Addr, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	return iface.Addrs()
}

var (
	applyExitNodeSetting    = setExitNode
	applySubnetRouteSetting = setSubnetRoute
)

type routingSnapshot struct {
	exitNode     bool
	subnetRoutes []string
}

type routingView struct {
	available bool
	exitNode  bool
	subnet    bool
	lanSubnet string
}

type managedRoutingState struct {
	Subnet string `json:"subnet,omitempty"`
}

type defaultIPv4Route struct {
	Interface string
	Gateway   netip.Addr
	Metric    int
}

type defaultIPv6Route struct {
	Interface string
	Metric    uint64
}

type routingPrefsReader interface {
	Prefs() (*TsPrefs, error)
}

type routingClient interface {
	routingPrefsReader
	SetRouting(exitNode bool, routes []string) error
}

type routingForwardingSync func(snapshot routingSnapshot) error

func (s *Service) SetExitNode(c *gin.Context) {
	req, ok := bindTailscaleRoutingRequest(c)
	if !ok {
		var rsp proto.Response
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	var rsp proto.Response
	release, ok := beginTailscaleMutation(c, &rsp, false)
	if !ok {
		return
	}
	defer release()

	if !requireTailscaleChallengeValue(
		c,
		&rsp,
		authservice.SensitiveActionTailscaleSetExitNode,
		req.Challenge,
	) {
		return
	}

	if err := applyExitNodeSetting(c.Request.Context(), *req.Enabled); err != nil {
		log.Errorf("failed to set tailscale exit node to %v: %v", *req.Enabled, err)
		rsp.ErrRsp(c, -1, "update exit node failed")
		return
	}

	rsp.OkRsp(c)
}

func (s *Service) SetSubnetRoute(c *gin.Context) {
	req, ok := bindTailscaleRoutingRequest(c)
	if !ok {
		var rsp proto.Response
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	var rsp proto.Response
	release, ok := beginTailscaleMutation(c, &rsp, false)
	if !ok {
		return
	}
	defer release()

	if !requireTailscaleChallengeValue(
		c,
		&rsp,
		authservice.SensitiveActionTailscaleSetSubnetRoute,
		req.Challenge,
	) {
		return
	}

	if err := applySubnetRouteSetting(c.Request.Context(), *req.Enabled); err != nil {
		log.Errorf("failed to set tailscale subnet route to %v: %v", *req.Enabled, err)
		rsp.ErrRsp(c, -1, "update subnet route failed")
		return
	}

	rsp.OkRsp(c)
}

func bindTailscaleRoutingRequest(c *gin.Context) (*proto.SetTailscaleRoutingReq, bool) {
	if c == nil || c.Request == nil {
		return nil, false
	}

	var req proto.SetTailscaleRoutingReq
	if err := c.ShouldBind(&req); err != nil || req.Enabled == nil {
		return nil, false
	}
	return &req, true
}

func setExitNode(ctx context.Context, enabled bool) error {
	operation := NewLocalClientWithContext(ctx)
	recovery := NewLocalClient()
	previous, err := readRoutingSnapshot(operation)
	if err != nil {
		return err
	}

	desired := routingSnapshot{
		exitNode:     enabled,
		subnetRoutes: append([]string(nil), previous.subnetRoutes...),
	}
	changed, forwardingReady, err := applyRoutingPreferences(
		operation,
		recovery,
		previous,
		desired,
		syncRoutingForwarding,
	)
	if err != nil {
		return err
	}
	if !forwardingReady {
		if err := syncRoutingForwarding(desired); err != nil {
			return rollbackRouting(err, recovery, previous, desired, changed, syncRoutingForwarding)
		}
	}
	return nil
}

func setSubnetRoute(ctx context.Context, enabled bool) error {
	operation := NewLocalClientWithContext(ctx)
	recovery := NewLocalClient()
	previous, err := readRoutingSnapshot(operation)
	if err != nil {
		return err
	}

	managedSubnet, err := loadManagedSubnet(routingStateFile)
	if err != nil {
		return err
	}
	candidate, _, candidateErr := detectLANSubnet()

	desiredRoutes := append([]string(nil), previous.subnetRoutes...)
	if enabled {
		if candidateErr != nil || candidate == "" {
			if candidateErr == nil {
				candidateErr = errors.New("no safe LAN subnet detected")
			}
			return candidateErr
		}

		if managedSubnet != "" && managedSubnet != candidate {
			desiredRoutes = removeRoute(desiredRoutes, managedSubnet)
		}
		desiredRoutes = addRoute(desiredRoutes, candidate)
	} else {
		toRemove := managedSubnet
		if toRemove == "" && candidateErr == nil && containsRoute(desiredRoutes, candidate) {
			toRemove = candidate
		}
		if toRemove != "" {
			desiredRoutes = removeRoute(desiredRoutes, toRemove)
		}
	}

	desired := routingSnapshot{
		exitNode:     previous.exitNode,
		subnetRoutes: desiredRoutes,
	}
	changed, forwardingReady, err := applyRoutingPreferences(
		operation,
		recovery,
		previous,
		desired,
		syncRoutingForwarding,
	)
	if err != nil {
		return err
	}
	if !forwardingReady {
		if err := syncRoutingForwarding(desired); err != nil {
			return rollbackRouting(err, recovery, previous, desired, changed, syncRoutingForwarding)
		}
	}

	stateSubnet := ""
	if enabled {
		stateSubnet = candidate
	}
	if err := saveManagedSubnet(routingStateFile, stateSubnet); err != nil {
		return rollbackRouting(err, recovery, previous, desired, changed, syncRoutingForwarding)
	}
	return nil
}

func prepareRoutingForUninstall() error {
	operation := NewLocalClient()
	recovery := NewLocalClient()
	previous, err := readRoutingSnapshot(operation)
	if err != nil {
		return fmt.Errorf("read routing preferences before uninstall: %w", err)
	}
	managedSubnet, err := loadManagedSubnet(routingStateFile)
	if err != nil {
		return fmt.Errorf("read managed subnet before uninstall: %w", err)
	}

	desired := routingSnapshotForUninstall(previous, managedSubnet)

	changed, forwardingReady, err := applyRoutingPreferences(
		operation,
		recovery,
		previous,
		desired,
		syncRoutingForwarding,
	)
	if err != nil {
		return err
	}
	if !forwardingReady {
		if err := syncRoutingForwarding(desired); err != nil {
			return rollbackRouting(err, recovery, previous, desired, changed, syncRoutingForwarding)
		}
	}
	if err := saveManagedSubnet(routingStateFile, ""); err != nil {
		return rollbackRouting(err, recovery, previous, desired, changed, syncRoutingForwarding)
	}
	return nil
}

func routingSnapshotForUninstall(previous routingSnapshot, managedSubnet string) routingSnapshot {
	desired := routingSnapshot{
		exitNode:     false,
		subnetRoutes: append([]string(nil), previous.subnetRoutes...),
	}
	if managedSubnet != "" {
		desired.subnetRoutes = removeRoute(desired.subnetRoutes, managedSubnet)
	}
	return desired
}

func applyRoutingPreferences(
	operation routingClient,
	recovery routingClient,
	previous routingSnapshot,
	desired routingSnapshot,
	syncForwarding routingForwardingSync,
) (changed bool, forwardingReady bool, err error) {
	changed = !sameRoutingSnapshots(previous, desired)
	forwardingReady = routingExpands(previous, desired)
	if forwardingReady {
		if err := syncForwarding(desired); err != nil {
			return false, false, rollbackRouting(err, recovery, previous, desired, false, syncForwarding)
		}
	}
	if !changed {
		return false, forwardingReady, nil
	}

	if err := operation.SetRouting(desired.exitNode, desired.subnetRoutes); err != nil {
		return true, forwardingReady, rollbackRouting(err, recovery, previous, desired, true, syncForwarding)
	}
	updated, err := readRoutingSnapshot(operation)
	if err != nil {
		return true, forwardingReady, rollbackRouting(err, recovery, previous, desired, true, syncForwarding)
	}
	if !sameRoutingSnapshots(updated, desired) {
		err := fmt.Errorf("tailscale routing preference mismatch: got %+v, want %+v", updated, desired)
		return true, forwardingReady, rollbackRouting(err, recovery, previous, desired, true, syncForwarding)
	}
	return true, forwardingReady, nil
}

func rollbackRouting(
	cause error,
	recovery routingClient,
	previous routingSnapshot,
	desired routingSnapshot,
	prefsMayHaveChanged bool,
	syncForwarding routingForwardingSync,
) error {
	errs := []error{cause}
	if prefsMayHaveChanged {
		if err := recovery.SetRouting(previous.exitNode, previous.subnetRoutes); err != nil {
			errs = append(errs, fmt.Errorf("rollback tailscale routing preferences: %w", err))
		}
	}

	actual, err := readRoutingSnapshot(recovery)
	if err != nil {
		errs = append(errs, fmt.Errorf("read tailscale preferences after rollback: %w", err))
		actual = conservativeRoutingSnapshot(previous, desired)
	}
	if err := syncForwarding(actual); err != nil {
		errs = append(errs, fmt.Errorf("reconcile routing forwarding after rollback: %w", err))
	}
	return errors.Join(errs...)
}

func readRoutingSnapshot(reader routingPrefsReader) (routingSnapshot, error) {
	prefs, err := reader.Prefs()
	if err != nil {
		return routingSnapshot{}, err
	}
	return snapshotFromPrefs(prefs)
}

func sameRoutingSnapshots(left, right routingSnapshot) bool {
	return left.exitNode == right.exitNode && sameRoutes(left.subnetRoutes, right.subnetRoutes)
}

func routingExpands(previous, desired routingSnapshot) bool {
	if desired.exitNode && !previous.exitNode {
		return true
	}
	for _, route := range desired.subnetRoutes {
		if !containsRoute(previous.subnetRoutes, route) {
			return true
		}
	}
	return false
}

func conservativeRoutingSnapshot(left, right routingSnapshot) routingSnapshot {
	routes := append([]string(nil), left.subnetRoutes...)
	for _, route := range right.subnetRoutes {
		routes = addRoute(routes, route)
	}
	return routingSnapshot{
		exitNode:     left.exitNode || right.exitNode,
		subnetRoutes: routes,
	}
}

func readRoutingView(client *LocalClient) (routingView, error) {
	prefs, err := client.Prefs()
	if err != nil {
		return routingView{}, err
	}
	snapshot, err := snapshotFromPrefs(prefs)
	if err != nil {
		return routingView{}, err
	}

	managedSubnet, err := loadManagedSubnet(routingStateFile)
	if err != nil {
		return routingView{}, err
	}
	candidate, _, candidateErr := detectLANSubnet()

	view := routingView{
		available: true,
		exitNode:  snapshot.exitNode,
	}
	if managedSubnet != "" && containsRoute(snapshot.subnetRoutes, managedSubnet) {
		view.subnet = true
		view.lanSubnet = managedSubnet
		return view, nil
	}
	if candidateErr == nil {
		view.lanSubnet = candidate
		view.subnet = containsRoute(snapshot.subnetRoutes, candidate)
	}
	return view, nil
}

func snapshotFromPrefs(prefs *TsPrefs) (routingSnapshot, error) {
	if prefs == nil {
		return routingSnapshot{}, errors.New("tailscale prefs are nil")
	}

	snapshot := routingSnapshot{}
	seen := make(map[string]struct{}, len(prefs.AdvertiseRoutes))
	for _, raw := range prefs.AdvertiseRoutes {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(raw))
		if err != nil || prefix != prefix.Masked() {
			return routingSnapshot{}, fmt.Errorf("invalid tailscale advertised route %q", raw)
		}

		route := prefix.String()
		if prefix.Bits() == 0 {
			snapshot.exitNode = true
			continue
		}
		if _, ok := seen[route]; ok {
			continue
		}
		seen[route] = struct{}{}
		snapshot.subnetRoutes = append(snapshot.subnetRoutes, route)
	}
	sort.Strings(snapshot.subnetRoutes)
	return snapshot, nil
}

func addRoute(routes []string, route string) []string {
	if containsRoute(routes, route) {
		return append([]string(nil), routes...)
	}
	result := append(append([]string(nil), routes...), route)
	sort.Strings(result)
	return result
}

func removeRoute(routes []string, route string) []string {
	result := make([]string, 0, len(routes))
	for _, existing := range routes {
		if existing != route {
			result = append(result, existing)
		}
	}
	sort.Strings(result)
	return result
}

func containsRoute(routes []string, route string) bool {
	for _, existing := range routes {
		if existing == route {
			return true
		}
	}
	return false
}

func sameRoutes(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for index := range leftCopy {
		if leftCopy[index] != rightCopy[index] {
			return false
		}
	}
	return true
}

func detectLANSubnet() (string, string, error) {
	file, err := os.Open(procNetRouteFile)
	if err != nil {
		return "", "", err
	}
	defer func() {
		_ = file.Close()
	}()

	routes, err := parseDefaultIPv4Routes(file)
	if err != nil {
		return "", "", err
	}
	route, err := selectDefaultIPv4Route(routes)
	if err != nil {
		return "", "", err
	}
	if !isSafeUplinkInterface(route.Interface) {
		return "", "", fmt.Errorf("unsupported default-route interface %q", route.Interface)
	}

	addresses, err := routingInterfaceAddrs(route.Interface)
	if err != nil {
		return "", "", err
	}
	prefix, err := selectPrivateLANPrefix(addresses, route.Gateway)
	if err != nil {
		return "", "", err
	}
	return prefix.String(), route.Interface, nil
}

func detectDefaultUplink() (string, error) {
	file, err := os.Open(procNetRouteFile)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	routes, err := parseDefaultIPv4Routes(file)
	if err != nil {
		return "", err
	}
	route, err := selectDefaultIPv4Route(routes)
	if err != nil {
		return "", err
	}
	if !isSafeUplinkInterface(route.Interface) {
		return "", fmt.Errorf("unsupported default-route interface %q", route.Interface)
	}
	return route.Interface, nil
}

func detectDefaultUplinks() ([]string, error) {
	uplinks := make(map[string]struct{}, 2)
	ipv4Uplink, ipv4Err := detectDefaultUplink()
	if ipv4Err == nil {
		uplinks[ipv4Uplink] = struct{}{}
	}

	file, err := os.Open(procNetIPv6RouteFile)
	if errors.Is(err, os.ErrNotExist) {
		if len(uplinks) == 0 {
			return nil, ipv4Err
		}
		return sortedRoutingUplinks(uplinks), nil
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	routes, err := parseDefaultIPv6Routes(file)
	if err != nil {
		return nil, err
	}
	ipv6Uplinks, err := selectDefaultIPv6Uplinks(routes)
	if err != nil {
		return nil, err
	}
	for _, uplink := range ipv6Uplinks {
		uplinks[uplink] = struct{}{}
	}
	if len(uplinks) == 0 {
		return nil, fmt.Errorf("no safe default uplink: %w", ipv4Err)
	}
	return sortedRoutingUplinks(uplinks), nil
}

func sortedRoutingUplinks(uplinks map[string]struct{}) []string {
	result := make([]string, 0, len(uplinks))
	for uplink := range uplinks {
		result = append(result, uplink)
	}
	sort.Strings(result)
	return result
}

func parseDefaultIPv4Routes(reader io.Reader) ([]defaultIPv4Route, error) {
	scanner := bufio.NewScanner(reader)
	routes := make([]defaultIPv4Route, 0, 2)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 || fields[0] == "Iface" || fields[1] != "00000000" || fields[7] != "00000000" {
			continue
		}

		flags, err := strconv.ParseUint(fields[3], 16, 32)
		if err != nil || flags&0x1 == 0 {
			continue
		}
		metric, err := strconv.Atoi(fields[6])
		if err != nil || metric < 0 {
			continue
		}
		gateway, err := parseProcRouteIPv4(fields[2])
		if err != nil {
			continue
		}

		routes = append(routes, defaultIPv4Route{
			Interface: fields[0],
			Gateway:   gateway,
			Metric:    metric,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(routes) == 0 {
		return nil, errors.New("no IPv4 default route")
	}
	return routes, nil
}

func parseDefaultIPv6Routes(reader io.Reader) ([]defaultIPv6Route, error) {
	const zeroIPv6 = "00000000000000000000000000000000"

	scanner := bufio.NewScanner(reader)
	routes := make([]defaultIPv6Route, 0, 2)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 || !strings.EqualFold(fields[0], zeroIPv6) || fields[1] != "00" {
			continue
		}
		metric, err := strconv.ParseUint(fields[5], 16, 64)
		if err != nil {
			continue
		}
		flags, err := strconv.ParseUint(fields[8], 16, 64)
		if err != nil || flags&0x1 == 0 {
			continue
		}
		routes = append(routes, defaultIPv6Route{
			Interface: fields[9],
			Metric:    metric,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return routes, nil
}

func selectDefaultIPv6Uplinks(routes []defaultIPv6Route) ([]string, error) {
	sorted := append([]defaultIPv6Route(nil), routes...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Metric == sorted[j].Metric {
			return sorted[i].Interface < sorted[j].Interface
		}
		return sorted[i].Metric < sorted[j].Metric
	})

	seen := make(map[string]struct{}, len(sorted))
	uplinks := make([]string, 0, len(sorted))
	for _, route := range sorted {
		if !isSafeUplinkInterface(route.Interface) {
			return nil, fmt.Errorf("unsupported IPv6 default-route interface %q", route.Interface)
		}
		if _, ok := seen[route.Interface]; ok {
			continue
		}
		seen[route.Interface] = struct{}{}
		uplinks = append(uplinks, route.Interface)
	}
	return uplinks, nil
}

func parseProcRouteIPv4(value string) (netip.Addr, error) {
	number, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return netip.Addr{}, err
	}
	return netip.AddrFrom4([4]byte{
		byte(number),
		byte(number >> 8),
		byte(number >> 16),
		byte(number >> 24),
	}), nil
}

func selectDefaultIPv4Route(routes []defaultIPv4Route) (defaultIPv4Route, error) {
	if len(routes) == 0 {
		return defaultIPv4Route{}, errors.New("no IPv4 default route")
	}
	sorted := append([]defaultIPv4Route(nil), routes...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Metric == sorted[j].Metric {
			return sorted[i].Interface < sorted[j].Interface
		}
		return sorted[i].Metric < sorted[j].Metric
	})
	best := sorted[0]
	for _, route := range sorted[1:] {
		if route.Metric != best.Metric {
			break
		}
		if route.Interface != best.Interface || route.Gateway != best.Gateway {
			return defaultIPv4Route{}, errors.New("ambiguous IPv4 default route")
		}
	}
	return best, nil
}

func selectPrivateLANPrefix(addresses []net.Addr, gateway netip.Addr) (netip.Prefix, error) {
	var selected netip.Prefix
	for _, address := range addresses {
		prefix, err := netip.ParsePrefix(address.String())
		if err != nil {
			continue
		}
		prefix = prefix.Masked()
		if !isSafeLANPrefix(prefix) {
			continue
		}
		if gateway.IsValid() && !gateway.IsUnspecified() && !prefix.Contains(gateway) {
			continue
		}
		if selected.IsValid() && selected != prefix {
			return netip.Prefix{}, errors.New("ambiguous private LAN subnet")
		}
		selected = prefix
	}
	if !selected.IsValid() {
		return netip.Prefix{}, errors.New("no private LAN subnet on default interface")
	}
	return selected, nil
}

func isSafeLANPrefix(prefix netip.Prefix) bool {
	return prefix.IsValid() && prefix.Addr().Is4() && prefix.Addr().IsPrivate() &&
		prefix.Bits() > 0 && prefix.Bits() < 32
}

func isSafeUplinkInterface(name string) bool {
	if name == "" || strings.ContainsAny(name, "/\\\x00") {
		return false
	}
	return strings.HasPrefix(name, "eth") || strings.HasPrefix(name, "en") ||
		strings.HasPrefix(name, "wlan") || strings.HasPrefix(name, "wl")
}

func loadManagedSubnet(path string) (string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	var state managedRoutingState
	if err := json.Unmarshal(data, &state); err != nil {
		return "", err
	}
	if state.Subnet == "" {
		return "", nil
	}
	prefix, err := netip.ParsePrefix(state.Subnet)
	if err != nil || prefix != prefix.Masked() || !isSafeLANPrefix(prefix) {
		return "", fmt.Errorf("invalid managed tailscale subnet %q", state.Subnet)
	}
	return prefix.String(), nil
}

func saveManagedSubnet(path, subnet string) error {
	if subnet == "" {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}

	prefix, err := netip.ParsePrefix(subnet)
	if err != nil || prefix != prefix.Masked() || !isSafeLANPrefix(prefix) {
		return fmt.Errorf("invalid managed tailscale subnet %q", subnet)
	}
	data, err := json.Marshal(managedRoutingState{Subnet: prefix.String()})
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomically(path, data, 0o600)
}

func syncRoutingForwarding(snapshot routingSnapshot) error {
	needsIPv4, needsIPv6, err := routingForwardingNeeds(snapshot)
	if err != nil {
		return err
	}
	if !needsIPv4 && !needsIPv6 {
		if err := os.Remove(routingSysctlConfigFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}

	var uplinks []string
	if needsIPv6 {
		uplinks, err = detectDefaultUplinks()
		if err != nil {
			return err
		}
		// Linux ignores normal router advertisements when forwarding is on.
		// Override that behavior before enabling forwarding so the uplink does
		// not lose its IPv6 default route (and Tailscale control connection).
		if err := writeKernelParameter(procIPv6DefaultAccept, 2); err != nil {
			return err
		}
		for _, uplink := range uplinks {
			if err := writeKernelParameter(ipv6AcceptRAPath(uplink), 2); err != nil {
				return err
			}
		}
	}
	if needsIPv4 {
		if err := writeKernelParameter(procIPv4Forward, 1); err != nil {
			return err
		}
	}
	if needsIPv6 {
		if err := writeKernelParameter(procIPv6Forward, 1); err != nil {
			return err
		}
	}

	content, err := routingSysctlContent(needsIPv4, needsIPv6, uplinks)
	if err != nil {
		return err
	}
	return writeFileAtomically(routingSysctlConfigFile, content, 0o644)
}

func routingForwardingNeeds(snapshot routingSnapshot) (bool, bool, error) {
	needsIPv4 := snapshot.exitNode
	needsIPv6 := snapshot.exitNode
	for _, route := range snapshot.subnetRoutes {
		prefix, err := netip.ParsePrefix(route)
		if err != nil || prefix != prefix.Masked() || prefix.Bits() == 0 {
			return false, false, fmt.Errorf("invalid advertised subnet route %q", route)
		}
		if prefix.Addr().Is4() {
			needsIPv4 = true
		} else if prefix.Addr().Is6() {
			needsIPv6 = true
		}
	}
	return needsIPv4, needsIPv6, nil
}

func routingSysctlContent(needsIPv4, needsIPv6 bool, uplinks []string) ([]byte, error) {
	if !needsIPv4 && !needsIPv6 {
		return nil, nil
	}
	if needsIPv6 && len(uplinks) == 0 {
		return nil, errors.New("no safe tailscale uplink")
	}
	normalizedUplinks := make([]string, 0, len(uplinks))
	seenUplinks := make(map[string]struct{}, len(uplinks))
	for _, uplink := range uplinks {
		if !isSafeUplinkInterface(uplink) {
			return nil, fmt.Errorf("invalid tailscale uplink %q", uplink)
		}
		if _, ok := seenUplinks[uplink]; ok {
			continue
		}
		seenUplinks[uplink] = struct{}{}
		normalizedUplinks = append(normalizedUplinks, uplink)
	}
	sort.Strings(normalizedUplinks)

	lines := []string{
		"# Managed by NanoKVM for Tailscale routing",
	}
	if needsIPv6 {
		lines = append(lines,
			"# Accept router advertisements before enabling IPv6 forwarding",
			"net.ipv6.conf.default.accept_ra = 2",
		)
		for _, uplink := range normalizedUplinks {
			// Slash syntax preserves dots inside VLAN interface names.
			lines = append(lines, fmt.Sprintf("net/ipv6/conf/%s/accept_ra = 2", uplink))
		}
	}
	if needsIPv4 {
		lines = append(lines, "net.ipv4.ip_forward = 1")
	}
	if needsIPv6 {
		lines = append(lines, "net.ipv6.conf.all.forwarding = 1")
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

func ipv6AcceptRAPath(uplink string) string {
	return filepath.Join("/proc/sys/net/ipv6/conf", uplink, "accept_ra")
}

func writeKernelParameter(path string, value int) error {
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "%d\n", value); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func writeFileAtomically(path string, data []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = os.Remove(temporaryPath)
	}()

	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return nil
}
