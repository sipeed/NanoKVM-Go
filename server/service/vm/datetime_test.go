package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

const validSetSystemTimeRequest = `{"datetime":"2026-08-13T21:02:03"}`

type setSystemTimeTestResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Timestamp int64 `json:"timestamp"`
	} `json:"data"`
}

func TestSetTimeParsesDatetimeInCurrentDeviceTimezone(t *testing.T) {
	setupSetTimeHandlerTest(t)
	location := time.FixedZone("Asia/Shanghai test", 8*60*60)
	getSystemTimeLocation = func() (*time.Location, error) { return location, nil }
	want := time.Date(2026, time.August, 13, 21, 2, 3, 0, location)
	prepared := false
	completed := false
	prepareSystemTimeSessions = func() (func(bool), error) {
		prepared = true
		return func(timeChanged bool) {
			if !timeChanged {
				t.Fatal("successful clock change completed as failed")
			}
			completed = true
		}, nil
	}
	set := false
	setSystemTime = func(value time.Time) error {
		if !prepared {
			t.Fatal("system time changed before session invalidation preparation")
		}
		if !value.Equal(want) || value.Location() != location {
			t.Fatalf("system time = %s in %s, want %s in %s", value, value.Location(), want, location)
		}
		set = true
		return nil
	}

	rsp := performSetTimeRequest(t, validSetSystemTimeRequest, "application/json")
	if rsp.Code != 0 {
		t.Fatalf("set time response: code=%d msg=%q", rsp.Code, rsp.Msg)
	}
	if !set || !prepared || !completed {
		t.Fatalf("set=%t prepared=%t completed=%t, want all true", set, prepared, completed)
	}
	if rsp.Data.Timestamp != want.UnixMilli() {
		t.Fatalf("timestamp = %d, want %d", rsp.Data.Timestamp, want.UnixMilli())
	}
}

func TestSetTimeStrictlyRejectsInvalidJSONBeforeSideEffects(t *testing.T) {
	setupSetTimeHandlerTest(t)
	getSystemTimeLocation = func() (*time.Location, error) {
		t.Fatal("invalid input must not load the device timezone")
		return nil, nil
	}
	prepareSystemTimeSessions = func() (func(bool), error) {
		t.Fatal("invalid input must not prepare session invalidation")
		return nil, nil
	}
	setSystemTime = func(time.Time) error {
		t.Fatal("invalid input must not change the system time")
		return nil
	}

	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "empty body", contentType: "application/json", body: ""},
		{name: "missing datetime", contentType: "application/json", body: `{}`},
		{name: "empty datetime", contentType: "application/json", body: `{"datetime":""}`},
		{name: "unknown field", contentType: "application/json", body: `{"datetime":"2026-08-13T21:02:03","timezone":"UTC"}`},
		{name: "legacy challenge field", contentType: "application/json", body: `{"datetime":"2026-08-13T21:02:03","challenge":"set-time-challenge"}`},
		{name: "wrong datetime type", contentType: "application/json", body: `{"datetime":1786626123000}`},
		{name: "multiple JSON values", contentType: "application/json", body: validSetSystemTimeRequest + ` {}`},
		{name: "trailing garbage", contentType: "application/json", body: validSetSystemTimeRequest + `x`},
		{name: "oversized body", contentType: "application/json", body: `{"datetime":"` + strings.Repeat("x", setSystemTimeRequestMaxBytes) + `"}`},
		{name: "form request", contentType: "application/x-www-form-urlencoded", body: "datetime=2026-08-13T21%3A02%3A03"},
		{name: "missing content type", contentType: "", body: validSetSystemTimeRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rsp := performSetTimeRequest(t, test.body, test.contentType)
			if rsp.Code == 0 {
				t.Fatalf("invalid request accepted: content-type=%q body=%s", test.contentType, test.body)
			}
		})
	}
}

func TestSetTimeRejectsInvalidLocalDatetimeBeforeSideEffects(t *testing.T) {
	setupSetTimeHandlerTest(t)
	location := time.FixedZone("UTC+8", 8*60*60)
	getSystemTimeLocation = func() (*time.Location, error) { return location, nil }
	prepareSystemTimeSessions = func() (func(bool), error) {
		t.Fatal("invalid datetime must not prepare session invalidation")
		return nil, nil
	}
	setSystemTime = func(time.Time) error {
		t.Fatal("invalid datetime must not change the system time")
		return nil
	}

	invalidDatetimes := []string{
		"2019-12-31T23:59:59",
		"2038-01-19T11:14:08",
		"2026-02-30T12:00:00",
		"2026-08-13 21:02:03",
		"2026-08-13T21:02",
		"2026-08-13T21:02:03Z",
		"2026-08-13T21:02:03+08:00",
	}
	for _, datetime := range invalidDatetimes {
		t.Run(datetime, func(t *testing.T) {
			body, err := json.Marshal(map[string]string{"datetime": datetime})
			if err != nil {
				t.Fatal(err)
			}
			rsp := performSetTimeRequest(t, string(body), "application/json")
			if rsp.Code == 0 {
				t.Fatalf("invalid local datetime %q was accepted", datetime)
			}
		})
	}
}

func TestSetTimeRejectsAmbiguousDaylightSavingTime(t *testing.T) {
	setupSetTimeHandlerTest(t)
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load DST timezone: %v", err)
	}
	getSystemTimeLocation = func() (*time.Location, error) { return location, nil }
	prepareSystemTimeSessions = func() (func(bool), error) {
		t.Fatal("ambiguous datetime must not prepare session invalidation")
		return nil, nil
	}
	setSystemTime = func(time.Time) error {
		t.Fatal("ambiguous datetime must not change the system time")
		return nil
	}

	body := `{"datetime":"2026-11-01T01:30:00"}`
	rsp := performSetTimeRequest(t, body, "application/json")
	if rsp.Code == 0 {
		t.Fatal("ambiguous daylight-saving datetime was accepted")
	}
}

func TestSetTimeAcceptsUnambiguousDaylightSavingTime(t *testing.T) {
	setupSetTimeHandlerTest(t)
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load DST timezone: %v", err)
	}
	getSystemTimeLocation = func() (*time.Location, error) { return location, nil }
	prepareSystemTimeSessions = func() (func(bool), error) { return func(bool) {}, nil }
	setSystemTime = func(time.Time) error { return nil }

	body := `{"datetime":"2026-11-01T03:30:00"}`
	rsp := performSetTimeRequest(t, body, "application/json")
	if rsp.Code != 0 {
		t.Fatalf("unambiguous daylight-saving datetime rejected: code=%d msg=%q", rsp.Code, rsp.Msg)
	}
}

func TestSetTimeStopsBeforeClockChangeWhenTimezoneLoadFails(t *testing.T) {
	setupSetTimeHandlerTest(t)
	getSystemTimeLocation = func() (*time.Location, error) {
		return nil, errors.New("read timezone failed")
	}
	prepareSystemTimeSessions = func() (func(bool), error) {
		t.Fatal("timezone failure must not prepare session invalidation")
		return nil, nil
	}
	setSystemTime = func(time.Time) error {
		t.Fatal("timezone failure must not change the system time")
		return nil
	}

	rsp := performSetTimeRequest(t, validSetSystemTimeRequest, "application/json")
	if rsp.Code == 0 {
		t.Fatal("timezone failure was accepted")
	}
}

func TestSetTimeFailureDoesNotInvalidateSessions(t *testing.T) {
	setupSetTimeHandlerTest(t)
	getSystemTimeLocation = func() (*time.Location, error) { return time.UTC, nil }
	completed := false
	prepareSystemTimeSessions = func() (func(bool), error) {
		return func(timeChanged bool) {
			if timeChanged {
				t.Fatal("failed clock change completed as successful")
			}
			completed = true
		}, nil
	}
	setSystemTime = func(time.Time) error { return errors.New("clock_settime failed") }

	rsp := performSetTimeRequest(t, validSetSystemTimeRequest, "application/json")
	if rsp.Code == 0 {
		t.Fatal("clock_settime failure was accepted")
	}
	if !completed {
		t.Fatal("prepared key rotation was not completed after clock failure")
	}
}

func TestSetTimeStopsBeforeClockChangeWhenSessionInvalidationPreparationFails(t *testing.T) {
	setupSetTimeHandlerTest(t)
	getSystemTimeLocation = func() (*time.Location, error) { return time.UTC, nil }
	prepareSystemTimeSessions = func() (func(bool), error) {
		return nil, errors.New("session invalidation preparation failed")
	}
	setSystemTime = func(time.Time) error {
		t.Fatal("clock changed after session invalidation preparation failed")
		return nil
	}

	rsp := performSetTimeRequest(t, validSetSystemTimeRequest, "application/json")
	if rsp.Code == 0 {
		t.Fatal("session invalidation preparation failure was reported as success")
	}
}

func setupSetTimeHandlerTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	originalGetLocation := getSystemTimeLocation
	originalPrepareSessions := prepareSystemTimeSessions
	originalSetTime := setSystemTime

	t.Cleanup(func() {
		getSystemTimeLocation = originalGetLocation
		prepareSystemTimeSessions = originalPrepareSessions
		setSystemTime = originalSetTime
	})
}

func performSetTimeRequest(t *testing.T, body, contentType string) setSystemTimeTestResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/vm/time", bytes.NewBufferString(body))
	if contentType != "" {
		ctx.Request.Header.Set("Content-Type", contentType)
	}
	ctx.Set("username", "admin")

	NewService().SetTime(ctx)

	var rsp setSystemTimeTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("unmarshal response: %v body=%s", err, recorder.Body.String())
	}
	return rsp
}
