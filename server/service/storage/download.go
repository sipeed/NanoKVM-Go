package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

var (
	sentinelPath         = "/tmp/.download_in_progress"
	downloadDataDir      = "/data"
	downloadStatfs       = unix.Statfs
	createDownloadTemp   = func(dir, pattern string) (downloadFile, error) { return os.CreateTemp(dir, pattern) }
	renameDownloadFile   = os.Rename
	removeDownloadFile   = os.Remove
	downloadPolicy       = defaultDownloadPolicy
	downloadResolver     = defaultResolveIPAddrs
	downloadHTTPClient   = newDownloadHTTPClient
	downloadURLValidator = validateDownloadURL
)

var (
	errDownloadSizeLimitExceeded = errors.New("download exceeds configured size limit")
	errDownloadInsufficientSpace = errors.New("insufficient storage for download")
)

const (
	maxDownloadAttempts                        = 10
	downloadInsufficientStorageResponseMessage = "insufficient storage"
	cancelImageDownloadTimeout                 = 5 * time.Second
)

func downloadAdmissionFailureMessage(err error) string {
	if errors.Is(err, errDownloadInsufficientSpace) {
		return downloadInsufficientStorageResponseMessage
	}

	return "failed when checking the url"
}

type ipResolver func(context.Context, string) ([]netip.Addr, error)

type downloadURLPolicy struct {
	AllowPrivate      bool
	AllowedHosts      []string
	AllowedCIDRs      []netip.Prefix
	NoProgressTimeout time.Duration
	MaxImageBytes     int64
	ReservedFreeBytes int64
}

func defaultResolveIPAddrs(ctx context.Context, host string) ([]netip.Addr, error) {
	if ip, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{ip.Unmap()}, nil
	}

	resolved, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}

	addrs := make([]netip.Addr, 0, len(resolved))
	for _, item := range resolved {
		addr, ok := netip.AddrFromSlice(item.IP)
		if !ok {
			continue
		}
		addrs = append(addrs, addr.Unmap())
	}

	if len(addrs) == 0 {
		return nil, errors.New("host did not resolve to an IP address")
	}

	return addrs, nil
}

func defaultDownloadPolicy() downloadURLPolicy {
	conf := config.GetInstance()

	return downloadURLPolicy{
		AllowPrivate:      conf.Security.Download.AllowPrivate,
		AllowedHosts:      append([]string(nil), conf.Security.Download.AllowedHosts...),
		AllowedCIDRs:      parseDownloadCIDRs(conf.Security.Download.AllowedCIDRs),
		NoProgressTimeout: config.DownloadNoProgressTimeout(conf.Security.Download.NoProgressTimeout),
		MaxImageBytes:     int64(conf.Security.Download.MaxImageBytes),
		ReservedFreeBytes: int64(conf.Security.Download.ReservedFreeBytes),
	}
}

func parseDownloadCIDRs(values []string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			log.Warnf("ignoring invalid download CIDR allowlist entry %q: %v", value, err)
			continue
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes
}

func newDownloadHTTPClient(resolve ipResolver, policy downloadURLPolicy) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		Proxy:              nil,
		DisableCompression: true,
		DialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}

			addrs, err := resolveAndValidateHost(ctx, host, resolve, policy)
			if err != nil {
				return nil, err
			}

			var lastErr error
			for _, addr := range addrs {
				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}

			if lastErr != nil {
				return nil, lastErr
			}

			return nil, errors.New("host did not resolve to a usable IP address")
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       30 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many redirects")
			}
			_, err := validateDownloadURL(req.URL.String(), resolve, policy)
			return err
		},
	}
}

func validateDownloadURL(rawURL string, resolve ipResolver, policy downloadURLPolicy) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u == nil || u.Host == "" {
		return nil, errors.New("invalid url")
	}

	if u.User != nil {
		return nil, errors.New("url userinfo is not allowed")
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, errors.New("unsupported url scheme")
	}
	u.Scheme = scheme

	if _, err := resolveAndValidateHost(context.Background(), u.Hostname(), resolve, policy); err != nil {
		return nil, err
	}

	return u, nil
}

func resolveAndValidateHost(ctx context.Context, host string, resolve ipResolver, policy downloadURLPolicy) ([]netip.Addr, error) {
	if host == "" || strings.Contains(host, "%") {
		return nil, errors.New("invalid host")
	}

	addrs, err := resolve(ctx, host)
	if err != nil {
		return nil, err
	}

	if len(addrs) == 0 {
		return nil, errors.New("host did not resolve to an IP address")
	}

	for _, addr := range addrs {
		if !isAllowedDownloadIP(addr, host, policy) {
			return nil, fmt.Errorf("blocked address: %s", addr)
		}
	}

	return addrs, nil
}

func isAllowedDownloadIP(addr netip.Addr, host string, policy downloadURLPolicy) bool {
	addr = addr.Unmap()
	if !addr.IsValid() {
		return false
	}
	if isHardBlockedDownloadIP(addr) {
		return false
	}

	if isPublicDownloadIP(addr) {
		return true
	}

	if policy.AllowPrivate && isPrivateDownloadIP(addr) {
		return true
	}

	if isAllowedDownloadHost(host, policy) && isPrivateDownloadIP(addr) {
		return true
	}

	return isAllowedDownloadCIDR(addr, policy)
}

func isPublicDownloadIP(addr netip.Addr) bool {
	return addr.IsGlobalUnicast() && !addr.IsPrivate() && !isPrivateDownloadIP(addr)
}

func isPrivateDownloadIP(addr netip.Addr) bool {
	addr = addr.Unmap()
	if addr.IsPrivate() {
		return true
	}
	return netip.MustParsePrefix("100.64.0.0/10").Contains(addr)
}

func isAllowedDownloadHost(host string, policy downloadURLPolicy) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if host == "" {
		return false
	}

	for _, allowed := range policy.AllowedHosts {
		allowed = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(allowed), "."))
		if allowed != "" && host == allowed {
			return true
		}
	}

	return false
}

func isAllowedDownloadCIDR(addr netip.Addr, policy downloadURLPolicy) bool {
	for _, prefix := range policy.AllowedCIDRs {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}

func isHardBlockedDownloadIP(addr netip.Addr) bool {
	if addr.IsLoopback() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified() {
		return true
	}

	blockedPrefixes := []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("192.0.0.0/24"),
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("224.0.0.0/4"),
		netip.MustParsePrefix("240.0.0.0/4"),
		netip.MustParsePrefix("::/128"),
		netip.MustParsePrefix("::1/128"),
		netip.MustParsePrefix("64:ff9b::/96"),
		netip.MustParsePrefix("100::/64"),
		netip.MustParsePrefix("2001::/23"),
		netip.MustParsePrefix("2001:db8::/32"),
		netip.MustParsePrefix("fe80::/10"),
		netip.MustParsePrefix("ff00::/8"),
	}

	for _, prefix := range blockedPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}

func getWithNoProgressTimeout(client *http.Client, rawURL string, timeout time.Duration) (*http.Response, error) {
	return getWithNoProgressTimeoutFromContext(context.Background(), client, rawURL, timeout, 0)
}

func getWithNoProgressTimeoutFrom(
	client *http.Client,
	rawURL string,
	timeout time.Duration,
	offset int64,
) (*http.Response, error) {
	return getWithNoProgressTimeoutFromContext(context.Background(), client, rawURL, timeout, offset)
}

func getWithNoProgressTimeoutFromContext(
	parent context.Context,
	client *http.Client,
	rawURL string,
	timeout time.Duration,
	offset int64,
) (*http.Response, error) {
	ctx, cancel := context.WithCancel(parent)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		cancel()
		return nil, err
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	resp, err := doDownloadRequestWithHTTPFallback(client, req)
	if err != nil {
		cancel()
		return nil, err
	}

	if timeout > 0 {
		resp.Body = newNoProgressReadCloser(resp.Body, timeout, cancel)
	} else {
		resp.Body = &cancelOnCloseReadCloser{ReadCloser: resp.Body, cancel: cancel}
	}

	return resp, nil
}

func doDownloadRequestWithHTTPFallback(
	client *http.Client,
	request *http.Request,
) (*http.Response, error) {
	response, err := client.Do(request)
	if err == nil || request.URL == nil || request.URL.Scheme != "https" {
		return response, err
	}
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}

	fallbackRequest := request.Clone(request.Context())
	fallbackURL := *request.URL
	fallbackURL.Scheme = "http"
	fallbackRequest.URL = &fallbackURL

	response, fallbackErr := client.Do(fallbackRequest)
	if fallbackErr != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, fmt.Errorf(
			"HTTPS request failed (%v); HTTP fallback failed: %w",
			err,
			fallbackErr,
		)
	}
	return response, nil
}

type cancelOnCloseReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (rc *cancelOnCloseReadCloser) Close() error {
	err := rc.ReadCloser.Close()
	rc.cancel()
	return err
}

type noProgressReadCloser struct {
	body    io.ReadCloser
	cancel  context.CancelFunc
	timeout time.Duration
	timer   *time.Timer
	mu      sync.Mutex
	stopped bool
}

func newNoProgressReadCloser(body io.ReadCloser, timeout time.Duration, cancel context.CancelFunc) io.ReadCloser {
	rc := &noProgressReadCloser{
		body:    body,
		cancel:  cancel,
		timeout: timeout,
	}
	rc.timer = time.AfterFunc(timeout, cancel)
	return rc
}

func (rc *noProgressReadCloser) Read(p []byte) (int, error) {
	n, err := rc.body.Read(p)
	if n > 0 {
		rc.reset()
	}
	if err != nil {
		rc.stopTimer()
	}
	return n, err
}

func (rc *noProgressReadCloser) Close() error {
	rc.stopTimer()
	err := rc.body.Close()
	rc.cancel()
	return err
}

func (rc *noProgressReadCloser) reset() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if !rc.stopped {
		rc.timer.Reset(rc.timeout)
	}
}

func (rc *noProgressReadCloser) stopTimer() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if !rc.stopped {
		rc.timer.Stop()
		rc.stopped = true
	}
}

type downloadReadError struct {
	err error
}

func (e *downloadReadError) Error() string {
	return e.err.Error()
}

func (e *downloadReadError) Unwrap() error {
	return e.err
}

type downloadWriteError struct {
	err error
}

func (e *downloadWriteError) Error() string {
	return e.err.Error()
}

func (e *downloadWriteError) Unwrap() error {
	return e.err
}

func saveDownloadWithResume(
	client *http.Client,
	rawURL string,
	timeout time.Duration,
	out downloadOutput,
	progress *loggingWriter,
) error {
	return saveDownloadWithResumeFromResponseContext(
		context.Background(),
		client,
		rawURL,
		timeout,
		out,
		progress,
		nil,
		nil,
	)
}

type downloadResponseAdmitter func(total, written int64) error

func saveDownloadWithResumeFromResponse(
	client *http.Client,
	rawURL string,
	timeout time.Duration,
	out downloadOutput,
	progress *loggingWriter,
	firstResponse *http.Response,
	admit downloadResponseAdmitter,
) error {
	return saveDownloadWithResumeFromResponseContext(
		context.Background(),
		client,
		rawURL,
		timeout,
		out,
		progress,
		firstResponse,
		admit,
	)
}

func saveDownloadWithResumeFromResponseContext(
	ctx context.Context,
	client *http.Client,
	rawURL string,
	timeout time.Duration,
	out downloadOutput,
	progress *loggingWriter,
	firstResponse *http.Response,
	admit downloadResponseAdmitter,
) error {
	var lastErr error
	offset := int64(0)
	totalSize := int64(-1)
	progress.setProgress(offset, totalSize)

	for attempt := 1; attempt <= maxDownloadAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			if firstResponse != nil && firstResponse.Body != nil {
				_ = firstResponse.Body.Close()
			}
			return err
		}
		resp := firstResponse
		firstResponse = nil
		if resp == nil {
			var err error
			resp, err = getWithNoProgressTimeoutFromContext(ctx, client, rawURL, timeout, offset)
			if err != nil {
				if contextErr := ctx.Err(); contextErr != nil {
					return contextErr
				}
				lastErr = fmt.Errorf("download request failed: %w", err)
				if attempt < maxDownloadAttempts {
					log.Warnf(
						"download request failed at byte %d (attempt %d/%d): %v; retrying",
						offset,
						attempt,
						maxDownloadAttempts,
						err,
					)
					continue
				}
				break
			}
		}

		responseTotal, restart, err := validateDownloadResponse(resp, offset, totalSize)
		if err != nil {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			return err
		}

		if restart {
			log.Warnf("download server ignored Range at byte %d; restarting from byte 0", offset)
			if err := out.Truncate(0); err != nil {
				_ = resp.Body.Close()
				return &downloadWriteError{err: fmt.Errorf("truncate destination file: %w", err)}
			}
			offset = 0
		}
		if admit != nil {
			if err := admit(responseTotal, offset); err != nil {
				_ = resp.Body.Close()
				return err
			}
		}

		if _, err := out.Seek(offset, io.SeekStart); err != nil {
			_ = resp.Body.Close()
			return &downloadWriteError{err: fmt.Errorf("seek destination file: %w", err)}
		}

		totalSize = responseTotal
		progress.setProgress(offset, totalSize)
		written, copyErr := copyDownloadBody(progress, resp.Body)
		closeErr := resp.Body.Close()
		offset += written
		if contextErr := ctx.Err(); contextErr != nil {
			return contextErr
		}
		var writeErr *downloadWriteError
		if errors.As(copyErr, &writeErr) {
			return copyErr
		}

		if totalSize >= 0 {
			if offset > totalSize {
				return fmt.Errorf(
					"download size exceeds expected total: wrote %d bytes, expected %d",
					offset,
					totalSize,
				)
			}
			if offset == totalSize && copyErr == nil && closeErr == nil {
				return nil
			}
		}

		if copyErr == nil && closeErr != nil {
			copyErr = &downloadReadError{err: fmt.Errorf("close response body: %w", closeErr)}
		}
		if copyErr == nil {
			if totalSize < 0 {
				return nil
			}
			copyErr = &downloadReadError{err: io.ErrUnexpectedEOF}
		}

		var readErr *downloadReadError
		if !errors.As(copyErr, &readErr) {
			return copyErr
		}
		lastErr = readErr.err

		if attempt < maxDownloadAttempts {
			log.Warnf(
				"download interrupted at byte %d (attempt %d/%d): %v; resuming",
				offset,
				attempt,
				maxDownloadAttempts,
				readErr.err,
			)
		}
	}

	if lastErr == nil {
		lastErr = errors.New("download did not complete")
	}
	return fmt.Errorf(
		"download interrupted after %d attempts at byte %d: %w",
		maxDownloadAttempts,
		offset,
		lastErr,
	)
}

func validateDownloadResponse(
	resp *http.Response,
	offset int64,
	expectedTotal int64,
) (total int64, restart bool, err error) {
	if resp == nil || resp.Body == nil {
		return 0, false, errors.New("download returned an empty response")
	}

	if offset == 0 {
		if resp.StatusCode != http.StatusOK {
			return 0, false, fmt.Errorf(
				"download returned status %d, expected %d",
				resp.StatusCode,
				http.StatusOK,
			)
		}
		return resp.ContentLength, false, nil
	}

	if resp.StatusCode == http.StatusOK {
		return resp.ContentLength, true, nil
	}
	if resp.StatusCode != http.StatusPartialContent {
		return 0, false, fmt.Errorf(
			"resume returned status %d, expected %d",
			resp.StatusCode,
			http.StatusPartialContent,
		)
	}

	start, end, responseTotal, err := parseDownloadContentRange(resp.Header.Get("Content-Range"))
	if err != nil {
		return 0, false, err
	}
	if start != offset {
		return 0, false, fmt.Errorf(
			"resume Content-Range starts at %d, expected %d",
			start,
			offset,
		)
	}
	if expectedTotal >= 0 && responseTotal != expectedTotal {
		return 0, false, fmt.Errorf(
			"resume Content-Range total is %d, expected %d",
			responseTotal,
			expectedTotal,
		)
	}
	segmentSize := end - start + 1
	if resp.ContentLength >= 0 && resp.ContentLength != segmentSize {
		return 0, false, fmt.Errorf(
			"resume Content-Length is %d, expected %d from Content-Range",
			resp.ContentLength,
			segmentSize,
		)
	}

	return responseTotal, false, nil
}

func parseDownloadContentRange(value string) (start int64, end int64, total int64, err error) {
	unit, value, ok := strings.Cut(strings.TrimSpace(value), " ")
	if !ok || strings.ToLower(unit) != "bytes" {
		return 0, 0, 0, errors.New("resume response has an invalid Content-Range")
	}

	rangeValue, totalValue, ok := strings.Cut(value, "/")
	if !ok || strings.Contains(totalValue, "/") || totalValue == "*" {
		return 0, 0, 0, errors.New("resume response has an invalid Content-Range")
	}
	startValue, endValue, ok := strings.Cut(rangeValue, "-")
	if !ok || strings.Contains(endValue, "-") {
		return 0, 0, 0, errors.New("resume response has an invalid Content-Range")
	}

	start, err = strconv.ParseInt(startValue, 10, 64)
	if err != nil {
		return 0, 0, 0, errors.New("resume response has an invalid Content-Range")
	}
	end, err = strconv.ParseInt(endValue, 10, 64)
	if err != nil {
		return 0, 0, 0, errors.New("resume response has an invalid Content-Range")
	}
	total, err = strconv.ParseInt(totalValue, 10, 64)
	if err != nil || start < 0 || end < start || total <= end {
		return 0, 0, 0, errors.New("resume response has an invalid Content-Range")
	}

	return start, end, total, nil
}

func copyDownloadBody(dst io.Writer, src io.Reader) (int64, error) {
	buffer := make([]byte, 32*1024)
	var written int64
	emptyReads := 0

	for {
		n, readErr := src.Read(buffer)
		if n > 0 {
			emptyReads = 0
			writeN, writeErr := dst.Write(buffer[:n])
			written += int64(writeN)
			if writeErr != nil {
				return written, &downloadWriteError{err: writeErr}
			}
			if writeN != n {
				return written, &downloadWriteError{err: io.ErrShortWrite}
			}
		} else if readErr == nil {
			emptyReads++
			if emptyReads >= 100 {
				return written, &downloadReadError{err: io.ErrNoProgress}
			}
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return written, nil
			}
			return written, &downloadReadError{err: readErr}
		}
	}
}

func (s *Service) ImageEnabled(c *gin.Context) {
	var rsp proto.Response
	release, err := acquireStorageAccess(true)
	if err != nil {
		rsp.OkRspWithData(c, &proto.ImageEnabledRsp{Enabled: false})
		return
	}
	defer release()

	if err := ensureImageDirectory(imageDirectory); err != nil {
		rsp.OkRspWithData(c, &proto.ImageEnabledRsp{Enabled: false})
		return
	}
	// Check if /data mount is RO/RW
	testFile := "/data/.testfile"
	file, err := os.Create(testFile)
	if err != nil {
		if os.IsPermission(err) {
			rsp.OkRspWithData(c, &proto.ImageEnabledRsp{
				Enabled: false,
			})
			return
		}
		rsp.OkRspWithData(c, &proto.ImageEnabledRsp{
			Enabled: false,
		})
		return
	}
	defer file.Close()
	defer os.Remove(testFile)

	rsp.OkRspWithData(c, &proto.ImageEnabledRsp{
		Enabled: true,
	})
}

func (s *Service) StatusImage(c *gin.Context) {
	var rsp proto.Response

	// Check if the sentinel file exists
	log.Debug("StatusImage")
	if _, err := os.Stat(sentinelPath); err == nil {
		content, err := os.ReadFile(sentinelPath)
		if err != nil {
			log.Error("Failed to read sentinel file")
			status := newImageDownloadStatus("in_progress", "", "")
			rsp.OkRspWithData(c, &status)
			return
		}
		status := parseImageDownloadStatus(string(content))
		if operation := s.currentImageDownload(); operation != nil && operation.isPaused() {
			status.Status = "paused"
		}
		rsp.OkRspWithData(c, &status)

		return
	}
	status := newImageDownloadStatus("idle", "", "")
	rsp.OkRspWithData(c, &status)
}

func (s *Service) PauseImageDownload(c *gin.Context) {
	var rsp proto.Response
	operation := s.currentImageDownload()
	if operation == nil || !operation.pause() {
		status := newImageDownloadStatus("idle", "", "")
		rsp.OkRspWithData(c, &status)
		return
	}

	status := newImageDownloadStatus("paused", "", "")
	rsp.OkRspWithData(c, &status)
}

func (s *Service) ResumeImageDownload(c *gin.Context) {
	var rsp proto.Response
	operation := s.currentImageDownload()
	if operation == nil || !operation.resume() {
		status := newImageDownloadStatus("idle", "", "")
		rsp.OkRspWithData(c, &status)
		return
	}

	status := newImageDownloadStatus("in_progress", "", "")
	rsp.OkRspWithData(c, &status)
}

func (s *Service) CancelImageDownload(c *gin.Context) {
	var rsp proto.Response
	operation := s.currentImageDownload()
	if operation == nil {
		status := newImageDownloadStatus("idle", "", "")
		rsp.OkRspWithData(c, &status)
		return
	}

	operation.cancel()
	timer := time.NewTimer(cancelImageDownloadTimeout)
	defer timer.Stop()

	select {
	case <-operation.done:
		status := newImageDownloadStatus("idle", "", "")
		rsp.OkRspWithData(c, &status)
	case <-timer.C:
		rsp.ErrRsp(c, -1, "failed to cancel download")
	case <-c.Request.Context().Done():
		return
	}
}

func newImageDownloadStatus(status, file, percentage string) proto.StatusImageRsp {
	return proto.StatusImageRsp{
		Status:           status,
		File:             file,
		Percentage:       percentage,
		DownloadedBytes:  -1,
		TotalBytes:       -1,
		BytesPerSecond:   -1,
		RemainingSeconds: -1,
	}
}

func parseImageDownloadStatus(content string) proto.StatusImageRsp {
	status := newImageDownloadStatus("in_progress", content, "")
	parts := strings.Split(content, ";")

	if len(parts) >= 6 {
		metricsStart := len(parts) - 5
		downloadedBytes, downloadedErr := strconv.ParseInt(parts[metricsStart+1], 10, 64)
		totalBytes, totalErr := strconv.ParseInt(parts[metricsStart+2], 10, 64)
		bytesPerSecond, speedErr := strconv.ParseInt(parts[metricsStart+3], 10, 64)
		remainingSeconds, remainingErr := strconv.ParseInt(parts[metricsStart+4], 10, 64)
		if downloadedErr == nil && totalErr == nil && speedErr == nil && remainingErr == nil {
			status.File = strings.Join(parts[:metricsStart], ";")
			status.Percentage = parts[metricsStart]
			status.DownloadedBytes = downloadedBytes
			status.TotalBytes = totalBytes
			status.BytesPerSecond = bytesPerSecond
			status.RemainingSeconds = remainingSeconds
			return status
		}
	}

	if len(parts) >= 2 && strings.HasSuffix(parts[len(parts)-1], "%") {
		status.File = strings.Join(parts[:len(parts)-1], ";")
		status.Percentage = parts[len(parts)-1]
	}

	return status
}

func (s *Service) DownloadImage(c *gin.Context) {
	var req proto.DownloadImageReq
	var rsp proto.Response

	log.Debug("DownloadImage")

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if req.File == "" || !validChecksum(req.SHA256) {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	policy := downloadPolicy()
	client := downloadHTTPClient(downloadResolver, policy)
	u, err := downloadURLValidator(req.File, downloadResolver, policy)
	if err != nil {
		rsp.ErrRsp(c, -1, "invalid url")
		return
	}
	releaseStorage, err := acquireStorageAccess(true)
	if err != nil {
		rsp.ErrRsp(c, -1, err.Error())
		return
	}
	handoffStorage := false
	defer func() {
		if !handoffStorage {
			releaseStorage()
		}
	}()

	lease, err := acquireDownloadLease(sentinelPath, req.File)
	if errors.Is(err, os.ErrExist) {
		log.Debug("Download in progress")
		rsp.ErrRsp(c, -1, "download in progress")
		return
	}
	if err != nil {
		log.Errorf("failed to create download sentinel: %v", err)
		rsp.ErrRsp(c, -1, "failed to create sentinel file")
		return
	}
	operation, err := s.beginImageDownload(lease)
	if err != nil {
		releaseDownloadLease(lease)
		log.Debug("Download in progress")
		rsp.ErrRsp(c, -1, "download in progress")
		return
	}
	operation.releaseStorage = releaseStorage
	handoffStorage = true
	handoffOperation := false
	defer func() {
		if !handoffOperation {
			s.finishImageDownload(operation)
		}
	}()

	if _, err := downloadImageBudget(downloadDataDir, policy); err != nil {
		log.Errorf("failed to admit image download: %v", err)
		rsp.ErrRsp(c, -1, downloadAdmissionFailureMessage(err))
		return
	}

	headRequest, err := http.NewRequestWithContext(operation.ctx, http.MethodHead, u.String(), nil)
	if err != nil {
		rsp.ErrRsp(c, -1, "failed when checking the url")
		log.Errorf("Failed to create URL check request: %v", err)
		return
	}
	headResp, err := doDownloadRequestWithHTTPFallback(client, headRequest)
	if err != nil {
		if errors.Is(operation.ctx.Err(), context.Canceled) {
			rsp.ErrRsp(c, -1, "download canceled")
			return
		}
		rsp.ErrRsp(c, -1, "failed when checking the url")
		log.Errorf("failed to check image download URL: %v", err)
		return
	}
	if headResp == nil {
		rsp.ErrRsp(c, -1, "failed when checking the url")
		log.Error("failed to check image download URL: empty response")
		return
	}

	if _, err := admitDownloadResponse(headResp, downloadDataDir, policy); err != nil {
		rsp.ErrRsp(c, -1, downloadAdmissionFailureMessage(err))
		log.Errorf("failed to admit image download: %v", err)
		if headResp.Body != nil {
			_ = headResp.Body.Close()
		}
		return
	}
	if headResp.Body != nil {
		if err := headResp.Body.Close(); err != nil {
			rsp.ErrRsp(c, -1, "failed when checking the url")
			log.Errorf("failed to close image download HEAD response: %v", err)
			return
		}
	}
	if err := operation.ctx.Err(); err != nil {
		rsp.ErrRsp(c, -1, "download canceled")
		return
	}

	go func() {
		defer s.finishImageDownload(operation)
		err := downloadImageToDataWithControl(
			operation.ctx,
			client,
			u,
			policy,
			lease,
			operation.waitUntilResumed,
			req.SHA256,
		)
		result := newImageDownloadStatus("success", req.File, "100%")
		if err != nil {
			result.Status = "failed"
			if errors.Is(err, errChecksumMismatch) {
				result.Status = "checksum_failed"
			}
			if errors.Is(err, context.Canceled) {
				result.Status = "idle"
			}
			log.Errorf("failed to download image: %v", err)
		}
		s.downloadMu.Lock()
		if s.download == operation {
			s.lastDownload = result
		}
		s.downloadMu.Unlock()
	}()
	handoffOperation = true
	status := newImageDownloadStatus("in_progress", req.File, "")
	rsp.OkRspWithData(c, &status)
}

type downloadFile interface {
	downloadOutput
	Name() string
	Sync() error
	Close() error
}

type downloadOutput interface {
	io.Writer
	io.Seeker
	Truncate(size int64) error
}

type downloadLease struct {
	path     string
	file     *os.File
	mu       sync.Mutex
	released bool
}

func acquireDownloadLease(path, source string) (*downloadLease, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, err
	}
	if _, err := file.WriteString(source); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, err
	}

	return &downloadLease{path: path, file: file}, nil
}

func (lease *downloadLease) update(contents string) error {
	lease.mu.Lock()
	defer lease.mu.Unlock()
	if lease.released {
		return os.ErrClosed
	}
	if err := lease.file.Truncate(0); err != nil {
		return err
	}
	if _, err := lease.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err := lease.file.WriteString(contents)
	return err
}

func (lease *downloadLease) release() error {
	lease.mu.Lock()
	if lease.released {
		lease.mu.Unlock()
		return nil
	}
	lease.released = true
	fileInfo, statErr := lease.file.Stat()
	closeErr := lease.file.Close()
	lease.mu.Unlock()

	if statErr != nil {
		return errors.Join(closeErr, statErr)
	}
	pathInfo, err := os.Lstat(lease.path)
	if errors.Is(err, os.ErrNotExist) {
		return closeErr
	}
	if err != nil {
		return errors.Join(closeErr, err)
	}
	if !os.SameFile(fileInfo, pathInfo) {
		return errors.Join(closeErr, errors.New("download sentinel ownership changed"))
	}
	if err := os.Remove(lease.path); err != nil {
		return errors.Join(closeErr, err)
	}
	return closeErr
}

func releaseDownloadLease(lease *downloadLease) {
	if err := lease.release(); err != nil {
		log.Errorf("failed to remove image download sentinel: %v", err)
	}
}

func downloadImageBudget(directory string, policy downloadURLPolicy) (int64, error) {
	budget, err := imageStorageBudget(directory, downloadStatfs, policy.MaxImageBytes, policy.ReservedFreeBytes, 0)
	switch {
	case errors.Is(err, errImageStorageInsufficientSpace):
		return 0, errDownloadInsufficientSpace
	case errors.Is(err, errImageStorageSizeLimit):
		return 0, errDownloadSizeLimitExceeded
	}
	return budget, err
}

func validateDeclaredDownloadSize(contentLength, budget int64, policy downloadURLPolicy) error {
	if contentLength < 0 {
		return nil
	}
	if contentLength > policy.MaxImageBytes {
		return errDownloadSizeLimitExceeded
	}
	if contentLength > budget {
		return errDownloadInsufficientSpace
	}
	return nil
}

func admitDownloadResponse(response *http.Response, directory string, policy downloadURLPolicy) (int64, error) {
	if response == nil {
		return 0, errors.New("empty download response")
	}
	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected download response status: %d", response.StatusCode)
	}

	budget, err := downloadImageBudget(directory, policy)
	if err != nil {
		return 0, err
	}
	if err := validateDeclaredDownloadSize(response.ContentLength, budget, policy); err != nil {
		return 0, err
	}
	return budget, nil
}

func admitDownloadTotal(total, written int64, directory string, policy downloadURLPolicy) error {
	if written < 0 || written > policy.MaxImageBytes {
		return errDownloadSizeLimitExceeded
	}

	budget, err := downloadImageBudget(directory, policy)
	if err != nil {
		return err
	}
	if budget > policy.MaxImageBytes-written {
		budget = policy.MaxImageBytes
	} else {
		budget += written
	}
	return validateDeclaredDownloadSize(total, budget, policy)
}

func downloadImageToData(client *http.Client, source *url.URL, policy downloadURLPolicy, lease *downloadLease) error {
	return downloadImageToDataWithContext(context.Background(), client, source, policy, lease)
}

func downloadImageToDataWithContext(
	ctx context.Context,
	client *http.Client,
	source *url.URL,
	policy downloadURLPolicy,
	lease *downloadLease,
) error {
	return downloadImageToDataWithControl(ctx, client, source, policy, lease, nil)
}

func downloadImageToDataWithControl(
	ctx context.Context,
	client *http.Client,
	source *url.URL,
	policy downloadURLPolicy,
	lease *downloadLease,
	waitForResume func() error,
	expectedChecksum ...string,
) (err error) {
	if waitForResume != nil {
		if err = waitForResume(); err != nil {
			return err
		}
	}
	response, err := getWithNoProgressTimeoutFromContext(
		ctx,
		client,
		source.String(),
		policy.NoProgressTimeout,
		0,
	)
	if err != nil {
		return err
	}
	if response == nil || response.Body == nil {
		return errors.New("empty download response")
	}
	responseClosed := false
	defer func() {
		if !responseClosed {
			_ = response.Body.Close()
		}
	}()

	initialTotal, restart, err := validateDownloadResponse(response, 0, -1)
	if err != nil {
		return err
	}
	if restart {
		return errors.New("initial image download response unexpectedly requested a restart")
	}
	budget, err := admitDownloadResponse(response, downloadDataDir, policy)
	if err != nil {
		return err
	}

	file, err := createDownloadTemp(downloadDataDir, ".image-download-*")
	if err != nil {
		return err
	}
	temporaryPath := file.Name()
	fileClosed := false
	published := false
	defer func() {
		if !fileClosed {
			closeErr := file.Close()
			if closeErr != nil && err == nil {
				err = fmt.Errorf("close image download temporary file: %w", closeErr)
			}
		}
		if !published {
			removeErr := removeDownloadFile(temporaryPath)
			if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) && err == nil {
				err = fmt.Errorf("remove image download temporary file: %w", removeErr)
			}
		}
	}()

	speedTracker := downloadSpeedTracker{
		lastAt: time.Now(),
	}
	writer := &loggingWriter{
		writer:        file,
		limit:         budget,
		totalSize:     initialTotal,
		waitForResume: waitForResume,
		onProgress: func(total, totalSize int64) error {
			percentage := ""
			if totalSize > 0 {
				percentage = fmt.Sprintf("%.2f%%", float64(total)/float64(totalSize)*100)
			}
			bytesPerSecond, remainingSeconds := speedTracker.update(total, totalSize, time.Now())
			return lease.update(fmt.Sprintf(
				"%s;%s;%d;%d;%d;%d",
				source.String(),
				percentage,
				total,
				totalSize,
				bytesPerSecond,
				remainingSeconds,
			))
		},
	}
	writer.startTicker()
	defer writer.stopTicker()

	err = saveDownloadWithResumeFromResponseContext(
		ctx,
		client,
		source.String(),
		policy.NoProgressTimeout,
		file,
		writer,
		response,
		func(total, written int64) error {
			return admitDownloadTotal(total, written, downloadDataDir, policy)
		},
	)
	responseClosed = true
	if err != nil {
		return fmt.Errorf("save image download response: %w", err)
	}
	if waitForResume != nil {
		if err = waitForResume(); err != nil {
			return err
		}
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return fmt.Errorf("sync image download temporary file: %w", err)
	}
	if err = file.Close(); err != nil {
		fileClosed = true
		return fmt.Errorf("close image download temporary file: %w", err)
	}
	fileClosed = true
	if waitForResume != nil {
		if err = waitForResume(); err != nil {
			return err
		}
	}
	if err = ctx.Err(); err != nil {
		return err
	}

	if len(expectedChecksum) > 0 {
		if err = verifyImageChecksum(ctx, temporaryPath, expectedChecksum[0]); err != nil {
			return err
		}
	}
	destinationPath := filepath.Join(downloadDataDir, filepath.Base(source.Path))
	mediumOperationMu.Lock()
	defer mediumOperationMu.Unlock()
	if err = guardMediaDelete(destinationPath); err != nil {
		return err
	}
	if err = renameDownloadFile(temporaryPath, destinationPath); err != nil {
		return fmt.Errorf("publish image download: %w", err)
	}
	published = true
	return nil
}

type loggingWriter struct {
	writer        io.Writer
	limit         int64
	total         int64
	totalSize     int64
	waitForResume func() error
	onProgress    func(total, totalSize int64) error
	ticker        *time.Ticker
	done          chan struct{}
	stopped       chan struct{}
	stopOnce      sync.Once
	mu            sync.Mutex
}

type downloadSpeedTracker struct {
	lastAt    time.Time
	lastTotal int64
}

func (tracker *downloadSpeedTracker) update(total, totalSize int64, now time.Time) (int64, int64) {
	elapsed := now.Sub(tracker.lastAt)
	delta := total - tracker.lastTotal
	tracker.lastAt = now
	tracker.lastTotal = total

	if elapsed <= 0 || delta <= 0 {
		return 0, -1
	}

	bytesPerSecond := int64(math.Round(float64(delta) / elapsed.Seconds()))
	if bytesPerSecond <= 0 {
		return 0, -1
	}

	remainingSeconds := int64(-1)
	if totalSize > 0 {
		remainingBytes := totalSize - total
		if remainingBytes <= 0 {
			remainingSeconds = 0
		} else {
			remainingSeconds = int64(math.Ceil(float64(remainingBytes) / float64(bytesPerSecond)))
		}
	}

	return bytesPerSecond, remainingSeconds
}

func (lw *loggingWriter) startTicker() {
	if lw.onProgress == nil {
		return
	}

	lw.ticker = time.NewTicker(2500 * time.Millisecond)
	lw.done = make(chan struct{})
	lw.stopped = make(chan struct{})
	lw.updateProgress()
	go func() {
		defer close(lw.stopped)
		for {
			select {
			case <-lw.done:
				return
			case <-lw.ticker.C:
				lw.updateProgress()
			}
		}
	}()
}

func (lw *loggingWriter) stopTicker() {
	if lw.done == nil {
		return
	}
	lw.stopOnce.Do(func() {
		lw.ticker.Stop()
		close(lw.done)
		<-lw.stopped
	})
}

func (lw *loggingWriter) updateProgress() {
	lw.mu.Lock()
	total := lw.total
	totalSize := lw.totalSize
	lw.mu.Unlock()
	if err := lw.onProgress(total, totalSize); err != nil {
		log.Errorf("failed to update image download progress: %v", err)
	}
}

func (lw *loggingWriter) Write(data []byte) (int, error) {
	if lw.waitForResume != nil {
		if err := lw.waitForResume(); err != nil {
			return 0, err
		}
	}

	lw.mu.Lock()
	defer lw.mu.Unlock()

	if lw.limit > 0 && (lw.total > lw.limit || int64(len(data)) > lw.limit-lw.total) {
		return 0, errDownloadSizeLimitExceeded
	}

	n, err := lw.writer.Write(data)
	lw.total += int64(n)
	return n, err
}

func (lw *loggingWriter) setProgress(total int64, totalSize int64) {
	lw.mu.Lock()
	lw.total = total
	lw.totalSize = totalSize
	lw.mu.Unlock()
}
