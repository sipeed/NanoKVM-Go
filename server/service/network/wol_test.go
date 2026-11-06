package network

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"syscall"
	"testing"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

func TestWakeOnLANCommandUsesFixedExecutableAndArguments(t *testing.T) {
	cmd := wakeOnLANCommand("AA:BB:CC:DD:EE:FF")
	want := []string{"ether-wake", "-b", "AA:BB:CC:DD:EE:FF"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("wake command arguments = %#v, want %#v", cmd.Args, want)
	}
}

func TestParseMACNormalizesSupportedForms(t *testing.T) {
	for _, input := range []string{
		"aa:bb:cc:dd:ee:ff",
		"AA-BB-CC-DD-EE-FF",
		"aabb.ccdd.eeff",
	} {
		got, err := parseMAC(input)
		if err != nil {
			t.Fatalf("parseMAC(%q): %v", input, err)
		}
		if got != "AA:BB:CC:DD:EE:FF" {
			t.Fatalf("parseMAC(%q) = %q", input, got)
		}
	}
}

func TestParseMACRejectsCommandContent(t *testing.T) {
	for _, input := range []string{"AA:BB:CC:DD:EE:FF;reboot", "$(reboot)", "AA:BB:CC:DD:EE"} {
		if _, err := parseMAC(input); err == nil {
			t.Fatalf("parseMAC(%q) unexpectedly succeeded", input)
		}
	}
}

func TestGetMacHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, tt := range []struct {
		name     string
		content  string
		readErr  error
		wantCode int
		wantMacs []string
	}{
		{
			name:     "missing history returns an empty list",
			readErr:  os.ErrNotExist,
			wantMacs: []string{},
		},
		{
			name:     "empty history returns an empty list",
			wantMacs: []string{},
		},
		{
			name:     "saved history preserves addresses and names",
			content:  "AA:BB:CC:DD:EE:FF Work PC\n11:22:33:44:55:66\n",
			wantMacs: []string{"AA:BB:CC:DD:EE:FF Work PC", "11:22:33:44:55:66", ""},
		},
		{
			name:     "permission failure remains an error",
			readErr:  os.ErrPermission,
			wantCode: -2,
		},
		{
			name:     "IO failure remains an error",
			readErr:  syscall.EIO,
			wantCode: -2,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			previousReadFile := wolReadFile
			wolReadFile = func(path string) ([]byte, error) {
				if path != WolMacFile {
					t.Fatalf("history path = %q, want %q", path, WolMacFile)
				}
				if tt.readErr != nil {
					return nil, &os.PathError{Op: "open", Path: path, Err: tt.readErr}
				}
				return []byte(tt.content), nil
			}
			t.Cleanup(func() { wolReadFile = previousReadFile })

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/network/wol/mac", nil)
			NewService().GetMac(ctx)

			if recorder.Code != http.StatusOK {
				t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusOK)
			}
			var response struct {
				Code int              `json:"code"`
				Msg  string           `json:"msg"`
				Data *proto.GetMacRsp `json:"data"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.Code != tt.wantCode {
				t.Fatalf("response code = %d, want %d; body = %s", response.Code, tt.wantCode, recorder.Body.String())
			}
			if tt.wantCode != 0 {
				if response.Msg != "open file error" || response.Data != nil {
					t.Fatalf("unexpected error response: %s", recorder.Body.String())
				}
				return
			}
			if response.Data == nil || !reflect.DeepEqual(response.Data.Macs, tt.wantMacs) {
				t.Fatalf("response body = %s, want macs = %#v", recorder.Body.String(), tt.wantMacs)
			}
		})
	}
}
