package tailscale

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestManagedLocalAPIProfileAndStartContract(t *testing.T) {
	type capturedRequest struct {
		method      string
		path        string
		host        string
		contentType string
		body        string
	}
	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		requests = append(requests, capturedRequest{
			method:      request.Method,
			path:        request.URL.Path,
			host:        request.Host,
			contentType: request.Header.Get("Content-Type"),
			body:        string(body),
		})
		switch {
		case request.Method == http.MethodGet && request.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIProfilesPath:
			_, _ = io.WriteString(w, `[{"ID":"1ab3","Name":"Sipeed","NodeID":"node-1","ControlURL":"`+sipeedControlURL+`"}]`)
		case request.Method == http.MethodGet && request.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPICurrentProfilePath:
			_, _ = io.WriteString(w, `{"ID":"beef","Name":"official","NodeID":"node-2","ControlURL":"https://controlplane.tailscale.com"}`)
		case request.Method == http.MethodPut && request.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIProfilesPath:
			w.WriteHeader(http.StatusCreated)
		case request.Method == http.MethodPost && request.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIProfilesPath+"1ab3":
			w.WriteHeader(http.StatusNoContent)
		case request.Method == http.MethodPost && request.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIStartPath:
			w.WriteHeader(http.StatusNoContent)
		case request.Method == http.MethodPost && request.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPILoginInteractivePath:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))
	profiles, err := client.Profiles()
	if err != nil || len(profiles) != 1 || profiles[0].ID != "1ab3" || profiles[0].ControlURL != sipeedControlURL {
		t.Fatalf("Profiles() = %+v, %v", profiles, err)
	}
	current, err := client.CurrentProfile()
	if err != nil || current.ID != "beef" {
		t.Fatalf("CurrentProfile() = %+v, %v", current, err)
	}
	if err := client.NewProfile(); err != nil {
		t.Fatal(err)
	}
	if err := client.SwitchProfile("1ab3"); err != nil {
		t.Fatal(err)
	}
	const authKey = "hskey-auth-memory-only"
	if err := client.StartManaged(authKey, sipeedControlURL, "nano-kvm", true); err != nil {
		t.Fatal(err)
	}

	want := []capturedRequest{
		{method: http.MethodGet, path: tailscaleLocalAPIBasePath + tailscaleLocalAPIProfilesPath, host: tailscaleLocalAPIHost},
		{method: http.MethodGet, path: tailscaleLocalAPIBasePath + tailscaleLocalAPICurrentProfilePath, host: tailscaleLocalAPIHost},
		{method: http.MethodPut, path: tailscaleLocalAPIBasePath + tailscaleLocalAPIProfilesPath, host: tailscaleLocalAPIHost},
		{method: http.MethodPost, path: tailscaleLocalAPIBasePath + tailscaleLocalAPIProfilesPath + "1ab3", host: tailscaleLocalAPIHost},
		{
			method:      http.MethodPost,
			path:        tailscaleLocalAPIBasePath + tailscaleLocalAPIStartPath,
			host:        tailscaleLocalAPIHost,
			contentType: "application/json",
			body:        `{"UpdatePrefs":{"ControlURL":"` + sipeedControlURL + `","RouteAll":false,"CorpDNS":false,"WantRunning":true,"Hostname":"nano-kvm"},"AuthKey":"` + authKey + `"}`,
		},
		{
			method: http.MethodPost,
			path:   tailscaleLocalAPIBasePath + tailscaleLocalAPILoginInteractivePath,
			host:   tailscaleLocalAPIHost,
		},
	}
	if !reflect.DeepEqual(requests, want) {
		encoded, _ := json.MarshalIndent(requests, "", "  ")
		t.Fatalf("requests = %s, want %#v", encoded, want)
	}
}

func TestManagedLocalAPIRejectsInvalidProfileIDWithoutRequest(t *testing.T) {
	requested := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requested = true
	}))
	defer server.Close()
	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))
	for _, profileID := range []string{"", "../profile", "profile/id", "profile id"} {
		if err := client.SwitchProfile(profileID); err == nil {
			t.Fatalf("SwitchProfile(%q) succeeded", profileID)
		}
	}
	if requested {
		t.Fatal("invalid profile ID reached LocalAPI")
	}
}

func TestManagedLocalAPIStartErrorDoesNotLeakAuthKey(t *testing.T) {
	const authKey = "hskey-auth-do-not-log"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "start rejected")
	}))
	defer server.Close()
	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))
	err := client.StartManaged(authKey, sipeedControlURL, "nano-kvm", true)
	if err == nil {
		t.Fatal("StartManaged unexpectedly succeeded")
	}
	if strings.Contains(err.Error(), authKey) {
		t.Fatalf("StartManaged error leaked auth key: %v", err)
	}
}

func TestManagedLocalAPISkipsInteractiveLoginForExistingNodeKey(t *testing.T) {
	var starts, interactive int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case tailscaleLocalAPIBasePath + tailscaleLocalAPIStartPath:
			starts++
			w.WriteHeader(http.StatusNoContent)
		case tailscaleLocalAPIBasePath + tailscaleLocalAPILoginInteractivePath:
			interactive++
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))
	if err := client.StartManaged("hskey-auth-memory-only", sipeedControlURL, "nano-kvm", false); err != nil {
		t.Fatal(err)
	}
	if starts != 1 || interactive != 0 {
		t.Fatalf("managed reconnect requests = starts %d, interactive %d; want 1, 0", starts, interactive)
	}
}
