package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	"golang.org/x/sys/unix"
)

type downloadRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn downloadRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func testResolver(records map[string][]string) ipResolver {
	return func(ctx context.Context, host string) ([]netip.Addr, error) {
		if addr, err := netip.ParseAddr(host); err == nil {
			return []netip.Addr{addr.Unmap()}, nil
		}

		values := records[host]
		addrs := make([]netip.Addr, 0, len(values))
		for _, value := range values {
			addrs = append(addrs, netip.MustParseAddr(value).Unmap())
		}
		return addrs, nil
	}
}

func TestParseImageDownloadStatus(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    proto.StatusImageRsp
	}{
		{
			name:    "legacy source only",
			content: "https://example.com/image.iso",
			want:    newImageDownloadStatus("in_progress", "https://example.com/image.iso", ""),
		},
		{
			name:    "legacy progress with semicolon in URL",
			content: "https://example.com/image.iso?token=one;two;12.50%",
			want:    newImageDownloadStatus("in_progress", "https://example.com/image.iso?token=one;two", "12.50%"),
		},
		{
			name:    "extended progress",
			content: "https://example.com/image.iso;25.00%;33554432;2886218023;13002342;220",
			want: proto.StatusImageRsp{
				Status:           "in_progress",
				File:             "https://example.com/image.iso",
				Percentage:       "25.00%",
				DownloadedBytes:  33554432,
				TotalBytes:       2886218023,
				BytesPerSecond:   13002342,
				RemainingSeconds: 220,
			},
		},
		{
			name:    "extended unknown total with semicolon in URL",
			content: "https://example.com/image.iso?token=one;two;;33554432;-1;13002342;-1",
			want: proto.StatusImageRsp{
				Status:           "in_progress",
				File:             "https://example.com/image.iso?token=one;two",
				DownloadedBytes:  33554432,
				TotalBytes:       -1,
				BytesPerSecond:   13002342,
				RemainingSeconds: -1,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := parseImageDownloadStatus(test.content); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("parseImageDownloadStatus() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestDownloadSpeedTracker(t *testing.T) {
	startedAt := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	tracker := downloadSpeedTracker{lastAt: startedAt}

	tests := []struct {
		name          string
		total         int64
		totalSize     int64
		at            time.Time
		wantSpeed     int64
		wantRemaining int64
	}{
		{
			name:          "active transfer",
			total:         100,
			totalSize:     1000,
			at:            startedAt.Add(2 * time.Second),
			wantSpeed:     50,
			wantRemaining: 18,
		},
		{
			name:          "stalled transfer",
			total:         100,
			totalSize:     1000,
			at:            startedAt.Add(4 * time.Second),
			wantSpeed:     0,
			wantRemaining: -1,
		},
		{
			name:          "transfer resumes from stalled baseline",
			total:         200,
			totalSize:     1000,
			at:            startedAt.Add(6 * time.Second),
			wantSpeed:     50,
			wantRemaining: 16,
		},
		{
			name:          "unknown total omits ETA",
			total:         300,
			totalSize:     -1,
			at:            startedAt.Add(8 * time.Second),
			wantSpeed:     50,
			wantRemaining: -1,
		},
		{
			name:          "completed transfer",
			total:         400,
			totalSize:     400,
			at:            startedAt.Add(10 * time.Second),
			wantSpeed:     50,
			wantRemaining: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotSpeed, gotRemaining := tracker.update(test.total, test.totalSize, test.at)
			if gotSpeed != test.wantSpeed || gotRemaining != test.wantRemaining {
				t.Fatalf(
					"update() = (%d, %d), want (%d, %d)",
					gotSpeed,
					gotRemaining,
					test.wantSpeed,
					test.wantRemaining,
				)
			}
		})
	}
}

func TestPauseAndResumeImageDownloadStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalSentinel := sentinelPath
	sentinelPath = filepath.Join(t.TempDir(), "download.sentinel")
	t.Cleanup(func() { sentinelPath = originalSentinel })

	lease, err := acquireDownloadLease(sentinelPath, "https://example.com/image.iso")
	if err != nil {
		t.Fatalf("acquireDownloadLease: %v", err)
	}
	service := &Service{}
	operation, err := service.beginImageDownload(lease)
	if err != nil {
		t.Fatalf("beginImageDownload: %v", err)
	}
	t.Cleanup(func() { service.finishImageDownload(operation) })

	requestStatus := func(handler func(*gin.Context), path string) proto.StatusImageRsp {
		t.Helper()
		recorder := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(recorder)
		ginContext.Request = httptest.NewRequest(http.MethodPost, path, nil)
		handler(ginContext)

		var response struct {
			Code int                  `json:"code"`
			Data proto.StatusImageRsp `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode %s response: %v", path, err)
		}
		if response.Code != 0 {
			t.Fatalf("%s response code = %d, want 0", path, response.Code)
		}
		return response.Data
	}

	paused := requestStatus(service.PauseImageDownload, "/api/storage/download/image/pause")
	if paused.Status != "paused" || !operation.isPaused() {
		t.Fatalf("pause status = %q, paused = %v", paused.Status, operation.isPaused())
	}
	status := requestStatus(service.StatusImage, "/api/storage/download/image/status")
	if status.Status != "paused" {
		t.Fatalf("polled status = %q, want paused", status.Status)
	}

	resumed := requestStatus(service.ResumeImageDownload, "/api/storage/download/image/resume")
	if resumed.Status != "in_progress" || operation.isPaused() {
		t.Fatalf("resume status = %q, paused = %v", resumed.Status, operation.isPaused())
	}
}

func TestLoggingWriterWaitsWhilePausedAndContinuesAfterResume(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	operation := &imageDownloadOperation{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	if !operation.pause() {
		t.Fatal("failed to pause operation")
	}

	var destination bytes.Buffer
	waitStarted := make(chan struct{})
	var waitOnce sync.Once
	writer := &loggingWriter{
		writer: &destination,
		waitForResume: func() error {
			waitOnce.Do(func() { close(waitStarted) })
			return operation.waitUntilResumed()
		},
	}
	result := make(chan error, 1)
	go func() {
		_, err := writer.Write([]byte("image data"))
		result <- err
	}()

	<-waitStarted
	select {
	case err := <-result:
		t.Fatalf("paused write returned early: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if destination.Len() != 0 {
		t.Fatalf("paused destination bytes = %d, want 0", destination.Len())
	}

	if !operation.resume() {
		t.Fatal("failed to resume operation")
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("resumed write error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("resumed write did not finish")
	}
	if got := destination.String(); got != "image data" {
		t.Fatalf("resumed destination = %q, want image data", got)
	}
}

func TestPausedDownloadUnblocksOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	operation := &imageDownloadOperation{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	if !operation.pause() {
		t.Fatal("failed to pause operation")
	}

	result := make(chan error, 1)
	go func() { result <- operation.waitUntilResumed() }()
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("paused wait error = %v, want context cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled paused download remained blocked")
	}
}

func TestCancelImageDownloadStopsActiveOperationAndReleasesLease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalSentinel := sentinelPath
	sentinelPath = filepath.Join(t.TempDir(), "download.sentinel")
	t.Cleanup(func() { sentinelPath = originalSentinel })

	lease, err := acquireDownloadLease(sentinelPath, "https://example.com/image.iso")
	if err != nil {
		t.Fatalf("acquireDownloadLease: %v", err)
	}
	service := &Service{}
	operation, err := service.beginImageDownload(lease)
	if err != nil {
		t.Fatalf("beginImageDownload: %v", err)
	}
	go func() {
		<-operation.ctx.Done()
		service.finishImageDownload(operation)
	}()

	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/storage/download/image/cancel",
		nil,
	)
	service.CancelImageDownload(ginContext)

	var response struct {
		Code int                  `json:"code"`
		Data proto.StatusImageRsp `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode cancel response: %v", err)
	}
	if response.Code != 0 || response.Data.Status != "idle" {
		t.Fatalf("cancel response = %#v, want successful idle status", response)
	}
	if service.currentImageDownload() != nil {
		t.Fatal("active download remains after cancellation")
	}
	if _, err := os.Stat(sentinelPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("sentinel remains after cancellation: %v", err)
	}
}

func TestValidateDownloadURLAllowsHTTPAndHTTPSPublicTargets(t *testing.T) {
	resolve := testResolver(map[string][]string{
		"example.com": {"93.184.216.34"},
	})

	cases := []string{
		"http://example.com/image.iso",
		"https://example.com/image.iso",
		"http://93.184.216.34/image.iso",
	}

	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			if _, err := validateDownloadURL(tc, resolve, downloadURLPolicy{}); err != nil {
				t.Fatalf("expected URL to be allowed: %v", err)
			}
		})
	}
}

func TestValidateDownloadURLRejectsUnsafeTargets(t *testing.T) {
	resolve := testResolver(map[string][]string{
		"private.example": {"10.0.0.1"},
		"mixed.example":   {"93.184.216.34", "192.168.1.10"},
	})

	cases := []string{
		"http://127.0.0.1/image.iso",
		"http://[::1]/image.iso",
		"http://0.0.0.0/image.iso",
		"http://10.0.0.1/image.iso",
		"http://172.16.0.1/image.iso",
		"http://192.168.1.1/image.iso",
		"http://169.254.169.254/latest/meta-data/",
		"http://100.64.0.1/image.iso",
		"http://198.18.0.1/image.iso",
		"http://203.0.113.10/image.iso",
		"http://240.0.0.1/image.iso",
		"http://[fe80::1]/image.iso",
		"http://[fc00::1]/image.iso",
		"http://[2001:db8::1]/image.iso",
		"http://224.0.0.1/image.iso",
		"file:///etc/passwd",
		"gopher://127.0.0.1:22/_test",
		"ftp://example.com/image.iso",
		"http://private.example/image.iso",
		"http://mixed.example/image.iso",
	}

	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			if _, err := validateDownloadURL(tc, resolve, downloadURLPolicy{}); err == nil {
				t.Fatal("expected URL to be rejected")
			}
		})
	}
}

func TestValidateDownloadURLAllowsConfiguredPrivateTargets(t *testing.T) {
	resolve := testResolver(map[string][]string{
		"nas.local": {"192.168.1.20"},
	})

	hostPolicy := downloadURLPolicy{
		AllowedHosts: []string{"nas.local"},
	}
	if _, err := validateDownloadURL("http://nas.local/image.iso", resolve, hostPolicy); err != nil {
		t.Fatalf("expected host allowlist to permit private URL: %v", err)
	}

	cidrPolicy := downloadURLPolicy{
		AllowedCIDRs: []netip.Prefix{netip.MustParsePrefix("192.168.1.0/24")},
	}
	if _, err := validateDownloadURL("http://192.168.1.20/image.iso", resolve, cidrPolicy); err != nil {
		t.Fatalf("expected CIDR allowlist to permit private URL: %v", err)
	}
}

func TestValidateDownloadURLKeepsLoopbackBlockedEvenWhenConfigured(t *testing.T) {
	policy := downloadURLPolicy{
		AllowPrivate: true,
		AllowedCIDRs: []netip.Prefix{
			netip.MustParsePrefix("127.0.0.0/8"),
		},
	}

	if _, err := validateDownloadURL("http://127.0.0.1/image.iso", testResolver(nil), policy); err == nil {
		t.Fatal("expected loopback to remain blocked")
	}
}

func TestDownloadHTTPClientRejectsRedirectToPrivateAddress(t *testing.T) {
	resolve := testResolver(map[string][]string{
		"example.com": {"93.184.216.34"},
	})
	client := newDownloadHTTPClient(resolve, downloadURLPolicy{})

	redirectReq := &http.Request{URL: &url.URL{Scheme: "http", Host: "127.0.0.1", Path: "/internal"}}
	previousReq := &http.Request{URL: &url.URL{Scheme: "http", Host: "example.com", Path: "/image.iso"}}

	if err := client.CheckRedirect(redirectReq, []*http.Request{previousReq}); err == nil {
		t.Fatal("expected redirect to private address to be rejected")
	}
}

func TestDoDownloadRequestWithHTTPFallback(t *testing.T) {
	const rawURL = "https://downloads.example:8443/releases/Windows%2011.iso?token=a%2Fb&channel=preview"

	t.Run("HTTPS success does not request HTTP", func(t *testing.T) {
		var requested []*http.Request
		client := downloadTestClient(func(request *http.Request) (*http.Response, error) {
			requested = append(requested, request.Clone(request.Context()))
			return downloadTestResponse(request, "https"), nil
		})
		request, err := http.NewRequest(http.MethodHead, rawURL, nil)
		if err != nil {
			t.Fatal(err)
		}

		response, err := doDownloadRequestWithHTTPFallback(client, request)
		if err != nil {
			t.Fatalf("HTTPS request: %v", err)
		}
		defer response.Body.Close()

		if len(requested) != 1 || requested[0].URL.Scheme != "https" {
			t.Fatalf("requests = %v, want one HTTPS request", downloadRequestURLs(requested))
		}
	})

	t.Run("HTTPS failure retries identical HEAD over HTTP", func(t *testing.T) {
		var requested []*http.Request
		type contextKey struct{}
		contextValue := &struct{}{}
		client := downloadTestClient(func(request *http.Request) (*http.Response, error) {
			requested = append(requested, request.Clone(request.Context()))
			if len(requested) == 1 {
				return nil, errors.New("TLS certificate is not yet valid")
			}
			return downloadTestResponse(request, "http"), nil
		})
		request, err := http.NewRequestWithContext(
			context.WithValue(context.Background(), contextKey{}, contextValue),
			http.MethodHead,
			rawURL,
			nil,
		)
		if err != nil {
			t.Fatal(err)
		}

		response, err := doDownloadRequestWithHTTPFallback(client, request)
		if err != nil {
			t.Fatalf("HTTP fallback: %v", err)
		}
		defer response.Body.Close()

		assertDownloadHTTPSFallbackRequests(t, requested, http.MethodHead)
		if requested[1].Context().Value(contextKey{}) != contextValue {
			t.Fatal("HTTP fallback did not preserve the request context")
		}
	})

	t.Run("HTTPS and HTTP failures return an error", func(t *testing.T) {
		var requested []*http.Request
		client := downloadTestClient(func(request *http.Request) (*http.Response, error) {
			requested = append(requested, request.Clone(request.Context()))
			return nil, errors.New("network failed")
		})
		request, err := http.NewRequest(http.MethodHead, rawURL, nil)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := doDownloadRequestWithHTTPFallback(client, request); err == nil {
			t.Fatal("HTTPS and HTTP failures were accepted")
		}
		assertDownloadHTTPSFallbackRequests(t, requested, http.MethodHead)
	})

	t.Run("plain HTTP failure is not retried", func(t *testing.T) {
		var requested []*http.Request
		client := downloadTestClient(func(request *http.Request) (*http.Response, error) {
			requested = append(requested, request.Clone(request.Context()))
			return nil, errors.New("network failed")
		})
		request, err := http.NewRequest(
			http.MethodHead,
			"http://downloads.example:8080/releases/image.iso?token=a%2Fb",
			nil,
		)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := doDownloadRequestWithHTTPFallback(client, request); err == nil {
			t.Fatal("HTTP failure was accepted")
		}
		if len(requested) != 1 || requested[0].URL.Scheme != "http" {
			t.Fatalf("requests = %v, want one HTTP request", downloadRequestURLs(requested))
		}
	})
}

func TestGetWithNoProgressTimeoutUsesHTTPFallbackAndWrapsBody(t *testing.T) {
	const rawURL = "https://downloads.example:8443/releases/image.iso?token=a%2Fb"
	payload := []byte("image payload")
	var requested []*http.Request
	client := downloadTestClient(func(request *http.Request) (*http.Response, error) {
		requested = append(requested, request.Clone(request.Context()))
		if len(requested) == 1 {
			return nil, errors.New("TLS certificate is not yet valid")
		}
		response := downloadTestResponse(request, string(payload))
		response.ContentLength = int64(len(payload))
		return response, nil
	})

	response, err := getWithNoProgressTimeout(client, rawURL, time.Minute)
	if err != nil {
		t.Fatalf("GET through HTTP fallback: %v", err)
	}
	defer response.Body.Close()

	assertDownloadHTTPSFallbackRequests(t, requested, http.MethodGet)
	if _, ok := response.Body.(*noProgressReadCloser); !ok {
		t.Fatalf("response body type = %T, want *noProgressReadCloser", response.Body)
	}
	got, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read fallback response: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("body = %q, want %q", got, payload)
	}
}

func TestSaveDownloadWithResumeCompletesAfterUnexpectedEOF(t *testing.T) {
	payload := []byte("complete image payload")
	split := 8
	requests := 0
	client := downloadTestClient(func(request *http.Request) (*http.Response, error) {
		requests++
		switch requests {
		case 1:
			if got := request.Header.Get("Range"); got != "" {
				t.Fatalf("initial Range = %q, want empty", got)
			}
			return downloadBodyResponse(
				request,
				http.StatusOK,
				int64(len(payload)),
				io.NopCloser(io.MultiReader(
					bytes.NewReader(payload[:split]),
					downloadErrorReader{err: io.ErrUnexpectedEOF},
				)),
			), nil
		case 2:
			if got, want := request.Header.Get("Range"), "bytes=8-"; got != want {
				t.Fatalf("resume Range = %q, want %q", got, want)
			}
			response := downloadBodyResponse(
				request,
				http.StatusPartialContent,
				int64(len(payload)-split),
				io.NopCloser(bytes.NewReader(payload[split:])),
			)
			response.Header.Set(
				"Content-Range",
				fmt.Sprintf("bytes %d-%d/%d", split, len(payload)-1, len(payload)),
			)
			return response, nil
		default:
			t.Fatalf("unexpected request %d", requests)
			return nil, errors.New("unexpected request")
		}
	})

	file, progress := newDownloadTestDestination(t)
	if err := saveDownloadWithResume(client, "http://example.com/image.iso", time.Minute, file, progress); err != nil {
		t.Fatalf("save download: %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	assertDownloadOpenFileContents(t, file, payload)
}

func TestSaveDownloadWithResumeRejectsMismatchedContentRange(t *testing.T) {
	payload := []byte("0123456789")

	tests := []struct {
		name         string
		contentRange string
		wantError    string
	}{
		{
			name:         "start",
			contentRange: "bytes 3-9/10",
			wantError:    "starts at 3, expected 4",
		},
		{
			name:         "total",
			contentRange: "bytes 4-10/11",
			wantError:    "total is 11, expected 10",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			client := downloadTestClient(func(request *http.Request) (*http.Response, error) {
				requests++
				if requests == 1 {
					return downloadBodyResponse(
						request,
						http.StatusOK,
						int64(len(payload)),
						io.NopCloser(io.MultiReader(
							bytes.NewReader(payload[:4]),
							downloadErrorReader{err: io.ErrUnexpectedEOF},
						)),
					), nil
				}

				response := downloadBodyResponse(
					request,
					http.StatusPartialContent,
					-1,
					io.NopCloser(bytes.NewReader(payload[4:])),
				)
				response.Header.Set("Content-Range", test.contentRange)
				return response, nil
			})

			file, progress := newDownloadTestDestination(t)
			err := saveDownloadWithResume(
				client,
				"http://example.com/image.iso",
				time.Minute,
				file,
				progress,
			)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want substring %q", err, test.wantError)
			}
			if requests != 2 {
				t.Fatalf("requests = %d, want 2", requests)
			}
		})
	}
}

func TestAdmitDownloadResponseRejectsDeclaredSizeAndInsufficientBudget(t *testing.T) {
	dir := t.TempDir()
	setDownloadTestEnvironment(t, dir, 100, 1)
	policy := testDownloadPolicy(10)

	response := &http.Response{StatusCode: http.StatusOK, ContentLength: 11}
	if _, err := admitDownloadResponse(response, dir, policy); !errors.Is(err, errDownloadSizeLimitExceeded) {
		t.Fatalf("admitDownloadResponse error = %v, want size-limit error", err)
	}

	policy = testDownloadPolicy(100)
	policy.ReservedFreeBytes = 10
	response.ContentLength = 11
	setDownloadStatfs(t, 20, 1)
	if _, err := admitDownloadResponse(response, dir, policy); !errors.Is(err, errDownloadInsufficientSpace) {
		t.Fatalf("admitDownloadResponse error = %v, want insufficient-space error", err)
	}
}

func TestDownloadImageBudgetRejectsOverflowingFilesystemCapacity(t *testing.T) {
	dir := t.TempDir()
	setDownloadTestEnvironment(t, dir, ^uint64(0), 2)
	if _, err := downloadImageBudget(dir, testDownloadPolicy(10)); err == nil {
		t.Fatal("expected overflow error")
	}
}

func TestDownloadImageRejectsOversizedHEADWithoutGET(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	setDownloadTestEnvironment(t, dir, 100, 1)
	originalSentinel := sentinelPath
	sentinelPath = filepath.Join(t.TempDir(), "download.sentinel")
	t.Cleanup(func() { sentinelPath = originalSentinel })

	getRequests := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method == http.MethodGet {
			getRequests++
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			ContentLength: 11,
			Body:          io.NopCloser(strings.NewReader("")),
			Request:       request,
		}, nil
	})}
	setDownloadHandlerDependencies(t, testDownloadPolicy(10), client)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/storage/download/image", strings.NewReader("file=https%3A%2F%2Fexample.com%2Fimage.iso"))
	context.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	(&Service{}).DownloadImage(context)

	assertDownloadImageHandlerResponse(t, recorder, -1, "failed when checking the url")
	if getRequests != 0 {
		t.Fatalf("GET requests = %d, want 0 after oversized HEAD", getRequests)
	}
	if _, err := os.Stat(sentinelPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("sentinel remains after rejected HEAD: %v", err)
	}
}

func TestDownloadImageRejectsInsufficientBudgetWithoutRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	setDownloadTestEnvironment(t, dir, 10, 1)
	originalSentinel := sentinelPath
	sentinelPath = filepath.Join(t.TempDir(), "download.sentinel")
	t.Cleanup(func() { sentinelPath = originalSentinel })

	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode:    http.StatusOK,
			ContentLength: 1,
			Body:          io.NopCloser(strings.NewReader("")),
			Request:       request,
		}, nil
	})}
	policy := testDownloadPolicy(10)
	policy.ReservedFreeBytes = 10
	setDownloadHandlerDependencies(t, policy, client)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/storage/download/image", strings.NewReader("file=https%3A%2F%2Fexample.com%2Fimage.iso"))
	context.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	(&Service{}).DownloadImage(context)

	assertDownloadImageHandlerResponse(t, recorder, -1, downloadInsufficientStorageResponseMessage)
	if requests != 0 {
		t.Fatalf("requests = %d, want 0 after insufficient-budget rejection", requests)
	}
	if _, err := os.Stat(sentinelPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("sentinel remains after insufficient-budget rejection: %v", err)
	}
}

func TestDownloadImageRejectsInsufficientSpaceDeclaredByHEADWithoutGET(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	setDownloadTestEnvironment(t, dir, 20, 1)
	originalSentinel := sentinelPath
	sentinelPath = filepath.Join(t.TempDir(), "download.sentinel")
	t.Cleanup(func() { sentinelPath = originalSentinel })

	headRequests := 0
	getRequests := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.Method {
		case http.MethodHead:
			headRequests++
		case http.MethodGet:
			getRequests++
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			ContentLength: 11,
			Body:          io.NopCloser(strings.NewReader("")),
			Request:       request,
		}, nil
	})}
	policy := testDownloadPolicy(100)
	policy.ReservedFreeBytes = 10
	setDownloadHandlerDependencies(t, policy, client)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/storage/download/image", strings.NewReader("file=https%3A%2F%2Fexample.com%2Fimage.iso"))
	context.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	(&Service{}).DownloadImage(context)

	assertDownloadImageHandlerResponse(t, recorder, -1, downloadInsufficientStorageResponseMessage)
	if headRequests != 1 {
		t.Fatalf("HEAD requests = %d, want 1 before declared-size rejection", headRequests)
	}
	if getRequests != 0 {
		t.Fatalf("GET requests = %d, want 0 after insufficient-space HEAD rejection", getRequests)
	}
	if _, err := os.Stat(sentinelPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("sentinel remains after insufficient-space HEAD rejection: %v", err)
	}
}

func assertDownloadImageHandlerResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantCode int, wantMessage string) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("response status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode handler response: %v", err)
	}
	if response.Code != wantCode {
		t.Fatalf("response code = %d, want %d", response.Code, wantCode)
	}
	if response.Msg != wantMessage {
		t.Fatalf("response message = %q, want %q", response.Msg, wantMessage)
	}
}

func TestDownloadImageToDataRejectsChangedGETBeforeCreatingTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	setDownloadTestEnvironment(t, dir, 100, 1)
	oldPath := filepath.Join(dir, "image.iso")
	if err := os.WriteFile(oldPath, []byte("old image"), 0o600); err != nil {
		t.Fatalf("write old image: %v", err)
	}

	created := false
	originalCreateTemp := createDownloadTemp
	createDownloadTemp = func(string, string) (downloadFile, error) {
		created = true
		return nil, errors.New("temporary file should not be created")
	}
	t.Cleanup(func() { createDownloadTemp = originalCreateTemp })

	lease := acquireTestDownloadLease(t)
	err := downloadImageToData(
		staticDownloadClient(http.StatusOK, 11, []byte("01234567890")),
		mustParseDownloadURL(t, "https://example.com/image.iso"),
		testDownloadPolicy(10),
		lease,
	)
	if !errors.Is(err, errDownloadSizeLimitExceeded) {
		t.Fatalf("downloadImageToData error = %v, want size-limit error", err)
	}
	if created {
		t.Fatal("created a temporary file for a rejected GET response")
	}
	assertDownloadFileContents(t, oldPath, "old image")
}

func TestDownloadImageToDataRejectsNonOKGETBeforeCreatingTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	setDownloadTestEnvironment(t, dir, 100, 1)
	oldPath := filepath.Join(dir, "image.iso")
	if err := os.WriteFile(oldPath, []byte("old image"), 0o600); err != nil {
		t.Fatalf("write old image: %v", err)
	}

	created := false
	originalCreateTemp := createDownloadTemp
	createDownloadTemp = func(string, string) (downloadFile, error) {
		created = true
		return nil, errors.New("temporary file should not be created")
	}
	t.Cleanup(func() { createDownloadTemp = originalCreateTemp })

	lease := acquireTestDownloadLease(t)
	err := downloadImageToData(
		staticDownloadClient(http.StatusForbidden, 0, nil),
		mustParseDownloadURL(t, "https://example.com/image.iso"),
		testDownloadPolicy(10),
		lease,
	)
	if err == nil {
		t.Fatal("expected non-OK GET response to be rejected")
	}
	if created {
		t.Fatal("created a temporary file for a rejected GET response")
	}
	assertDownloadFileContents(t, oldPath, "old image")
}

func TestDownloadImageToDataRejectsUnknownAndFalseDeclaredOversizeBodies(t *testing.T) {
	cases := []struct {
		name          string
		contentLength int64
		body          []byte
	}{
		{name: "unknown length", contentLength: -1, body: []byte("012345")},
		{name: "false small length", contentLength: 3, body: []byte("012345")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			setDownloadTestEnvironment(t, dir, 100, 1)
			oldPath := filepath.Join(dir, "image.iso")
			if err := os.WriteFile(oldPath, []byte("old image"), 0o600); err != nil {
				t.Fatalf("write old image: %v", err)
			}

			lease := acquireTestDownloadLease(t)
			err := downloadImageToData(
				staticDownloadClient(http.StatusOK, tc.contentLength, tc.body),
				mustParseDownloadURL(t, "https://example.com/image.iso"),
				testDownloadPolicy(5),
				lease,
			)
			if !errors.Is(err, errDownloadSizeLimitExceeded) {
				t.Fatalf("downloadImageToData error = %v, want size-limit error", err)
			}
			assertDownloadFileContents(t, oldPath, "old image")
			assertNoDownloadTemporaryFiles(t, dir)
		})
	}
}

func TestLoggingWriterRejectsOversizedBlockBeforeWriting(t *testing.T) {
	var destination bytes.Buffer
	writer := &loggingWriter{writer: &destination, limit: 5}

	n, err := writer.Write([]byte("123456"))
	if !errors.Is(err, errDownloadSizeLimitExceeded) {
		t.Fatalf("Write error = %v, want size-limit error", err)
	}
	if n != 0 {
		t.Fatalf("Write bytes = %d, want 0", n)
	}
	if got := destination.Len(); got != 0 {
		t.Fatalf("destination bytes = %d, want 0", got)
	}
}

func TestDownloadImageToDataPublishesOnlyCompleteBoundedFile(t *testing.T) {
	dir := t.TempDir()
	setDownloadTestEnvironment(t, dir, 100, 1)
	destination := filepath.Join(dir, "image.iso")
	if err := os.WriteFile(destination, []byte("old image"), 0o600); err != nil {
		t.Fatalf("write old image: %v", err)
	}

	lease := acquireTestDownloadLease(t)
	if err := downloadImageToData(
		staticDownloadClient(http.StatusOK, 5, []byte("12345")),
		mustParseDownloadURL(t, "https://example.com/image.iso"),
		testDownloadPolicy(5),
		lease,
	); err != nil {
		t.Fatalf("downloadImageToData: %v", err)
	}

	assertDownloadFileContents(t, destination, "12345")
	assertNoDownloadTemporaryFiles(t, dir)
}

func TestDownloadImageToDataCancellationCleansTemporaryFileAndPreservesPublishedImage(t *testing.T) {
	dir := t.TempDir()
	setDownloadTestEnvironment(t, dir, 1024, 1)
	destination := filepath.Join(dir, "image.iso")
	if err := os.WriteFile(destination, []byte("old image"), 0o600); err != nil {
		t.Fatalf("write old image: %v", err)
	}

	readStarted := make(chan struct{})
	requests := 0
	client := downloadTestClient(func(request *http.Request) (*http.Response, error) {
		requests++
		return downloadBodyResponse(
			request,
			http.StatusOK,
			1024,
			&contextBlockingReadCloser{
				ctx:     request.Context(),
				started: readStarted,
			},
		), nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	lease := acquireTestDownloadLease(t)
	source := mustParseDownloadURL(t, "https://example.com/image.iso")
	result := make(chan error, 1)
	go func() {
		result <- downloadImageToDataWithContext(
			ctx,
			client,
			source,
			testDownloadPolicy(1024),
			lease,
		)
	}()

	select {
	case <-readStarted:
	case <-time.After(time.Second):
		t.Fatal("download did not begin reading the response body")
	}
	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("download error = %v, want context cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled download did not stop")
	}
	if requests != 1 {
		t.Fatalf("download requests = %d, want no retry after cancellation", requests)
	}
	assertDownloadFileContents(t, destination, "old image")
	assertNoDownloadTemporaryFiles(t, dir)
}

func TestDownloadImageToDataDoesNotPublishWhenResponseCloseFails(t *testing.T) {
	dir := t.TempDir()
	setDownloadTestEnvironment(t, dir, 100, 1)
	destination := filepath.Join(dir, "image.iso")
	if err := os.WriteFile(destination, []byte("old image"), 0o600); err != nil {
		t.Fatalf("write old image: %v", err)
	}

	closeErr := errors.New("response close failed")
	client := downloadTestClient(func(request *http.Request) (*http.Response, error) {
		return downloadBodyResponse(
			request,
			http.StatusOK,
			5,
			downloadCloseErrorReadCloser{
				ReadCloser: io.NopCloser(bytes.NewReader([]byte("12345"))),
				err:        closeErr,
			},
		), nil
	})

	lease := acquireTestDownloadLease(t)
	err := downloadImageToData(
		client,
		mustParseDownloadURL(t, "https://example.com/image.iso"),
		testDownloadPolicy(5),
		lease,
	)
	if !errors.Is(err, closeErr) {
		t.Fatalf("downloadImageToData error = %v, want response close error", err)
	}
	assertDownloadFileContents(t, destination, "old image")
	assertNoDownloadTemporaryFiles(t, dir)
}

func TestDownloadImageToDataResumesViaHTTPFallbackAndPublishesBoundedFile(t *testing.T) {
	dir := t.TempDir()
	setDownloadTestEnvironment(t, dir, 100, 1)
	destination := filepath.Join(dir, "image.iso")
	if err := os.WriteFile(destination, []byte("old image"), 0o600); err != nil {
		t.Fatalf("write old image: %v", err)
	}

	payload := []byte("complete image payload")
	split := 8
	requests := 0
	client := downloadTestClient(func(request *http.Request) (*http.Response, error) {
		requests++
		switch requests {
		case 1:
			if request.URL.Scheme != "https" || request.Header.Get("Range") != "" {
				t.Fatalf("initial request = %s with Range %q", request.URL, request.Header.Get("Range"))
			}
			return nil, errors.New("TLS certificate is not yet valid")
		case 2:
			if request.URL.Scheme != "http" || request.Header.Get("Range") != "" {
				t.Fatalf("fallback request = %s with Range %q", request.URL, request.Header.Get("Range"))
			}
			return downloadBodyResponse(
				request,
				http.StatusOK,
				int64(len(payload)),
				io.NopCloser(io.MultiReader(
					bytes.NewReader(payload[:split]),
					downloadErrorReader{err: io.ErrUnexpectedEOF},
				)),
			), nil
		case 3:
			if request.URL.Scheme != "https" || request.Header.Get("Range") != "bytes=8-" {
				t.Fatalf("resume request = %s with Range %q", request.URL, request.Header.Get("Range"))
			}
			return nil, errors.New("TLS certificate is not yet valid")
		case 4:
			if request.URL.Scheme != "http" || request.Header.Get("Range") != "bytes=8-" {
				t.Fatalf("resume fallback request = %s with Range %q", request.URL, request.Header.Get("Range"))
			}
			response := downloadBodyResponse(
				request,
				http.StatusPartialContent,
				int64(len(payload)-split),
				io.NopCloser(bytes.NewReader(payload[split:])),
			)
			response.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", split, len(payload)-1, len(payload)))
			return response, nil
		default:
			t.Fatalf("unexpected request %d", requests)
			return nil, errors.New("unexpected request")
		}
	})

	lease := acquireTestDownloadLease(t)
	if err := downloadImageToData(
		client,
		mustParseDownloadURL(t, "https://example.com/image.iso"),
		testDownloadPolicy(int64(len(payload))),
		lease,
	); err != nil {
		t.Fatalf("downloadImageToData: %v", err)
	}
	if requests != 4 {
		t.Fatalf("requests = %d, want 4", requests)
	}
	assertDownloadFileContents(t, destination, string(payload))
	assertNoDownloadTemporaryFiles(t, dir)
}

func TestDownloadImageToDataCleansTemporaryFileOnSyncCloseAndRenameFailure(t *testing.T) {
	cases := []struct {
		name          string
		configureFile func(t *testing.T, dir string)
	}{
		{
			name: "sync failure",
			configureFile: func(t *testing.T, dir string) {
				setFakeDownloadTempFile(t, filepath.Join(dir, ".image-download-sync"), errors.New("sync failed"), nil)
			},
		},
		{
			name: "close failure",
			configureFile: func(t *testing.T, dir string) {
				setFakeDownloadTempFile(t, filepath.Join(dir, ".image-download-close"), nil, errors.New("close failed"))
			},
		},
		{
			name: "rename failure",
			configureFile: func(t *testing.T, _ string) {
				originalRename := renameDownloadFile
				renameDownloadFile = func(string, string) error { return errors.New("rename failed") }
				t.Cleanup(func() { renameDownloadFile = originalRename })
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			setDownloadTestEnvironment(t, dir, 100, 1)
			destination := filepath.Join(dir, "image.iso")
			if err := os.WriteFile(destination, []byte("old image"), 0o600); err != nil {
				t.Fatalf("write old image: %v", err)
			}
			tc.configureFile(t, dir)

			lease := acquireTestDownloadLease(t)
			err := downloadImageToData(
				staticDownloadClient(http.StatusOK, 5, []byte("12345")),
				mustParseDownloadURL(t, "https://example.com/image.iso"),
				testDownloadPolicy(5),
				lease,
			)
			if err == nil {
				t.Fatal("expected download failure")
			}
			assertDownloadFileContents(t, destination, "old image")
			assertNoDownloadTemporaryFiles(t, dir)
		})
	}
}

func TestSaveDownloadWithResumeRestartsWhenRangeIsIgnored(t *testing.T) {
	payload := []byte("complete image payload")
	split := 7
	requests := 0
	client := downloadTestClient(func(request *http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			return downloadBodyResponse(
				request,
				http.StatusOK,
				int64(len(payload)),
				io.NopCloser(io.MultiReader(
					bytes.NewReader(payload[:split]),
					downloadErrorReader{err: io.ErrUnexpectedEOF},
				)),
			), nil
		}

		if got, want := request.Header.Get("Range"), "bytes=7-"; got != want {
			t.Fatalf("resume Range = %q, want %q", got, want)
		}
		return downloadBodyResponse(
			request,
			http.StatusOK,
			int64(len(payload)),
			io.NopCloser(bytes.NewReader(payload)),
		), nil
	})

	file, progress := newDownloadTestDestination(t)
	if err := saveDownloadWithResume(client, "http://example.com/image.iso", time.Minute, file, progress); err != nil {
		t.Fatalf("save download: %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	assertDownloadOpenFileContents(t, file, payload)
}

func TestSaveDownloadWithResumeHasHardAttemptLimit(t *testing.T) {
	payload := []byte("0123456789")
	requests := 0
	client := downloadTestClient(func(request *http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			return downloadBodyResponse(
				request,
				http.StatusOK,
				int64(len(payload)),
				io.NopCloser(io.MultiReader(
					bytes.NewReader(payload[:1]),
					downloadErrorReader{err: io.ErrUnexpectedEOF},
				)),
			), nil
		}

		response := downloadBodyResponse(
			request,
			http.StatusPartialContent,
			int64(len(payload)-1),
			io.NopCloser(downloadErrorReader{err: io.ErrUnexpectedEOF}),
		)
		response.Header.Set("Content-Range", "bytes 1-9/10")
		return response, nil
	})

	file, progress := newDownloadTestDestination(t)
	err := saveDownloadWithResume(client, "http://example.com/image.iso", time.Minute, file, progress)
	wantError := fmt.Sprintf("after %d attempts", maxDownloadAttempts)
	if err == nil || !strings.Contains(err.Error(), wantError) {
		t.Fatalf("error = %v, want hard-attempt-limit error", err)
	}
	if requests != maxDownloadAttempts {
		t.Fatalf("requests = %d, want %d", requests, maxDownloadAttempts)
	}
}

func TestSaveDownloadWithResumeDoesNotRetryDiskWriteErrors(t *testing.T) {
	payload := []byte("image payload")
	requests := 0
	client := downloadTestClient(func(request *http.Request) (*http.Response, error) {
		requests++
		return downloadBodyResponse(
			request,
			http.StatusOK,
			int64(len(payload)),
			io.NopCloser(bytes.NewReader(payload)),
		), nil
	})

	file, _ := newDownloadTestDestination(t)
	progress := &loggingWriter{writer: downloadFailingWriter{err: syscall.ENOSPC}}
	err := saveDownloadWithResume(client, "http://example.com/image.iso", time.Minute, file, progress)
	if !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("error = %v, want ENOSPC", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

type downloadErrorReader struct {
	err error
}

func (reader downloadErrorReader) Read([]byte) (int, error) {
	return 0, reader.err
}

type downloadCloseErrorReadCloser struct {
	io.ReadCloser
	err error
}

func (reader downloadCloseErrorReadCloser) Close() error {
	return errors.Join(reader.ReadCloser.Close(), reader.err)
}

type contextBlockingReadCloser struct {
	ctx     context.Context
	started chan struct{}
	once    sync.Once
}

func (reader *contextBlockingReadCloser) Read([]byte) (int, error) {
	reader.once.Do(func() { close(reader.started) })
	<-reader.ctx.Done()
	return 0, reader.ctx.Err()
}

func (reader *contextBlockingReadCloser) Close() error {
	return nil
}

type downloadFailingWriter struct {
	err error
}

func (writer downloadFailingWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func downloadBodyResponse(
	request *http.Request,
	statusCode int,
	contentLength int64,
	body io.ReadCloser,
) *http.Response {
	return &http.Response{
		StatusCode:    statusCode,
		Status:        fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Header:        make(http.Header),
		Body:          body,
		ContentLength: contentLength,
		Request:       request,
	}
}

func newDownloadTestDestination(t *testing.T) (*os.File, *loggingWriter) {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "download-*.iso")
	if err != nil {
		t.Fatalf("create destination file: %v", err)
	}
	t.Cleanup(func() {
		_ = file.Close()
	})
	return file, &loggingWriter{writer: file}
}

func assertDownloadOpenFileContents(t *testing.T, file *os.File, want []byte) {
	t.Helper()

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("seek destination file: %v", err)
	}
	got, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read destination file: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("destination = %q, want %q", got, want)
	}
}

func downloadTestClient(roundTrip downloadRoundTripFunc) *http.Client {
	return &http.Client{Transport: roundTrip}
}

func downloadTestResponse(request *http.Request, body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Request:    request,
	}
}

func assertDownloadHTTPSFallbackRequests(t *testing.T, requested []*http.Request, method string) {
	t.Helper()
	if len(requested) != 2 {
		t.Fatalf("requests = %d, want HTTPS followed by HTTP", len(requested))
	}

	httpsRequest := requested[0]
	httpRequest := requested[1]
	if httpsRequest.Method != method || httpRequest.Method != method {
		t.Fatalf("methods = %q, %q; want %q", httpsRequest.Method, httpRequest.Method, method)
	}
	if httpsRequest.URL.Scheme != "https" || httpRequest.URL.Scheme != "http" {
		t.Fatalf(
			"schemes = %q, %q; want https, http",
			httpsRequest.URL.Scheme,
			httpRequest.URL.Scheme,
		)
	}
	if !reflect.DeepEqual(
		[]string{httpRequest.URL.Host, httpRequest.URL.EscapedPath(), httpRequest.URL.RawQuery},
		[]string{httpsRequest.URL.Host, httpsRequest.URL.EscapedPath(), httpsRequest.URL.RawQuery},
	) {
		t.Fatalf("HTTP fallback changed URL beyond scheme: HTTPS=%s HTTP=%s", httpsRequest.URL, httpRequest.URL)
	}
}

func downloadRequestURLs(requests []*http.Request) []string {
	urls := make([]string, 0, len(requests))
	for _, request := range requests {
		urls = append(urls, request.URL.String())
	}
	return urls
}

func TestAcquireDownloadLeaseAllowsOnlyOneConcurrentOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "download.sentinel")
	const contenders = 16
	start := make(chan struct{})
	results := make(chan *downloadLease, contenders)
	var wg sync.WaitGroup
	for range contenders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			lease, err := acquireDownloadLease(path, "https://example.com/image.iso")
			if err == nil {
				results <- lease
				return
			}
			if !errors.Is(err, os.ErrExist) {
				t.Errorf("acquireDownloadLease: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	var leases []*downloadLease
	for lease := range results {
		leases = append(leases, lease)
	}
	if len(leases) != 1 {
		t.Fatalf("successful leases = %d, want 1", len(leases))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat sentinel: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("sentinel permissions = %#o, want 0600", got)
	}
	if err := leases[0].release(); err != nil {
		t.Fatalf("release lease: %v", err)
	}
}

func TestDownloadLeaseReleaseDoesNotRemoveAReplacementSentinel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "download.sentinel")
	lease, err := acquireDownloadLease(path, "https://example.com/image.iso")
	if err != nil {
		t.Fatalf("acquireDownloadLease: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove owned sentinel: %v", err)
	}
	if err := os.WriteFile(path, []byte("replacement"), 0o600); err != nil {
		t.Fatalf("write replacement sentinel: %v", err)
	}
	if err := lease.release(); err == nil {
		t.Fatal("expected ownership-change error")
	}
	assertDownloadFileContents(t, path, "replacement")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func staticDownloadClient(status int, contentLength int64, body []byte) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    status,
			ContentLength: contentLength,
			Body:          io.NopCloser(bytes.NewReader(body)),
			Request:       request,
		}, nil
	})}
}

type fakeDownloadFile struct {
	bytes.Buffer
	path     string
	syncErr  error
	closeErr error
}

func (file *fakeDownloadFile) Name() string { return file.path }

func (file *fakeDownloadFile) Sync() error { return file.syncErr }

func (file *fakeDownloadFile) Close() error { return file.closeErr }

func (file *fakeDownloadFile) Seek(offset int64, whence int) (int64, error) {
	if offset == 0 && whence == io.SeekStart {
		return 0, nil
	}
	return 0, errors.New("unsupported fake download file seek")
}

func (file *fakeDownloadFile) Truncate(size int64) error {
	if size != 0 {
		return errors.New("unsupported fake download file truncate")
	}
	file.Reset()
	return nil
}

func testDownloadPolicy(maxImageBytes int64) downloadURLPolicy {
	return downloadURLPolicy{
		MaxImageBytes:     maxImageBytes,
		ReservedFreeBytes: 0,
	}
}

func setDownloadTestEnvironment(t *testing.T, dataDir string, availableBlocks uint64, blockSize int64) {
	t.Helper()
	originalDataDir := downloadDataDir
	originalStatfs := downloadStatfs
	downloadDataDir = dataDir
	setDownloadStatfs(t, availableBlocks, blockSize)
	t.Cleanup(func() {
		downloadDataDir = originalDataDir
		downloadStatfs = originalStatfs
	})
}

func setDownloadStatfs(t *testing.T, availableBlocks uint64, blockSize int64) {
	t.Helper()
	original := downloadStatfs
	downloadStatfs = func(_ string, stat *unix.Statfs_t) error {
		*stat = unix.Statfs_t{Bavail: availableBlocks, Bsize: blockSize}
		return nil
	}
	t.Cleanup(func() { downloadStatfs = original })
}

func setFakeDownloadTempFile(t *testing.T, path string, syncErr, closeErr error) {
	t.Helper()
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("create temporary download file fixture: %v", err)
	}
	original := createDownloadTemp
	createDownloadTemp = func(string, string) (downloadFile, error) {
		return &fakeDownloadFile{path: path, syncErr: syncErr, closeErr: closeErr}, nil
	}
	t.Cleanup(func() { createDownloadTemp = original })
}

func setDownloadHandlerDependencies(t *testing.T, policy downloadURLPolicy, client *http.Client) {
	t.Helper()
	allowTestStorageAccess(t)
	originalPolicy := downloadPolicy
	originalResolver := downloadResolver
	originalClient := downloadHTTPClient
	originalValidator := downloadURLValidator
	downloadPolicy = func() downloadURLPolicy { return policy }
	downloadResolver = testResolver(map[string][]string{"example.com": {"93.184.216.34"}})
	downloadHTTPClient = func(ipResolver, downloadURLPolicy) *http.Client { return client }
	downloadURLValidator = validateDownloadURL
	t.Cleanup(func() {
		downloadPolicy = originalPolicy
		downloadResolver = originalResolver
		downloadHTTPClient = originalClient
		downloadURLValidator = originalValidator
	})
}

func acquireTestDownloadLease(t *testing.T) *downloadLease {
	t.Helper()
	lease, err := acquireDownloadLease(filepath.Join(t.TempDir(), "download.sentinel"), "https://example.com/image.iso")
	if err != nil {
		t.Fatalf("acquireDownloadLease: %v", err)
	}
	t.Cleanup(func() { _ = lease.release() })
	return lease
}

func mustParseDownloadURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	return parsed
}

func assertDownloadFileContents(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if got := string(data); got != want {
		t.Fatalf("contents of %s = %q, want %q", path, got, want)
	}
}

func assertNoDownloadTemporaryFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read directory: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".image-download-") {
			t.Fatalf("unexpected temporary download file %q", entry.Name())
		}
	}
}
