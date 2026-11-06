package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
)

func TestUSBRestartCommandUsesFixedScriptArgumentsWithoutShellCode(t *testing.T) {
	command := usbRestartCommand()
	want := []string{"/bin/bash", "/opt/scripts/usbdev.sh", "restart"}
	if len(command.Args) != len(want) {
		t.Fatalf("command args=%q, want %q", command.Args, want)
	}
	for index := range want {
		if command.Args[index] != want[index] {
			t.Fatalf("command args=%q, want %q", command.Args, want)
		}
	}
	for _, arg := range command.Args {
		if arg == "-c" {
			t.Fatalf("command unexpectedly evaluates shell code: %q", command.Args)
		}
	}
}

func TestEnsureImageDirectoryCreatesMissingDirectory(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "data")
	if err := ensureImageDirectory(directory); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(directory)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() || info.Mode().Perm() != 0o755 {
		t.Fatalf("image directory mode = %v", info.Mode())
	}
}

func TestEnsureImageDirectoryRejectsNonDirectory(t *testing.T) {
	base := t.TempDir()

	regularFile := filepath.Join(base, "regular")
	if err := os.WriteFile(regularFile, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureImageDirectory(regularFile); err == nil {
		t.Fatal("regular file accepted as image directory")
	}

	target := filepath.Join(base, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(base, "symlink")
	if err := os.Symlink(target, symlink); err != nil {
		t.Fatal(err)
	}
	if err := ensureImageDirectory(symlink); err == nil {
		t.Fatal("symlink accepted as image directory")
	}
}

func TestSafeImagePathAllowsRegularIsoAndImgInAllowedDirectories(t *testing.T) {
	base := t.TempDir()
	dataDir := filepath.Join(base, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("create data dir: %v", err)
	}

	for _, name := range []string{"boot.iso", "disk.img"} {
		path := filepath.Join(dataDir, name)
		if err := os.WriteFile(path, []byte("image"), 0o600); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}

		got, ok := safeImagePath(path, []string{dataDir})
		if !ok {
			t.Fatalf("expected %s to be allowed", path)
		}
		if got != path {
			t.Fatalf("expected normalized path %s, got %s", path, got)
		}
	}
}

func TestSafeImagePathRejectsEtcPasswd(t *testing.T) {
	if path, ok := safeImagePath("/etc/passwd", []string{imageDirectory}); ok {
		t.Fatalf("expected /etc/passwd to be rejected, got %s", path)
	}
}

func TestSafeImagePathRejectsTraversal(t *testing.T) {
	base := t.TempDir()
	dataDir := filepath.Join(base, "data")
	targetDir := filepath.Join(base, "tmp")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("create data dir: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("create target dir: %v", err)
	}
	target := filepath.Join(targetDir, "escape.iso")
	if err := os.WriteFile(target, []byte("image"), 0o600); err != nil {
		t.Fatalf("create target: %v", err)
	}

	if path, ok := safeImagePath(filepath.Join(dataDir, "..", "tmp", "escape.iso"), []string{dataDir}); ok {
		t.Fatalf("expected traversal to be rejected, got %s", path)
	}
}

func TestSafeImagePathRejectsWrongExtension(t *testing.T) {
	dataDir := t.TempDir()
	target := filepath.Join(dataDir, "notes.txt")
	if err := os.WriteFile(target, []byte("not an image"), 0o600); err != nil {
		t.Fatalf("create target: %v", err)
	}

	if path, ok := safeImagePath(target, []string{dataDir}); ok {
		t.Fatalf("expected wrong extension to be rejected, got %s", path)
	}
}

func TestSafeImagePathRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink setup requires privileges on some Windows environments")
	}

	base := t.TempDir()
	dataDir := filepath.Join(base, "data")
	outsideDir := filepath.Join(base, "outside")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("create data dir: %v", err)
	}
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatalf("create outside dir: %v", err)
	}
	outsideImage := filepath.Join(outsideDir, "secret.iso")
	if err := os.WriteFile(outsideImage, []byte("secret"), 0o600); err != nil {
		t.Fatalf("create outside image: %v", err)
	}
	link := filepath.Join(dataDir, "linked.iso")
	if err := os.Symlink(outsideImage, link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	if path, ok := safeImagePath(link, []string{dataDir}); ok {
		t.Fatalf("expected symlink escape to be rejected, got %s", path)
	}
}

func TestSafeImagePathRejectsNonRegularFile(t *testing.T) {
	dataDir := t.TempDir()
	dirWithImageExtension := filepath.Join(dataDir, "folder.iso")
	if err := os.Mkdir(dirWithImageExtension, 0o755); err != nil {
		t.Fatalf("create directory: %v", err)
	}

	if path, ok := safeImagePath(dirWithImageExtension, []string{dataDir}); ok {
		t.Fatalf("expected non-regular path to be rejected, got %s", path)
	}
}

func TestMountImageRejectsInvalidPathBeforeUSBRestart(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{"file":"/etc/passwd","readOnly":true}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/storage/image/mount", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	NewService().MountImage(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 response, got %d", recorder.Code)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"code":-2`)) {
		t.Fatalf("expected invalid arguments response, got %s", recorder.Body.String())
	}
}

func TestMountedImageRequiresExistingNonEmptyBackingFile(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing")
	file, mounted, err := readMountedImageFile(missingPath)
	if err != nil {
		t.Fatalf("missing backing file returned an error: %v", err)
	}
	if file != "" || mounted {
		t.Fatalf("missing backing file returned file=%q mounted=%v", file, mounted)
	}

	tests := []struct {
		name        string
		content     string
		wantFile    string
		wantMounted bool
	}{
		{name: "empty", content: "\n", wantFile: "", wantMounted: false},
		{name: "whitespace", content: " \t\n", wantFile: "", wantMounted: false},
		{
			name:        "image path",
			content:     "/data/ventoy-1.1.12-livecd.iso\n",
			wantFile:    "/data/ventoy-1.1.12-livecd.iso",
			wantMounted: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "file")
			if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
				t.Fatal(err)
			}

			file, mounted, err := readMountedImageFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if file != test.wantFile {
				t.Fatalf("file = %q, want %q", file, test.wantFile)
			}
			if mounted != test.wantMounted {
				t.Fatalf("mounted = %v, want %v", mounted, test.wantMounted)
			}
		})
	}
}

func TestSafeImageDeletePathRejectsTraversal(t *testing.T) {
	if path, ok := safeImageDeletePath("/data/../tmp/traversal-delete.iso"); ok {
		t.Fatalf("expected traversal to be rejected, got %s", path)
	}
}

type deleteImageTestResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func TestDeleteImageRejectsMissingChallengeBeforeRemove(t *testing.T) {
	setupDeleteImageHandlerTest(t)
	deleteImageActionRequiresChallenge = func() bool { return true }
	verifyDeleteImageActionChallenge = func(*gin.Context, string, string) error {
		return errors.New("challenge required")
	}
	resolveDeleteImagePath = func(string) (string, bool) {
		t.Fatal("path resolution must not run when challenge verification fails")
		return "", false
	}
	removeImageFile = func(string) error {
		t.Fatal("image file must not be removed when challenge verification fails")
		return nil
	}

	rsp := performDeleteImageRequest(t, "/data/boot.iso", "")
	if rsp.Code != -3 {
		t.Fatalf("expected challenge rejection code -3, got %d", rsp.Code)
	}
	if rsp.Msg != "sensitive action verification required" {
		t.Fatalf("unexpected rejection message: %q", rsp.Msg)
	}
}

func TestDeleteImageVerifiesChallengeAndRemovesFile(t *testing.T) {
	setupDeleteImageHandlerTest(t)
	deleteImageActionRequiresChallenge = func() bool { return true }
	verified := false
	verifyDeleteImageActionChallenge = func(_ *gin.Context, challenge, action string) error {
		if challenge != "delete-image-challenge" {
			t.Fatalf("expected delete image challenge, got %q", challenge)
		}
		if action != authservice.SensitiveActionDeleteImage {
			t.Fatalf("expected %s action, got %q", authservice.SensitiveActionDeleteImage, action)
		}
		verified = true
		return nil
	}
	resolved := false
	resolveDeleteImagePath = func(file string) (string, bool) {
		if file != "/data/boot.iso" {
			t.Fatalf("unexpected file path %q", file)
		}
		resolved = true
		return "/tmp/boot.iso", true
	}
	removed := false
	removeImageFile = func(path string) error {
		if path != "/tmp/boot.iso" {
			t.Fatalf("unexpected remove path %q", path)
		}
		removed = true
		return nil
	}

	rsp := performDeleteImageRequest(t, "/data/boot.iso", "delete-image-challenge")
	if rsp.Code != 0 {
		t.Fatalf("expected successful delete request, got %d (%s)", rsp.Code, rsp.Msg)
	}
	if !verified {
		t.Fatal("expected delete image challenge to be verified")
	}
	if !resolved {
		t.Fatal("expected delete path to be resolved after challenge verification")
	}
	if !removed {
		t.Fatal("expected image file to be removed after challenge verification")
	}
}

func TestDeleteImageSkipsChallengeWhenPolicyDisabled(t *testing.T) {
	setupDeleteImageHandlerTest(t)
	deleteImageActionRequiresChallenge = func() bool { return false }
	verifyDeleteImageActionChallenge = func(*gin.Context, string, string) error {
		t.Fatal("delete image challenge verifier must not run when policy is disabled")
		return nil
	}
	resolveDeleteImagePath = func(string) (string, bool) {
		return "/tmp/boot.iso", true
	}
	removed := false
	removeImageFile = func(string) error {
		removed = true
		return nil
	}

	rsp := performDeleteImageRequest(t, "/data/boot.iso", "")
	if rsp.Code != 0 {
		t.Fatalf("expected successful delete request, got %d", rsp.Code)
	}
	if !removed {
		t.Fatal("expected image file to be removed when policy is disabled")
	}
}

func setupDeleteImageHandlerTest(t *testing.T) {
	t.Helper()
	allowTestStorageAccess(t)

	gin.SetMode(gin.TestMode)
	originalRequires := deleteImageActionRequiresChallenge
	originalVerify := verifyDeleteImageActionChallenge
	originalRemove := removeImageFile
	originalResolve := resolveDeleteImagePath

	t.Cleanup(func() {
		deleteImageActionRequiresChallenge = originalRequires
		verifyDeleteImageActionChallenge = originalVerify
		removeImageFile = originalRemove
		resolveDeleteImagePath = originalResolve
	})
}

func performDeleteImageRequest(t *testing.T, file, challenge string) deleteImageTestResponse {
	t.Helper()

	body, err := json.Marshal(map[string]string{"file": file, "challenge": challenge})
	if err != nil {
		t.Fatalf("Marshal request: %v", err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/storage/image/delete", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	NewService().DeleteImage(ctx)

	var rsp deleteImageTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("Unmarshal response: %v body=%s", err, recorder.Body.String())
	}
	return rsp
}
