package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type memoryFabricResponse struct {
	Code int `json:"code"`
	Data struct {
		Enabled bool `json:"enabled"`
		Running bool `json:"running"`
	} `json:"data"`
}

func TestServiceStateManagerMemoryFabricHookSuccess(t *testing.T) {
	running := false
	var calls []string
	setTestAIRuntime(t, nil, &calls)
	setTestMemoryFabricManager(t, &running, nil, nil, nil, &calls)

	manager, path := newTestServiceStateManager(t)
	manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)

	snapshot, err := manager.Set(aiServiceMemoryFabric, true, "web")
	if err != nil {
		t.Fatalf("set memoryFabric enabled: %v", err)
	}
	if !memoryFabricCallsEqual(calls, []string{
		"runtime:true:true:false:true:0s",
		"start:memfab.service:true",
		"start:nano_ocr.service:true",
	}) {
		t.Fatalf("unexpected hook calls after enable: %#v", calls)
	}
	entry := snapshot.Services[aiServiceMemoryFabric]
	if snapshot.Revision != 2 || !entry.Enabled || entry.Source != "web" {
		t.Fatalf("unexpected enabled snapshot: %#v", snapshot)
	}

	_, saved := readServiceStateFile(t, path)
	if saved.Revision != 2 || saved.Services[aiServiceMemoryFabric] != entry {
		t.Fatalf("unexpected saved memoryFabric state after enable: %#v", saved)
	}

	snapshot, err = manager.Set(aiServiceMemoryFabric, false, "screen")
	if err != nil {
		t.Fatalf("set memoryFabric disabled: %v", err)
	}
	if !memoryFabricCallsEqual(calls, []string{
		"runtime:true:true:false:true:0s",
		"start:memfab.service:true",
		"start:nano_ocr.service:true",
		"stop:memfab.service:true",
		"stop:nano_ocr.service:true",
		"runtime:false:false:false:true:0s",
	}) {
		t.Fatalf("unexpected hook calls after disable: %#v", calls)
	}
	entry = snapshot.Services[aiServiceMemoryFabric]
	if snapshot.Revision != 3 || entry.Enabled || entry.Source != "screen" {
		t.Fatalf("unexpected disabled snapshot: %#v", snapshot)
	}
}

func TestServiceStateManagerMemoryFabricHookFailureDoesNotPersistOrBroadcast(t *testing.T) {
	running := false
	var calls []string
	setTestAIRuntime(t, nil, &calls)
	setTestMemoryFabricManager(t, &running, nil, errors.New("start failed"), nil, &calls)

	manager, path := newTestServiceStateManager(t)
	manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)
	if err := manager.Save(); err != nil {
		t.Fatalf("save initial state: %v", err)
	}
	before, beforeState := readServiceStateFile(t, path)
	ch := manager.Subscribe()

	if _, err := manager.Set(aiServiceMemoryFabric, true, "web"); err == nil {
		t.Fatalf("expected memoryFabric hook error")
	}
	if !memoryFabricCallsEqual(calls, []string{
		"runtime:true:true:false:true:0s",
		"start:memfab.service:true",
		"runtime:false:false:false:true:0s",
	}) {
		t.Fatalf("unexpected hook calls: %#v", calls)
	}

	after, afterState := readServiceStateFile(t, path)
	if !bytes.Equal(before, after) {
		t.Fatalf("expected state file unchanged after hook failure")
	}
	if afterState.Revision != beforeState.Revision ||
		afterState.Services[aiServiceMemoryFabric] != beforeState.Services[aiServiceMemoryFabric] {
		t.Fatalf("expected state unchanged, before=%#v after=%#v", beforeState, afterState)
	}

	select {
	case event := <-ch:
		t.Fatalf("unexpected event after hook failure: %#v", event)
	default:
	}
}

func TestServiceStateManagerMemoryFabricOCRStartFailureRollsBack(t *testing.T) {
	var calls []string
	setTestAIRuntime(t, nil, &calls)
	setTestMemoryFabricManagerWithOptions(t, testMemoryFabricManagerOptions{
		startErrs: map[string]error{
			memoryFabricOCRUnitName: errors.New("ocr start failed"),
		},
		calls: &calls,
	})

	manager, path := newTestServiceStateManager(t)
	manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)
	if err := manager.Save(); err != nil {
		t.Fatalf("save initial state: %v", err)
	}
	before, beforeState := readServiceStateFile(t, path)
	ch := manager.Subscribe()

	if _, err := manager.Set(aiServiceMemoryFabric, true, "web"); err == nil {
		t.Fatalf("expected memoryFabric OCR start error")
	}
	if !memoryFabricCallsEqual(calls, []string{
		"runtime:true:true:false:true:0s",
		"start:memfab.service:true",
		"start:nano_ocr.service:true",
		"stop:memfab.service:true",
		"runtime:false:false:false:true:0s",
	}) {
		t.Fatalf("unexpected hook calls after OCR start failure: %#v", calls)
	}

	after, afterState := readServiceStateFile(t, path)
	if !bytes.Equal(before, after) {
		t.Fatalf("expected state file unchanged after OCR start failure")
	}
	if afterState.Revision != beforeState.Revision ||
		afterState.Services[aiServiceMemoryFabric] != beforeState.Services[aiServiceMemoryFabric] {
		t.Fatalf("expected state unchanged, before=%#v after=%#v", beforeState, afterState)
	}

	select {
	case event := <-ch:
		t.Fatalf("unexpected event after OCR start failure: %#v", event)
	default:
	}
}

func TestServiceStateManagerMemoryFabricOCRStopFailureRollsBack(t *testing.T) {
	var calls []string
	setTestAIRuntime(t, nil, &calls)
	setTestMemoryFabricManagerWithOptions(t, testMemoryFabricManagerOptions{
		running: map[string]bool{
			memoryFabricUnitName:    true,
			memoryFabricOCRUnitName: true,
		},
		stopErrs: map[string]error{
			memoryFabricOCRUnitName: errors.New("ocr stop failed"),
		},
		calls: &calls,
	})

	manager, path := newTestServiceStateManager(t)
	initial := defaultServiceStateSnapshot()
	initial.Revision = 7
	initial.Services[aiServiceMemoryFabric] = ServiceStateEntry{
		Enabled:   true,
		UpdatedAt: "2026-06-18T00:00:00Z",
		Source:    "test",
	}
	if err := saveServiceStateToPath(path, initial); err != nil {
		t.Fatalf("write initial state: %v", err)
	}
	if err := manager.Load(); err != nil {
		t.Fatalf("load initial state: %v", err)
	}
	manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)
	before, beforeState := readServiceStateFile(t, path)
	ch := manager.Subscribe()

	if _, err := manager.Set(aiServiceMemoryFabric, false, "web"); err == nil {
		t.Fatalf("expected memoryFabric OCR stop error")
	}
	if !memoryFabricCallsEqual(calls, []string{
		"stop:memfab.service:true",
		"stop:nano_ocr.service:true",
		"start:memfab.service:true",
	}) {
		t.Fatalf("unexpected hook calls after OCR stop failure: %#v", calls)
	}

	after, afterState := readServiceStateFile(t, path)
	if !bytes.Equal(before, after) {
		t.Fatalf("expected state file unchanged after OCR stop failure")
	}
	if afterState.Revision != beforeState.Revision ||
		afterState.Services[aiServiceMemoryFabric] != beforeState.Services[aiServiceMemoryFabric] {
		t.Fatalf("expected state unchanged, before=%#v after=%#v", beforeState, afterState)
	}

	select {
	case event := <-ch:
		t.Fatalf("unexpected event after OCR stop failure: %#v", event)
	default:
	}
}

func TestMemoryFabricStatusRequiresBothUnitsRunning(t *testing.T) {
	tests := []struct {
		name    string
		running map[string]bool
		want    bool
	}{
		{
			name: "both running",
			running: map[string]bool{
				memoryFabricUnitName:    true,
				memoryFabricOCRUnitName: true,
			},
			want: true,
		},
		{
			name: "memfab stopped",
			running: map[string]bool{
				memoryFabricUnitName:    false,
				memoryFabricOCRUnitName: true,
			},
			want: false,
		},
		{
			name: "ocr stopped",
			running: map[string]bool{
				memoryFabricUnitName:    true,
				memoryFabricOCRUnitName: false,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setTestMemoryFabricManagerWithOptions(t, testMemoryFabricManagerOptions{
				running: tt.running,
			})

			got, err := newMemoryFabricManager().status()
			if err != nil {
				t.Fatalf("get status: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected running=%t, got %t", tt.want, got)
			}
		})
	}
}

func TestMemoryFabricWebUsesServiceStateAndRunningStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withAIStateFiles(t)
	running := false
	var calls []string
	setTestMemoryFabricManager(t, &running, nil, nil, nil, &calls)
	writeAIServiceState(t, aiServiceMemoryFabric, true)

	service := &Service{}
	c, recorder := newTestContext(http.MethodGet, "/api/ai/memory-fabric", "")
	service.GetMemoryFabric(c)
	rsp := decodeMemoryFabricResponse(t, recorder)
	if rsp.Code != 0 || !rsp.Data.Enabled || rsp.Data.Running {
		t.Fatalf("expected enabled=true running=false response, got %#v", rsp)
	}

	c, recorder = newTestContext(http.MethodPost, "/api/ai/memory-fabric", `{"enabled":true}`)
	service.SetMemoryFabric(c)
	rsp = decodeMemoryFabricResponse(t, recorder)
	if rsp.Code != 0 || !rsp.Data.Enabled || !rsp.Data.Running {
		t.Fatalf("expected enabled=true running=true response after start, got %#v", rsp)
	}
	if len(calls) < 4 || !memoryFabricCallsEqual(calls[len(calls)-4:len(calls)-2], []string{
		"start:memfab.service:true",
		"start:nano_ocr.service:true",
	}) {
		t.Fatalf("expected start call before status, calls=%#v", calls)
	}

	snapshot, err := DefaultServiceStateManager().List()
	if err != nil {
		t.Fatalf("list service state after memoryFabric enable: %v", err)
	}
	entry := snapshot.Services[aiServiceMemoryFabric]
	if !entry.Enabled || entry.Source != "web" {
		t.Fatalf("expected memoryFabric service state enabled from web, got %#v", entry)
	}

	c, recorder = newTestContext(http.MethodPost, "/api/ai/memory-fabric", `{"enabled":false}`)
	service.SetMemoryFabric(c)
	rsp = decodeMemoryFabricResponse(t, recorder)
	if rsp.Code != 0 || rsp.Data.Enabled || rsp.Data.Running {
		t.Fatalf("expected enabled=false running=false response after stop, got %#v", rsp)
	}
	if len(calls) < 4 || !memoryFabricCallsEqual(calls[len(calls)-4:len(calls)-2], []string{
		"stop:memfab.service:true",
		"stop:nano_ocr.service:true",
	}) {
		t.Fatalf("expected stop call before status, calls=%#v", calls)
	}
}

func TestMemoryFabricWebSetFailureDoesNotWriteServiceState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withAIStateFiles(t)
	running := false
	var calls []string
	setTestMemoryFabricManager(t, &running, nil, errors.New("start failed"), nil, &calls)

	if err := DefaultServiceStateManager().Save(); err != nil {
		t.Fatalf("save initial service state: %v", err)
	}
	before, beforeState := readServiceStateFile(t, serviceStateFilePath)

	service := &Service{}
	c, recorder := newTestContext(http.MethodPost, "/api/ai/memory-fabric", `{"enabled":true}`)
	service.SetMemoryFabric(c)
	rsp := decodeMemoryFabricResponse(t, recorder)
	if rsp.Code == 0 {
		t.Fatalf("expected failed response, got %#v", rsp)
	}

	after, afterState := readServiceStateFile(t, serviceStateFilePath)
	if !bytes.Equal(before, after) {
		t.Fatalf("expected service state file unchanged after failed memoryFabric set")
	}
	if afterState.Revision != beforeState.Revision ||
		afterState.Services[aiServiceMemoryFabric] != beforeState.Services[aiServiceMemoryFabric] {
		t.Fatalf("expected service state unchanged, before=%#v after=%#v", beforeState, afterState)
	}
	if len(calls) != 1 || calls[0] != "start:memfab.service:true" {
		t.Fatalf("unexpected calls after failed memoryFabric set: %#v", calls)
	}
}

func TestStateSocketMemoryFabricIncludesRunningAndUsesHook(t *testing.T) {
	env := newTestServiceStateEnv(t)
	setTestStateSocketCapabilities(t, 512, chipTypeAX620QE)
	running := false
	var calls []string
	setTestAIRuntime(t, nil, nil)
	setTestMemoryFabricManager(t, &running, nil, nil, nil, &calls)
	env.manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)

	_, client, _ := startTestStateSocketServer(t, env)

	resp := doStateSocketRequest(t, client, http.MethodGet, "/v1/services", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var data stateSocketServicesResponse
	if err := decodeJSONResponse(resp, &data); err != nil {
		t.Fatalf("decode services response: %v", err)
	}
	memoryFabric := data.Services[aiServiceMemoryFabric]
	if memoryFabric.Enabled || memoryFabric.Running == nil || *memoryFabric.Running {
		t.Fatalf("unexpected memoryFabric socket response before set: %#v", memoryFabric)
	}

	resp = doStateSocketRequest(t, client, http.MethodPut, "/v1/services/memoryFabric", []byte(`{"enabled":true,"source":"screen"}`))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 after socket PUT, got %d", resp.StatusCode)
	}
	data = stateSocketServicesResponse{}
	if err := decodeJSONResponse(resp, &data); err != nil {
		t.Fatalf("decode put response: %v", err)
	}
	memoryFabric = data.Services[aiServiceMemoryFabric]
	if !memoryFabric.Enabled || memoryFabric.Running == nil || !*memoryFabric.Running ||
		memoryFabric.Source != "screen" {
		t.Fatalf("unexpected memoryFabric socket response after set: %#v", memoryFabric)
	}
	if len(calls) < 4 || !memoryFabricCallsEqual(calls[len(calls)-4:len(calls)-2], []string{
		"start:memfab.service:true",
		"start:nano_ocr.service:true",
	}) {
		t.Fatalf("expected socket PUT to start memoryFabric, calls=%#v", calls)
	}
}

func TestStateSocketMemoryFabricSetFailureDoesNotWriteState(t *testing.T) {
	env := newTestServiceStateEnv(t)
	setTestStateSocketCapabilities(t, 512, chipTypeAX620QE)
	running := false
	setTestAIRuntime(t, nil, nil)
	setTestMemoryFabricManager(t, &running, nil, errors.New("start failed"), nil, nil)
	env.manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)

	_, client, _ := startTestStateSocketServer(t, env)
	before, beforeState := readServiceStateFile(t, env.statePath)

	resp := doStateSocketRequest(t, client, http.MethodPut, "/v1/services/memoryFabric", []byte(`{"enabled":true,"source":"screen"}`))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.StatusCode)
	}

	after, afterState := readServiceStateFile(t, env.statePath)
	if !bytes.Equal(before, after) {
		t.Fatalf("expected service state file unchanged after socket set failure")
	}
	if afterState.Revision != beforeState.Revision ||
		afterState.Services[aiServiceMemoryFabric] != beforeState.Services[aiServiceMemoryFabric] {
		t.Fatalf("expected service state unchanged, before=%#v after=%#v", beforeState, afterState)
	}
}

func decodeMemoryFabricResponse(t *testing.T, recorder *httptest.ResponseRecorder) memoryFabricResponse {
	t.Helper()

	var rsp memoryFabricResponse
	if err := decodeJSONBytes(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return rsp
}

func decodeJSONResponse(resp *http.Response, v interface{}) error {
	return json.NewDecoder(resp.Body).Decode(v)
}

func decodeJSONBytes(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

type testMemoryFabricManagerOptions struct {
	running    map[string]bool
	statusErrs map[string]error
	startErrs  map[string]error
	stopErrs   map[string]error
	calls      *[]string
}

func setTestMemoryFabricManager(
	t *testing.T,
	running *bool,
	statusErr error,
	startErr error,
	stopErr error,
	calls *[]string,
) {
	t.Helper()

	if running == nil {
		defaultRunning := false
		running = &defaultRunning
	}

	statusErrs := make(map[string]error)
	if statusErr != nil {
		statusErrs[memoryFabricUnitName] = statusErr
	}

	startErrs := make(map[string]error)
	if startErr != nil {
		startErrs[memoryFabricUnitName] = startErr
	}

	stopErrs := make(map[string]error)
	if stopErr != nil {
		stopErrs[memoryFabricUnitName] = stopErr
	}

	setTestMemoryFabricManagerWithOptions(t, testMemoryFabricManagerOptions{
		running: map[string]bool{
			memoryFabricUnitName:    *running,
			memoryFabricOCRUnitName: *running,
		},
		statusErrs: statusErrs,
		startErrs:  startErrs,
		stopErrs:   stopErrs,
		calls:      calls,
	})
}

func setTestMemoryFabricManagerWithOptions(t *testing.T, opts testMemoryFabricManagerOptions) {
	t.Helper()

	oldNewMemoryFabricManager := newMemoryFabricManager
	if opts.running == nil {
		opts.running = map[string]bool{
			memoryFabricUnitName:    false,
			memoryFabricOCRUnitName: false,
		}
	}
	if opts.statusErrs == nil {
		opts.statusErrs = make(map[string]error)
	}
	if opts.startErrs == nil {
		opts.startErrs = make(map[string]error)
	}
	if opts.stopErrs == nil {
		opts.stopErrs = make(map[string]error)
	}

	newMemoryFabricManager = func() memoryFabricManager {
		return memoryFabricManager{
			unitName:    memoryFabricUnitName,
			ocrUnitName: memoryFabricOCRUnitName,
			isRunning: func(unit string) (bool, error) {
				appendMemoryFabricCall(opts.calls, "status:%s", unit)
				if err := opts.statusErrs[unit]; err != nil {
					return false, err
				}
				return opts.running[unit], nil
			},
			startService: func(unit string, enable bool) error {
				appendMemoryFabricCall(opts.calls, "start:%s:%t", unit, enable)
				if err := opts.startErrs[unit]; err != nil {
					return err
				}
				opts.running[unit] = true
				return nil
			},
			stopService: func(unit string, disable bool) error {
				appendMemoryFabricCall(opts.calls, "stop:%s:%t", unit, disable)
				if err := opts.stopErrs[unit]; err != nil {
					return err
				}
				opts.running[unit] = false
				return nil
			},
		}
	}

	t.Cleanup(func() {
		newMemoryFabricManager = oldNewMemoryFabricManager
	})
}

func appendMemoryFabricCall(calls *[]string, format string, args ...interface{}) {
	if calls == nil {
		return
	}
	*calls = append(*calls, fmt.Sprintf(format, args...))
}

func memoryFabricCallsEqual(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}
