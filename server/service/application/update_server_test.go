package application

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/updateverify"

	"github.com/gin-gonic/gin"
)

func TestApplicationUpdateServerChannels(t *testing.T) {
	for _, test := range []struct {
		server  string
		preview bool
		want    string
	}{
		{"", false, StableURL},
		{"", true, PreviewURL},
		{" http://192.168.1.20:8080/releases/// ", false, "http://192.168.1.20:8080/releases"},
		{"http://192.168.1.20:8080/releases/", true, "http://192.168.1.20:8080/releases/preview"},
	} {
		got, err := applicationUpdateBaseURL(test.server, test.preview)
		if err != nil || got != test.want {
			t.Fatalf("server=%q preview=%t: %q, %v; want %q", test.server, test.preview, got, err, test.want)
		}
	}
}

func TestConfiguredServerDiscoveryAndDownload(t *testing.T) {
	archive := []byte("test update transport")
	digest := sha512.Sum512(archive)
	metadata := latestMetadataBytes(t, "1.2.4", "nanokvm-go_1.2.4.tar.gz", base64.StdEncoding.EncodeToString(digest[:]), int64(len(archive)))
	channel := "/releases"
	if isPreviewEnabled() {
		channel += "/preview"
	}
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		switch r.URL.Path {
		case channel + "/latest.json":
			_, _ = w.Write(metadata)
		case channel + "/nanokvm-go_1.2.4.tar.gz":
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	previous := getConfiguredUpdateServer
	getConfiguredUpdateServer = func() string { return server.URL + "/releases/" }
	t.Cleanup(func() { getConfiguredUpdateServer = previous })

	latest, err := getLatest()
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodGet, latest.Url, nil)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), latest.Name)
	if err := downloadApplicationArtifact(request, target, latest.Size, server.Client(), nil); err != nil {
		t.Fatal(err)
	}
	if err := updateverify.VerifyTransportArtifact(target, updateverify.TransportMetadata{
		Name: latest.Name, SHA512: latest.Sha512, Size: latest.Size,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(got, archive) {
		t.Fatalf("download = %q, %v", got, err)
	}
	if len(requests) != 2 || requests[0] != channel+"/latest.json" || requests[1] != channel+"/"+latest.Name {
		t.Fatalf("unexpected server requests: %v", requests)
	}
}

func TestInvalidConfiguredUpdateServerDoesNotFallBack(t *testing.T) {
	previous := getConfiguredUpdateServer
	getConfiguredUpdateServer = func() string { return "file:///tmp/releases" }
	t.Cleanup(func() { getConfiguredUpdateServer = previous })
	useApplicationHTTPClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid configuration must not issue any network request")
		return nil, errors.New("unexpected request")
	}))
	if _, err := getLatest(); err == nil {
		t.Fatal("invalid configured update server accepted")
	}
}

func TestUpdateDiscoveryHasDeadline(t *testing.T) {
	useApplicationHTTPClient(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		deadline, ok := request.Context().Deadline()
		if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > latestMetadataTimeout {
			t.Fatal("metadata request has no bounded deadline")
		}
		return latestHTTPResponse(request, http.StatusOK, io.NopCloser(strings.NewReader("{}"))), nil
	}))
	if _, _, err := fetchApplicationUpdateFile("http://updates.invalid/latest.json"); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateServerSettingsHandlers(t *testing.T) {
	previousGet, previousSave := getConfiguredUpdateServer, saveConfiguredUpdateServer
	t.Cleanup(func() {
		getConfiguredUpdateServer, saveConfiguredUpdateServer = previousGet, previousSave
	})
	configured := "https://previous.example.com/releases"
	getConfiguredUpdateServer = func() string { return configured }
	saveError := false
	saves := 0
	saveConfiguredUpdateServer = func(server string) error {
		saves++
		if saveError {
			return errors.New("disk full")
		}
		configured = server
		return nil
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	service := NewService()
	router.GET("/server", service.GetUpdateServer)
	router.POST("/server", service.SetUpdateServer)
	request := func(method, body string) (int, string) {
		t.Helper()
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/server", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, req)
		var rsp struct {
			Code int                      `json:"code"`
			Data proto.GetUpdateServerRsp `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
			t.Fatal(err)
		}
		return rsp.Code, rsp.Data.Server
	}
	if code, server := request(http.MethodGet, ""); code != 0 || server != configured {
		t.Fatalf("get server = %d, %q", code, server)
	}
	for _, body := range []string{`{}`, `{"server":null}`, `{"server":42}`, `{"server":"file:///tmp"}`, `{"server":`} {
		if code, _ := request(http.MethodPost, body); code != -1 {
			t.Fatalf("invalid body %q returned %d", body, code)
		}
	}
	if saves != 0 {
		t.Fatal("invalid requests reached persistence")
	}
	if code, server := request(http.MethodPost, `{"server":" http://192.168.1.20:8080/releases/// "}`); code != 0 || server != "http://192.168.1.20:8080/releases" || configured != server {
		t.Fatalf("save server = %d, %q; configured=%q", code, server, configured)
	}
	saveError = true
	if code, _ := request(http.MethodPost, `{"server":""}`); code != -2 || configured == "" {
		t.Fatalf("failed save = %d; configured=%q", code, configured)
	}
	saveError = false
	if code, server := request(http.MethodPost, `{"server":""}`); code != 0 || server != "" || configured != "" {
		t.Fatalf("restore official server = %d, %q; configured=%q", code, server, configured)
	}
}
