package application

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestEnsureApplicationCacheRootCreatesPrivateRealDirectory(t *testing.T) {
	root := filepath.Join(applicationStatusTestDirectory(t), "cache")
	if err := ensureApplicationCacheRoot(root); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(root)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Fatalf("cache root mode = %v", info.Mode())
	}
	target := filepath.Join(applicationStatusTestDirectory(t), "target")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(applicationStatusTestDirectory(t), "cache-link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := ensureApplicationCacheRoot(link); err == nil {
		t.Fatal("symlink cache root was accepted")
	}
}

func TestApplicationStorageErrorsHaveStableClassification(t *testing.T) {
	if code := classifyApplicationStorageError(os.ErrPermission); code != UpdateErrorInterrupted {
		t.Fatalf("permission classified as %s", code)
	}
	if code := classifyApplicationStorageError(&os.PathError{Op: "write", Path: "cache", Err: unix.ENOSPC}); code != UpdateErrorNoSpace {
		t.Fatalf("ENOSPC classified as %s", code)
	}
	if code := classifyApplicationStorageError(&os.PathError{Op: "write", Path: "cache", Err: unix.EDQUOT}); code != UpdateErrorNoSpace {
		t.Fatalf("EDQUOT classified as %s", code)
	}
}

func TestApplicationDownloadBudgetErrorsRemainTypedNoSpace(t *testing.T) {
	requests := 0
	oldClient := applicationHTTPClient
	applicationHTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return nil, errors.New("unexpected HTTP request")
	})}
	t.Cleanup(func() { applicationHTTPClient = oldClient })
	target := filepath.Join(applicationStatusTestDirectory(t), "archive.tar.gz")
	err := downloadOnlineApplicationArtifact(
		"https://updates.invalid/archive.tar.gz",
		target,
		int64(^uint64(0)>>1),
		"operation",
	)
	if !errors.Is(err, unix.ENOSPC) {
		t.Fatalf("oversized download error = %v, want typed ENOSPC", err)
	}
	if code := classifyApplicationStorageError(err); code != UpdateErrorNoSpace {
		t.Fatalf("oversized download classified as %s", code)
	}
	if requests != 0 {
		t.Fatalf("permanent local storage error made %d HTTP requests", requests)
	}
}

func TestDownloadOnlineApplicationArtifactHasDeadline(t *testing.T) {
	oldClient := applicationHTTPClient
	applicationHTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		deadline, ok := request.Context().Deadline()
		if !ok {
			t.Fatal("online application request has no deadline")
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > applicationOnlineDownloadTimeout {
			t.Fatalf("online application deadline remaining = %v", remaining)
		}
		return nil, context.DeadlineExceeded
	})}
	t.Cleanup(func() { applicationHTTPClient = oldClient })
	target := filepath.Join(applicationStatusTestDirectory(t), "archive.tar.gz")
	if err := downloadOnlineApplicationArtifact("https://updates.invalid/archive.tar.gz", target, 1, "operation"); err == nil {
		t.Fatal("expected deadline transport failure")
	}
}

func TestSequentialInstallScriptForcesPackageConffiles(t *testing.T) {
	const want = `/usr/bin/dpkg --force-confnew --force-confmiss -i "$deb"`
	if !strings.Contains(sequentialInstallScript, want) {
		t.Fatalf("install script does not contain %q", want)
	}
	if strings.Contains(sequentialInstallScript, "--force-confdef") {
		t.Fatal("install script must not prefer the default conffile action")
	}
	for _, want := range []string{
		`install_deb "$deb" || install_failed=1`,
		`/usr/bin/dpkg-query -W -f='${Version}' "$package"`,
		`verify_installed_version kvmcomm || exit 1`,
		`verify_installed_version nanokvm-go || exit 1`,
	} {
		if !strings.Contains(sequentialInstallScript, want) {
			t.Fatalf("install script does not contain %q", want)
		}
	}
}

func TestValidateExtractedApplicationRootRequiresSingleCanonicalDirectory(t *testing.T) {
	root := t.TempDir()
	extractDir := filepath.Join(root, "extracted")
	expected := filepath.Join(extractDir, updateCacheDirName("1.2.3"))
	createDir(t, expected)

	if err := validateExtractedApplicationRoot(extractDir, expected, "1.2.3"); err != nil {
		t.Fatalf("validate canonical root: %v", err)
	}
	createDir(t, filepath.Join(extractDir, "extra"))
	if err := validateExtractedApplicationRoot(extractDir, expected, "1.2.3"); err == nil {
		t.Fatal("extra top-level directory was accepted")
	}
	if err := os.RemoveAll(filepath.Join(extractDir, "extra")); err != nil {
		t.Fatal(err)
	}
	if err := validateExtractedApplicationRoot(extractDir, filepath.Join(extractDir, "other"), "1.2.3"); err == nil {
		t.Fatal("non-canonical returned root was accepted")
	}
}

func TestValidateApplicationArchiveRejectsDuplicateAliasAndExtraTopLevel(t *testing.T) {
	version := "1.2.3"
	root := updateCacheDirName(version)
	tests := []struct {
		name    string
		entries []archiveTestEntry
	}{
		{
			name: "valid",
			entries: []archiveTestEntry{
				{name: root + "/", typeflag: tar.TypeDir},
				{name: root + "/file", content: "x"},
			},
		},
		{
			name: "duplicate directory",
			entries: []archiveTestEntry{
				{name: root + "/", typeflag: tar.TypeDir},
				{name: root + "/", typeflag: tar.TypeDir},
			},
		},
		{
			name: "path alias",
			entries: []archiveTestEntry{
				{name: root + "/./file", content: "x"},
			},
		},
		{
			name: "extra top level",
			entries: []archiveTestEntry{
				{name: root + "/file", content: "x"},
				{name: "extra/file", content: "x"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archivePath := filepath.Join(t.TempDir(), "bundle.tar.gz")
			writeArchiveForUpdateTest(t, archivePath, test.entries)
			err := validateApplicationArchive(archivePath, version)
			if test.name == "valid" && err != nil {
				t.Fatalf("valid archive rejected: %v", err)
			}
			if test.name != "valid" && err == nil {
				t.Fatal("invalid archive accepted")
			}
		})
	}
}

func TestDownloadApplicationArtifactIsExactAndPrivate(t *testing.T) {
	payload := []byte("signed application archive")
	for _, test := range []struct {
		name          string
		body          []byte
		contentLength int64
		wantError     bool
	}{
		{name: "exact", body: payload, contentLength: int64(len(payload))},
		{name: "unknown length exact", body: payload, contentLength: -1},
		{name: "short", body: payload[:len(payload)-1], contentLength: -1, wantError: true},
		{name: "long", body: append(append([]byte(nil), payload...), 'x'), contentLength: -1, wantError: true},
		{name: "wrong declared length", body: payload, contentLength: int64(len(payload) + 1), wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				response := httptest.NewRecorder()
				response.Code = http.StatusOK
				response.Body.Write(test.body)
				result := response.Result()
				result.Request = request
				result.ContentLength = test.contentLength
				result.Body = io.NopCloser(bytes.NewReader(test.body))
				return result, nil
			})}
			target := filepath.Join(t.TempDir(), "archive.tar.gz")
			req, err := http.NewRequest(http.MethodGet, "https://updates.invalid/archive", nil)
			if err != nil {
				t.Fatal(err)
			}
			err = downloadApplicationArtifact(req, target, int64(len(payload)), client, nil)
			if test.wantError {
				if err == nil {
					t.Fatal("expected download failure")
				}
				assertNotExists(t, target)
				return
			}
			if err != nil {
				t.Fatalf("download exact artifact: %v", err)
			}
			assertFileContent(t, target, string(payload))
			info, err := os.Stat(target)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0o600 {
				t.Fatalf("download mode = %o, want 600", info.Mode().Perm())
			}
		})
	}
}

func TestDownloadApplicationArtifactHTTPSFallback(t *testing.T) {
	payload := []byte("signed application archive")
	const archiveURL = "https://updates.invalid:8443/releases/archive.tar.gz?channel=preview&token=a%2Fb"

	t.Run("HTTPS success does not request HTTP", func(t *testing.T) {
		var requested []*http.Request
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			requested = append(requested, request.Clone(request.Context()))
			return applicationArchiveHTTPResponse(request, payload), nil
		})}
		target := filepath.Join(applicationStatusTestDirectory(t), "archive.tar.gz")
		request, err := http.NewRequest(http.MethodGet, archiveURL, nil)
		if err != nil {
			t.Fatal(err)
		}

		if err := downloadApplicationArtifact(request, target, int64(len(payload)), client, nil); err != nil {
			t.Fatalf("download archive over HTTPS: %v", err)
		}
		if len(requested) != 1 || requested[0].URL.Scheme != "https" {
			t.Fatalf("requests = %v, want one HTTPS request", requestURLs(requested))
		}
		assertFileContent(t, target, string(payload))
	})

	t.Run("HTTPS failure retries the same URL over HTTP", func(t *testing.T) {
		var requested []*http.Request
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			requested = append(requested, request.Clone(request.Context()))
			if len(requested) == 1 {
				return nil, errors.New("TLS certificate is not yet valid")
			}
			return applicationArchiveHTTPResponse(request, payload), nil
		})}
		target := filepath.Join(applicationStatusTestDirectory(t), "archive.tar.gz")
		request, err := http.NewRequest(http.MethodGet, archiveURL, nil)
		if err != nil {
			t.Fatal(err)
		}

		if err := downloadApplicationArtifact(request, target, int64(len(payload)), client, nil); err != nil {
			t.Fatalf("download archive through HTTP fallback: %v", err)
		}
		assertHTTPSFallbackRequests(t, requested)
		assertFileContent(t, target, string(payload))
	})

	t.Run("HTTPS and HTTP failures return an error", func(t *testing.T) {
		var requested []*http.Request
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			requested = append(requested, request.Clone(request.Context()))
			return nil, errors.New("network failed")
		})}
		target := filepath.Join(applicationStatusTestDirectory(t), "archive.tar.gz")
		request, err := http.NewRequest(http.MethodGet, archiveURL, nil)
		if err != nil {
			t.Fatal(err)
		}

		if err := downloadApplicationArtifact(request, target, int64(len(payload)), client, nil); err == nil {
			t.Fatal("HTTPS and HTTP archive failures were accepted")
		}
		assertHTTPSFallbackRequests(t, requested)
		assertNotExists(t, target)
	})
}

func applicationArchiveHTTPResponse(request *http.Request, payload []byte) *http.Response {
	response := latestHTTPResponse(request, http.StatusOK, io.NopCloser(bytes.NewReader(payload)))
	response.ContentLength = int64(len(payload))
	return response
}

func requestURLs(requests []*http.Request) []string {
	urls := make([]string, 0, len(requests))
	for _, request := range requests {
		urls = append(urls, request.URL.String())
	}
	return urls
}

func TestPublishUpdateCacheReplacesAndRollsBackExistingTarget(t *testing.T) {
	baseDir := t.TempDir()
	version := "2.0.0"
	targetDir := updateCacheDir(baseDir, version)
	extractedDir := filepath.Join(baseDir, "staged", updateCacheDirName(version))
	writeFile(t, filepath.Join(targetDir, "old"), "old")
	writeFile(t, filepath.Join(extractedDir, "new"), "new")

	cacheDir, published, err := publishUpdateCache(extractedDir, version, baseDir, os.Rename, exchangePaths)
	if err != nil {
		t.Fatalf("publish update cache: %v", err)
	}
	if cacheDir != targetDir {
		t.Fatalf("cache dir = %q, want %q", cacheDir, targetDir)
	}
	assertExists(t, filepath.Join(targetDir, "new"))
	assertNotExists(t, filepath.Join(targetDir, "old"))
	if err := published.Rollback(); err != nil {
		t.Fatalf("rollback update cache: %v", err)
	}
	assertExists(t, filepath.Join(targetDir, "old"))
	assertNotExists(t, filepath.Join(targetDir, "new"))
}

func TestPublishUpdateCacheRestoresExistingTargetWhenPublishRenameFails(t *testing.T) {
	baseDir := t.TempDir()
	version := "2.0.0"
	targetDir := updateCacheDir(baseDir, version)
	extractedDir := filepath.Join(baseDir, "staged", updateCacheDirName(version))
	writeFile(t, filepath.Join(targetDir, "old"), "old")
	writeFile(t, filepath.Join(extractedDir, "new"), "new")

	exchange := func(string, string) error {
		return errors.New("injected publish failure")
	}
	if _, _, err := publishUpdateCache(extractedDir, version, baseDir, os.Rename, exchange); err == nil {
		t.Fatal("expected publish failure")
	}
	assertExists(t, filepath.Join(targetDir, "old"))
	assertExists(t, filepath.Join(extractedDir, "new"))
}

func TestPublishedUpdateCacheRollbackFailureKeepsBothCachesRecoverable(t *testing.T) {
	baseDir := t.TempDir()
	version := "2.0.0"
	targetDir := updateCacheDir(baseDir, version)
	extractedDir := filepath.Join(baseDir, "staged", updateCacheDirName(version))
	writeFile(t, filepath.Join(targetDir, "old"), "old")
	writeFile(t, filepath.Join(extractedDir, "new"), "new")

	cacheDir, published, err := publishUpdateCache(extractedDir, version, baseDir, os.Rename, exchangePaths)
	if err != nil {
		t.Fatal(err)
	}
	published.exchange = func(string, string) error { return errors.New("injected rollback failure") }
	if err := published.Rollback(); err == nil {
		t.Fatal("expected rollback failure")
	}
	assertFileContent(t, filepath.Join(cacheDir, "new"), "new")
	assertFileContent(t, filepath.Join(extractedDir, "old"), "old")
}

func TestApplicationDebPathsUseCanonicalVerifiedVersionAndOrder(t *testing.T) {
	dir := t.TempDir()
	want := []string{
		filepath.Join(dir, "kvmcomm_3.0.0_armhf.deb"),
		filepath.Join(dir, "nanokvm-go_3.0.0_armhf.deb"),
	}
	if got := applicationDebPaths(dir, "3.0.0"); !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %#v, want %#v", got, want)
	}
}

func writeBundlePlaceholders(t *testing.T, dir, version string) {
	t.Helper()
	for _, appName := range appNames {
		writeFile(t, filepath.Join(dir, debFileName(appName, version)), appName+" deb")
		writeFile(t, filepath.Join(dir, appName+"_"+version+".json"), "{}")
	}
}

func createDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("create dir %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent dir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %s to not exist, stat err = %v", path, err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("content of %s = %q, want %q", path, data, want)
	}
}

type archiveTestEntry struct {
	name     string
	typeflag byte
	content  string
}

func writeArchiveForUpdateTest(t *testing.T, archivePath string, entries []archiveTestEntry) {
	t.Helper()
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		typeflag := entry.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		header := &tar.Header{
			Name:     entry.name,
			Typeflag: typeflag,
			Mode:     0o600,
			Size:     int64(len(entry.content)),
		}
		if typeflag == tar.TypeDir {
			header.Mode = 0o700
			header.Size = 0
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if entry.content != "" {
			if _, err := tarWriter.Write([]byte(entry.content)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
