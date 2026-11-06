package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateMonitorName(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty disables override", value: ""},
		{name: "printable ascii", value: "NanoKVM-Go"},
		{name: "thirteen bytes", value: "ABCDEFGHIJKLM"},
		{name: "fourteen bytes", value: "ABCDEFGHIJKLMN", wantErr: true},
		{name: "newline", value: "NanoKVM\nGo", wantErr: true},
		{name: "delete", value: "Nano\x7fKVM", wantErr: true},
		{name: "non ascii", value: "Nano显示器", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateMonitorName(test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateMonitorName(%q) error = %v, wantErr %v", test.value, err, test.wantErr)
			}
		})
	}
}

func TestApplyMonitorNameOverrideFormatsNameAndPreservesSource(t *testing.T) {
	source := makeMonitorNameTestEdid(3, 90)
	original := append([]byte(nil), source...)
	extensions := append([]byte(nil), source[edidBlockSize:]...)

	patched, err := applyMonitorNameOverride(source, "NanoKVM")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(source, original) {
		t.Fatal("source EDID was modified")
	}
	if !bytes.Equal(patched[edidBlockSize:], extensions) {
		t.Fatal("extension blocks were modified")
	}

	wantPayload := append([]byte("NanoKVM\n"), bytes.Repeat([]byte{' '}, 5)...)
	if got := patched[95:108]; !bytes.Equal(got, wantPayload) {
		t.Fatalf("monitor name payload = % x, want % x", got, wantPayload)
	}
	assertValidEdidBlockChecksums(t, patched)
}

func TestApplyMonitorNameOverrideUsesAllThirteenBytesWithoutLineFeed(t *testing.T) {
	source := makeMonitorNameTestEdid(1, 90)
	name := "ABCDEFGHIJKLM"

	patched, err := applyMonitorNameOverride(source, name)
	if err != nil {
		t.Fatal(err)
	}
	if got := patched[95:108]; string(got) != name {
		t.Fatalf("monitor name payload = %q, want %q", got, name)
	}
	assertValidEdidBlockChecksums(t, patched)
}

func TestApplyMonitorNameOverrideValidatesEdidStructure(t *testing.T) {
	t.Run("bad header", func(t *testing.T) {
		data := makeMonitorNameTestEdid(1, 90)
		data[0] = 0x01
		updateEdidBlockChecksum(data[:edidBlockSize])
		if _, err := applyMonitorNameOverride(data, "Name"); err == nil || !strings.Contains(err.Error(), "header") {
			t.Fatalf("error = %v, want invalid header", err)
		}
	})

	t.Run("non block length", func(t *testing.T) {
		data := append(makeMonitorNameTestEdid(1, 90), 0x00)
		if _, err := applyMonitorNameOverride(data, "Name"); err == nil || !strings.Contains(err.Error(), "length") {
			t.Fatalf("error = %v, want invalid length", err)
		}
	})

	t.Run("extension count mismatch", func(t *testing.T) {
		data := makeMonitorNameTestEdid(2, 90)
		data[126] = 0
		updateEdidBlockChecksum(data[:edidBlockSize])
		if _, err := applyMonitorNameOverride(data, "Name"); err == nil || !strings.Contains(err.Error(), "extension count") {
			t.Fatalf("error = %v, want extension count mismatch", err)
		}
	})

	t.Run("bad checksum", func(t *testing.T) {
		data := makeMonitorNameTestEdid(2, 90)
		data[edidBlockSize+10]++
		if _, err := applyMonitorNameOverride(data, "Name"); err == nil || !strings.Contains(err.Error(), "checksum in block 1") {
			t.Fatalf("error = %v, want block checksum failure", err)
		}
	})

	t.Run("descriptor missing from base block", func(t *testing.T) {
		data := makeMonitorNameTestEdid(2)
		copy(data[edidBlockSize+54:], []byte{0x00, 0x00, 0x00, 0xfc, 0x00})
		updateEdidBlockChecksum(data[edidBlockSize : 2*edidBlockSize])
		if _, err := applyMonitorNameOverride(data, "Name"); !errors.Is(err, errMonitorNameDescriptorNotFound) {
			t.Fatalf("error = %v, want descriptor not found", err)
		}
	})

	t.Run("multiple descriptors", func(t *testing.T) {
		data := makeMonitorNameTestEdid(1, 90, 108)
		if _, err := applyMonitorNameOverride(data, "Name"); !errors.Is(err, errMultipleMonitorNameDescriptors) {
			t.Fatalf("error = %v, want multiple descriptors", err)
		}
	})
}

func TestApplyMonitorNameOverrideEmptyValidatesAndCopiesEdid(t *testing.T) {
	source := makeMonitorNameTestEdid(1, 90)
	got, err := applyMonitorNameOverride(source, "")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, source) {
		t.Fatal("empty override changed the EDID")
	}
	got[8]++
	if bytes.Equal(got, source) {
		t.Fatal("returned EDID aliases the source")
	}
}

func TestGetEdidMonitorName(t *testing.T) {
	setupEdidTestPaths(t)

	rsp := getEdidTest(t, NewService().GetEdidMonitorName)
	if rsp.Code != 0 {
		t.Fatalf("GetEdidMonitorName failed: code=%d msg=%s", rsp.Code, rsp.Msg)
	}
	if got := decodeMonitorNameResponse(t, rsp); got != "" {
		t.Fatalf("monitorName = %q, want empty", got)
	}

	if err := saveEdidConfig(edidConfig{MonitorName: "NanoKVM-Test"}); err != nil {
		t.Fatal(err)
	}
	rsp = getEdidTest(t, NewService().GetEdidMonitorName)
	if got := decodeMonitorNameResponse(t, rsp); got != "NanoKVM-Test" {
		t.Fatalf("monitorName = %q, want NanoKVM-Test", got)
	}
}

func TestSetEdidMonitorNameAppliesAndPersistsWithoutChangingSource(t *testing.T) {
	setupEdidTestPaths(t)

	name := "current"
	source := makeMonitorNameTestEdid(2, 90)
	sourcePath := filepath.Join(EdidDir, name+".bin")
	writeEdidTestFile(t, sourcePath, source)
	writeEdidTestState(t, edidState{Name: name, Source: edidSourceDefault, File: sourcePath})
	if err := os.MkdirAll(filepath.Dir(LT7911Edid), 0o755); err != nil {
		t.Fatal(err)
	}

	rsp := postEdidTestForm(t, NewService().SetEdidMonitorName, url.Values{
		"monitorName": {"Desk-KVM"},
	})
	if rsp.Code != 0 {
		t.Fatalf("SetEdidMonitorName failed: code=%d msg=%s", rsp.Code, rsp.Msg)
	}
	if got := decodeMonitorNameResponse(t, rsp); got != "Desk-KVM" {
		t.Fatalf("response monitorName = %q, want Desk-KVM", got)
	}

	config, err := loadEdidConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.MonitorName != "Desk-KVM" {
		t.Fatalf("saved monitor name = %q, want Desk-KVM", config.MonitorName)
	}
	if got := readEdidTestFile(t, sourcePath); !bytes.Equal(got, source) {
		t.Fatal("source EDID was modified")
	}
	want, err := applyMonitorNameOverride(source, "Desk-KVM")
	if err != nil {
		t.Fatal(err)
	}
	if got := readEdidTestFile(t, LT7911Edid); !bytes.Equal(got, want) {
		t.Fatal("hardware EDID does not contain the configured monitor name")
	}
	if info, err := os.Stat(EdidConfigFile); err != nil {
		t.Fatal(err)
	} else if info.Mode().Perm() != 0o644 {
		t.Fatalf("config mode = %o, want 644", info.Mode().Perm())
	}
}

func TestSetEdidMonitorNameHonorsVirtualSpeakerMarker(t *testing.T) {
	tests := []struct {
		name           string
		speakerEnabled bool
	}{
		{name: "speaker disabled keeps display audio", speakerEnabled: false},
		{name: "speaker enabled removes display audio", speakerEnabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupEdidTestPaths(t)

			name := "current"
			source := makeMonitorNameAudioTestEdid()
			sourcePath := filepath.Join(EdidDir, name+".bin")
			writeEdidTestFile(t, sourcePath, source)
			writeEdidTestState(t, edidState{Name: name, Source: edidSourceDefault, File: sourcePath})

			oldSpeakerMarkerExists := virtualSpeakerMarkerExists
			oldWrite := writeEdidHardware
			t.Cleanup(func() {
				virtualSpeakerMarkerExists = oldSpeakerMarkerExists
				writeEdidHardware = oldWrite
			})
			virtualSpeakerMarkerExists = func() bool { return tt.speakerEnabled }

			var hardwareEdid []byte
			writeEdidHardware = func(data []byte) error {
				hardwareEdid = append([]byte(nil), data...)
				return nil
			}

			rsp := postEdidTestForm(t, NewService().SetEdidMonitorName, url.Values{
				"monitorName": {"Desk-KVM"},
			})
			if rsp.Code != 0 {
				t.Fatalf("SetEdidMonitorName failed: code=%d msg=%s", rsp.Code, rsp.Msg)
			}

			want, err := applyMonitorNameOverride(source, "Desk-KVM")
			if err != nil {
				t.Fatal(err)
			}
			if tt.speakerEnabled {
				want, err = stripEdidAudio(want)
				if err != nil {
					t.Fatal(err)
				}
			}
			if !bytes.Equal(hardwareEdid, want) {
				t.Fatal("hardware EDID does not match the virtual speaker audio policy")
			}
			if got := readEdidTestFile(t, sourcePath); !bytes.Equal(got, source) {
				t.Fatal("source EDID was modified")
			}
		})
	}
}

func TestSetEdidMonitorNameEmptyDisablesOverride(t *testing.T) {
	setupEdidTestPaths(t)

	name := "current"
	source := makeMonitorNameTestEdid(1, 90)
	sourcePath := filepath.Join(EdidDir, name+".bin")
	writeEdidTestFile(t, sourcePath, source)
	writeEdidTestState(t, edidState{Name: name, Source: edidSourceDefault, File: sourcePath})
	if err := saveEdidConfig(edidConfig{MonitorName: "Old-Name"}); err != nil {
		t.Fatal(err)
	}
	oldEffective, err := applyMonitorNameOverride(source, "Old-Name")
	if err != nil {
		t.Fatal(err)
	}
	writeEdidTestFile(t, LT7911Edid, oldEffective)

	rsp := postEdidTestForm(t, NewService().SetEdidMonitorName, url.Values{
		"monitorName": {""},
	})
	if rsp.Code != 0 {
		t.Fatalf("SetEdidMonitorName failed: code=%d msg=%s", rsp.Code, rsp.Msg)
	}
	if got := decodeMonitorNameResponse(t, rsp); got != "" {
		t.Fatalf("response monitorName = %q, want empty", got)
	}
	config, err := loadEdidConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.MonitorName != "" {
		t.Fatalf("saved monitor name = %q, want empty", config.MonitorName)
	}
	if got := readEdidTestFile(t, LT7911Edid); !bytes.Equal(got, source) {
		t.Fatal("clearing monitor name did not restore the original EDID")
	}
}

func TestSetEdidMonitorNameRejectsInvalidValueBeforeChangingState(t *testing.T) {
	setupEdidTestPaths(t)

	rsp := postEdidTestForm(t, NewService().SetEdidMonitorName, url.Values{
		"monitorName": {"ABCDEFGHIJKLMN"},
	})
	if rsp.Code != -2 || rsp.Msg != "invalid monitor name" {
		t.Fatalf("response = code %d msg %q, want invalid monitor name", rsp.Code, rsp.Msg)
	}
	if _, err := os.Stat(CurrentEdidStateFile); !os.IsNotExist(err) {
		t.Fatalf("current state stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(EdidConfigFile); !os.IsNotExist(err) {
		t.Fatalf("config stat error = %v, want not exist", err)
	}
}

func TestSetEdidMonitorNameRejectsMissingDescriptorWithoutChangingState(t *testing.T) {
	setupEdidTestPaths(t)

	name := "current"
	source := makeMonitorNameTestEdid(1)
	sourcePath := filepath.Join(EdidDir, name+".bin")
	writeEdidTestFile(t, sourcePath, source)
	writeEdidTestState(t, edidState{Name: name, Source: edidSourceDefault, File: sourcePath})
	oldHardware := []byte("current-hardware-edid")
	writeEdidTestFile(t, LT7911Edid, oldHardware)

	rsp := postEdidTestForm(t, NewService().SetEdidMonitorName, url.Values{
		"monitorName": {"Desk-KVM"},
	})
	if rsp.Code == 0 || rsp.Msg != "monitor name descriptor not found" {
		t.Fatalf("response = code %d msg %q, want descriptor error", rsp.Code, rsp.Msg)
	}
	if _, err := os.Stat(EdidConfigFile); !os.IsNotExist(err) {
		t.Fatalf("config stat error = %v, want not exist", err)
	}
	if got := readEdidTestFile(t, LT7911Edid); !bytes.Equal(got, oldHardware) {
		t.Fatal("hardware EDID changed after descriptor rejection")
	}
}

func TestSetEdidMonitorNameHardwareFailureRollsBackAndKeepsConfig(t *testing.T) {
	setupEdidTestPaths(t)

	name := "current"
	source := makeMonitorNameTestEdid(1, 90)
	sourcePath := filepath.Join(EdidDir, name+".bin")
	writeEdidTestFile(t, sourcePath, source)
	writeEdidTestState(t, edidState{Name: name, Source: edidSourceDefault, File: sourcePath})
	if err := saveEdidConfig(edidConfig{MonitorName: "Old"}); err != nil {
		t.Fatal(err)
	}
	oldEffective, err := applyMonitorNameOverride(source, "Old")
	if err != nil {
		t.Fatal(err)
	}
	writeEdidTestFile(t, LT7911Edid, oldEffective)

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

	rsp := postEdidTestForm(t, NewService().SetEdidMonitorName, url.Values{
		"monitorName": {"New"},
	})
	if rsp.Code != -6 {
		t.Fatalf("response code = %d, want -6", rsp.Code)
	}
	if writes != 2 {
		t.Fatalf("hardware writes = %d, want apply plus rollback", writes)
	}
	if got := readEdidTestFile(t, LT7911Edid); !bytes.Equal(got, oldEffective) {
		t.Fatal("hardware EDID was not rolled back")
	}
	config, err := loadEdidConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.MonitorName != "Old" {
		t.Fatalf("config monitor name = %q, want Old", config.MonitorName)
	}
}

func TestSetEdidMonitorNameConfigFailureRollsBackHardware(t *testing.T) {
	setupEdidTestPaths(t)

	name := "current"
	source := makeMonitorNameTestEdid(1, 90)
	sourcePath := filepath.Join(EdidDir, name+".bin")
	writeEdidTestFile(t, sourcePath, source)
	writeEdidTestState(t, edidState{Name: name, Source: edidSourceDefault, File: sourcePath})
	if err := saveEdidConfig(edidConfig{MonitorName: "Old"}); err != nil {
		t.Fatal(err)
	}
	oldEffective, err := applyMonitorNameOverride(source, "Old")
	if err != nil {
		t.Fatal(err)
	}
	writeEdidTestFile(t, LT7911Edid, oldEffective)

	oldSave := saveEdidConfigAction
	t.Cleanup(func() { saveEdidConfigAction = oldSave })
	saveEdidConfigAction = func(edidConfig) error {
		return errors.New("injected config failure")
	}

	rsp := postEdidTestForm(t, NewService().SetEdidMonitorName, url.Values{
		"monitorName": {"New"},
	})
	if rsp.Code != -7 {
		t.Fatalf("response code = %d, want -7", rsp.Code)
	}
	if got := readEdidTestFile(t, LT7911Edid); !bytes.Equal(got, oldEffective) {
		t.Fatal("hardware EDID was not rolled back")
	}
	config, err := loadEdidConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.MonitorName != "Old" {
		t.Fatalf("config monitor name = %q, want Old", config.MonitorName)
	}
}

func TestSwitchEdidAppliesGlobalMonitorNameWithoutChangingSource(t *testing.T) {
	setupEdidTestPaths(t)

	name := "selected"
	source := makeMonitorNameTestEdid(2, 90)
	sourcePath := filepath.Join(EdidDir, name+".bin")
	writeEdidTestFile(t, sourcePath, source)
	writeDefaultEdidTestFile(t, makeMonitorNameTestEdid(2, 90))
	if err := os.MkdirAll(filepath.Dir(LT7911Edid), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := saveEdidConfig(edidConfig{MonitorName: "Global-Name"}); err != nil {
		t.Fatal(err)
	}

	rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
	if rsp.Code != 0 {
		t.Fatalf("SwitchEdid failed: code=%d msg=%s", rsp.Code, rsp.Msg)
	}
	if got := readEdidTestFile(t, sourcePath); !bytes.Equal(got, source) {
		t.Fatal("source EDID was modified")
	}
	want, err := applyMonitorNameOverride(source, "Global-Name")
	if err != nil {
		t.Fatal(err)
	}
	if got := readEdidTestFile(t, LT7911Edid); !bytes.Equal(got, want) {
		t.Fatal("hardware EDID does not contain global monitor name")
	}
}

func TestSwitchEdidHonorsVirtualSpeakerMarker(t *testing.T) {
	tests := []struct {
		name           string
		speakerEnabled bool
	}{
		{name: "speaker disabled keeps display audio", speakerEnabled: false},
		{name: "speaker enabled removes display audio", speakerEnabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupEdidTestPaths(t)

			name := "selected"
			source := makeMonitorNameAudioTestEdid()
			sourcePath := filepath.Join(EdidDir, name+".bin")
			writeEdidTestFile(t, sourcePath, source)
			writeDefaultEdidTestFile(t, makeMonitorNameAudioTestEdid())
			if err := saveEdidConfig(edidConfig{MonitorName: "Global-Name"}); err != nil {
				t.Fatal(err)
			}

			oldSpeakerMarkerExists := virtualSpeakerMarkerExists
			oldWrite := writeEdidHardware
			t.Cleanup(func() {
				virtualSpeakerMarkerExists = oldSpeakerMarkerExists
				writeEdidHardware = oldWrite
			})
			virtualSpeakerMarkerExists = func() bool { return tt.speakerEnabled }

			var hardwareEdid []byte
			writeEdidHardware = func(data []byte) error {
				hardwareEdid = append([]byte(nil), data...)
				return nil
			}

			rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
			if rsp.Code != 0 {
				t.Fatalf("SwitchEdid failed: code=%d msg=%s", rsp.Code, rsp.Msg)
			}

			want, err := applyMonitorNameOverride(source, "Global-Name")
			if err != nil {
				t.Fatal(err)
			}
			if tt.speakerEnabled {
				want, err = stripEdidAudio(want)
				if err != nil {
					t.Fatal(err)
				}
			}
			if !bytes.Equal(hardwareEdid, want) {
				t.Fatal("hardware EDID does not match the virtual speaker audio policy")
			}
			if got := readEdidTestFile(t, sourcePath); !bytes.Equal(got, source) {
				t.Fatal("source EDID was modified")
			}
		})
	}
}

func TestSwitchCustomEdidAppliesGlobalMonitorNameWithoutChangingSource(t *testing.T) {
	setupEdidTestPaths(t)

	name := "custom.bin"
	source := makeMonitorNameTestEdid(2, 90)
	sourcePath := filepath.Join(CustomEdidDir, name)
	writeEdidTestFile(t, sourcePath, source)
	writeDefaultEdidTestFile(t, makeMonitorNameTestEdid(2, 90))
	if err := os.MkdirAll(filepath.Dir(LT7911Edid), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := saveEdidConfig(edidConfig{MonitorName: "Global-Name"}); err != nil {
		t.Fatal(err)
	}

	rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
	if rsp.Code != 0 {
		t.Fatalf("SwitchEdid failed: code=%d msg=%s", rsp.Code, rsp.Msg)
	}
	if got := readEdidTestFile(t, sourcePath); !bytes.Equal(got, source) {
		t.Fatal("custom source EDID was modified")
	}
	want, err := applyMonitorNameOverride(source, "Global-Name")
	if err != nil {
		t.Fatal(err)
	}
	if got := readEdidTestFile(t, LT7911Edid); !bytes.Equal(got, want) {
		t.Fatal("hardware EDID does not contain global monitor name")
	}
	state := readEdidTestState(t)
	if state.Source != edidSourceCustom || state.File != sourcePath {
		t.Fatalf("state = %+v, want custom EDID %s", state, sourcePath)
	}
}

func TestSwitchEdidRejectsOverrideWhenDescriptorIsMissing(t *testing.T) {
	setupEdidTestPaths(t)

	name := "selected"
	sourcePath := filepath.Join(EdidDir, name+".bin")
	writeEdidTestFile(t, sourcePath, makeMonitorNameTestEdid(1))
	oldHardware := []byte("unchanged")
	writeEdidTestFile(t, LT7911Edid, oldHardware)
	if err := saveEdidConfig(edidConfig{MonitorName: "Global-Name"}); err != nil {
		t.Fatal(err)
	}

	rsp := postEdidTestForm(t, NewService().SwitchEdid, url.Values{"edid": {name}})
	if rsp.Code == 0 || rsp.Msg != "monitor name descriptor not found" {
		t.Fatalf("response = code %d msg %q, want descriptor error", rsp.Code, rsp.Msg)
	}
	if got := readEdidTestFile(t, LT7911Edid); !bytes.Equal(got, oldHardware) {
		t.Fatal("hardware EDID changed after descriptor rejection")
	}
	if _, err := os.Stat(CurrentEdidStateFile); !os.IsNotExist(err) {
		t.Fatalf("current state stat error = %v, want not exist", err)
	}
}

func makeMonitorNameTestEdid(blockCount int, descriptorOffsets ...int) []byte {
	data := make([]byte, blockCount*edidBlockSize)
	copy(data, []byte{0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00})
	data[18] = 1
	data[19] = 4
	data[126] = byte(blockCount - 1)

	for _, offset := range descriptorOffsets {
		copy(data[offset:offset+5], []byte{0x00, 0x00, 0x00, 0xfc, 0x00})
		payload := data[offset+5 : offset+18]
		for i := range payload {
			payload[i] = 0x20
		}
		copy(payload, []byte("Original\n"))
	}

	for block := 0; block < blockCount; block++ {
		start := block * edidBlockSize
		updateEdidBlockChecksum(data[start : start+edidBlockSize])
	}
	return data
}

func makeMonitorNameAudioTestEdid() []byte {
	data := makeMonitorNameTestEdid(2, 90)
	cta := data[edidBlockSize:]
	cta[0] = 0x02
	cta[1] = 0x03
	cta[2] = 8
	cta[3] = 0x40
	cta[4] = 0x23
	copy(cta[5:8], []byte{0x09, 0x07, 0x07})
	updateEdidBlockChecksum(cta)
	return data
}

func assertValidEdidBlockChecksums(t *testing.T, data []byte) {
	t.Helper()
	for block := 0; block < len(data)/edidBlockSize; block++ {
		sum := 0
		start := block * edidBlockSize
		for _, value := range data[start : start+edidBlockSize] {
			sum += int(value)
		}
		if sum&0xff != 0 {
			t.Fatalf("block %d checksum sum = 0x%02x, want 0", block, sum&0xff)
		}
	}
}

func decodeMonitorNameResponse(t *testing.T, rsp edidTestResponse) string {
	t.Helper()
	var data struct {
		MonitorName string `json:"monitorName"`
	}
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatal(err)
	}
	return data.MonitorName
}
