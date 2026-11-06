package vm

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type edidTestResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func TestStripEdidAudio(t *testing.T) {
	input, err := os.ReadFile(filepath.Join("../../../debhelper/packages/kvmcomm/rootfs/kvmcomm/edid", "nanokvm-go-1920x1080-60.bin"))
	if err != nil {
		t.Fatal(err)
	}

	output, err := stripEdidAudio(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(output) != len(input) {
		t.Fatalf("output size = %d, want %d", len(output), len(input))
	}
	if string(output[:128]) != string(input[:128]) {
		t.Fatal("base EDID block was modified")
	}

	cta := output[128:256]
	if cta[3]&0x40 != 0 {
		t.Fatal("CTA Basic Audio flag is still set")
	}
	for pos, end := 4, int(cta[2]); pos < end; {
		header := cta[pos]
		length := int(header & 0x1f)
		tag := header >> 5
		if tag == 1 || tag == 4 {
			t.Fatalf("audio-related CTA block tag %d is still present", tag)
		}
		pos += 1 + length
	}

	var sum byte
	for _, value := range cta {
		sum += value
	}
	if sum != 0 {
		t.Fatalf("CTA checksum = %#x, want 0", sum)
	}
}

func TestStripEdidAudioRejectsMalformedInput(t *testing.T) {
	if _, err := stripEdidAudio(make([]byte, 127)); err == nil {
		t.Fatal("expected invalid EDID size error")
	}

	input := make([]byte, 128)
	input[126] = 1
	if _, err := stripEdidAudio(input); err == nil {
		t.Fatal("expected invalid extension count error")
	}
}

func TestSwitchEdidDefaultSavesCurrentState(t *testing.T) {
	setupEdidTestPaths(t)

	name := "E54-1080P60FPS"
	content := makeEdidTestData(3)
	defaultFile := filepath.Join(EdidDir, name+".bin")
	writeEdidTestFile(t, defaultFile, content)
	writeDefaultEdidTestFile(t, makeEdidTestData(4))
	if err := os.MkdirAll(filepath.Dir(LT7911Edid), 0o755); err != nil {
		t.Fatal(err)
	}

	rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
	if rsp.Code != 0 {
		t.Fatalf("SwitchEdid failed: code=%d msg=%s", rsp.Code, rsp.Msg)
	}

	if got := readEdidTestFile(t, LT7911Edid); string(got) != string(content) {
		t.Fatalf("hardware EDID = %q, want %q", got, content)
	}

	state := readEdidTestState(t)
	if state.Name != name || state.Source != edidSourceDefault || state.File != defaultFile {
		t.Fatalf("state = %+v, want name=%s source=%s file=%s", state, name, edidSourceDefault, defaultFile)
	}

	info, err := os.Stat(CurrentEdidStateFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("state file mode = %o, want 644", info.Mode().Perm())
	}
}

func TestSwitchEdidCustomSavesCurrentState(t *testing.T) {
	setupEdidTestPaths(t)

	name := "custom.bin"
	content := makeEdidTestData(5)
	customFile := filepath.Join(CustomEdidDir, name)
	writeEdidTestFile(t, customFile, content)
	writeDefaultEdidTestFile(t, makeEdidTestData(6))
	if err := os.MkdirAll(filepath.Dir(LT7911Edid), 0o755); err != nil {
		t.Fatal(err)
	}

	rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
	if rsp.Code != 0 {
		t.Fatalf("SwitchEdid failed: code=%d msg=%s", rsp.Code, rsp.Msg)
	}

	if got := readEdidTestFile(t, LT7911Edid); string(got) != string(content) {
		t.Fatalf("hardware EDID = %q, want %q", got, content)
	}

	state := readEdidTestState(t)
	if state.Name != name || state.Source != edidSourceCustom || state.File != customFile {
		t.Fatalf("state = %+v, want name=%s source=%s file=%s", state, name, edidSourceCustom, customFile)
	}

	if _, err := os.Stat(filepath.Join(CustomEdidDir, "edid_flag")); !os.IsNotExist(err) {
		t.Fatalf("edid_flag stat err = %v, want not exist", err)
	}
}

func TestSwitchEdidMissingPreviousStateDoesNotWriteHardware(t *testing.T) {
	setupEdidTestPaths(t)

	name := "E54-1080P60FPS"
	writeEdidTestFile(t, filepath.Join(EdidDir, name+".bin"), makeEdidTestData(7))

	oldWrite := writeEdidHardware
	t.Cleanup(func() { writeEdidHardware = oldWrite })
	writes := 0
	writeEdidHardware = func([]byte) error {
		writes++
		return nil
	}

	rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
	if rsp.Code == 0 {
		t.Fatal("SwitchEdid succeeded, want failure")
	}
	if writes != 0 {
		t.Fatalf("hardware writes = %d, want 0", writes)
	}

	if _, err := os.Stat(CurrentEdidStateFile); !os.IsNotExist(err) {
		t.Fatalf("current state stat err = %v, want not exist", err)
	}
}

func TestSwitchEdidHardwareFailureRollsBackAndKeepsCurrentState(t *testing.T) {
	setupEdidTestPaths(t)

	previousContent := makeEdidTestData(8)
	previousFile := writeDefaultEdidTestFile(t, previousContent)
	previousState := edidState{
		Name:   defaultCurrentEdidName,
		Source: edidSourceDefault,
		File:   previousFile,
	}
	writeEdidTestState(t, previousState)
	writeEdidTestFile(t, LT7911Edid, previousContent)

	name := "E54-1080P60FPS"
	writeEdidTestFile(t, filepath.Join(EdidDir, name+".bin"), makeEdidTestData(9))

	oldWrite := writeEdidHardware
	t.Cleanup(func() { writeEdidHardware = oldWrite })
	writes := 0
	writeEdidHardware = func(data []byte) error {
		writes++
		if writes == 1 {
			if err := os.WriteFile(LT7911Edid, []byte("partial"), 0o644); err != nil {
				t.Fatal(err)
			}
			return errors.New("injected hardware failure")
		}
		return os.WriteFile(LT7911Edid, data, 0o644)
	}

	rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
	if rsp.Code == 0 {
		t.Fatal("SwitchEdid succeeded, want failure")
	}
	if writes != 2 {
		t.Fatalf("hardware writes = %d, want apply plus rollback", writes)
	}
	if got := readEdidTestFile(t, LT7911Edid); string(got) != string(previousContent) {
		t.Fatalf("hardware EDID = %q, want %q", got, previousContent)
	}
	if got := readEdidTestState(t); got != previousState {
		t.Fatalf("current state = %+v, want %+v", got, previousState)
	}
}

func TestSwitchEdidStateSaveFailureRollsBackHardware(t *testing.T) {
	setupEdidTestPaths(t)

	previousContent := makeEdidTestData(10)
	previousFile := writeDefaultEdidTestFile(t, previousContent)
	previousState := edidState{
		Name:   defaultCurrentEdidName,
		Source: edidSourceDefault,
		File:   previousFile,
	}
	writeEdidTestState(t, previousState)
	writeEdidTestFile(t, LT7911Edid, previousContent)

	name := "E54-1080P60FPS"
	writeEdidTestFile(t, filepath.Join(EdidDir, name+".bin"), makeEdidTestData(11))

	oldSave := saveCurrentEdidStateAction
	t.Cleanup(func() { saveCurrentEdidStateAction = oldSave })
	saveCurrentEdidStateAction = func(edidState) error {
		return errors.New("injected state failure")
	}

	rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
	if rsp.Code != -5 {
		t.Fatalf("response code = %d, want -5", rsp.Code)
	}
	if got := readEdidTestFile(t, LT7911Edid); string(got) != string(previousContent) {
		t.Fatalf("hardware EDID = %q, want %q", got, previousContent)
	}
	if got := readEdidTestState(t); got != previousState {
		t.Fatalf("current state = %+v, want %+v", got, previousState)
	}
}

func TestGetEdidReadsCurrentState(t *testing.T) {
	setupEdidTestPaths(t)

	name := "E54-1080P60FPS"
	defaultFile := filepath.Join(EdidDir, name+".bin")
	writeEdidTestFile(t, defaultFile, []byte("default-edid"))
	writeEdidTestState(t, edidState{
		Name:   name,
		Source: edidSourceDefault,
		File:   defaultFile,
	})

	rsp := getEdidTest(t, NewService().GetEdid)
	if rsp.Code != 0 {
		t.Fatalf("GetEdid failed: code=%d msg=%s", rsp.Code, rsp.Msg)
	}

	var data struct {
		Edid string `json:"edid"`
	}
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.Edid != name {
		t.Fatalf("edid = %q, want %q", data.Edid, name)
	}
}

func TestGetEdidMissingStateCreatesDefaultWithoutWritingHardware(t *testing.T) {
	setupEdidTestPaths(t)

	defaultFile := filepath.Join(EdidDir, defaultCurrentEdidName+".bin")
	writeEdidTestFile(t, defaultFile, []byte("default-edid"))

	rsp := getEdidTest(t, NewService().GetEdid)
	if rsp.Code != 0 {
		t.Fatalf("GetEdid failed: code=%d msg=%s", rsp.Code, rsp.Msg)
	}

	var data struct {
		Edid string `json:"edid"`
	}
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.Edid != defaultCurrentEdidName {
		t.Fatalf("edid = %q, want %q", data.Edid, defaultCurrentEdidName)
	}

	state := readEdidTestState(t)
	if state.Name != defaultCurrentEdidName || state.Source != edidSourceDefault || state.File != defaultFile {
		t.Fatalf("state = %+v, want name=%s source=%s file=%s", state, defaultCurrentEdidName, edidSourceDefault, defaultFile)
	}

	if _, err := os.Stat(LT7911Edid); !os.IsNotExist(err) {
		t.Fatalf("hardware EDID stat err = %v, want not exist", err)
	}
}

func TestGetEdidMissingStateFailsWhenDefaultFileMissing(t *testing.T) {
	setupEdidTestPaths(t)

	rsp := getEdidTest(t, NewService().GetEdid)
	if rsp.Code == 0 {
		t.Fatal("GetEdid succeeded, want failure")
	}
	if rsp.Msg != "get edid failed" {
		t.Fatalf("message = %q, want get edid failed", rsp.Msg)
	}

	if _, err := os.Stat(CurrentEdidStateFile); !os.IsNotExist(err) {
		t.Fatalf("current state stat err = %v, want not exist", err)
	}
}

func TestGetEdidInvalidCurrentStateFails(t *testing.T) {
	tests := []struct {
		name    string
		arrange func(t *testing.T)
	}{
		{
			name: "invalid json",
			arrange: func(t *testing.T) {
				writeEdidTestFile(t, CurrentEdidStateFile, []byte("{"))
			},
		},
		{
			name: "missing field",
			arrange: func(t *testing.T) {
				writeEdidTestFile(t, CurrentEdidStateFile, []byte(`{"name":"E54-1080P60FPS","source":"default"}`))
			},
		},
		{
			name: "invalid source",
			arrange: func(t *testing.T) {
				writeEdidTestFile(t, CurrentEdidStateFile, []byte(`{"name":"E54-1080P60FPS","source":"bad","file":"/tmp/edid.bin"}`))
			},
		},
		{
			name: "missing referenced file",
			arrange: func(t *testing.T) {
				name := "E54-1080P60FPS"
				writeEdidTestState(t, edidState{
					Name:   name,
					Source: edidSourceDefault,
					File:   filepath.Join(EdidDir, name+".bin"),
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupEdidTestPaths(t)
			tt.arrange(t)

			rsp := getEdidTest(t, NewService().GetEdid)
			if rsp.Code == 0 {
				t.Fatal("GetEdid succeeded, want failure")
			}
			if rsp.Msg != "get edid failed" {
				t.Fatalf("message = %q, want get edid failed", rsp.Msg)
			}
		})
	}
}

func TestDeleteEdidRejectsCurrentCustom(t *testing.T) {
	setupEdidTestPaths(t)

	name := "current.bin"
	customFile := filepath.Join(CustomEdidDir, name)
	writeEdidTestFile(t, customFile, []byte("custom-edid"))
	writeEdidTestState(t, edidState{
		Name:   name,
		Source: edidSourceCustom,
		File:   customFile,
	})

	rsp := postEdidTestForm(t, NewService().DeleteEdid, url.Values{"edid": {name}})
	if rsp.Code == 0 {
		t.Fatal("DeleteEdid succeeded, want failure")
	}
	if rsp.Msg != "current EDID cannot be deleted" {
		t.Fatalf("message = %q, want current EDID cannot be deleted", rsp.Msg)
	}
	if _, err := os.Stat(customFile); err != nil {
		t.Fatalf("current custom EDID was removed: %s", err)
	}
}

func TestDeleteEdidAllowsNonCurrentCustom(t *testing.T) {
	setupEdidTestPaths(t)

	currentName := "current.bin"
	currentFile := filepath.Join(CustomEdidDir, currentName)
	otherName := "other.bin"
	otherFile := filepath.Join(CustomEdidDir, otherName)
	writeEdidTestFile(t, currentFile, []byte("current-edid"))
	writeEdidTestFile(t, otherFile, []byte("other-edid"))
	writeEdidTestState(t, edidState{
		Name:   currentName,
		Source: edidSourceCustom,
		File:   currentFile,
	})

	rsp := postEdidTestForm(t, NewService().DeleteEdid, url.Values{"edid": {otherName}})
	if rsp.Code != 0 {
		t.Fatalf("DeleteEdid failed: code=%d msg=%s", rsp.Code, rsp.Msg)
	}
	if _, err := os.Stat(otherFile); !os.IsNotExist(err) {
		t.Fatalf("other EDID stat err = %v, want not exist", err)
	}
	if _, err := os.Stat(currentFile); err != nil {
		t.Fatalf("current EDID stat failed: %s", err)
	}
}

func TestSwitchEdidRejectsUnsafeName(t *testing.T) {
	setupEdidTestPaths(t)

	rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {"../escape.bin"}})
	if rsp.Code == 0 {
		t.Fatal("SwitchEdid succeeded, want failure")
	}
	if rsp.Msg != "invalid EDID" {
		t.Fatalf("message = %q, want invalid EDID", rsp.Msg)
	}
}

func TestSafeCustomEdidPathRejectsTraversal(t *testing.T) {
	setupEdidTestPaths(t)

	if path, ok := safeCustomEdidPath("../escape.bin"); ok {
		t.Fatalf("expected traversal to be rejected, got %s", path)
	}
	if path, ok := safeCustomEdidPath("nested/escape.bin"); ok {
		t.Fatalf("expected nested path to be rejected, got %s", path)
	}
}

func setupEdidTestPaths(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	root := t.TempDir()
	oldEdidDir := EdidDir
	oldLT7911Edid := LT7911Edid
	oldCustomEdidDir := CustomEdidDir
	oldCurrentEdidStateFile := CurrentEdidStateFile
	oldEdidConfigFile := EdidConfigFile

	EdidDir = filepath.Join(root, "kvmcomm", "edid")
	LT7911Edid = filepath.Join(root, "proc", "lt7911_info", "edid")
	CustomEdidDir = filepath.Join(root, "etc", "kvm", "edid")
	CurrentEdidStateFile = filepath.Join(CustomEdidDir, "current.json")
	EdidConfigFile = filepath.Join(CustomEdidDir, "config.json")

	t.Cleanup(func() {
		EdidDir = oldEdidDir
		LT7911Edid = oldLT7911Edid
		CustomEdidDir = oldCustomEdidDir
		CurrentEdidStateFile = oldCurrentEdidStateFile
		EdidConfigFile = oldEdidConfigFile
	})
}

func postEdidTestForm(t *testing.T, handler gin.HandlerFunc, values url.Values) edidTestResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return performEdidTestRequest(t, handler, req)
}

func getEdidTest(t *testing.T, handler gin.HandlerFunc) edidTestResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	return performEdidTestRequest(t, handler, req)
}

func performEdidTestRequest(t *testing.T, handler gin.HandlerFunc, req *http.Request) edidTestResponse {
	t.Helper()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var rsp edidTestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response %q failed: %s", w.Body.String(), err)
	}

	return rsp
}

func writeEdidTestState(t *testing.T, state edidState) {
	t.Helper()

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}

	writeEdidTestFile(t, CurrentEdidStateFile, data)
}

func writeDefaultEdidTestFile(t *testing.T, data []byte) string {
	t.Helper()

	path := filepath.Join(EdidDir, defaultCurrentEdidName+".bin")
	writeEdidTestFile(t, path, data)
	return path
}

func readEdidTestState(t *testing.T) edidState {
	t.Helper()

	var state edidState
	data := readEdidTestFile(t, CurrentEdidStateFile)
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}

	return state
}

func writeEdidTestFile(t *testing.T, path string, data []byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readEdidTestFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func makeEdidTestData(marker byte) []byte {
	data := makeMonitorNameTestEdid(1, 90)
	data[8] = marker
	updateEdidBlockChecksum(data[:edidBlockSize])
	return data
}
