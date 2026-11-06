package zerotier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"NanoKVM-Server/proto"
)

var networkIDPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)
var nodeIDPattern = regexp.MustCompile(`^[0-9a-f]{10}$`)

func normalizeNetworkID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if !networkIDPattern.MatchString(id) || id == "0000000000000000" {
		return "", errors.New("invalidNetworkID")
	}
	return id, nil
}

type nodeStatus struct {
	Address string `json:"address"`
	Online  bool   `json:"online"`
	Version string `json:"version"`
}

type networkStatus struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Addresses []string `json:"assignedAddresses"`
}

// The local API token is never sent to the browser or through an HTTP proxy.
func (s *Service) request(ctx context.Context, method, path string, body any, result any) error {
	token, err := readSmallFile(filepath.Join(s.paths.state, "authtoken.secret"))
	if err != nil || len(bytes.TrimSpace(token)) == 0 {
		return errors.New("apiUnavailable")
	}
	var data []byte
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return errors.New("apiFailed")
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, s.apiURL+path, bytes.NewReader(data))
	if err != nil {
		return errors.New("apiFailed")
	}
	req.Header.Set("X-ZT1-Auth", strings.TrimSpace(string(token)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return errors.New("apiUnavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("apiFailed")
	}
	data, err = io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return errors.New("apiFailed")
	}
	if result != nil && json.Unmarshal(data, result) != nil {
		return errors.New("apiFailed")
	}
	return nil
}

func readSmallFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 4096 {
		return nil, errors.New("invalidFile")
	}
	return os.ReadFile(path)
}

// Offline status reads only the public node ID and membership filenames.
func (s *Service) savedState() (string, []proto.ZeroTierNetwork, error) {
	var id string
	if data, err := readSmallFile(filepath.Join(s.paths.state, "identity.public")); err == nil {
		id = strings.SplitN(string(data), ":", 2)[0]
		if !nodeIDPattern.MatchString(id) {
			id = ""
		}
	}
	networks := []proto.ZeroTierNetwork{}
	entries, err := os.ReadDir(filepath.Join(s.paths.state, "networks.d"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return id, networks, errors.New("configInvalid")
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.Type().IsRegular() || !strings.HasSuffix(name, ".conf") {
			continue
		}
		networkID, err := normalizeNetworkID(strings.TrimSuffix(name, ".conf"))
		if err == nil {
			networks = append(networks, proto.ZeroTierNetwork{ID: networkID, Status: "STOPPED", Addresses: []string{}, AccessURLs: []string{}})
		}
	}
	return id, networks, nil
}

func networkView(n networkStatus, online bool, scheme string, port int) proto.ZeroTierNetwork {
	out := proto.ZeroTierNetwork{ID: n.ID, Name: n.Name, Status: n.Status, Addresses: []string{}, AccessURLs: []string{}}
	if !online {
		out.Status = "OFFLINE"
		return out
	}
	if n.Status != "OK" {
		return out
	}
	seen := map[netip.Addr]bool{}
	for _, cidr := range n.Addresses {
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			continue
		}
		ip := prefix.Addr().Unmap()
		if !ip.IsGlobalUnicast() || ip.IsLoopback() || seen[ip] {
			continue
		}
		seen[ip] = true
		out.Addresses = append(out.Addresses, ip.String())
		if scheme != "http" && scheme != "https" || port < 1 || port > 65535 {
			continue
		}
		host := net.JoinHostPort(ip.String(), strconv.Itoa(port))
		if scheme == "http" && port == 80 || scheme == "https" && port == 443 {
			host = ip.String()
			if ip.Is6() {
				host = "[" + host + "]"
			}
		}
		out.AccessURLs = append(out.AccessURLs, (&url.URL{Scheme: scheme, Host: host}).String())
	}
	if len(out.Addresses) == 0 {
		out.Status = "WAITING_ADDRESS"
	}
	return out
}
