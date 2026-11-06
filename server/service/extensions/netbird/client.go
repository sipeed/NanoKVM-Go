package netbird

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
	"strconv"
	"strings"
)

const managementURL = "https://api.netbird.io:443"
const adminURL = "https://app.netbird.io:443"

type daemonStatus struct {
	Status        string `json:"status"`
	DaemonVersion string `json:"daemonVersion"`
	FullStatus    struct {
		ManagementState struct {
			Connected bool `json:"connected"`
		} `json:"managementState"`
		LocalPeerState struct {
			IP   string `json:"IP"`
			FQDN string `json:"fqdn"`
		} `json:"localPeerState"`
	} `json:"fullStatus"`
}

type loginResponse struct {
	NeedsSSOLogin           bool   `json:"needsSSOLogin"`
	UserCode                string `json:"userCode"`
	VerificationURI         string `json:"verificationURI"`
	VerificationURIComplete string `json:"verificationURIComplete"`
}

type rpcError struct {
	Code int `json:"code"`
}

func (e *rpcError) Error() string { return "daemonRequestFailed" }

func (s *Service) request(ctx context.Context, method string, input, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost/daemon.DaemonService/"+method, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if err != nil || len(data) > 2<<20 {
		return errors.New("invalidResponse")
	}
	if resp.StatusCode != http.StatusOK {
		var failure rpcError
		_ = json.Unmarshal(data, &failure)
		return &failure
	}
	if output != nil {
		return json.Unmarshal(data, output)
	}
	return nil
}

func loginOptions() map[string]any {
	// Keep configuration server-owned. Neither credentials nor server URLs are
	// accepted from the browser. This peer only exposes the NanoKVM itself.
	return map[string]any{
		"managementUrl": managementURL, "adminURL": adminURL,
		"isUnixDesktopClient": false, "profileName": "default",
	}
}

func peerConfig() map[string]any {
	// v0.80.0 Login only persists the management URL and preshared key.
	// Other preferences must be written using SetConfig before Login/Up.
	return map[string]any{
		"profileName": "default", "managementUrl": managementURL, "adminURL": adminURL,
		"disableAutoConnect": false,
		"disableDns":         true, "disableClientRoutes": true, "disableServerRoutes": true,
		"serverSSHAllowed": false, "remoteJobsAllowed": false,
		"rosenpassEnabled": false, "enableLocalMetrics": false,
	}
}

func authorizationURL(reply loginResponse) (string, error) {
	value := reply.VerificationURIComplete
	if value == "" {
		value = reply.VerificationURI
	}
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return "", errors.New("loginFailed")
	}
	return u.String(), nil
}

func peerAddress(value string) string {
	if prefix, err := netip.ParsePrefix(value); err == nil {
		value = prefix.Addr().String()
	}
	ip, err := netip.ParseAddr(value)
	if err != nil || !ip.IsGlobalUnicast() || ip.IsLoopback() || ip.Zone() != "" {
		return ""
	}
	return ip.String()
}

func accessURL(scheme string, port int, ip string) string {
	host := ip
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if scheme == "https" && port != 443 || scheme == "http" && port != 80 {
		host = net.JoinHostPort(ip, strconv.Itoa(port))
	}
	return (&url.URL{Scheme: scheme, Host: host}).String()
}

func readSmallFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 4097))
	if len(data) > 4096 {
		return "", errors.New("invalidFile")
	}
	return string(data), err
}
