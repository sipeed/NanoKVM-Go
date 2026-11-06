package tailscale

import (
	"errors"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectsTargetLANSubnetFromProcRouteData(t *testing.T) {
	routes, err := parseDefaultIPv4Routes(strings.NewReader(`Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT
wlan0 00000000 0100A8C0 0003 0 0 600 00000000 0 0 0
wlan0 0000A8C0 00000000 0001 0 0 600 00FCFFFF 0 0 0
`))
	if err != nil {
		t.Fatalf("parseDefaultIPv4Routes() error = %v", err)
	}
	route, err := selectDefaultIPv4Route(routes)
	if err != nil {
		t.Fatalf("selectDefaultIPv4Route() error = %v", err)
	}
	if route.Interface != "wlan0" || route.Gateway.String() != "192.168.0.1" || route.Metric != 600 {
		t.Fatalf("default route = %#v", route)
	}

	prefix, err := selectPrivateLANPrefix([]net.Addr{
		&net.IPNet{IP: net.ParseIP("192.168.3.16"), Mask: net.CIDRMask(22, 32)},
	}, route.Gateway)
	if err != nil {
		t.Fatalf("selectPrivateLANPrefix() error = %v", err)
	}
	if got, want := prefix.String(), "192.168.0.0/22"; got != want {
		t.Fatalf("LAN subnet = %q, want %q", got, want)
	}
}

func TestSelectDefaultIPv4RouteUsesMetricAndRejectsAmbiguity(t *testing.T) {
	routes := []defaultIPv4Route{
		{Interface: "wlan0", Gateway: netip.MustParseAddr("192.168.0.1"), Metric: 600},
		{Interface: "eth0", Gateway: netip.MustParseAddr("10.0.0.1"), Metric: 100},
	}
	selected, err := selectDefaultIPv4Route(routes)
	if err != nil {
		t.Fatalf("selectDefaultIPv4Route() error = %v", err)
	}
	if selected.Interface != "eth0" {
		t.Fatalf("selected interface = %q, want eth0", selected.Interface)
	}

	routes[0].Metric = 100
	if _, err := selectDefaultIPv4Route(routes); err == nil {
		t.Fatal("ambiguous default routes unexpectedly accepted")
	}
}

func TestParseDefaultIPv6RoutesFindsActualUplinks(t *testing.T) {
	routes, err := parseDefaultIPv6Routes(strings.NewReader(`00000000000000000000000000000000 00 00000000000000000000000000000000 00 FE800000000000000000000000000001 00000064 00000000 00000000 00000003 enp1s0.100
00000000000000000000000000000000 00 00000000000000000000000000000000 00 FE800000000000000000000000000002 00000032 00000000 00000000 00000003 wlan0
20010DB8000000000000000000000000 40 00000000000000000000000000000000 00 00000000000000000000000000000000 00000001 00000000 00000000 00000001 eth0
`))
	if err != nil {
		t.Fatalf("parseDefaultIPv6Routes() error = %v", err)
	}
	uplinks, err := selectDefaultIPv6Uplinks(routes)
	if err != nil {
		t.Fatalf("selectDefaultIPv6Uplinks() error = %v", err)
	}
	if got, want := strings.Join(uplinks, ","), "wlan0,enp1s0.100"; got != want {
		t.Fatalf("IPv6 default uplinks = %q, want %q", got, want)
	}
}

func TestSelectPrivateLANPrefixRejectsUnsafeOrAmbiguousNetworks(t *testing.T) {
	gateway := netip.MustParseAddr("192.168.0.1")
	if _, err := selectPrivateLANPrefix([]net.Addr{
		&net.IPNet{IP: net.ParseIP("100.64.0.2"), Mask: net.CIDRMask(24, 32)},
	}, gateway); err == nil {
		t.Fatal("CGNAT subnet unexpectedly accepted")
	}

	if _, err := selectPrivateLANPrefix([]net.Addr{
		&net.IPNet{IP: net.ParseIP("192.168.3.16"), Mask: net.CIDRMask(22, 32)},
		&net.IPNet{IP: net.ParseIP("192.168.0.10"), Mask: net.CIDRMask(24, 32)},
	}, gateway); err == nil {
		t.Fatal("ambiguous private subnets unexpectedly accepted")
	}
}

func TestSnapshotSeparatesExitNodeAndSubnetRoutes(t *testing.T) {
	snapshot, err := snapshotFromPrefs(&TsPrefs{AdvertiseRoutes: []string{
		"0.0.0.0/0",
		"::/0",
		"192.168.0.0/22",
		"10.0.0.0/24",
	}})
	if err != nil {
		t.Fatalf("snapshotFromPrefs() error = %v", err)
	}
	if !snapshot.exitNode {
		t.Fatal("exit node was not detected")
	}
	if !sameRoutes(snapshot.subnetRoutes, []string{"10.0.0.0/24", "192.168.0.0/22"}) {
		t.Fatalf("subnet routes = %#v", snapshot.subnetRoutes)
	}
}

func TestRoutingSnapshotComparisonIncludesAllPreferences(t *testing.T) {
	base := routingSnapshot{
		exitNode:     true,
		subnetRoutes: []string{"10.0.0.0/24", "192.168.0.0/22"},
	}
	if !sameRoutingSnapshots(base, routingSnapshot{
		exitNode:     true,
		subnetRoutes: []string{"192.168.0.0/22", "10.0.0.0/24"},
	}) {
		t.Fatal("equivalent routing snapshots were not equal")
	}
	if sameRoutingSnapshots(base, routingSnapshot{
		exitNode:     false,
		subnetRoutes: append([]string(nil), base.subnetRoutes...),
	}) {
		t.Fatal("exit-node mismatch was not detected")
	}
	if sameRoutingSnapshots(base, routingSnapshot{
		exitNode:     true,
		subnetRoutes: []string{"192.168.0.0/22"},
	}) {
		t.Fatal("missing unrelated subnet route was not detected")
	}
}

func TestUninstallRoutingSnapshotPreservesManualRoutes(t *testing.T) {
	previous := routingSnapshot{
		exitNode: true,
		subnetRoutes: []string{
			"10.0.0.0/24",
			"192.168.0.0/22",
		},
	}
	desired := routingSnapshotForUninstall(previous, "192.168.0.0/22")
	if desired.exitNode {
		t.Fatal("uninstall snapshot still advertises an exit node")
	}
	if !sameRoutes(desired.subnetRoutes, []string{"10.0.0.0/24"}) {
		t.Fatalf("uninstall routes = %#v, want only manual route", desired.subnetRoutes)
	}
}

func TestManagedSubnetRouteMergePreservesOtherRoutes(t *testing.T) {
	routes := []string{"10.0.0.0/24", "192.168.10.0/24"}
	routes = removeRoute(routes, "192.168.10.0/24")
	routes = addRoute(routes, "192.168.0.0/22")
	if !sameRoutes(routes, []string{"10.0.0.0/24", "192.168.0.0/22"}) {
		t.Fatalf("migrated routes = %#v", routes)
	}
	routes = removeRoute(routes, "192.168.0.0/22")
	if !sameRoutes(routes, []string{"10.0.0.0/24"}) {
		t.Fatalf("disabled routes = %#v", routes)
	}
}

func TestRoutingSysctlContentSetsRABeforeIPv6Forwarding(t *testing.T) {
	content, err := routingSysctlContent(true, true, []string{"wlan0", "enp1s0.100"})
	if err != nil {
		t.Fatalf("routingSysctlContent() error = %v", err)
	}
	text := string(content)
	acceptIndex := strings.Index(text, "net/ipv6/conf/wlan0/accept_ra = 2")
	forwardIndex := strings.Index(text, "net.ipv6.conf.all.forwarding = 1")
	if acceptIndex < 0 || forwardIndex < 0 || acceptIndex >= forwardIndex {
		t.Fatalf("unsafe sysctl ordering:\n%s", text)
	}
	if !strings.Contains(text, "net.ipv4.ip_forward = 1") {
		t.Fatalf("IPv4 forwarding missing:\n%s", text)
	}
	if !strings.Contains(text, "net/ipv6/conf/enp1s0.100/accept_ra = 2") {
		t.Fatalf("VLAN uplink does not use slash sysctl syntax:\n%s", text)
	}

	subnetOnly, err := routingSysctlContent(true, false, nil)
	if err != nil {
		t.Fatalf("subnet routingSysctlContent() error = %v", err)
	}
	if strings.Contains(string(subnetOnly), "ipv6") {
		t.Fatalf("subnet-only config unexpectedly changes IPv6:\n%s", subnetOnly)
	}

	ipv6Only, err := routingSysctlContent(false, true, []string{"wlan0"})
	if err != nil {
		t.Fatalf("IPv6 subnet routingSysctlContent() error = %v", err)
	}
	if strings.Contains(string(ipv6Only), "net.ipv4.ip_forward") ||
		!strings.Contains(string(ipv6Only), "net.ipv6.conf.all.forwarding = 1") {
		t.Fatalf("IPv6-only forwarding config is incorrect:\n%s", ipv6Only)
	}
}

func TestRoutingForwardingNeedsFollowSubnetAddressFamilies(t *testing.T) {
	tests := []struct {
		name     string
		snapshot routingSnapshot
		ipv4     bool
		ipv6     bool
	}{
		{name: "none", snapshot: routingSnapshot{}},
		{
			name:     "IPv4 subnet",
			snapshot: routingSnapshot{subnetRoutes: []string{"192.168.0.0/22"}},
			ipv4:     true,
		},
		{
			name:     "IPv6 subnet",
			snapshot: routingSnapshot{subnetRoutes: []string{"fd00::/64"}},
			ipv6:     true,
		},
		{
			name:     "exit node",
			snapshot: routingSnapshot{exitNode: true},
			ipv4:     true,
			ipv6:     true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			ipv4, ipv6, err := routingForwardingNeeds(testCase.snapshot)
			if err != nil {
				t.Fatalf("routingForwardingNeeds() error = %v", err)
			}
			if ipv4 != testCase.ipv4 || ipv6 != testCase.ipv6 {
				t.Fatalf("forwarding needs = (%v, %v), want (%v, %v)", ipv4, ipv6, testCase.ipv4, testCase.ipv6)
			}
		})
	}
}

func TestManagedSubnetStateIsAtomicAndPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tailscale-routing.json")
	if err := saveManagedSubnet(path, "192.168.0.0/22"); err != nil {
		t.Fatalf("saveManagedSubnet() error = %v", err)
	}
	got, err := loadManagedSubnet(path)
	if err != nil {
		t.Fatalf("loadManagedSubnet() error = %v", err)
	}
	if got != "192.168.0.0/22" {
		t.Fatalf("managed subnet = %q", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat managed subnet state: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("managed subnet mode = %o, want 600", mode)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".tailscale-routing.json.*"))
	if err != nil {
		t.Fatalf("glob temporary files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("atomic write left temporary files: %#v", matches)
	}

	if err := saveManagedSubnet(path, ""); err != nil {
		t.Fatalf("clear managed subnet: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("managed subnet state still exists: %v", err)
	}
}

type fakeRoutingPrefsResult struct {
	prefs *TsPrefs
	err   error
}

type fakeRoutingClient struct {
	prefsResults []fakeRoutingPrefsResult
	setCalls     []routingSnapshot
	setErr       error
}

func (f *fakeRoutingClient) Prefs() (*TsPrefs, error) {
	if len(f.prefsResults) == 0 {
		return nil, errors.New("unexpected Prefs call")
	}
	result := f.prefsResults[0]
	f.prefsResults = f.prefsResults[1:]
	return result.prefs, result.err
}

func (f *fakeRoutingClient) SetRouting(exitNode bool, routes []string) error {
	f.setCalls = append(f.setCalls, routingSnapshot{
		exitNode:     exitNode,
		subnetRoutes: append([]string(nil), routes...),
	})
	return f.setErr
}

func TestRollbackRoutingReconcilesForwardingFromActualPrefs(t *testing.T) {
	cause := errors.New("verification failed")
	rollbackErr := errors.New("rollback command failed")
	previous := routingSnapshot{subnetRoutes: []string{"10.0.0.0/24"}}
	desired := routingSnapshot{
		exitNode:     true,
		subnetRoutes: []string{"10.0.0.0/24", "192.168.0.0/22"},
	}
	actual := routingSnapshot{
		exitNode:     true,
		subnetRoutes: []string{"172.16.0.0/16"},
	}
	client := &fakeRoutingClient{
		prefsResults: []fakeRoutingPrefsResult{{prefs: &TsPrefs{AdvertiseRoutes: []string{
			"0.0.0.0/0",
			"::/0",
			"172.16.0.0/16",
		}}}},
		setErr: rollbackErr,
	}
	var synced routingSnapshot
	err := rollbackRouting(cause, client, previous, desired, true, func(snapshot routingSnapshot) error {
		synced = snapshot
		return nil
	})
	if !errors.Is(err, cause) || !errors.Is(err, rollbackErr) {
		t.Fatalf("rollback error = %v, want original and rollback errors", err)
	}
	if len(client.setCalls) != 1 || !sameRoutingSnapshots(client.setCalls[0], previous) {
		t.Fatalf("rollback set calls = %#v", client.setCalls)
	}
	if !sameRoutingSnapshots(synced, actual) {
		t.Fatalf("forwarding reconciled to %#v, want actual %#v", synced, actual)
	}
}

func TestRollbackRoutingUsesConservativeUnionWhenPrefsCannotBeRead(t *testing.T) {
	cause := errors.New("update failed")
	readErr := errors.New("prefs unavailable")
	syncErr := errors.New("forwarding unavailable")
	previous := routingSnapshot{subnetRoutes: []string{"10.0.0.0/24"}}
	desired := routingSnapshot{
		exitNode:     true,
		subnetRoutes: []string{"192.168.0.0/22"},
	}
	client := &fakeRoutingClient{
		prefsResults: []fakeRoutingPrefsResult{{err: readErr}},
	}
	var synced routingSnapshot
	err := rollbackRouting(cause, client, previous, desired, false, func(snapshot routingSnapshot) error {
		synced = snapshot
		return syncErr
	})
	if !errors.Is(err, cause) || !errors.Is(err, readErr) || !errors.Is(err, syncErr) {
		t.Fatalf("rollback error = %v, want all recovery errors", err)
	}
	want := routingSnapshot{
		exitNode:     true,
		subnetRoutes: []string{"10.0.0.0/24", "192.168.0.0/22"},
	}
	if !sameRoutingSnapshots(synced, want) {
		t.Fatalf("conservative forwarding = %#v, want %#v", synced, want)
	}
}

func TestApplyRoutingPreferencesRejectsPartialSnapshotUpdate(t *testing.T) {
	previous := routingSnapshot{subnetRoutes: []string{"10.0.0.0/24"}}
	desired := routingSnapshot{
		exitNode:     true,
		subnetRoutes: []string{"10.0.0.0/24", "192.168.0.0/22"},
	}
	operation := &fakeRoutingClient{prefsResults: []fakeRoutingPrefsResult{{prefs: &TsPrefs{AdvertiseRoutes: []string{
		"0.0.0.0/0",
		"::/0",
		"192.168.0.0/22",
	}}}}}
	recovery := &fakeRoutingClient{prefsResults: []fakeRoutingPrefsResult{{prefs: &TsPrefs{AdvertiseRoutes: []string{
		"10.0.0.0/24",
	}}}}}
	var synced []routingSnapshot
	_, _, err := applyRoutingPreferences(operation, recovery, previous, desired, func(snapshot routingSnapshot) error {
		synced = append(synced, snapshot)
		return nil
	})
	if err == nil {
		t.Fatal("partial routing snapshot update unexpectedly accepted")
	}
	if len(operation.setCalls) != 1 || !sameRoutingSnapshots(operation.setCalls[0], desired) {
		t.Fatalf("operation set calls = %#v", operation.setCalls)
	}
	if len(recovery.setCalls) != 1 || !sameRoutingSnapshots(recovery.setCalls[0], previous) {
		t.Fatalf("recovery set calls = %#v", recovery.setCalls)
	}
	if len(synced) != 2 || !sameRoutingSnapshots(synced[0], desired) || !sameRoutingSnapshots(synced[1], previous) {
		t.Fatalf("forwarding snapshots = %#v", synced)
	}
}
