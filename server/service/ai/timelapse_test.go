package ai

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

func ptrInt64(value int64) *int64 {
	return &value
}

func ptrInt(value int) *int {
	return &value
}

func TestParseTimelapseIndexReadsCanonicalFieldsAndTSMetadata(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "2026-06-01.ts.idx")
	content := "{\"frame\":0,\"pts\":0,\"duration\":3000,\"captured_at\":\"2026-06-01T00:00:00Z\",\"capture_id\":\"session:6\",\"width\":1920,\"height\":1080,\"h264_bytes\":123,\"ts_offset\":0,\"segment_id\":0}\n" +
		"{\"frame\":1,\"pts\":3000,\"duration\":3000,\"captured_at\":\"2026-06-01T00:00:01Z\",\"capture_id\":\"session:7\",\"width\":1920,\"height\":1080,\"h264_bytes\":456,\"ts_offset\":188,\"segment_id\":0}\n"
	if err := os.WriteFile(indexPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	frames, err := parseTimelapseIndex(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}
	if frames[1].TSOffset == nil || *frames[1].TSOffset != 188 {
		t.Fatalf("unexpected ts_offset: %#v", frames[1].TSOffset)
	}
	if frames[1].SegmentID == nil || *frames[1].SegmentID != 0 {
		t.Fatalf("unexpected segment_id: %#v", frames[1].SegmentID)
	}
	if frames[1].CapturedAtRaw != "2026-06-01T00:00:01Z" {
		t.Fatalf("unexpected captured_at: %q", frames[1].CapturedAtRaw)
	}
	if frames[1].CaptureID != "session:7" {
		t.Fatalf("unexpected capture_id: %q", frames[1].CaptureID)
	}
	if got := frames[1].CapturedAt.Format(time.RFC3339Nano); got != frames[1].CapturedAtRaw {
		t.Fatalf("unexpected parsed captured_at: %q", got)
	}

	framesRsp := buildFramesResponse("2026-06-01", frames)
	if len(framesRsp.Frames) != 2 || framesRsp.Frames[1].CapturedAt != frames[1].CapturedAtRaw {
		t.Fatalf("frames response did not use captured_at: %#v", framesRsp)
	}
	day := buildDay("2026-06-01", frames, 376, time.Date(2026, 6, 2, 0, 0, 0, 0, time.Local))
	if day.StartTime != frames[0].CapturedAtRaw || day.EndTime != frames[1].CapturedAtRaw {
		t.Fatalf("day response did not use captured_at: %#v", day)
	}
	if !day.Deletable {
		t.Fatalf("historical day should be deletable: %#v", day)
	}
}

func TestBuildDayAllowsDeletingCurrentDate(t *testing.T) {
	now := time.Date(2026, 6, 1, 23, 59, 59, 0, time.FixedZone("CST", 8*60*60))
	day := buildDay("2026-06-01", nil, 0, now)
	if !day.Deletable {
		t.Fatalf("current day should be manually deletable: %#v", day)
	}
}

func TestDeleteTimelapseDayRemovesDirectoryAndReturnsTotalBytes(t *testing.T) {
	root := t.TempDir()
	date := "2026-06-01"
	dayPath := filepath.Join(root, date)
	if err := os.MkdirAll(filepath.Join(dayPath, "metadata"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(dayPath, date+".ts"):               "video-data",
		filepath.Join(dayPath, date+".ts.idx"):           "index",
		filepath.Join(dayPath, date+".ts.stable"):        "188\n",
		filepath.Join(dayPath, "metadata", "extra.json"): "{}",
	}
	var wantBytes int64
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		wantBytes += int64(len(content))
	}

	deletedBytes, err := deleteTimelapseDay(root, date, time.Date(2026, 6, 2, 0, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	if deletedBytes != wantBytes {
		t.Fatalf("unexpected deleted bytes: got %d want %d", deletedBytes, wantBytes)
	}
	if _, err := os.Stat(dayPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("timelapse directory still exists: %v", err)
	}
}

func TestDeleteTimelapseDayIsIdempotentWhenDirectoryIsMissing(t *testing.T) {
	deletedBytes, err := deleteTimelapseDay(t.TempDir(), "2026-06-01", time.Date(2026, 6, 2, 0, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	if deletedBytes != 0 {
		t.Fatalf("missing directory deleted %d bytes", deletedBytes)
	}
}

func TestDeleteTimelapseDayAllowsCurrentDate(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	date := now.Format("2006-01-02")
	dayPath := filepath.Join(root, date)
	if err := os.MkdirAll(dayPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dayPath, date+".ts"), []byte("today"), 0o644); err != nil {
		t.Fatal(err)
	}

	deletedBytes, err := deleteTimelapseDay(root, date, now)
	if err != nil {
		t.Fatalf("delete current timelapse day: %v", err)
	}
	if deletedBytes != int64(len("today")) {
		t.Fatalf("unexpected deleted bytes: got %d want %d", deletedBytes, len("today"))
	}
	if _, err := os.Stat(dayPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("current timelapse directory still exists: %v", err)
	}
}

func TestDeleteTimelapseDayRejectsFutureAndInvalidDates(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	for _, date := range []string{"2026-06-03", "../2026-06-01", "2026-02-30"} {
		t.Run(date, func(t *testing.T) {
			if _, err := deleteTimelapseDay(root, date, now); !errors.Is(err, errTimelapseDayNotDeletable) {
				t.Fatalf("expected not-deletable error for %q, got %v", date, err)
			}
		})
	}
}

func TestProtectedScreenTimelapsePlaybackEndpointsRequireAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withAIStateFiles(t)

	service := &Service{}
	verificationCalls := 0
	verifyScreenTimelapsePlaybackChallenge = func(*gin.Context, string) error {
		verificationCalls++
		return errors.New("authorization required")
	}

	jsonEndpoints := []struct {
		name    string
		method  string
		target  string
		body    string
		handler gin.HandlerFunc
	}{
		{name: "access", method: http.MethodPost, target: "/api/ai/screen-timelapse/access", body: `{}`, handler: service.AuthorizeScreenTimelapsePlayback},
		{name: "days", method: http.MethodGet, target: "/api/ai/screen-timelapse/days?recent=30", handler: service.GetScreenTimelapseDays},
		{name: "frames", method: http.MethodGet, target: "/api/ai/screen-timelapse/frames?date=2026-06-01", handler: service.GetScreenTimelapseFrames},
		{name: "search", method: http.MethodGet, target: "/api/ai/screen-timelapse/search?query=needle", handler: service.SearchScreenTimelapse},
		{name: "delete", method: http.MethodDelete, target: "/api/ai/screen-timelapse/day?date=2026-06-01", handler: service.DeleteScreenTimelapseDay},
	}

	for _, tt := range jsonEndpoints {
		t.Run(tt.name, func(t *testing.T) {
			c, recorder := newTestContext(tt.method, tt.target, tt.body)
			tt.handler(c)
			var rsp struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
				t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
			}
			if rsp.Code != -3 {
				t.Fatalf("expected authorization rejection, got %#v", rsp)
			}
		})
	}

	c, recorder := newTestContext(http.MethodGet, "/api/ai/screen-timelapse/video?date=2026-06-01", "")
	service.GetScreenTimelapseVideo(c)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected protected video to return 403, got %d", recorder.Code)
	}
	if verificationCalls != len(jsonEndpoints)+1 {
		t.Fatalf("expected every endpoint to verify playback access, got %d checks", verificationCalls)
	}
}

type fakeTimelapseDayFileInfo struct {
	mode os.FileMode
}

func (info fakeTimelapseDayFileInfo) Name() string       { return "2026-06-01" }
func (info fakeTimelapseDayFileInfo) Size() int64        { return 0 }
func (info fakeTimelapseDayFileInfo) Mode() os.FileMode  { return info.mode }
func (info fakeTimelapseDayFileInfo) ModTime() time.Time { return time.Time{} }
func (info fakeTimelapseDayFileInfo) IsDir() bool        { return info.mode.IsDir() }
func (info fakeTimelapseDayFileInfo) Sys() any           { return nil }

func TestDeleteScreenTimelapseDayGenericFailureCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withAIStateFiles(t)

	const date = "2026-06-01"

	oldCache := screenTimelapseCache
	cache := newTimelapseCache()
	cache.now = func() time.Time { return time.Date(2026, 7, 1, 12, 0, 0, 0, time.Local) }
	screenTimelapseCache = cache
	t.Cleanup(func() { screenTimelapseCache = oldCache })

	oldCoordinator := screenTimelapseMutationCoordinator
	coordinator := newTimelapseMutationCoordinator(
		filepath.Join(t.TempDir(), "timelapse.lock"),
		func() *timelapseCache { return screenTimelapseCache },
		newTimelapsePlaybackLeases(),
	)
	screenTimelapseMutationCoordinator = coordinator
	t.Cleanup(func() { screenTimelapseMutationCoordinator = oldCoordinator })

	service := &Service{}
	deleteDay := func() (int, proto.DeleteScreenTimelapseDayRsp) {
		c, recorder := newTestContext(http.MethodDelete, "/api/ai/screen-timelapse/day?date="+date, "")
		service.DeleteScreenTimelapseDay(c)
		var rsp struct {
			Code int                               `json:"code"`
			Data proto.DeleteScreenTimelapseDayRsp `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
			t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
		}
		return rsp.Code, rsp.Data
	}

	t.Run("generic failure returns -6 not -3", func(t *testing.T) {
		// Must not wrap os.ErrNotExist: a missing day is a silent success.
		simulatedErr := errors.New("simulated io failure")
		coordinator.lstat = func(string) (os.FileInfo, error) { return nil, simulatedErr }

		code, _ := deleteDay()
		if code == -3 {
			t.Fatal("generic deletion failure must not reuse authorization code -3")
		}
		if code != -6 {
			t.Fatalf("expected generic failure code -6, got %d", code)
		}
	})

	t.Run("success reports deleted bytes", func(t *testing.T) {
		const wantBytes int64 = 4096
		coordinator.lstat = func(string) (os.FileInfo, error) {
			return fakeTimelapseDayFileInfo{mode: os.ModeDir | 0o755}, nil
		}
		coordinator.directorySize = func(string) (int64, error) { return wantBytes, nil }
		coordinator.rename = func(string, string) error { return nil }
		coordinator.removeAll = func(string) error { return nil }

		code, data := deleteDay()
		if code != 0 {
			t.Fatalf("expected success code 0, got %d", code)
		}
		if data.Date != date || data.DeletedBytes != wantBytes {
			t.Fatalf("unexpected delete response: %#v", data)
		}
	})
}

func TestAuthorizeScreenTimelapsePlaybackForwardsChallenge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withAIStateFiles(t)

	service := &Service{}
	var gotChallenge string
	verifyScreenTimelapsePlaybackChallenge = func(_ *gin.Context, challenge string) error {
		gotChallenge = challenge
		return nil
	}

	c, recorder := newTestContext(http.MethodPost, "/api/ai/screen-timelapse/access", `{"challenge":"screen-timelapse-playback-challenge"}`)
	service.AuthorizeScreenTimelapsePlayback(c)
	var rsp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	if rsp.Code != 0 || gotChallenge != "screen-timelapse-playback-challenge" {
		t.Fatalf("unexpected access authorization response: code=%d challenge=%q", rsp.Code, gotChallenge)
	}
}

func TestTimelapseCacheInvalidateDayClearsOnlyAffectedDate(t *testing.T) {
	tc := newTimelapseCache()
	tc.now = func() time.Time { return time.Date(2026, 6, 2, 0, 0, 0, 0, time.Local) }
	date := "2026-06-01"
	otherDate := "2026-05-31"
	root := filepath.Join(t.TempDir(), "timelapse")
	dayPath := filepath.Join(root, date)
	videoPath := filepath.Join(dayPath, date+".ts")
	otherVideoPath := filepath.Join(root, otherDate, otherDate+".ts")

	tc.setDays(30, []proto.ScreenTimelapseDay{{Date: date}, {Date: otherDate}})
	tc.setDays(90, []proto.ScreenTimelapseDay{{Date: date}})
	tc.setFrames(date, []timelapseFrame{{CaptureID: "deleted"}})
	tc.setFrames(otherDate, []timelapseFrame{{CaptureID: "kept"}})
	tc.tsIndexes[timelapseTSIndexCacheKey(videoPath, 188)] = cacheEntry[timelapseTSIndex]{
		value:     timelapseTSIndex{Size: 188},
		expiresAt: tc.now().Add(time.Minute),
	}
	tc.tsIndexes[timelapseTSIndexCacheKey(otherVideoPath, 376)] = cacheEntry[timelapseTSIndex]{
		value:     timelapseTSIndex{Size: 376},
		expiresAt: tc.now().Add(time.Minute),
	}
	deletedLoad := &timelapseTSIndexLoad{path: videoPath}
	keptLoad := &timelapseTSIndexLoad{path: otherVideoPath}
	tc.tsIndexLoads[timelapseTSIndexCacheKey(videoPath, 564)] = deletedLoad
	tc.tsIndexLoads[timelapseTSIndexCacheKey(otherVideoPath, 752)] = keptLoad

	tc.invalidateDay(date, dayPath)

	tc.mu.Lock()
	defer tc.mu.Unlock()
	if len(tc.days) != 0 {
		t.Fatalf("days cache was not cleared: %#v", tc.days)
	}
	if _, ok := tc.frames[date]; ok {
		t.Fatalf("deleted date frames remained cached: %#v", tc.frames[date])
	}
	if _, ok := tc.frames[otherDate]; !ok {
		t.Fatal("unrelated date frames were removed")
	}
	if _, ok := tc.tsIndexes[timelapseTSIndexCacheKey(videoPath, 188)]; ok {
		t.Fatal("deleted date TS index remained cached")
	}
	if _, ok := tc.tsIndexes[timelapseTSIndexCacheKey(otherVideoPath, 376)]; !ok {
		t.Fatal("unrelated TS index was removed")
	}
	if !deletedLoad.invalidated {
		t.Fatal("deleted date in-flight TS index load was not invalidated")
	}
	if keptLoad.invalidated {
		t.Fatal("unrelated in-flight TS index load was invalidated")
	}
}

func TestWaitForTSIndexLoadRejectsLoadInvalidatedWhileWaiting(t *testing.T) {
	tc := newTimelapseCache()
	load := &timelapseTSIndexLoad{
		index: timelapseTSIndex{Size: 188},
	}
	load.wg.Add(1)

	result := make(chan error, 1)
	go func() {
		_, err := tc.waitForTSIndexLoad(load)
		result <- err
	}()

	tc.mu.Lock()
	load.invalidated = true
	tc.mu.Unlock()
	load.wg.Done()

	if err := <-result; !errors.Is(err, errTimelapseIndexInvalidated) {
		t.Fatalf("expected invalidated load error, got %v", err)
	}
}

func TestParseTimelapseIndexSkipsInvalidCapturedAt(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "2026-06-01.ts.idx")
	content := `{"frame":0,"pts":0,"duration":3000,"captured_at":"invalid","capture_id":"session:7","ts_offset":0,"segment_id":0}` + "\n" +
		`{"frame":1,"pts":3000,"duration":3000,"captured_at":"2026-06-01T00:00:01Z","capture_id":"  ","ts_offset":188,"segment_id":0}` + "\n"
	if err := os.WriteFile(indexPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	frames, err := parseTimelapseIndex(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 0 {
		t.Fatalf("expected invalid captured_at row to be skipped, got %#v", frames)
	}
}
func TestReadTimelapseStableOffsetValidation(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "2026-06-01.ts")
	if err := os.WriteFile(videoPath, make([]byte, mpegTSPacketSize*3), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(timelapseStablePath(videoPath), []byte("376\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	offset, ok := readTimelapseStableOffset(videoPath, mpegTSPacketSize*3)
	if !ok || offset != mpegTSPacketSize*2 {
		t.Fatalf("expected valid stable offset %d, got offset=%d ok=%t", mpegTSPacketSize*2, offset, ok)
	}

	if err := os.WriteFile(timelapseStablePath(videoPath), []byte("377\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if offset, ok := readTimelapseStableOffset(videoPath, mpegTSPacketSize*3); ok {
		t.Fatalf("expected invalid unaligned stable offset, got %d", offset)
	}

	if err := os.WriteFile(timelapseStablePath(videoPath), []byte("752\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if offset, ok := readTimelapseStableOffset(videoPath, mpegTSPacketSize*3); ok {
		t.Fatalf("expected invalid oversize stable offset, got %d", offset)
	}
}

func TestFindTimelapseTSStableEndUsesSidecarAndFallsBack(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "2026-06-01.ts")
	if err := os.WriteFile(videoPath, make([]byte, mpegTSPacketSize*2), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(timelapseStablePath(videoPath), []byte("188\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	offset, err := findTimelapseTSStableEnd(videoPath, mpegTSPacketSize*2)
	if err != nil || offset != mpegTSPacketSize {
		t.Fatalf("expected sidecar stable offset, got offset=%d err=%v", offset, err)
	}

	if err := os.WriteFile(timelapseStablePath(videoPath), []byte("189\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if offset, err := findTimelapseTSStableEnd(videoPath, mpegTSPacketSize*2); err == nil || offset != 0 {
		t.Fatalf("expected invalid sidecar to fall back to TS scan error, got offset=%d err=%v", offset, err)
	}
}

func TestSearchLineFromTextFindsMatchingLineCaseInsensitive(t *testing.T) {
	line, fallback := searchLineFromText("OPNM4520\nOPNM4520 Datasheet\nOPNS3030A", "datasheet")
	if line != "OPNM4520 Datasheet" {
		t.Fatalf("unexpected matched line: %q", line)
	}
	if fallback != "OPNM4520" {
		t.Fatalf("unexpected fallback line: %q", fallback)
	}
}

func TestPickSearchSnippetUsesOCRPreview(t *testing.T) {
	result := memfabSearchResult{
		OCRPreview: "OPNM4520\nOPNM4520 Datasheet\nOPNS3030A",
	}

	if got := pickSearchSnippet(result, "datasheet"); got != "OPNM4520 Datasheet" {
		t.Fatalf("unexpected search snippet: %q", got)
	}
}

func TestMemfabSearchResponseDecodesOCRPreview(t *testing.T) {
	var response memfabSearchResponse
	if err := json.Unmarshal([]byte(`{"query":"SPD27","count":1,"results":[{"captured_at":"2026-07-15T06:17:33.852545112Z","capture_id":"session:1","ocr_preview":"搜索 [SPD27]"}]}`), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 1 || response.Results[0].OCRPreview != "搜索 [SPD27]" {
		t.Fatalf("unexpected memfab response: %#v", response)
	}
}

func TestEnrichSearchResultUsesCaptureIDWhenVideoFrameExists(t *testing.T) {
	oldCache := screenTimelapseCache
	screenTimelapseCache = newTimelapseCache()
	t.Cleanup(func() {
		screenTimelapseCache = oldCache
	})

	capturedRaw := "2000-01-02T15:27:36+08:00"
	capturedAt, err := time.Parse(time.RFC3339, capturedRaw)
	if err != nil {
		t.Fatal(err)
	}
	screenTimelapseCache.setFrames("2000-01-02", []timelapseFrame{
		{
			PTS:           9000,
			CapturedAtRaw: capturedRaw,
			CaptureID:     "session:1",
			CapturedAt:    capturedAt,
			CurrentTime:   0.1,
			TSOffset:      ptrInt64(0),
		},
	})

	raw := memfabSearchResult{
		CapturedAt: capturedRaw,
		CaptureID:  "session:1",
		OCRPreview: "搜索 SPD27",
	}
	checks := searchResultFileChecks{
		videoExists: func(date string) bool { return date == "2000-01-02" },
	}

	result, _ := enrichSearchResult(raw, "SPD27", checks)
	if !result.Available {
		t.Fatalf("expected result available, got reason=%q", result.UnavailableReason)
	}
	if result.Date != "2000-01-02" || result.PTS != 9000 || result.CurrentTime != 0.1 {
		t.Fatalf("unexpected matched result: %#v", result)
	}
	if result.Snippet != "搜索 SPD27" {
		t.Fatalf("expected OCR preview in result snippet, got %q", result.Snippet)
	}
}

func TestEnrichSearchResultRequiresExactCaptureID(t *testing.T) {
	oldCache := screenTimelapseCache
	screenTimelapseCache = newTimelapseCache()
	t.Cleanup(func() {
		screenTimelapseCache = oldCache
	})

	capturedRaw := "2000-01-02T15:27:36+08:00"
	capturedAt, err := time.Parse(time.RFC3339, capturedRaw)
	if err != nil {
		t.Fatal(err)
	}
	exactCaptureTime := capturedAt.Add(5 * time.Minute)
	screenTimelapseCache.setFrames("2000-01-02", []timelapseFrame{
		{
			PTS:           9000,
			CapturedAtRaw: capturedRaw,
			CaptureID:     "session:old",
			CapturedAt:    capturedAt,
			CurrentTime:   0.1,
			TSOffset:      ptrInt64(0),
		},
		{
			PTS:           18000,
			CapturedAtRaw: exactCaptureTime.Format(time.RFC3339),
			CaptureID:     "session:7",
			CapturedAt:    exactCaptureTime,
			CurrentTime:   0.2,
			TSOffset:      ptrInt64(188),
		},
	})
	checks := searchResultFileChecks{
		videoExists: func(date string) bool { return date == "2000-01-02" },
	}

	var searchResponse memfabSearchResponse
	if err := json.Unmarshal([]byte(`{"query":"SPD27","count":1,"results":[{"captured_at":"2000-01-02T15:27:36+08:00","capture_id":"session:7"}]}`), &searchResponse); err != nil {
		t.Fatal(err)
	}
	if len(searchResponse.Results) != 1 {
		t.Fatalf("unexpected memfab response: %#v", searchResponse)
	}
	result, _ := enrichSearchResult(searchResponse.Results[0], "SPD27", checks)
	if !result.Available || result.PTS != 18000 || result.MatchedFrameTime != exactCaptureTime.Format(time.RFC3339) {
		t.Fatalf("expected exact capture_id match, got %#v", result)
	}

	result, _ = enrichSearchResult(memfabSearchResult{
		CapturedAt: capturedRaw,
		CaptureID:  "unknown-session:9",
	}, "SPD27", checks)
	if result.Available || result.UnavailableReason != "no_nearby_frame" {
		t.Fatalf("expected missing exact capture_id to be unavailable, got %#v", result)
	}
}

func TestEnrichSearchResultFindsFrameInAdjacentDateDirectory(t *testing.T) {
	oldCache := screenTimelapseCache
	screenTimelapseCache = newTimelapseCache()
	t.Cleanup(func() {
		screenTimelapseCache = oldCache
	})

	capturedRaw := "2000-01-02T18:27:36Z"
	capturedAt, err := time.Parse(time.RFC3339, capturedRaw)
	if err != nil {
		t.Fatal(err)
	}
	screenTimelapseCache.setFrames("2000-01-03", []timelapseFrame{
		{
			PTS:           9000,
			CapturedAtRaw: capturedRaw,
			CaptureID:     "session:adjacent",
			CapturedAt:    capturedAt,
			CurrentTime:   0.1,
			TSOffset:      ptrInt64(0),
		},
	})

	result, _ := enrichSearchResult(memfabSearchResult{
		CapturedAt: capturedRaw,
		CaptureID:  "session:adjacent",
	}, "SPD27", searchResultFileChecks{
		videoExists: func(date string) bool { return date == "2000-01-03" },
	})
	if !result.Available || result.Date != "2000-01-03" || result.PTS != 9000 {
		t.Fatalf("expected adjacent-date match, got %#v", result)
	}
}

func TestEnrichSearchResultReturnsNoNearbyFrameWithoutTimelapse(t *testing.T) {
	oldCache := screenTimelapseCache
	screenTimelapseCache = newTimelapseCache()
	t.Cleanup(func() {
		screenTimelapseCache = oldCache
	})

	result, skipped := enrichSearchResult(memfabSearchResult{
		CapturedAt: "1900-01-02T15:27:36+08:00",
		CaptureID:  "session:missing",
	}, "SPD27", searchResultFileChecks{})
	if result.Available || result.UnavailableReason != "no_nearby_frame" {
		t.Fatalf("expected no_nearby_frame, got %#v", result)
	}
	if skipped.missingFile {
		t.Fatalf("unexpected skipped flags: %#v", skipped)
	}
}

func TestFilterTimelapseFramesByStableOffset(t *testing.T) {
	frames := []timelapseFrame{
		{PTS: 0, TSOffset: ptrInt64(0)},
		{PTS: 3000, TSOffset: ptrInt64(188)},
		{PTS: 6000, TSOffset: ptrInt64(376)},
		{PTS: 9000},
	}

	stable := filterTimelapseFramesByStableOffset(frames, 376)
	if len(stable) != 2 {
		t.Fatalf("expected 2 stable frames, got %d", len(stable))
	}
	if stable[0].PTS != 0 || stable[1].PTS != 3000 {
		t.Fatalf("unexpected stable frames: %#v", stable)
	}
}

func TestTimelapseSeekStartFromFramesUsesTSOffset(t *testing.T) {
	frames := []timelapseFrame{
		{PTS: 0, TSOffset: ptrInt64(0), SegmentID: ptrInt(0)},
		{PTS: 3000, TSOffset: ptrInt64(188), SegmentID: ptrInt(0)},
		{PTS: 6000, TSOffset: ptrInt64(376), SegmentID: ptrInt(0)},
	}

	seekStart, ok := timelapseSeekStartFromFrames(frames, 564, 4500)
	if !ok || seekStart.Offset != 188 || len(seekStart.Prelude) != 0 {
		t.Fatalf("unexpected seek start: %#v ok=%t", seekStart, ok)
	}
}
