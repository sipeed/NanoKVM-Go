package frp

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"

	"github.com/gorilla/websocket"
)

// Opt-in because this downloads and executes the pinned official release. The
// default suite remains offline. Everything listens on loopback and uses test
// directories; no installed binary, unit, or saved device configuration changes.
func TestOfficialReleaseTunnel(t *testing.T) {
	if os.Getenv("NANOKVM_FRP_INTEGRATION") != "1" {
		t.Skip("set NANOKVM_FRP_INTEGRATION=1 to exercise official frpc/frps")
	}
	checksum, ok := releaseChecksums[packageArch()]
	if !ok {
		t.Fatal("unsupported integration-test architecture")
	}
	dir := t.TempDir()
	packageName := fmt.Sprintf("frp_%s_linux_%s", FRPVersion, packageArch())
	archive := filepath.Join(dir, "release.tgz")
	downloadURL := releaseURL + packageName + ".tar.gz"
	if source := os.Getenv("NANOKVM_FRP_TEST_ARCHIVE"); source != "" {
		// Devices without GitHub access may receive the archive over SSH first.
		// It still passes through the same size and pinned-checksum validation.
		fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, source)
		}))
		defer fixture.Close()
		downloadURL = fixture.URL
	}
	if err := downloadArchive(context.Background(), downloadURL, archive, checksum); err != nil {
		t.Fatal(err)
	}
	if err := extractClient(archive, dir, packageName); err != nil {
		t.Fatal(err)
	}
	serverBinary := filepath.Join(dir, "frps")
	clientBinary := filepath.Join(dir, "frpc")
	version, err := exec.Command(clientBinary, "--version").Output()
	if err != nil || strings.TrimSpace(string(version)) != FRPVersion {
		t.Fatalf("client version: %q, %v", version, err)
	}
	t.Logf("official client package: %s", packageName)
	if os.Getenv("NANOKVM_FRP_PUBLIC_ONLY") == "1" {
		if os.Getenv("NANOKVM_FRP_PUBLIC_SERVER") == "" {
			t.Fatal("public test server required")
		}
		port, err := strconv.Atoi(os.Getenv("NANOKVM_FRP_DEVICE_HTTPS_PORT"))
		if err != nil || port < 1 || port > 65535 {
			t.Fatal("invalid device HTTPS port")
		}
		s := testService(t)
		s.paths.binary = clientBinary
		cfg, err := makeConfig(validForm(), nil, port)
		if err != nil {
			t.Fatal(err)
		}
		testPublicServer(t, s, cfg)
		return
	}
	extractTestServer(t, archive, packageName+"/frps", serverBinary)

	adminProbe, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(adminPort))
	if err != nil {
		t.Fatal("test requires a free local admin port", err)
	}
	adminProbe.Close()
	controlPort, remotePort := freePort(t), freePort(t)
	for remotePort == controlPort {
		remotePort = freePort(t)
	}
	serverConfig := filepath.Join(dir, "frps.toml")
	serverText := fmt.Sprintf("bindAddr = '127.0.0.1'\nbindPort = %d\nauth.method = 'token'\nauth.token = 'integration-token'\ntransport.tls.force = true\nallowPorts = [{single = %d}]\n", controlPort, remotePort)
	if err := os.WriteFile(serverConfig, []byte(serverText), 0600); err != nil {
		t.Fatal(err)
	}
	stopServer := startTestProcess(t, serverBinary, serverConfig)
	waitFor(t, "frps listener", func() bool {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", controlPort), 200*time.Millisecond)
		if err == nil {
			conn.Close()
		}
		return err == nil
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("nanokvm-tls-test")) })
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: utils.CheckWebSocketOrigin}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		kind, payload, err := conn.ReadMessage()
		if err == nil {
			_ = conn.WriteMessage(kind, payload)
		}
	})
	target := httptest.NewTLSServer(mux)
	defer target.Close()
	targetPort := target.Listener.Addr().(*net.TCPAddr).Port
	s := testService(t)
	s.paths.binary = clientBinary
	s.localEndpoint = func() (string, int) { return "https", targetPort }
	token := "integration-token"
	form := proto.SetFRPConfigReq{ServerAddr: "127.0.0.1", ServerPort: controlPort, RemotePort: remotePort, Token: &token}
	if rsp, _ := request(t, s.SetConfig, form); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	// The form-generated schema is checked against the real pinned client in
	// this integration test, rather than spawning a verifier for every UI save.
	verifyContext, verifyCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer verifyCancel()
	if err := exec.CommandContext(verifyContext, clientBinary, "verify", "-c", s.paths.config).Run(); err != nil {
		t.Fatal("official client rejected generated configuration", err)
	}
	s.system.running = func() (bool, error) { return true, nil }
	s.system.enabled = func() (bool, error) { return true, nil }
	stopClient := startTestProcess(t, clientBinary, s.paths.config)
	waitFor(t, "registered TCP proxy", func() bool { return s.status(context.Background()).State == "connected" })
	publicURL := fmt.Sprintf("https://127.0.0.1:%d", remotePort)
	response, err := target.Client().Get(publicURL)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || string(body) != "nanokvm-tls-test" {
		t.Fatalf("TLS forwarding: %q, %v", body, err)
	}
	dialer := websocket.Dialer{TLSClientConfig: target.Client().Transport.(*http.Transport).TLSClientConfig}
	ws, _, err := dialer.Dial(strings.Replace(publicURL, "https:", "wss:", 1)+"/ws", http.Header{"Origin": []string{publicURL}})
	if err != nil {
		t.Fatal("same-origin WebSocket through TCP proxy", err)
	}
	payload := bytes.Repeat([]byte("video-input-test"), 2048)
	if err := ws.WriteMessage(websocket.BinaryMessage, payload); err != nil {
		t.Fatal(err)
	}
	_, echoed, err := ws.ReadMessage()
	ws.Close()
	if err != nil || !bytes.Equal(payload, echoed) {
		t.Fatal("WebSocket payload did not survive forwarding", err)
	}
	t.Log("HTTPS and same-origin binary WebSocket forwarding passed")

	stopServer()
	waitFor(t, "connection loss", func() bool { return s.status(context.Background()).State != "connected" })
	stopServer = startTestProcess(t, serverBinary, serverConfig)
	waitFor(t, "automatic reconnection", func() bool { return s.status(context.Background()).State == "connected" })
	t.Log("automatic reconnection after frps restart passed")
	stopClient()

	cfg, err := readConfig(s.paths.config)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Auth.Token = "incorrect-test-token"
	if err := writeConfig(s.paths.config, cfg); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = exec.CommandContext(ctx, clientBinary, "-c", s.paths.config).Run()
	if err == nil || ctx.Err() != nil {
		t.Fatal("wrong token must exit unsuccessfully for systemd to retry", err)
	}
	cancel()
	cfg.Auth.Token = token
	if err := writeConfig(s.paths.config, cfg); err != nil {
		t.Fatal(err)
	}
	var blocker net.Listener
	waitFor(t, "released public port", func() bool {
		blocker, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", remotePort))
		return err == nil
	})
	defer blocker.Close()
	stopClient = startTestProcess(t, clientBinary, s.paths.config)
	waitFor(t, "port conflict", func() bool {
		status := s.status(context.Background())
		return status.State == "error" && status.Error == "proxyFailed"
	})
	stopClient()
	blocker.Close()
	t.Log("incorrect token and public port conflict passed")

	// On the target device, optionally verify its current HTTPS web page through
	// the real tunnel. This reads the running app without replacing or restarting it.
	if devicePort := os.Getenv("NANOKVM_FRP_DEVICE_HTTPS_PORT"); devicePort != "" {
		port, err := strconv.Atoi(devicePort)
		if err != nil || port < 1 || port > 65535 {
			t.Fatal("invalid device HTTPS port")
		}
		cfg.Proxies[0].LocalPort = port
		if err := writeConfig(s.paths.config, cfg); err != nil {
			t.Fatal(err)
		}
		if os.Getenv("NANOKVM_FRP_SYSTEMD") == "1" {
			stopClient = startTestUnit(t, s)
		} else {
			stopClient = startTestProcess(t, clientBinary, s.paths.config)
		}
		waitFor(t, "device proxy", func() bool { return s.status(context.Background()).State == "connected" })
		// The device uses its own certificate; this test is confined to loopback.
		client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
		response, err := client.Get(publicURL)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		response.Body.Close()
		if err != nil || response.StatusCode != http.StatusOK || !bytes.Contains(bytes.ToLower(body), []byte("<html")) {
			t.Fatal("device web page could not be accessed through frp", err)
		}
		stopClient()
		t.Log("current device HTTPS web page forwarded successfully")
	}
	stopServer()
	if os.Getenv("NANOKVM_FRP_PUBLIC_SERVER") != "" {
		testPublicServer(t, s, cfg)
	}
}

func testPublicServer(t *testing.T, s *Service, cfg *clientConfig) {
	t.Helper()
	certificate, err := os.ReadFile(os.Getenv("NANOKVM_FRP_DEVICE_CERT"))
	if err != nil {
		t.Fatal("read device certificate", err)
	}
	block, _ := pem.Decode(certificate)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatal("invalid device certificate")
	}
	client := &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{
		// Match the actual device certificate instead of relying on a public CA
		// or domain match for the device's self-signed HTTPS certificate.
		InsecureSkipVerify: true,
		VerifyConnection: func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 || !bytes.Equal(state.PeerCertificates[0].Raw, block.Bytes) {
				return errors.New("forwarded certificate does not match the device")
			}
			return nil
		},
	}}}
	defer client.CloseIdleConnections()
	localURL := fmt.Sprintf("https://127.0.0.1:%d", cfg.Proxies[0].LocalPort)
	response, err := client.Get(localURL + "/api/extensions/tailscale/status")
	if err != nil {
		t.Fatal("check local authentication", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatal("public test requires device login protection")
	}
	secret, err := os.ReadFile(os.Getenv("NANOKVM_FRP_TOKEN_FILE"))
	if err != nil {
		t.Fatal("read test token file", err)
	}
	token := strings.TrimSpace(string(secret))
	form := proto.SetFRPConfigReq{ServerAddr: os.Getenv("NANOKVM_FRP_PUBLIC_SERVER"), ServerPort: 7000, RemotePort: 8443, Token: &token}
	if err := validateForm(&form); err != nil {
		t.Fatal("invalid public test settings")
	}
	localPort := cfg.Proxies[0].LocalPort
	s.localEndpoint = func() (string, int) { return "https", localPort }
	s.system.running = func() (bool, error) { return false, nil }
	s.system.enabled = func() (bool, error) { return false, nil }
	started := time.Now()
	if rsp, _ := request(t, s.SetConfig, form); rsp.Code != 0 {
		t.Fatal("save public test configuration", rsp.Code, rsp.Msg)
	}
	t.Logf("form configuration saved in %s", time.Since(started))
	cfg, err = readConfig(s.paths.config)
	if err != nil {
		t.Fatal(err)
	}
	stop := startTestUnit(t, s)
	defer stop()
	s.system.running = func() (bool, error) { return true, nil }
	s.system.enabled = func() (bool, error) { return true, nil }
	waitFor(t, "public server proxy registration", func() bool { return s.status(context.Background()).State == "connected" })
	response, err = client.Get(cfg.accessURL("https"))
	if err != nil {
		t.Fatal("public HTTPS forwarding", err)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	response.Body.Close()
	if err != nil || response.StatusCode != http.StatusOK || !bytes.Contains(bytes.ToLower(body), []byte("<html")) {
		t.Fatal("public device web page unavailable")
	}
	response, err = client.Get(cfg.accessURL("https") + "/api/extensions/tailscale/status")
	if err != nil {
		t.Fatal("public authentication check", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatal("public access bypassed device authentication")
	}
	t.Log("public HTTPS forwarding, device certificate pin and unauthenticated API rejection passed")
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func waitFor(t *testing.T, description string, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("timed out waiting for", description)
}

func startTestProcess(t *testing.T, binary, config string) func() {
	t.Helper()
	command := exec.Command(binary, "-c", config)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { _ = command.Wait(); close(done) }()
	var once sync.Once
	stop := func() { once.Do(func() { _ = command.Process.Kill(); <-done }) }
	t.Cleanup(stop)
	return stop
}

func extractTestServer(t *testing.T, archive, name, destination string) {
	t.Helper()
	file, err := os.Open(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err != nil {
			t.Fatal("missing frps in verified archive", err)
		}
		if header.Name != name {
			continue
		}
		if header.Typeflag != tar.TypeReg || header.Size > maxArchiveBytes {
			t.Fatal("invalid server executable")
		}
		output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.CopyN(output, reader, header.Size)
		closeErr := output.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
		return
	}
}

// Exercise the actual generated systemd unit under a unique runtime-only name.
// Device tests use a disk-backed TMPDIR outside /tmp and /var/tmp so PrivateTmp
// can stay enabled exactly as it is in the production unit.
func startTestUnit(t *testing.T, s *Service) func() {
	t.Helper()
	id, err := randomID()
	if err != nil {
		t.Fatal(err)
	}
	name := "nanokvm-frp-validation-" + id + ".service"
	path := filepath.Join("/run/systemd/system", name)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.WriteString(s.unitContents())
	closeErr := file.Close()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			if output, err := exec.Command("systemctl", "stop", name).CombinedOutput(); err != nil {
				t.Errorf("stop validation unit: %s: %v", output, err)
			}
			if err := os.Remove(path); err != nil {
				t.Error(err)
			}
			if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
				t.Error(err)
			}
		})
	}
	t.Cleanup(stop)
	if writeErr != nil || closeErr != nil {
		t.Fatal(writeErr, closeErr)
	}
	if output, err := exec.Command("systemd-analyze", "verify", path).CombinedOutput(); err != nil {
		t.Fatalf("unit verification: %s: %v", output, err)
	}
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("systemctl", "start", name).CombinedOutput(); err != nil {
		t.Fatalf("start validation unit: %s: %v", output, err)
	}
	t.Log("generated systemd unit started with production restrictions")
	return stop
}
