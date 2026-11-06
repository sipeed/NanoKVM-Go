package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"NanoKVM-Server/proto"
)

func TestGetCustomEdidListKeepsLegacyInvalidFilesVisible(t *testing.T) {
	setupEdidTestPaths(t)

	writeEdidTestFile(t, filepath.Join(CustomEdidDir, "legacy-invalid.bin"), make([]byte, 127))
	rsp := getEdidTest(t, NewService().GetCustomEdidList)
	if rsp.Code != 0 {
		t.Fatalf("GetCustomEdidList failed: code=%d msg=%q", rsp.Code, rsp.Msg)
	}
	var data struct {
		EdidList []string `json:"edidList"`
	}
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatal(err)
	}
	if len(data.EdidList) != 1 || data.EdidList[0] != "legacy-invalid.bin" {
		t.Fatalf("EDID list = %v, want legacy invalid file", data.EdidList)
	}
}

func TestSwitchEdidRejectsInvalidSourceBeforeHardwareWrite(t *testing.T) {
	setupEdidTestPaths(t)

	writeDefaultEdidTestFile(t, makeEdidTestData(30))
	name := "invalid.bin"
	path := filepath.Join(CustomEdidDir, name)
	invalid := makeEdidTestData(31)
	invalid[0] = 1
	updateEdidBlockChecksum(invalid[:edidBlockSize])
	writeEdidTestFile(t, path, invalid)

	oldWrite := writeEdidHardware
	writes := 0
	writeEdidHardware = func([]byte) error {
		writes++
		return nil
	}
	t.Cleanup(func() { writeEdidHardware = oldWrite })

	rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
	if rsp.Code == 0 {
		t.Fatal("SwitchEdid accepted invalid source")
	}
	if got := decodeEdidTestReason(t, rsp); got != edidReasonInvalidHeader {
		t.Fatalf("reason = %q, want %q", got, edidReasonInvalidHeader)
	}
	if writes != 0 {
		t.Fatalf("hardware writes = %d, want 0", writes)
	}
}

func TestSetEdidMonitorNameReturnsValidationReasonBeforeHardwareWrite(t *testing.T) {
	setupEdidTestPaths(t)

	name := "invalid.bin"
	path := filepath.Join(CustomEdidDir, name)
	invalid := makeEdidTestData(32)
	invalid[20]++
	writeEdidTestFile(t, path, invalid)
	writeEdidTestState(t, edidState{Name: name, Source: edidSourceCustom, File: path})

	oldWrite := writeEdidHardware
	writes := 0
	writeEdidHardware = func([]byte) error {
		writes++
		return nil
	}
	t.Cleanup(func() { writeEdidHardware = oldWrite })

	rsp := postEdidTestForm(t, NewService().SetEdidMonitorName, url.Values{"monitorName": {"Desk-KVM"}})
	if rsp.Code == 0 {
		t.Fatal("SetEdidMonitorName accepted invalid current source")
	}
	if got := decodeEdidTestReason(t, rsp); got != edidReasonInvalidChecksum {
		t.Fatalf("reason = %q, want %q", got, edidReasonInvalidChecksum)
	}
	if writes != 0 {
		t.Fatalf("hardware writes = %d, want 0", writes)
	}
}

func TestApplyCurrentEdidInvalidSourceDoesNotWriteHardwareOrResetAudio(t *testing.T) {
	setupEdidTestPaths(t)
	setupVirtualDeviceHandlerTest(t)

	name := "legacy-invalid.bin"
	path := filepath.Join(CustomEdidDir, name)
	writeEdidTestFile(t, path, make([]byte, 127))
	writeEdidTestState(t, edidState{Name: name, Source: edidSourceCustom, File: path})

	oldWrite := writeEdidHardware
	writes := 0
	writeEdidHardware = func([]byte) error {
		writes++
		return nil
	}
	t.Cleanup(func() { writeEdidHardware = oldWrite })

	applyVirtualAudioEdid = applyCurrentEdid
	virtualMicMarkerExists = func() bool { return false }
	virtualSpeakerMarkerExists = func() bool { return false }
	virtualMicRuntimeMounted = func() bool { return false }
	virtualSpeakerRuntimeMounted = func() bool { return false }
	runVirtualDeviceCommands = func([]string) error {
		t.Fatal("USB gadget changed after invalid EDID rejection")
		return nil
	}
	resetVirtualAudioCapture = func() {
		t.Fatal("audio capture reset after invalid EDID rejection")
	}

	rsp := performVirtualDeviceUpdate(t, proto.UpdateVirtualDeviceReq{
		Device: "audio",
		Mode:   virtualAudioModeDuplex,
	})
	if rsp.Code != -4 {
		t.Fatalf("response code = %d, want -4", rsp.Code)
	}
	if writes != 0 {
		t.Fatalf("hardware writes = %d, want 0", writes)
	}
}

func TestReadEffectiveEdidValidatesProcessedMonitorNameAndAudioResult(t *testing.T) {
	setupEdidTestPaths(t)

	path := filepath.Join(EdidDir, "processed.bin")
	source := makeMonitorNameTestEdid(2, 90)
	cta := source[edidBlockSize:]
	cta[0] = 0x02
	cta[2] = 8
	cta[3] = 0x40
	cta[4] = 0x23 // Audio data block with a three-byte payload.
	copy(cta[5:8], []byte{1, 2, 3})
	updateEdidBlockChecksum(cta)
	writeEdidTestFile(t, path, source)

	processed, err := readEffectiveEdid(path, "Desk-KVM", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateEdid(processed); err != nil {
		t.Fatalf("processed EDID failed validation: %v", err)
	}
	if bytes.Equal(processed, source) {
		t.Fatal("monitor-name and audio processing did not change the EDID")
	}
}

func TestReadEffectiveEdidRejectsInvalidPostProcessingResults(t *testing.T) {
	setupEdidTestPaths(t)

	path := filepath.Join(EdidDir, "source.bin")
	writeEdidTestFile(t, path, makeEdidTestData(38))

	originalMonitorAction := applyMonitorNameAction
	originalAudioAction := stripEdidAudioAction
	originalWriter := writeEdidHardware
	t.Cleanup(func() {
		applyMonitorNameAction = originalMonitorAction
		stripEdidAudioAction = originalAudioAction
		writeEdidHardware = originalWriter
	})

	tests := []struct {
		name           string
		monitorName    string
		dpAudioEnabled bool
		inject         func()
	}{
		{
			name:           "monitor name result",
			monitorName:    "Desk-KVM",
			dpAudioEnabled: true,
			inject: func() {
				applyMonitorNameAction = func(data []byte, _ string) ([]byte, error) {
					result := append([]byte(nil), data...)
					result[20]++
					return result, nil
				}
			},
		},
		{
			name:           "audio result",
			dpAudioEnabled: false,
			inject: func() {
				stripEdidAudioAction = func(data []byte) ([]byte, error) {
					result := append([]byte(nil), data...)
					result[20]++
					return result, nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyMonitorNameAction = originalMonitorAction
			stripEdidAudioAction = originalAudioAction
			tt.inject()

			_, err := readEffectiveEdid(path, tt.monitorName, tt.dpAudioEnabled)
			assertEdidValidationReason(t, err, edidReasonInvalidChecksum)

			writes := 0
			writeEdidHardware = func([]byte) error {
				writes++
				return nil
			}
			err = applyEdidFile(path, tt.monitorName, tt.dpAudioEnabled)
			assertEdidValidationReason(t, err, edidReasonInvalidChecksum)
			if writes != 0 {
				t.Fatalf("hardware writes = %d, want 0", writes)
			}
		})
	}
}

func TestSwitchEdidCanLeaveLegacyInvalidCurrentSource(t *testing.T) {
	setupEdidTestPaths(t)

	defaultData := makeEdidTestData(33)
	writeDefaultEdidTestFile(t, defaultData)
	legacyName := "legacy-invalid.bin"
	legacyPath := filepath.Join(CustomEdidDir, legacyName)
	writeEdidTestFile(t, legacyPath, make([]byte, 127))
	writeEdidTestState(t, edidState{Name: legacyName, Source: edidSourceCustom, File: legacyPath})

	name := "valid"
	path := filepath.Join(EdidDir, name+".bin")
	data := makeEdidTestData(34)
	writeEdidTestFile(t, path, data)
	if err := os.MkdirAll(filepath.Dir(LT7911Edid), 0o755); err != nil {
		t.Fatal(err)
	}

	rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
	if rsp.Code != 0 {
		t.Fatalf("SwitchEdid failed: code=%d msg=%q", rsp.Code, rsp.Msg)
	}
	if got := readEdidTestFile(t, LT7911Edid); !bytes.Equal(got, data) {
		t.Fatal("valid target EDID was not written")
	}
	wantState := edidState{Name: name, Source: edidSourceDefault, File: path}
	if got := readEdidTestState(t); got != wantState {
		t.Fatalf("state = %+v, want %+v", got, wantState)
	}
}

func TestSwitchEdidCanRecoverUnavailableOrMalformedCurrentState(t *testing.T) {
	tests := []struct {
		name    string
		arrange func(t *testing.T)
	}{
		{
			name: "missing current source",
			arrange: func(t *testing.T) {
				name := "missing.bin"
				writeEdidTestState(t, edidState{
					Name:   name,
					Source: edidSourceCustom,
					File:   filepath.Join(CustomEdidDir, name),
				})
			},
		},
		{
			name: "non regular current source",
			arrange: func(t *testing.T) {
				name := "directory.bin"
				path := filepath.Join(CustomEdidDir, name)
				if err := os.MkdirAll(path, 0o755); err != nil {
					t.Fatal(err)
				}
				writeEdidTestState(t, edidState{Name: name, Source: edidSourceCustom, File: path})
			},
		},
		{
			name: "malformed current state",
			arrange: func(t *testing.T) {
				writeEdidTestFile(t, CurrentEdidStateFile, []byte("{"))
			},
		},
	}

	for index, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupEdidTestPaths(t)
			writeDefaultEdidTestFile(t, makeEdidTestData(byte(40+index)))
			tt.arrange(t)

			name := "valid"
			path := filepath.Join(EdidDir, name+".bin")
			data := makeEdidTestData(byte(50 + index))
			writeEdidTestFile(t, path, data)
			if err := os.MkdirAll(filepath.Dir(LT7911Edid), 0o755); err != nil {
				t.Fatal(err)
			}

			rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
			if rsp.Code != 0 {
				t.Fatalf("SwitchEdid failed: code=%d msg=%q", rsp.Code, rsp.Msg)
			}
			if got := readEdidTestFile(t, LT7911Edid); !bytes.Equal(got, data) {
				t.Fatal("valid target EDID was not written")
			}
			wantState := edidState{Name: name, Source: edidSourceDefault, File: path}
			if got := readEdidTestState(t); got != wantState {
				t.Fatalf("state = %+v, want %+v", got, wantState)
			}
		})
	}
}

func TestSwitchEdidWriteFailureRollsLegacyInvalidCurrentBackToDefaultAndSyncsState(t *testing.T) {
	setupEdidTestPaths(t)

	defaultData := makeEdidTestData(35)
	defaultPath := writeDefaultEdidTestFile(t, defaultData)
	legacyName := "legacy-invalid.bin"
	legacyPath := filepath.Join(CustomEdidDir, legacyName)
	writeEdidTestFile(t, legacyPath, make([]byte, 127))
	writeEdidTestState(t, edidState{Name: legacyName, Source: edidSourceCustom, File: legacyPath})

	name := "valid"
	writeEdidTestFile(t, filepath.Join(EdidDir, name+".bin"), makeEdidTestData(36))

	oldWrite := writeEdidHardware
	writes := 0
	var rolledBack []byte
	writeEdidHardware = func(data []byte) error {
		writes++
		if writes == 1 {
			return errors.New("injected target write failure")
		}
		rolledBack = append([]byte(nil), data...)
		return nil
	}
	t.Cleanup(func() { writeEdidHardware = oldWrite })

	rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
	if rsp.Code == 0 {
		t.Fatal("SwitchEdid succeeded, want write failure")
	}
	if writes != 2 {
		t.Fatalf("hardware writes = %d, want target plus default rollback", writes)
	}
	if !bytes.Equal(rolledBack, defaultData) {
		t.Fatal("hardware rollback did not use validated default EDID")
	}
	wantState := edidState{
		Name:   defaultCurrentEdidName,
		Source: edidSourceDefault,
		File:   defaultPath,
	}
	if got := readEdidTestState(t); got != wantState {
		t.Fatalf("state = %+v, want default rollback state %+v", got, wantState)
	}
}

func TestSwitchEdidInvalidDefaultRollbackCandidateStopsBeforeHardwareWrite(t *testing.T) {
	setupEdidTestPaths(t)

	writeDefaultEdidTestFile(t, make([]byte, 128))
	legacyName := "legacy-invalid.bin"
	legacyPath := filepath.Join(CustomEdidDir, legacyName)
	writeEdidTestFile(t, legacyPath, make([]byte, 127))
	writeEdidTestState(t, edidState{Name: legacyName, Source: edidSourceCustom, File: legacyPath})

	name := "valid"
	writeEdidTestFile(t, filepath.Join(EdidDir, name+".bin"), makeEdidTestData(37))

	oldWrite := writeEdidHardware
	writes := 0
	writeEdidHardware = func([]byte) error {
		writes++
		return nil
	}
	t.Cleanup(func() { writeEdidHardware = oldWrite })

	rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
	if rsp.Code == 0 {
		t.Fatal("SwitchEdid succeeded without a valid rollback candidate")
	}
	if writes != 0 {
		t.Fatalf("hardware writes = %d, want 0", writes)
	}
}

func TestWriteValidatedEdidHardwareRejectsInvalidDataBeforeWriter(t *testing.T) {
	oldWrite := writeEdidHardware
	writes := 0
	writeEdidHardware = func([]byte) error {
		writes++
		return nil
	}
	t.Cleanup(func() { writeEdidHardware = oldWrite })

	if err := writeValidatedEdidHardware(make([]byte, edidBlockSize)); err == nil {
		t.Fatal("writeValidatedEdidHardware accepted invalid EDID")
	}
	if writes != 0 {
		t.Fatalf("hardware writer calls = %d, want 0", writes)
	}
}
