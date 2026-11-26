package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	updateEdidChecksum(invalid[:edidBlockSize])
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

func TestSwitchEdidHonorsVirtualSpeakerMarker(t *testing.T) {
	for _, speakerEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("speakerEnabled=%t", speakerEnabled), func(t *testing.T) {
			setupEdidTestPaths(t)
			source := readEdidTestFile(t, "../../../debhelper/packages/kvmcomm/rootfs/kvmcomm/edid/nanokvm-go-1920x1080-60.bin")
			writeDefaultEdidTestFile(t, makeEdidTestData(1))
			name := "selected.bin"
			path := filepath.Join(CustomEdidDir, name)
			writeEdidTestFile(t, path, source)
			oldMarker := virtualSpeakerMarkerExists
			virtualSpeakerMarkerExists = func() bool { return speakerEnabled }
			t.Cleanup(func() { virtualSpeakerMarkerExists = oldMarker })
			if err := os.MkdirAll(filepath.Dir(LT7911Edid), 0o755); err != nil {
				t.Fatal(err)
			}

			rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
			if rsp.Code != 0 {
				t.Fatalf("switch failed: %s", rsp.Msg)
			}
			got := readEdidTestFile(t, LT7911Edid)
			if !bytes.Equal(got[:edidBlockSize], source[:edidBlockSize]) {
				t.Fatal("profile identity changed")
			}
			if hasBasicAudio := got[edidBlockSize+3]&0x40 != 0; hasBasicAudio == speakerEnabled {
				t.Fatalf("unexpected display audio capability: %t", hasBasicAudio)
			}
			if !bytes.Equal(readEdidTestFile(t, path), source) {
				t.Fatal("source EDID was modified")
			}
		})
	}
}

func TestRestoreLegacyEdidRetriesAndRestoresSelectedProfileOnce(t *testing.T) {
	setupEdidTestPaths(t)
	name := "selected.bin"
	path := filepath.Join(CustomEdidDir, name)
	source := makeEdidTestData(37)
	writeEdidTestFile(t, path, source)
	writeEdidTestState(t, edidState{Name: name, Source: edidSourceCustom, File: path})
	legacy := filepath.Join(CustomEdidDir, "config.json")
	writeEdidTestFile(t, legacy, []byte(`{"monitorName":"Old override"}`))

	// The hardware path is unavailable on the first attempt.
	if err := RestoreLegacyEdid(); err == nil {
		t.Fatal("expected hardware write to fail")
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Fatal("lost migration marker after failed write", err)
	}
	if err := os.MkdirAll(filepath.Dir(LT7911Edid), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RestoreLegacyEdid(); err != nil {
		t.Fatal(err)
	}
	if got := readEdidTestFile(t, LT7911Edid); !bytes.Equal(got, source) {
		t.Fatal("selected profile was not restored")
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy config stat = %v, want file removed", err)
	}

	oldWrite := writeEdidHardware
	writeEdidHardware = func([]byte) error { return errors.New("unexpected repeat write") }
	t.Cleanup(func() { writeEdidHardware = oldWrite })
	if err := RestoreLegacyEdid(); err != nil {
		t.Fatal(err)
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

func TestBuild2EdidMigrationWithoutNameOverride(t *testing.T) {
	for _, profile := range []string{"stock", "custom"} {
		t.Run(profile, func(t *testing.T) {
			setupEdidTestPaths(t)
			source := makeEdidTestData(41)
			writeDefaultEdidTestFile(t, source)
			if profile == "custom" {
				path := filepath.Join(CustomEdidDir, "selected.bin")
				writeEdidTestFile(t, path, source)
				writeEdidTestState(t, edidState{Name: "selected.bin", Source: edidSourceCustom, File: path})
			}
			old := writeEdidHardware
			writes := 0
			writeEdidHardware = func(data []byte) error {
				if !bytes.Equal(data, source) {
					t.Fatal("wrong profile")
				}
				writes++
				return nil
			}
			t.Cleanup(func() { writeEdidHardware = old })
			for i := 0; i < 2; i++ {
				if err := RestoreLegacyEdid(); err != nil {
					t.Fatal(err)
				}
			}
			if (profile == "custom" && writes != 0) || (profile == "stock" && writes != 1) {
				t.Fatalf("hardware writes = %d", writes)
			}
		})
	}
}
