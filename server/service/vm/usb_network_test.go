package vm

import (
	"os"
	"path/filepath"
	"testing"

	"NanoKVM-Server/proto"
)

func TestNormalizeUsbNetworkConfig(t *testing.T) {
	tests := []struct {
		name    string
		req     proto.SetUsbNetworkReq
		want    proto.GetUsbNetworkRsp
		wantErr bool
	}{
		{
			name: "default server",
			req:  proto.SetUsbNetworkReq{Mode: "server"},
			want: proto.GetUsbNetworkRsp{Mode: "server"},
		},
		{
			name: "server with manual address",
			req: proto.SetUsbNetworkReq{
				Mode: " SERVER ",
				IPv4: " 10.20.30.1/24 ",
			},
			want: proto.GetUsbNetworkRsp{
				Mode: "server",
				IPv4: "10.20.30.1/24",
			},
		},
		{
			name: "client",
			req:  proto.SetUsbNetworkReq{Mode: "client"},
			want: proto.GetUsbNetworkRsp{Mode: "client"},
		},
		{
			name: "server with forwarding",
			req:  proto.SetUsbNetworkReq{Mode: "server", Forwarding: true},
			want: proto.GetUsbNetworkRsp{Mode: "server", Forwarding: true},
		},
		{name: "unknown mode", req: proto.SetUsbNetworkReq{Mode: "bridge"}, wantErr: true},
		{name: "client manual address", req: proto.SetUsbNetworkReq{Mode: "client", IPv4: "10.0.0.1/24"}, wantErr: true},
		{name: "client forwarding", req: proto.SetUsbNetworkReq{Mode: "client", Forwarding: true}, wantErr: true},
		{name: "invalid address", req: proto.SetUsbNetworkReq{Mode: "server", IPv4: "10.0.0.999/24"}, wantErr: true},
		{name: "loopback address", req: proto.SetUsbNetworkReq{Mode: "server", IPv4: "127.0.0.1/24"}, wantErr: true},
		{name: "link-local address", req: proto.SetUsbNetworkReq{Mode: "server", IPv4: "169.254.1.1/24"}, wantErr: true},
		{name: "IPv6 address", req: proto.SetUsbNetworkReq{Mode: "server", IPv4: "fd00::1/64"}, wantErr: true},
		{name: "hostless prefix", req: proto.SetUsbNetworkReq{Mode: "server", IPv4: "10.0.0.1/32"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeUsbNetworkConfig(tc.req)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("normalizeUsbNetworkConfig(%+v) expected error, got %+v", tc.req, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeUsbNetworkConfig(%+v) unexpected error: %v", tc.req, err)
			}
			if *got != tc.want {
				t.Fatalf("normalizeUsbNetworkConfig(%+v) = %+v, want %+v", tc.req, *got, tc.want)
			}
		})
	}
}

func TestReadWriteUsbNetworkConfig(t *testing.T) {
	dir := t.TempDir()
	paths := usbNetworkPaths{
		dhcp:    filepath.Join(dir, "ncm.dhcp"),
		ipv4:    filepath.Join(dir, "ncm.ipv4"),
		forward: filepath.Join(dir, "ncm.forward"),
	}

	config, err := readUsbNetworkConfig(paths)
	if err != nil {
		t.Fatalf("read default configuration: %v", err)
	}
	if config.Mode != usbNetworkModeServer || config.IPv4 != "" {
		t.Fatalf("unexpected default configuration: %+v", config)
	}
	current, err := isUsbNetworkConfigCurrent(paths, config)
	if err != nil {
		t.Fatalf("compare default configuration: %v", err)
	}
	if !current {
		t.Fatal("default server configuration with forwarding disabled should be current")
	}

	server := &proto.GetUsbNetworkRsp{
		Mode:       usbNetworkModeServer,
		IPv4:       "10.20.30.1/24",
		Forwarding: true,
	}
	if err := writeUsbNetworkConfig(paths, server); err != nil {
		t.Fatalf("write server configuration: %v", err)
	}
	config, err = readUsbNetworkConfig(paths)
	if err != nil {
		t.Fatalf("read server configuration: %v", err)
	}
	if *config != *server {
		t.Fatalf("read server configuration = %+v, want %+v", config, server)
	}
	forward, err := os.ReadFile(paths.forward)
	if err != nil {
		t.Fatalf("read server forwarding configuration: %v", err)
	}
	if string(forward) != usbNetworkDefaultForwardInterface+"\n" {
		t.Fatalf("server forwarding configuration = %q", string(forward))
	}
	current, err = isUsbNetworkConfigCurrent(paths, server)
	if err != nil {
		t.Fatalf("compare server configuration: %v", err)
	}
	if !current {
		t.Fatal("written server configuration should be current")
	}
	if err := os.WriteFile(paths.forward, []byte("eth0\n"), 0644); err != nil {
		t.Fatalf("replace forwarding interface: %v", err)
	}
	config, err = readUsbNetworkConfig(paths)
	if err != nil {
		t.Fatalf("read server configuration with legacy forwarding interface: %v", err)
	}
	if !config.Forwarding {
		t.Fatal("an existing forwarding interface should be reported as enabled")
	}
	current, err = isUsbNetworkConfigCurrent(paths, server)
	if err != nil {
		t.Fatalf("compare server configuration with legacy forwarding interface: %v", err)
	}
	if current {
		t.Fatal("enabled forwarding should use wlan0")
	}
	if err := writeUsbNetworkConfig(paths, server); err != nil {
		t.Fatalf("restore default forwarding interface: %v", err)
	}

	server.Forwarding = false
	if err := writeUsbNetworkConfig(paths, server); err != nil {
		t.Fatalf("disable server forwarding: %v", err)
	}
	if _, err := os.Stat(paths.forward); !os.IsNotExist(err) {
		t.Fatalf("disabling server forwarding should remove %s", paths.forward)
	}
	config, err = readUsbNetworkConfig(paths)
	if err != nil {
		t.Fatalf("read server configuration with forwarding disabled: %v", err)
	}
	if *config != *server {
		t.Fatalf("read server configuration = %+v, want %+v", config, server)
	}

	server.Forwarding = true
	if err := writeUsbNetworkConfig(paths, server); err != nil {
		t.Fatalf("re-enable server forwarding: %v", err)
	}

	client := &proto.GetUsbNetworkRsp{Mode: usbNetworkModeClient}
	if err := writeUsbNetworkConfig(paths, client); err != nil {
		t.Fatalf("write client configuration: %v", err)
	}
	config, err = readUsbNetworkConfig(paths)
	if err != nil {
		t.Fatalf("read client configuration: %v", err)
	}
	if *config != *client {
		t.Fatalf("read client configuration = %+v, want %+v", config, client)
	}
	if _, err := os.Stat(paths.ipv4); !os.IsNotExist(err) {
		t.Fatalf("client configuration should remove %s", paths.ipv4)
	}
	if _, err := os.Stat(paths.forward); !os.IsNotExist(err) {
		t.Fatalf("client configuration should remove %s", paths.forward)
	}
	current, err = isUsbNetworkConfigCurrent(paths, client)
	if err != nil {
		t.Fatalf("compare client configuration: %v", err)
	}
	if !current {
		t.Fatal("written client configuration should be current")
	}
	if err := os.WriteFile(paths.forward, nil, 0644); err != nil {
		t.Fatalf("create empty forwarding configuration: %v", err)
	}
	current, err = isUsbNetworkConfigCurrent(paths, client)
	if err != nil {
		t.Fatalf("compare client configuration with empty forwarding file: %v", err)
	}
	if current {
		t.Fatal("client configuration with a forwarding file should not be current")
	}
}

func TestRestoreUsbNetworkFiles(t *testing.T) {
	dir := t.TempDir()
	paths := usbNetworkPaths{
		dhcp:    filepath.Join(dir, "ncm.dhcp"),
		ipv4:    filepath.Join(dir, "ncm.ipv4"),
		forward: filepath.Join(dir, "ncm.forward"),
	}
	if err := os.WriteFile(paths.ipv4, []byte("10.0.0.1/24\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.forward, []byte("eth0\n"), 0640); err != nil {
		t.Fatal(err)
	}

	snapshot, err := snapshotUsbNetworkFiles(paths)
	if err != nil {
		t.Fatalf("snapshot configuration: %v", err)
	}
	if err := writeUsbNetworkConfig(paths, &proto.GetUsbNetworkRsp{Mode: usbNetworkModeClient}); err != nil {
		t.Fatalf("replace configuration: %v", err)
	}
	if err := restoreUsbNetworkFiles(snapshot); err != nil {
		t.Fatalf("restore configuration: %v", err)
	}

	content, err := os.ReadFile(paths.ipv4)
	if err != nil {
		t.Fatalf("read restored IPv4 configuration: %v", err)
	}
	if string(content) != "10.0.0.1/24\n" {
		t.Fatalf("restored IPv4 configuration = %q", string(content))
	}
	info, err := os.Stat(paths.ipv4)
	if err != nil {
		t.Fatalf("stat restored IPv4 configuration: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("restored IPv4 mode = %v", info.Mode().Perm())
	}
	forward, err := os.ReadFile(paths.forward)
	if err != nil {
		t.Fatalf("read restored forwarding configuration: %v", err)
	}
	if string(forward) != "eth0\n" {
		t.Fatalf("restored forwarding configuration = %q", string(forward))
	}
	info, err = os.Stat(paths.forward)
	if err != nil {
		t.Fatalf("stat restored forwarding configuration: %v", err)
	}
	if info.Mode().Perm() != 0640 {
		t.Fatalf("restored forwarding mode = %v", info.Mode().Perm())
	}
	if _, err := os.Stat(paths.dhcp); !os.IsNotExist(err) {
		t.Fatalf("restore should remove DHCP marker created after snapshot")
	}
}
