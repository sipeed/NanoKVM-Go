package tailscale

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSipeedPairingEnrollRequestContract(t *testing.T) {
	deviceID, _ := newSipeedCredential(sipeedDeviceIDBytes)
	token, _ := newSipeedCredential(sipeedDeviceTokenBytes)
	const authKey = "hskey-auth-test-secret"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != sipeedEnrollPath || request.URL.RawQuery != "" {
			t.Errorf("request = %s %s", request.Method, request.URL.String())
		}
		if got := request.Header.Get("Authorization"); got != "Bearer "+token {
			t.Errorf("Authorization = %q", got)
		}
		if got := request.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Errorf("Content-Type = %q", got)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if len(payload) != 2 || payload["device_id"] != deviceID || payload["hostname"] != "nano-kvm" {
			t.Errorf("enroll body = %s", body)
		}
		if strings.Contains(string(body), token) {
			t.Error("device token leaked into the enrollment body")
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = io.WriteString(w, `{"network_id":"net-test","device_id":"`+deviceID+`","auth_key":"`+authKey+`","expires_at":"2026-09-17T08:00:00Z"}`)
	}))
	defer server.Close()

	client := newHTTPSipeedPairingClient(server.URL, server.Client())
	response, err := client.Enroll(context.Background(), deviceID, token, "nano-kvm")
	if err != nil {
		t.Fatalf("Enroll() error = %v", err)
	}
	if response.NetworkID != "net-test" || response.DeviceID != deviceID || response.AuthKey != authKey {
		t.Fatalf("Enroll() response = %+v", response)
	}
}

func TestSipeedPairingJoinKeyRequestContract(t *testing.T) {
	token, _ := newSipeedCredential(sipeedDeviceTokenBytes)
	const authKey = "hskey-auth-test-secret"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != sipeedJoinKeyPath || request.URL.RawQuery != "" {
			t.Errorf("request = %s %s", request.Method, request.URL.String())
		}
		if got := request.Header.Get("Authorization"); got != "Bearer "+token {
			t.Errorf("Authorization = %q", got)
		}
		if got := request.Header.Get("Content-Type"); got != "" {
			t.Errorf("Content-Type = %q, want no request content type", got)
		}
		if request.ContentLength != 0 || len(request.TransferEncoding) != 0 {
			t.Errorf("join-key request carries a body: length=%d transfer=%v", request.ContentLength, request.TransferEncoding)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if len(body) != 0 {
			t.Errorf("join-key body = %q, want empty", body)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = io.WriteString(w, `{"auth_key":"`+authKey+`","expires_at":"2026-09-17T08:00:00Z"}`)
	}))
	defer server.Close()

	response, err := newHTTPSipeedPairingClient(server.URL, server.Client()).JoinKey(
		context.Background(),
		token,
	)
	if err != nil {
		t.Fatalf("JoinKey() error = %v", err)
	}
	if response.AuthKey != authKey || response.ExpiresAt != "2026-09-17T08:00:00Z" {
		t.Fatalf("JoinKey() response = %+v", response)
	}
}

func TestSipeedPairingJoinKeyRejectsInvalidSuccess(t *testing.T) {
	token, _ := newSipeedCredential(sipeedDeviceTokenBytes)
	for name, payload := range map[string]string{
		"missing auth key": `{"expires_at":"2026-09-17T08:00:00Z"}`,
		"invalid expiry":   `{"auth_key":"hskey-auth-test-secret","expires_at":"invalid"}`,
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, payload)
			}))
			defer server.Close()

			response, err := newHTTPSipeedPairingClient(server.URL, server.Client()).JoinKey(
				context.Background(),
				token,
			)
			if err == nil || response != nil {
				t.Fatalf("JoinKey() = %+v, %v; want invalid response error", response, err)
			}
			if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "hskey-auth-test-secret") {
				t.Fatalf("JoinKey() leaked a secret: %v", err)
			}
		})
	}
}

func TestSipeedPairingDeleteNetworkRequestContract(t *testing.T) {
	token, _ := newSipeedCredential(sipeedDeviceTokenBytes)
	const networkID = "net-test"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodDelete || request.URL.Path != sipeedNetworkPath || request.URL.RawQuery != "" {
			t.Errorf("request = %s %s", request.Method, request.URL.String())
		}
		if got := request.Header.Get("Authorization"); got != "Bearer "+token {
			t.Errorf("Authorization = %q", got)
		}
		if got := request.Header.Get("Content-Type"); got != "" {
			t.Errorf("Content-Type = %q, want no request content type", got)
		}
		if request.ContentLength != 0 || len(request.TransferEncoding) != 0 {
			t.Errorf("delete-network request carries a body: length=%d transfer=%v", request.ContentLength, request.TransferEncoding)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if len(body) != 0 {
			t.Errorf("delete-network body = %q, want empty", body)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = io.WriteString(w, `{"network_id":"net-test","destroyed":true,"devices_removed":3}`)
	}))
	defer server.Close()

	if err := newHTTPSipeedPairingClient(server.URL, server.Client()).DeleteNetwork(
		context.Background(),
		token,
		networkID,
	); err != nil {
		t.Fatalf("DeleteNetwork() error = %v", err)
	}
}

func TestSipeedPairingDeleteNetworkStatusHandling(t *testing.T) {
	token, _ := newSipeedCredential(sipeedDeviceTokenBytes)
	tests := []struct {
		name        string
		status      int
		code        string
		retryAfter  string
		wantSuccess bool
		wantRetry   time.Duration
	}{
		{name: "already destroyed", status: http.StatusUnauthorized, code: "invalid_token", wantSuccess: true},
		{name: "other unauthorized", status: http.StatusUnauthorized, code: "invalid_request"},
		{name: "not found", status: http.StatusNotFound, code: "not_found"},
		{name: "rate limited", status: http.StatusTooManyRequests, code: "rate_limited", retryAfter: "23", wantRetry: 23 * time.Second},
		{name: "internal failure", status: http.StatusInternalServerError, code: "internal_error"},
		{name: "upstream failure", status: http.StatusBadGateway, code: "upstream_error"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				if testCase.retryAfter != "" {
					w.Header().Set("Retry-After", testCase.retryAfter)
				}
				w.WriteHeader(testCase.status)
				_, _ = io.WriteString(w, `{"error":{"code":"`+testCase.code+`"}}`)
			}))
			defer server.Close()

			err := newHTTPSipeedPairingClient(server.URL, server.Client()).DeleteNetwork(
				context.Background(),
				token,
				"net-test",
			)
			if testCase.wantSuccess {
				if err != nil {
					t.Fatalf("DeleteNetwork() error = %v", err)
				}
			} else {
				if err == nil {
					t.Fatal("DeleteNetwork() unexpectedly succeeded")
				}
				var apiError *sipeedPairingAPIError
				if !errors.As(err, &apiError) || apiError.HTTPStatus != testCase.status ||
					apiError.RetryAfter != testCase.wantRetry {
					t.Fatalf("DeleteNetwork() error = %#v", err)
				}
			}
			if calls != 1 {
				t.Fatalf("DeleteNetwork() calls = %d, want 1", calls)
			}
		})
	}
}

func TestSipeedPairingDeleteNetworkRejectsInvalidSuccess(t *testing.T) {
	token, _ := newSipeedCredential(sipeedDeviceTokenBytes)
	tests := []struct {
		name    string
		status  int
		payload string
	}{
		{name: "missing network", status: http.StatusOK, payload: `{"destroyed":true,"devices_removed":1}`},
		{name: "wrong network", status: http.StatusOK, payload: `{"network_id":"other","destroyed":true,"devices_removed":1}`},
		{name: "not destroyed", status: http.StatusOK, payload: `{"network_id":"net-test","destroyed":false,"devices_removed":1}`},
		{name: "missing device count", status: http.StatusOK, payload: `{"network_id":"net-test","destroyed":true}`},
		{name: "negative device count", status: http.StatusOK, payload: `{"network_id":"net-test","destroyed":true,"devices_removed":-1}`},
		{name: "malformed JSON", status: http.StatusOK, payload: `{"network_id":`},
		{name: "trailing JSON", status: http.StatusOK, payload: `{"network_id":"net-test","destroyed":true,"devices_removed":0}{}`},
		{name: "non-200 success", status: http.StatusCreated, payload: `{"network_id":"net-test","destroyed":true,"devices_removed":1}`},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(testCase.status)
				_, _ = io.WriteString(w, testCase.payload)
			}))
			defer server.Close()

			err := newHTTPSipeedPairingClient(server.URL, server.Client()).DeleteNetwork(
				context.Background(),
				token,
				"net-test",
			)
			if err == nil {
				t.Fatal("DeleteNetwork() accepted an invalid success response")
			}
			if strings.Contains(err.Error(), token) {
				t.Fatalf("DeleteNetwork() leaked the bearer token: %v", err)
			}
		})
	}
}

func TestSipeedPairingDeleteNetworkHonorsContextTimeout(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
	}))
	defer server.Close()

	token, _ := newSipeedCredential(sipeedDeviceTokenBytes)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- newHTTPSipeedPairingClient(server.URL, server.Client()).DeleteNetwork(
			ctx,
			token,
			"net-test",
		)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("DeleteNetwork() request did not reach the server")
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("DeleteNetwork() error = %v, want deadline exceeded", err)
		}
	case <-time.After(time.Second):
		t.Fatal("DeleteNetwork() did not stop after its context timed out")
	}
}

func TestSipeedPairingSelfNodeRequestContract(t *testing.T) {
	token, _ := newSipeedCredential(sipeedDeviceTokenBytes)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != sipeedSelfNodePath {
			t.Errorf("request = %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer "+token {
			t.Error("missing bearer token")
		}
		body, _ := io.ReadAll(request.Body)
		if string(body) != `{"ip":"100.64.0.3"}` {
			t.Errorf("self-node body = %s", body)
		}
		if strings.Contains(string(body), token) {
			t.Error("device token leaked into the self-node body")
		}
		_, _ = io.WriteString(w, `{"node_id":"3","ip":"100.64.0.3"}`)
	}))
	defer server.Close()

	client := newHTTPSipeedPairingClient(server.URL, server.Client())
	nodeID, err := client.ReportSelfNode(context.Background(), token, "100.64.0.3")
	if err != nil {
		t.Fatalf("ReportSelfNode() error = %v", err)
	}
	if nodeID != "3" {
		t.Fatalf("ReportSelfNode() node ID = %q", nodeID)
	}
}

func TestSipeedPairingErrorMapping(t *testing.T) {
	token, _ := newSipeedCredential(sipeedDeviceTokenBytes)
	deviceID, _ := newSipeedCredential(sipeedDeviceIDBytes)
	tests := []struct {
		name        string
		status      int
		code        string
		retryAfter  string
		wantCode    string
		wantInvalid bool
		wantRetry   time.Duration
	}{
		{name: "invalid token", status: 401, code: "invalid_token", wantCode: "invalidToken", wantInvalid: true},
		{name: "rate limited", status: 429, code: "rate_limited", retryAfter: "17", wantCode: "rateLimited", wantRetry: 17 * time.Second},
		{name: "conflict is protocol failure", status: 409, code: "already_enrolled", wantCode: "protocolError"},
		{name: "upstream", status: 502, code: "upstream_error", wantCode: "pairingUpstreamError"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if testCase.retryAfter != "" {
					w.Header().Set("Retry-After", testCase.retryAfter)
				}
				w.WriteHeader(testCase.status)
				_, _ = io.WriteString(w, `{"error":{"code":"`+testCase.code+`","message":"must not be surfaced"}}`)
			}))
			defer server.Close()

			client := newHTTPSipeedPairingClient(server.URL, server.Client())
			_, err := client.Enroll(context.Background(), deviceID, token, "nano-kvm")
			if err == nil {
				t.Fatal("Enroll() unexpectedly succeeded")
			}
			code, invalid, retry := managedErrorFromPairing(err)
			if code != testCase.wantCode || invalid != testCase.wantInvalid || retry != testCase.wantRetry {
				t.Fatalf("mapping = %q, %v, %v", code, invalid, retry)
			}
			if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "must not be surfaced") {
				t.Fatalf("error leaked a secret or remote message: %v", err)
			}
		})
	}
}

func TestSipeedPairingRejectsMalformedSuccessWithoutLeakingSecrets(t *testing.T) {
	token, _ := newSipeedCredential(sipeedDeviceTokenBytes)
	deviceID, _ := newSipeedCredential(sipeedDeviceIDBytes)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"network_id":"net-test","device_id":"wrong","auth_key":"secret","expires_at":"bad"}`)
	}))
	defer server.Close()

	_, err := newHTTPSipeedPairingClient(server.URL, server.Client()).Enroll(
		context.Background(),
		deviceID,
		token,
		"nano-kvm",
	)
	if err == nil {
		t.Fatal("malformed response was accepted")
	}
	if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("malformed response error leaked secrets: %v", err)
	}
}

func TestSipeedPairingDoesNotSurfaceBearerEchoedAsRemoteErrorCode(t *testing.T) {
	token, _ := newSipeedCredential(sipeedDeviceTokenBytes)
	deviceID, _ := newSipeedCredential(sipeedDeviceIDBytes)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":{"code":"`+token+`"}}`)
	}))
	defer server.Close()

	_, err := newHTTPSipeedPairingClient(server.URL, server.Client()).Enroll(
		context.Background(),
		deviceID,
		token,
		"nano-kvm",
	)
	if err == nil || strings.Contains(err.Error(), token) {
		t.Fatalf("remote error leaked bearer token: %v", err)
	}
}

func TestSipeedPairingNetworkFailureUsesStableError(t *testing.T) {
	token, _ := newSipeedCredential(sipeedDeviceTokenBytes)
	deviceID, _ := newSipeedCredential(sipeedDeviceIDBytes)
	client := newHTTPSipeedPairingClient("https://127.0.0.1:1", &http.Client{Timeout: 100 * time.Millisecond})
	_, err := client.Enroll(context.Background(), deviceID, token, "nano-kvm")
	if err == nil {
		t.Fatal("network failure unexpectedly succeeded")
	}
	code, invalid, retry := managedErrorFromPairing(err)
	if code != "pairingUnavailable" || invalid || retry != 0 {
		t.Fatalf("network failure mapping = %q, %v, %v", code, invalid, retry)
	}
	var apiError *sipeedPairingAPIError
	if errors.As(err, &apiError) {
		t.Fatalf("network failure unexpectedly became API error: %+v", apiError)
	}
}

func TestSipeedPairingDoesNotForwardBearerAcrossRedirects(t *testing.T) {
	token, _ := newSipeedCredential(sipeedDeviceTokenBytes)
	deviceID, _ := newSipeedCredential(sipeedDeviceIDBytes)
	redirectReached := false
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/redirect-target" {
			redirectReached = true
			if request.Header.Get("Authorization") != "" {
				t.Error("redirect target received the bearer token")
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, request, server.URL+"/redirect-target", http.StatusFound)
	}))
	defer server.Close()

	client := newHTTPSipeedPairingClient(server.URL, nil)
	if _, err := client.Enroll(context.Background(), deviceID, token, "nano-kvm"); err == nil {
		t.Fatal("redirect response was accepted")
	}
	if redirectReached {
		t.Fatal("pairing client followed a redirect")
	}
}

func TestNormalizeSipeedClaimCodeAcceptsOnlySixASCIIDigits(t *testing.T) {
	for input, want := range map[string]string{
		"123456":  "123456",
		"123 456": "123456",
		"123-456": "123456",
	} {
		got, err := normalizeSipeedClaimCode(input)
		if err != nil || got != want {
			t.Fatalf("normalizeSipeedClaimCode(%q) = %q, %v; want %q", input, got, err, want)
		}
	}

	for _, input := range []string{"ABCDEF", "12A456", "12345", "1234567", "１２３４５６"} {
		if got, err := normalizeSipeedClaimCode(input); err == nil {
			t.Fatalf("normalizeSipeedClaimCode(%q) = %q, nil; want error", input, got)
		}
	}
}

func TestSipeedManagementPairingRequestContracts(t *testing.T) {
	token, err := newSipeedCredential(sipeedDeviceTokenBytes)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		method   string
		path     string
		body     string
		response string
		call     func(*httpSipeedPairingClient) error
	}{
		{
			name:     "claim code",
			method:   http.MethodPost,
			path:     sipeedClaimCodePath,
			response: `{"code":"123 456","expires_at":"2026-09-18T08:00:00Z","single_use":true}`,
			call: func(client *httpSipeedPairingClient) error {
				response, err := client.ClaimCode(context.Background(), token)
				if err == nil && (response.Code != "123 456" || !response.SingleUse) {
					return errors.New("unexpected claim code response")
				}
				return err
			},
		},
		{
			name:     "pending long poll",
			method:   http.MethodGet,
			path:     sipeedPendingPath + "?wait=20",
			response: `{"pending":[]}`,
			call: func(client *httpSipeedPairingClient) error {
				response, err := client.Pending(context.Background(), token, 20)
				if err == nil && len(response.Pending) != 0 {
					return errors.New("unexpected pending response")
				}
				return err
			},
		},
		{
			name:     "approve pending",
			method:   http.MethodPost,
			path:     sipeedPendingPath + "/pending-1/approve",
			body:     `{"name":"laptop"}`,
			response: `{"node_id":"node-1","name":"laptop","ips":["100.64.0.4"]}`,
			call: func(client *httpSipeedPairingClient) error {
				response, err := client.ApprovePending(context.Background(), token, "pending-1", "laptop")
				if err == nil && response.NodeID != "node-1" {
					return errors.New("unexpected approval response")
				}
				return err
			},
		},
		{
			name:     "reject pending",
			method:   http.MethodPost,
			path:     sipeedPendingPath + "/pending-1/reject",
			response: `{"pending_id":"pending-1","rejected":true}`,
			call: func(client *httpSipeedPairingClient) error {
				response, err := client.RejectPending(context.Background(), token, "pending-1")
				if err == nil && !response.Rejected {
					return errors.New("unexpected rejection response")
				}
				return err
			},
		},
		{
			name:     "authorize backup code",
			method:   http.MethodPost,
			path:     sipeedAuthorizePath,
			body:     `{"code":"2345678A","name":"phone"}`,
			response: `{"node_id":"node-2","name":"phone","ips":["100.64.0.5"]}`,
			call: func(client *httpSipeedPairingClient) error {
				response, err := client.Authorize(context.Background(), token, "2345-678A", "", "phone")
				if err == nil && response.NodeID != "node-2" {
					return errors.New("unexpected authorization response")
				}
				return err
			},
		},
		{
			name:     "devices",
			method:   http.MethodGet,
			path:     sipeedDevicesPath,
			response: `{"network_id":"network-1","devices":[{"node_id":"node-1","name":"self","ips":["100.64.0.3"],"online":true,"last_seen":"2026-09-18T08:00:00Z","is_self":true}]}`,
			call: func(client *httpSipeedPairingClient) error {
				response, err := client.Devices(context.Background(), token)
				if err == nil && (response.NetworkID != "network-1" || len(response.Devices) != 1) {
					return errors.New("unexpected devices response")
				}
				return err
			},
		},
		{
			name:     "rename device",
			method:   http.MethodPost,
			path:     sipeedDevicePathPrefix + "node-2/name",
			body:     `{"name":"new-name"}`,
			response: `{"node_id":"node-2","name":"new-name"}`,
			call: func(client *httpSipeedPairingClient) error {
				response, err := client.RenameDevice(context.Background(), token, "node-2", "new-name")
				if err == nil && response.Name != "new-name" {
					return errors.New("unexpected rename response")
				}
				return err
			},
		},
		{
			name:     "remove device",
			method:   http.MethodDelete,
			path:     sipeedDevicePathPrefix + "node-2",
			response: `{"node_id":"node-2","removed":true}`,
			call: func(client *httpSipeedPairingClient) error {
				return client.DeleteDevice(context.Background(), token, "node-2")
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method != testCase.method || request.URL.String() != testCase.path {
					t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.String(), testCase.method, testCase.path)
				}
				if got := request.Header.Get("Authorization"); got != "Bearer "+token {
					t.Errorf("Authorization = %q", got)
				}
				body, readErr := io.ReadAll(request.Body)
				if readErr != nil {
					t.Errorf("read request body: %v", readErr)
				}
				if string(body) != testCase.body {
					t.Errorf("request body = %q, want %q", body, testCase.body)
				}
				if testCase.body == "" && request.Header.Get("Content-Type") != "" {
					t.Errorf("empty request has Content-Type %q", request.Header.Get("Content-Type"))
				}
				if strings.Contains(string(body), token) {
					t.Error("request body leaked bearer token")
				}
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				_, _ = io.WriteString(w, testCase.response)
			}))
			defer server.Close()

			if err := testCase.call(newHTTPSipeedPairingClient(server.URL, server.Client())); err != nil {
				t.Fatalf("call() error = %v", err)
			}
		})
	}
}

func TestSipeedPendingHonorsContextCancellation(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
	}))
	defer server.Close()

	token, err := newSipeedCredential(sipeedDeviceTokenBytes)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, requestErr := newHTTPSipeedPairingClient(server.URL, server.Client()).Pending(ctx, token, 20)
		result <- requestErr
	}()
	<-started
	cancel()
	select {
	case requestErr := <-result:
		if !errors.Is(requestErr, context.Canceled) {
			t.Fatalf("Pending() error = %v, want context canceled", requestErr)
		}
	case <-time.After(time.Second):
		t.Fatal("Pending() did not stop after cancellation")
	}
}
