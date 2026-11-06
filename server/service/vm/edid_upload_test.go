package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestUploadEdidRejectsMalformedRequestsAndFilesWithoutSaving(t *testing.T) {
	tests := []struct {
		name             string
		filename         string
		data             []byte
		missing          bool
		oversizedRequest bool
		wantReason       edidErrorReason
	}{
		{name: "missing file", missing: true, wantReason: edidReasonBadRequest},
		{name: "invalid file name", filename: "display.edid", data: makeValidationTestEdid(1, 0), wantReason: edidReasonInvalidFileName},
		{name: "truncated 127 bytes", filename: "truncated.bin", data: make([]byte, 127), wantReason: edidReasonInvalidLength},
		{name: "invalid header", filename: "header.bin", data: invalidHeaderUploadTestEdid(), wantReason: edidReasonInvalidHeader},
		{name: "actual 513 bytes", filename: "large.bin", data: make([]byte, edidReadLimit), wantReason: edidReasonTooLarge},
		{
			name:             "oversized request with valid EDID file",
			filename:         "oversized.bin",
			data:             makeValidationTestEdid(1, 0),
			oversizedRequest: true,
			wantReason:       edidReasonTooLarge,
		},
		{name: "invalid CTA offset", filename: "cta.bin", data: invalidCTAUploadTestEdid(), wantReason: edidReasonInvalidCTAExtension},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupEdidTestPaths(t)

			var req *http.Request
			if tt.missing {
				req = newEdidUploadTestRequest(t, "", nil, false)
			} else if tt.oversizedRequest {
				req = newOversizedEdidUploadTestRequest(t, tt.filename, tt.data)
			} else {
				req = newEdidUploadTestRequest(t, tt.filename, tt.data, true)
			}
			rsp := performEdidTestRequest(t, NewService().UploadEdid, req)
			if rsp.Code == 0 {
				t.Fatal("UploadEdid succeeded, want rejection")
			}
			if got := decodeEdidTestReason(t, rsp); got != tt.wantReason {
				t.Fatalf("reason = %q, want %q", got, tt.wantReason)
			}
			if _, err := os.Stat(CustomEdidDir); !os.IsNotExist(err) {
				t.Fatalf("custom EDID directory was created for rejected upload: %v", err)
			}
		})
	}
}

func newOversizedEdidUploadTestRequest(t *testing.T, filename string, data []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	padding, err := writer.CreateFormField("padding")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := padding.Write(bytes.Repeat([]byte{'x'}, edidUploadRequestLimit)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestUploadEdidInvalidReplacementPreservesExistingFile(t *testing.T) {
	setupEdidTestPaths(t)

	name := "existing.bin"
	target := filepath.Join(CustomEdidDir, name)
	oldData := makeEdidTestData(20)
	writeEdidTestFile(t, target, oldData)

	rsp := performEdidTestRequest(
		t,
		NewService().UploadEdid,
		newEdidUploadTestRequest(t, name, make([]byte, 127), true),
	)
	if rsp.Code == 0 || decodeEdidTestReason(t, rsp) != edidReasonInvalidLength {
		t.Fatalf("response = code %d msg %q reason %q", rsp.Code, rsp.Msg, decodeEdidTestReason(t, rsp))
	}
	if got := readEdidTestFile(t, target); !bytes.Equal(got, oldData) {
		t.Fatal("invalid upload changed the existing EDID")
	}
	assertNoEdidUploadTemps(t)
}

func TestUploadEdidSavesValidFileAtomically(t *testing.T) {
	setupEdidTestPaths(t)

	oldWrite := writeEdidHardware
	hardwareWrites := 0
	writeEdidHardware = func([]byte) error {
		hardwareWrites++
		return errors.New("upload must not write hardware EDID")
	}
	t.Cleanup(func() { writeEdidHardware = oldWrite })

	name := "valid.bin"
	data := makeEdidTestData(21)
	rsp := performEdidTestRequest(
		t,
		NewService().UploadEdid,
		newEdidUploadTestRequest(t, name, data, true),
	)
	if rsp.Code != 0 {
		t.Fatalf("UploadEdid failed: code=%d msg=%q", rsp.Code, rsp.Msg)
	}

	target := filepath.Join(CustomEdidDir, name)
	if got := readEdidTestFile(t, target); !bytes.Equal(got, data) {
		t.Fatal("saved EDID differs from upload")
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("saved mode = %o, want 644", info.Mode().Perm())
	}
	if hardwareWrites != 0 {
		t.Fatalf("hardware writes = %d, want 0", hardwareWrites)
	}
	assertNoEdidUploadTemps(t)
}

func TestUploadEdidRejectsOverwriteOfCurrentCustomFile(t *testing.T) {
	setupEdidTestPaths(t)

	name := "current.bin"
	target := filepath.Join(CustomEdidDir, name)
	oldData := makeEdidTestData(22)
	writeEdidTestFile(t, target, oldData)
	writeEdidTestState(t, edidState{Name: name, Source: edidSourceCustom, File: target})

	rsp := performEdidTestRequest(
		t,
		NewService().UploadEdid,
		newEdidUploadTestRequest(t, name, makeEdidTestData(23), true),
	)
	if rsp.Code == 0 {
		t.Fatal("UploadEdid overwrote the current custom EDID")
	}
	if got := decodeEdidTestReason(t, rsp); got != edidReasonCurrentCannotBeOverwritten {
		t.Fatalf("reason = %q, want %q", got, edidReasonCurrentCannotBeOverwritten)
	}
	if got := readEdidTestFile(t, target); !bytes.Equal(got, oldData) {
		t.Fatal("current custom EDID content changed")
	}
	assertNoEdidUploadTemps(t)
}

func TestUploadEdidAtomicallyReplacesNonCurrentFile(t *testing.T) {
	setupEdidTestPaths(t)

	currentName := "current.bin"
	currentPath := filepath.Join(CustomEdidDir, currentName)
	writeEdidTestFile(t, currentPath, makeEdidTestData(24))
	writeEdidTestState(t, edidState{Name: currentName, Source: edidSourceCustom, File: currentPath})

	name := "other.bin"
	target := filepath.Join(CustomEdidDir, name)
	writeEdidTestFile(t, target, makeEdidTestData(25))
	newData := makeEdidTestData(26)

	rsp := performEdidTestRequest(
		t,
		NewService().UploadEdid,
		newEdidUploadTestRequest(t, name, newData, true),
	)
	if rsp.Code != 0 {
		t.Fatalf("UploadEdid failed: code=%d msg=%q", rsp.Code, rsp.Msg)
	}
	if got := readEdidTestFile(t, target); !bytes.Equal(got, newData) {
		t.Fatal("non-current EDID was not replaced")
	}
	assertNoEdidUploadTemps(t)
}

func TestUploadEdidCommitFailurePreservesExistingFileAndCleansTemp(t *testing.T) {
	setupEdidTestPaths(t)

	name := "existing.bin"
	target := filepath.Join(CustomEdidDir, name)
	oldData := makeEdidTestData(27)
	writeEdidTestFile(t, target, oldData)

	oldRename := renameEdidFileAction
	renameEdidFileAction = func(_, _ string) error {
		return errors.New("injected rename failure")
	}
	t.Cleanup(func() { renameEdidFileAction = oldRename })

	rsp := performEdidTestRequest(
		t,
		NewService().UploadEdid,
		newEdidUploadTestRequest(t, name, makeEdidTestData(28), true),
	)
	if rsp.Code == 0 || decodeEdidTestReason(t, rsp) != edidReasonSaveFailed {
		t.Fatalf("response = code %d msg %q reason %q", rsp.Code, rsp.Msg, decodeEdidTestReason(t, rsp))
	}
	if got := readEdidTestFile(t, target); !bytes.Equal(got, oldData) {
		t.Fatal("commit failure changed the existing EDID")
	}
	assertNoEdidUploadTemps(t)
}

func TestUploadEdidRejectsNonRegularTarget(t *testing.T) {
	setupEdidTestPaths(t)

	name := "directory.bin"
	target := filepath.Join(CustomEdidDir, name)
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	rsp := performEdidTestRequest(
		t,
		NewService().UploadEdid,
		newEdidUploadTestRequest(t, name, makeEdidTestData(29), true),
	)
	if rsp.Code == 0 || decodeEdidTestReason(t, rsp) != edidReasonSaveFailed {
		t.Fatalf("response = code %d msg %q reason %q", rsp.Code, rsp.Msg, decodeEdidTestReason(t, rsp))
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		t.Fatalf("non-regular target changed: info=%v err=%v", info, err)
	}
	assertNoEdidUploadTemps(t)
}

func newEdidUploadTestRequest(t *testing.T, filename string, data []byte, includeFile bool) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if includeFile {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatal(err)
		}
	} else if err := writer.WriteField("not-file", "value"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func invalidHeaderUploadTestEdid() []byte {
	data := makeValidationTestEdid(2, 0xf0)
	data[0] = 1
	updateEdidBlockChecksum(data[:edidBlockSize])
	return data
}

func invalidCTAUploadTestEdid() []byte {
	data := makeValidationTestEdid(2, 0x02)
	cta := data[edidBlockSize:]
	cta[2] = 255
	updateEdidBlockChecksum(cta)
	return data
}

func decodeEdidTestReason(t *testing.T, rsp edidTestResponse) edidErrorReason {
	t.Helper()
	if len(rsp.Data) == 0 || bytes.Equal(rsp.Data, []byte("null")) {
		return ""
	}
	var data edidErrorResponse
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatal(err)
	}
	return data.Reason
}

func assertNoEdidUploadTemps(t *testing.T) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(CustomEdidDir, ".edid-upload-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary upload files remain: %v", matches)
	}
}
