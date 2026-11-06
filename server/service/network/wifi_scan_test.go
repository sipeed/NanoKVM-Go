package network

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
)

type scanWifiTestResponse struct {
	Code int `json:"code"`
	Data struct {
		WifiList []*proto.WiFi `json:"wifiList"`
	} `json:"data"`
}

func performScanWifi(t *testing.T) scanWifiTestResponse {
	return performScanWifiWithHandler(t, NewService().ScanWifi, "")
}

func performScanWifiWithHandler(t *testing.T, handler func(*gin.Context), apKey string) scanWifiTestResponse {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/network/wifi/scan", nil)
	if apKey != "" {
		context.Request.Header.Set("X-AP-Key", apKey)
	}
	handler(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("scan status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response scanWifiTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode scan response: %v", err)
	}
	return response
}

func setScanAuthorizationSeams(t *testing.T, apMode bool, password string, tokenMiddleware func() gin.HandlerFunc) {
	t.Helper()

	previousAPMode := isAPConfigurationMode
	previousPassword := readAPPassword
	previousTokenMiddleware := scanTokenMiddleware
	isAPConfigurationMode = func() bool { return apMode }
	readAPPassword = func() string { return password }
	scanTokenMiddleware = tokenMiddleware
	t.Cleanup(func() {
		isAPConfigurationMode = previousAPMode
		readAPPassword = previousPassword
		scanTokenMiddleware = previousTokenMiddleware
	})
}

func setScanWifiSeams(t *testing.T, scan func() ([]byte, error), details func() (*utils.WiFiDetails, error)) {
	t.Helper()

	previousScan := runWiFiScan
	previousDetails := getWifiDetails
	runWiFiScan = scan
	getWifiDetails = details
	t.Cleanup(func() {
		runWiFiScan = previousScan
		getWifiDetails = previousDetails
	})
}

func TestScanWifiAPModeUsesScanCommand(t *testing.T) {
	called := false
	setScanWifiSeams(t,
		func() ([]byte, error) {
			called = true
			return []byte(`[{"ssid":"AP-nearby","bssid":"02:00:00:00:00:10","signal":-52,"frequency":2412,"security":"WPA2"}]`), nil
		},
		func() (*utils.WiFiDetails, error) {
			return &utils.WiFiDetails{WPAState: "DISCONNECTED"}, nil
		},
	)

	response := performScanWifi(t)
	if !called {
		t.Fatal("ScanWifi did not invoke the scan command")
	}
	if response.Code != 0 {
		t.Fatalf("scan code = %d, want 0", response.Code)
	}
	if len(response.Data.WifiList) != 1 || response.Data.WifiList[0].SSID != "AP-nearby" {
		t.Fatalf("scan list = %#v, want AP-nearby", response.Data.WifiList)
	}
}

func TestScanWifiScriptFailure(t *testing.T) {
	setScanWifiSeams(t,
		func() ([]byte, error) { return []byte("scan failed"), errors.New("exit status 1") },
		func() (*utils.WiFiDetails, error) { return nil, errors.New("not called") },
	)

	response := performScanWifi(t)
	if response.Code != -2 {
		t.Fatalf("scan code = %d, want -2", response.Code)
	}
}

func TestScanWifiMalformedJSON(t *testing.T) {
	setScanWifiSeams(t,
		func() ([]byte, error) { return []byte(`[{"ssid":]`), nil },
		func() (*utils.WiFiDetails, error) { return nil, errors.New("not called") },
	)

	response := performScanWifi(t)
	if response.Code != -3 {
		t.Fatalf("scan code = %d, want -3", response.Code)
	}
}

func TestScanWifiFiltersConnectedNetwork(t *testing.T) {
	setScanWifiSeams(t,
		func() ([]byte, error) {
			return []byte(`[
  {"ssid":"Connected","bssid":"02:00:00:00:00:01","signal":-42,"frequency":5180,"security":"WPA2"},
  null,
  {"ssid":"Available","bssid":"02:00:00:00:00:02","signal":-68,"frequency":2412,"security":"Open"}
]`), nil
		},
		func() (*utils.WiFiDetails, error) {
			return &utils.WiFiDetails{
				WPAState:  "COMPLETED",
				IPAddress: "192.0.2.20",
				BSSID:     "02:00:00:00:00:01",
			}, nil
		},
	)

	response := performScanWifi(t)
	if response.Code != 0 {
		t.Fatalf("scan code = %d, want 0", response.Code)
	}
	if len(response.Data.WifiList) != 1 || response.Data.WifiList[0].SSID != "Available" {
		t.Fatalf("scan list = %#v, want only Available", response.Data.WifiList)
	}
}

func TestScanWifiWithAPKeyOrTokenAcceptsAPKeyInConfigurationMode(t *testing.T) {
	setScanWifiSeams(t,
		func() ([]byte, error) {
			return []byte(`[{"ssid":"Portal Network","bssid":"02:00:00:00:00:03","signal":-54,"frequency":2412,"security":"WPA2"}]`), nil
		},
		func() (*utils.WiFiDetails, error) {
			return &utils.WiFiDetails{WPAState: "DISCONNECTED"}, nil
		},
	)
	calledTokenMiddleware := false
	setScanAuthorizationSeams(t, true, "portal-password", func() gin.HandlerFunc {
		return func(c *gin.Context) {
			calledTokenMiddleware = true
			c.AbortWithStatus(http.StatusUnauthorized)
		}
	})

	response := performScanWifiWithHandler(
		t,
		NewService().ScanWifiWithAPKeyOrToken,
		"portal-password",
	)
	if calledTokenMiddleware {
		t.Fatal("AP-key scan unexpectedly ran token middleware")
	}
	if response.Code != 0 || len(response.Data.WifiList) != 1 {
		t.Fatalf("AP-key scan response = %#v, want one successful network", response)
	}
}

func TestScanWifiWithAPKeyOrTokenRejectsInvalidAPKeyWithoutToken(t *testing.T) {
	setScanAuthorizationSeams(t, true, "portal-password", func() gin.HandlerFunc {
		return func(c *gin.Context) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
	})

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/network/wifi/scan", nil)
	context.Request.Header.Set("X-AP-Key", "wrong-password")
	NewService().ScanWifiWithAPKeyOrToken(context)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("invalid AP key status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestScanWifiWithAPKeyOrTokenFallsBackToTokenMiddleware(t *testing.T) {
	setScanWifiSeams(t,
		func() ([]byte, error) {
			return []byte(`[{"ssid":"Token Network","bssid":"02:00:00:00:00:04","signal":-60,"frequency":5180,"security":"WPA3"}]`), nil
		},
		func() (*utils.WiFiDetails, error) {
			return &utils.WiFiDetails{WPAState: "DISCONNECTED"}, nil
		},
	)
	calledTokenMiddleware := false
	setScanAuthorizationSeams(t, false, "", func() gin.HandlerFunc {
		return func(c *gin.Context) {
			calledTokenMiddleware = true
			c.Next()
		}
	})

	response := performScanWifiWithHandler(t, NewService().ScanWifiWithAPKeyOrToken, "")
	if !calledTokenMiddleware {
		t.Fatal("token middleware was not called outside AP configuration mode")
	}
	if response.Code != 0 || len(response.Data.WifiList) != 1 {
		t.Fatalf("token scan response = %#v, want one successful network", response)
	}
}
