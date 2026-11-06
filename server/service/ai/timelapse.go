package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

const (
	memfabRoot             = "/var/lib/memfab"
	timelapseDirName       = "timelapse"
	mpegTSClockHz          = 90000
	mpegTSTicksPerMs       = 90
	mpegTSPacketSize       = 188
	currentDateCacheTTL    = 10 * time.Second
	historicalDateCacheTTL = 60 * time.Second
)

var timelapseDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
var screenTimelapseSearchSlots = make(chan struct{}, 2)

var (
	errTimelapseDayNotDeletable  = errors.New("timelapse day is not deletable")
	errTimelapseIndexInvalidated = errors.New("timelapse index invalidated")
)

type timelapseFrame struct {
	Frame         int       `json:"frame"`
	PTS           int64     `json:"pts"`
	Duration      int64     `json:"duration"`
	CapturedAtRaw string    `json:"captured_at"`
	CaptureID     string    `json:"capture_id"`
	Width         int       `json:"width"`
	Height        int       `json:"height"`
	H264Bytes     int64     `json:"h264_bytes"`
	TSOffset      *int64    `json:"ts_offset"`
	SegmentID     *int      `json:"segment_id"`
	CapturedAt    time.Time `json:"-"`
	CurrentTime   float64   `json:"-"`
}

type memfabSearchResult struct {
	CapturedAt string `json:"captured_at"`
	CaptureID  string `json:"capture_id"`
	OCRPreview string `json:"ocr_preview"`
}

type memfabSearchResponse struct {
	Query                   string               `json:"query"`
	Count                   int                  `json:"count"`
	Results                 []memfabSearchResult `json:"results"`
	SkippedMissingFileCount int                  `json:"skipped_missing_file_count"`
}

type matchedTimelapseFrame struct {
	date  string
	frame timelapseFrame
}

type timelapseTSSeekStart struct {
	Offset  int64
	Prelude []byte
}

type timelapseTSSeekPoint struct {
	PTS    int64
	Offset int64
}

type timelapseTSIndex struct {
	Size       int64
	Prelude    []byte
	SeekPoints []timelapseTSSeekPoint
	StableEnd  int64
}

type timelapseTSIndexLoad struct {
	wg          sync.WaitGroup
	index       timelapseTSIndex
	err         error
	path        string
	invalidated bool
}

type searchResultFileChecks struct {
	videoExists func(date string) bool
}

type searchResultSkippedFlags struct {
	missingFile bool
}

type cacheEntry[T any] struct {
	value     T
	expiresAt time.Time
}

type timelapseCache struct {
	mu           sync.Mutex
	generation   uint64
	days         map[int]cacheEntry[[]proto.ScreenTimelapseDay]
	frames       map[string]cacheEntry[[]timelapseFrame]
	tsIndexes    map[string]cacheEntry[timelapseTSIndex]
	tsIndexLoads map[string]*timelapseTSIndexLoad
	now          func() time.Time
}

func newTimelapseCache() *timelapseCache {
	return &timelapseCache{
		days:         make(map[int]cacheEntry[[]proto.ScreenTimelapseDay]),
		frames:       make(map[string]cacheEntry[[]timelapseFrame]),
		tsIndexes:    make(map[string]cacheEntry[timelapseTSIndex]),
		tsIndexLoads: make(map[string]*timelapseTSIndexLoad),
		now:          time.Now,
	}
}

var screenTimelapseCache = newTimelapseCache()

func timelapseRoot() string {
	return filepath.Join(memfabRoot, timelapseDirName)
}

func isValidTimelapseDate(date string) bool {
	if !timelapseDatePattern.MatchString(date) {
		return false
	}
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

func isPlayableTimelapseDate(date string, now time.Time) bool {
	return isValidTimelapseDate(date) && date <= now.Format("2006-01-02")
}

func isCurrentTimelapseDate(date string, now time.Time) bool {
	return isValidTimelapseDate(date) && date == now.Format("2006-01-02")
}

func isDeletableTimelapseDate(date string, now time.Time) bool {
	return isPlayableTimelapseDate(date, now)
}

func timelapsePTSSeconds(pts int64) float64 {
	return float64(pts) / mpegTSClockHz
}

func timelapsePTSMilliseconds(pts int64) int64 {
	return pts / mpegTSTicksPerMs
}

func timelapseSecondsToPTS(seconds float64) int64 {
	if seconds <= 0 {
		return 0
	}
	return int64(seconds * mpegTSClockHz)
}

func timelapseTSIndexCacheKey(path string, size int64) string {
	return path + "\x00" + strconv.FormatInt(size, 10)
}

func cloneTimelapseTSIndex(index timelapseTSIndex) timelapseTSIndex {
	index.Prelude = append([]byte(nil), index.Prelude...)
	index.SeekPoints = append([]timelapseTSSeekPoint(nil), index.SeekPoints...)
	return index
}

func (index timelapseTSIndex) seekStart(targetPTS int64) (timelapseTSSeekStart, bool) {
	if len(index.SeekPoints) == 0 {
		return timelapseTSSeekStart{}, false
	}

	pointIndex := sort.Search(len(index.SeekPoints), func(i int) bool {
		return index.SeekPoints[i].PTS > targetPTS
	}) - 1
	if pointIndex < 0 {
		pointIndex = 0
	}

	return timelapseTSSeekStart{
		Offset:  index.SeekPoints[pointIndex].Offset,
		Prelude: append([]byte(nil), index.Prelude...),
	}, true
}

func searchLineFromText(text string, query string) (string, string) {
	content := strings.TrimSpace(text)
	if content == "" {
		return "", ""
	}

	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	fallback := ""
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if fallback == "" {
			fallback = line
		}
		if lowerQuery != "" && strings.Contains(strings.ToLower(line), lowerQuery) {
			return line, fallback
		}
	}
	if lowerQuery == "" {
		return fallback, fallback
	}
	return "", fallback
}

func pickSearchSnippet(result memfabSearchResult, query string) string {
	line, fallback := searchLineFromText(result.OCRPreview, query)
	if line != "" {
		return line
	}
	return fallback
}

func fileExists(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && !stat.IsDir()
}

func timelapseVideoExists(date string) bool {
	if !isPlayableTimelapseDate(date, screenTimelapseCache.now()) {
		return false
	}
	videoPath, err := timelapseVideoPath(date)
	return err == nil && fileExists(videoPath)
}

func defaultSearchResultFileChecks() searchResultFileChecks {
	return searchResultFileChecks{
		videoExists: timelapseVideoExists,
	}
}

func candidateDatesForCapturedAt(capturedAt time.Time) []string {
	date := capturedAt.Format("2006-01-02")
	return []string{
		date,
		capturedAt.AddDate(0, 0, -1).Format("2006-01-02"),
		capturedAt.AddDate(0, 0, 1).Format("2006-01-02"),
	}
}

func matchingFrameForSearch(capturedAt time.Time, captureID string) (matchedTimelapseFrame, bool) {
	now := screenTimelapseCache.now()
	captureID = strings.TrimSpace(captureID)
	if captureID == "" {
		return matchedTimelapseFrame{}, false
	}
	framesByDate := make([]struct {
		date   string
		frames []timelapseFrame
	}, 0, 3)
	for _, date := range candidateDatesForCapturedAt(capturedAt) {
		frames, err := getTimelapseFrames(date)
		if err != nil {
			continue
		}
		framesByDate = append(framesByDate, struct {
			date   string
			frames []timelapseFrame
		}{
			date:   date,
			frames: stableTimelapseFrames(date, frames, now),
		})
	}

	for _, framesForDate := range framesByDate {
		for _, frame := range framesForDate.frames {
			if frame.CaptureID == captureID {
				return matchedTimelapseFrame{date: framesForDate.date, frame: frame}, true
			}
		}
	}
	return matchedTimelapseFrame{}, false
}

func normalizeSearchLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func normalizeSearchQuery(query string) (string, bool) {
	query = strings.TrimSpace(query)
	if query == "" || len([]rune(query)) > 200 {
		return "", false
	}
	return query, true
}

func runMemfabSearch(ctx context.Context, query string, limit int) (memfabSearchResponse, error) {
	select {
	case screenTimelapseSearchSlots <- struct{}{}:
		defer func() { <-screenTimelapseSearchSlots }()
	case <-ctx.Done():
		return memfabSearchResponse{}, ctx.Err()
	}

	cmd := exec.CommandContext(ctx, "memfab", "replay", "search", "--query", query, "--limit", strconv.Itoa(limit))
	cmd.Env = append(os.Environ(), "MEMFAB_ROOT="+memfabRoot)
	out, err := cmd.Output()
	if err != nil {
		return memfabSearchResponse{}, err
	}
	var rsp memfabSearchResponse
	if err := json.Unmarshal(out, &rsp); err != nil {
		return memfabSearchResponse{}, err
	}
	return rsp, nil
}

func enrichSearchResult(raw memfabSearchResult, query string, checks searchResultFileChecks) (proto.ScreenTimelapseSearchResult, searchResultSkippedFlags) {
	var skipped searchResultSkippedFlags
	capturedRaw := raw.CapturedAt
	capturedAt, err := time.Parse(time.RFC3339, capturedRaw)
	if err != nil {
		return proto.ScreenTimelapseSearchResult{
			CapturedAt:        capturedRaw,
			Available:         false,
			UnavailableReason: "invalid_captured_at",
			Snippet:           pickSearchSnippet(raw, query),
		}, skipped
	}

	result := proto.ScreenTimelapseSearchResult{
		CapturedAt: capturedRaw,
		Date:       capturedAt.Format("2006-01-02"),
		Snippet:    pickSearchSnippet(raw, query),
	}

	match, ok := matchingFrameForSearch(capturedAt, raw.CaptureID)
	if !ok {
		result.Available = false
		result.UnavailableReason = "no_nearby_frame"
		return result, skipped
	}
	result.Date = match.date
	if checks.videoExists != nil && !checks.videoExists(match.date) {
		result.Available = false
		result.UnavailableReason = "missing_video"
		skipped.missingFile = true
		return result, skipped
	}

	result.Available = true
	result.PTS = match.frame.PTS
	result.CurrentTime = match.frame.CurrentTime
	result.MatchedFrameTime = match.frame.CapturedAtRaw
	return result, skipped
}

func timelapseVideoPath(date string) (string, error) {
	if !isValidTimelapseDate(date) {
		return "", errors.New("invalid date")
	}
	return filepath.Join(timelapseRoot(), date, date+".ts"), nil
}

func timelapseIndexPath(date string) (string, error) {
	if !isValidTimelapseDate(date) {
		return "", errors.New("invalid date")
	}
	return filepath.Join(timelapseRoot(), date, date+".ts.idx"), nil
}

func timelapseStablePath(videoPath string) string {
	return videoPath + ".stable"
}

func readTimelapseStableOffset(videoPath string, fileSize int64) (int64, bool) {
	if fileSize <= 0 {
		return 0, false
	}

	data, err := os.ReadFile(timelapseStablePath(videoPath))
	if err != nil {
		return 0, false
	}
	offset, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil || offset <= 0 || offset > fileSize || offset%mpegTSPacketSize != 0 {
		return 0, false
	}
	return offset, true
}

func parseTimelapseIndex(path string) ([]timelapseFrame, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var frames []timelapseFrame
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), "\x00 \t\r\n")
		if line == "" {
			continue
		}

		var frame timelapseFrame
		if err := json.Unmarshal([]byte(line), &frame); err != nil {
			continue
		}
		if frame.TSOffset == nil || frame.SegmentID == nil {
			continue
		}
		if *frame.TSOffset < 0 || *frame.TSOffset%mpegTSPacketSize != 0 || *frame.SegmentID != 0 {
			continue
		}
		capturedAtRaw := strings.TrimSpace(frame.CapturedAtRaw)
		capturedAt, err := time.Parse(time.RFC3339Nano, capturedAtRaw)
		if err != nil {
			continue
		}
		frame.CapturedAtRaw = capturedAtRaw
		frame.CaptureID = strings.TrimSpace(frame.CaptureID)
		if frame.CaptureID == "" {
			continue
		}
		frame.CapturedAt = capturedAt
		frame.CurrentTime = timelapsePTSSeconds(frame.PTS)
		frames = append(frames, frame)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(frames, func(i, j int) bool {
		return frames[i].PTS < frames[j].PTS
	})
	return frames, nil
}

func tsPacketPayload(packet []byte) ([]byte, bool) {
	if len(packet) != mpegTSPacketSize || packet[0] != 0x47 {
		return nil, false
	}

	adaptationFieldControl := (packet[3] & 0x30) >> 4
	if adaptationFieldControl == 0 || adaptationFieldControl == 2 {
		return nil, false
	}

	offset := 4
	if adaptationFieldControl == 3 {
		adaptationFieldLength := int(packet[offset])
		offset += 1 + adaptationFieldLength
		if offset >= len(packet) {
			return nil, false
		}
	}

	return packet[offset:], true
}

func tsSectionPayload(packet []byte) ([]byte, bool) {
	payload, ok := tsPacketPayload(packet)
	if !ok || len(payload) == 0 {
		return nil, false
	}

	payloadUnitStart := packet[1]&0x40 != 0
	if !payloadUnitStart {
		return payload, true
	}

	pointerField := int(payload[0])
	offset := 1 + pointerField
	if offset >= len(payload) {
		return nil, false
	}
	return payload[offset:], true
}

func parsePATPMTPID(packet []byte) (uint16, bool) {
	pid := uint16(packet[1]&0x1f)<<8 | uint16(packet[2])
	if pid != 0 {
		return 0, false
	}

	section, ok := tsSectionPayload(packet)
	if !ok || len(section) < 12 || section[0] != 0x00 {
		return 0, false
	}

	sectionLength := int(section[1]&0x0f)<<8 | int(section[2])
	end := 3 + sectionLength - 4
	if end > len(section) {
		end = len(section)
	}

	for offset := 8; offset+4 <= end; offset += 4 {
		programNumber := uint16(section[offset])<<8 | uint16(section[offset+1])
		if programNumber == 0 {
			continue
		}
		return uint16(section[offset+2]&0x1f)<<8 | uint16(section[offset+3]), true
	}
	return 0, false
}

func parsePMTVideoPID(packet []byte, pmtPID uint16) (uint16, bool) {
	pid := uint16(packet[1]&0x1f)<<8 | uint16(packet[2])
	if pid != pmtPID {
		return 0, false
	}

	section, ok := tsSectionPayload(packet)
	if !ok || len(section) < 17 || section[0] != 0x02 {
		return 0, false
	}

	sectionLength := int(section[1]&0x0f)<<8 | int(section[2])
	programInfoLength := int(section[10]&0x0f)<<8 | int(section[11])
	offset := 12 + programInfoLength
	end := 3 + sectionLength - 4
	if end > len(section) {
		end = len(section)
	}

	for offset+5 <= end {
		streamType := section[offset]
		elementaryPID := uint16(section[offset+1]&0x1f)<<8 | uint16(section[offset+2])
		esInfoLength := int(section[offset+3]&0x0f)<<8 | int(section[offset+4])
		switch streamType {
		case 0x02, 0x1b, 0x24:
			return elementaryPID, true
		}
		offset += 5 + esInfoLength
	}
	return 0, false
}

func parsePESPts(packet []byte, videoPID uint16) (int64, bool) {
	pid := uint16(packet[1]&0x1f)<<8 | uint16(packet[2])
	if pid != videoPID || packet[1]&0x40 == 0 {
		return 0, false
	}

	payload, ok := tsPacketPayload(packet)
	if !ok || len(payload) < 14 {
		return 0, false
	}
	if payload[0] != 0x00 || payload[1] != 0x00 || payload[2] != 0x01 {
		return 0, false
	}

	streamID := payload[3]
	if streamID < 0xe0 || streamID > 0xef {
		return 0, false
	}
	ptsDtsFlags := (payload[7] & 0xc0) >> 6
	if ptsDtsFlags != 0x02 && ptsDtsFlags != 0x03 {
		return 0, false
	}

	pts := int64((payload[9]>>1)&0x07)<<30 |
		int64(payload[10])<<22 |
		int64((payload[11]>>1)&0x7f)<<15 |
		int64(payload[12])<<7 |
		int64((payload[13]>>1)&0x7f)
	return pts, true
}

func parseTimelapseTSIndex(path string, maxSize int64) (timelapseTSIndex, error) {
	alignedSize := maxSize - maxSize%mpegTSPacketSize
	if alignedSize <= 0 {
		return timelapseTSIndex{}, errors.New("empty mpeg-ts stream")
	}

	file, err := os.Open(path)
	if err != nil {
		return timelapseTSIndex{}, err
	}
	defer file.Close()

	index := timelapseTSIndex{Size: alignedSize}
	packet := make([]byte, mpegTSPacketSize)
	var offset int64
	var pmtPID uint16
	var videoPID uint16
	var pmtFound bool
	var videoFound bool
	var patPacket []byte
	var pmtPacket []byte

	for offset < alignedSize {
		if _, err := io.ReadFull(file, packet); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return timelapseTSIndex{}, err
		}
		if packet[0] != 0x47 {
			return timelapseTSIndex{}, errors.New("invalid mpeg-ts sync byte")
		}

		if !pmtFound {
			if pid, ok := parsePATPMTPID(packet); ok {
				pmtPID = pid
				pmtFound = true
				patPacket = append([]byte(nil), packet...)
			}
		}
		if pmtFound && !videoFound {
			if pid, ok := parsePMTVideoPID(packet, pmtPID); ok {
				videoPID = pid
				videoFound = true
				pmtPacket = append([]byte(nil), packet...)
			}
		}
		if videoFound {
			if pts, ok := parsePESPts(packet, videoPID); ok {
				index.SeekPoints = append(index.SeekPoints, timelapseTSSeekPoint{
					PTS:    pts,
					Offset: offset,
				})
			}
		}

		offset += mpegTSPacketSize
	}

	if len(index.SeekPoints) == 0 {
		return timelapseTSIndex{}, errors.New("no video PES packet found")
	}
	index.Prelude = append(patPacket, pmtPacket...)
	if len(index.SeekPoints) >= 2 {
		index.StableEnd = index.SeekPoints[len(index.SeekPoints)-1].Offset + mpegTSPacketSize
	}
	return index, nil
}

func cacheTTLForDate(date string, now time.Time) time.Duration {
	if date == now.Format("2006-01-02") {
		return currentDateCacheTTL
	}
	return historicalDateCacheTTL
}

func cacheTTLForDays(days []proto.ScreenTimelapseDay, now time.Time) time.Duration {
	today := now.Format("2006-01-02")
	for _, day := range days {
		if day.Date == today {
			return currentDateCacheTTL
		}
	}
	return historicalDateCacheTTL
}

func (tc *timelapseCache) getDays(recent int) ([]proto.ScreenTimelapseDay, bool) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	entry, ok := tc.days[recent]
	if !ok || !tc.now().Before(entry.expiresAt) {
		if ok {
			delete(tc.days, recent)
		}
		return nil, false
	}
	return append([]proto.ScreenTimelapseDay(nil), entry.value...), true
}

func (tc *timelapseCache) setDays(recent int, days []proto.ScreenTimelapseDay) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.setDaysLocked(recent, days)
}

func (tc *timelapseCache) setDaysLocked(recent int, days []proto.ScreenTimelapseDay) {
	tc.days[recent] = cacheEntry[[]proto.ScreenTimelapseDay]{
		value:     append([]proto.ScreenTimelapseDay(nil), days...),
		expiresAt: tc.now().Add(cacheTTLForDays(days, tc.now())),
	}
}

func (tc *timelapseCache) currentGeneration() uint64 {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	return tc.generation
}

func (tc *timelapseCache) setDaysAtGeneration(recent int, days []proto.ScreenTimelapseDay, generation uint64) bool {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if tc.generation != generation {
		return false
	}
	tc.setDaysLocked(recent, days)
	return true
}

func (tc *timelapseCache) getFrames(date string) ([]timelapseFrame, bool) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	entry, ok := tc.frames[date]
	if !ok || !tc.now().Before(entry.expiresAt) {
		if ok {
			delete(tc.frames, date)
		}
		return nil, false
	}
	return append([]timelapseFrame(nil), entry.value...), true
}

func (tc *timelapseCache) setFrames(date string, frames []timelapseFrame) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.setFramesLocked(date, frames)
}

func (tc *timelapseCache) setFramesLocked(date string, frames []timelapseFrame) {
	tc.frames[date] = cacheEntry[[]timelapseFrame]{
		value:     append([]timelapseFrame(nil), frames...),
		expiresAt: tc.now().Add(cacheTTLForDate(date, tc.now())),
	}
}

func (tc *timelapseCache) setFramesAtGeneration(date string, frames []timelapseFrame, generation uint64) bool {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if tc.generation != generation {
		return false
	}
	tc.setFramesLocked(date, frames)
	return true
}

func (tc *timelapseCache) invalidateDay(date string, dayPath string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.generation++

	for recent := range tc.days {
		delete(tc.days, recent)
	}
	delete(tc.frames, date)

	cleanDayPath := filepath.Clean(dayPath)
	pathPrefix := cleanDayPath + string(os.PathSeparator)
	for key := range tc.tsIndexes {
		path := strings.SplitN(key, "\x00", 2)[0]
		if path == cleanDayPath || strings.HasPrefix(path, pathPrefix) {
			delete(tc.tsIndexes, key)
		}
	}
	for _, load := range tc.tsIndexLoads {
		if load.path == cleanDayPath || strings.HasPrefix(load.path, pathPrefix) {
			load.invalidated = true
		}
	}
}

func (tc *timelapseCache) pruneExpiredTSIndexesLocked(now time.Time) {
	for key, entry := range tc.tsIndexes {
		if !now.Before(entry.expiresAt) {
			delete(tc.tsIndexes, key)
		}
	}
}

func (tc *timelapseCache) waitForTSIndexLoad(load *timelapseTSIndexLoad) (timelapseTSIndex, error) {
	load.wg.Wait()
	tc.mu.Lock()
	invalidated := load.invalidated
	tc.mu.Unlock()
	if invalidated {
		return timelapseTSIndex{}, errTimelapseIndexInvalidated
	}
	if load.err != nil {
		return timelapseTSIndex{}, load.err
	}
	return cloneTimelapseTSIndex(load.index), nil
}

func (tc *timelapseCache) getTSIndex(path string, maxSize int64) (timelapseTSIndex, error) {
	key := timelapseTSIndexCacheKey(path, maxSize)

	tc.mu.Lock()
	entry, ok := tc.tsIndexes[key]
	if ok && tc.now().Before(entry.expiresAt) {
		tc.mu.Unlock()
		return cloneTimelapseTSIndex(entry.value), nil
	}
	if ok {
		delete(tc.tsIndexes, key)
	}
	if load, ok := tc.tsIndexLoads[key]; ok {
		tc.mu.Unlock()
		return tc.waitForTSIndexLoad(load)
	}

	load := &timelapseTSIndexLoad{path: path}
	load.wg.Add(1)
	tc.tsIndexLoads[key] = load
	tc.mu.Unlock()

	index, err := parseTimelapseTSIndex(path, maxSize)

	tc.mu.Lock()
	load.index = cloneTimelapseTSIndex(index)
	load.err = err
	if err == nil && !load.invalidated {
		now := tc.now()
		tc.pruneExpiredTSIndexesLocked(now)
		tc.tsIndexes[key] = cacheEntry[timelapseTSIndex]{
			value:     cloneTimelapseTSIndex(index),
			expiresAt: now.Add(historicalDateCacheTTL),
		}
	}
	delete(tc.tsIndexLoads, key)
	tc.mu.Unlock()

	load.wg.Done()
	if err != nil {
		return timelapseTSIndex{}, err
	}
	return cloneTimelapseTSIndex(index), nil
}

func getTimelapseTSIndex(path string, maxSize int64) (timelapseTSIndex, error) {
	return screenTimelapseCache.getTSIndex(path, maxSize)
}

func timelapseSeekStartFromFrames(frames []timelapseFrame, filesize int64, targetPTS int64) (timelapseTSSeekStart, bool) {
	if len(frames) == 0 || filesize <= 0 {
		return timelapseTSSeekStart{}, false
	}

	pointIndex := sort.Search(len(frames), func(i int) bool {
		return frames[i].PTS > targetPTS
	}) - 1
	if pointIndex < 0 {
		pointIndex = 0
	}

	for pointIndex >= 0 {
		offset := frames[pointIndex].TSOffset
		if offset != nil && *offset > 0 && *offset < filesize {
			return timelapseTSSeekStart{Offset: *offset}, true
		}
		pointIndex--
	}
	return timelapseTSSeekStart{}, false
}

func findTimelapseTSSeekStartFromIndex(date string, filesize int64, targetPTS int64) (timelapseTSSeekStart, bool) {
	frames, err := getTimelapseFrames(date)
	if err != nil {
		return timelapseTSSeekStart{}, false
	}
	frames = filterTimelapseFramesByStableOffset(frames, filesize)
	return timelapseSeekStartFromFrames(frames, filesize, targetPTS)
}

func findTimelapseTSSeekStart(path string, maxSize int64, targetPTS int64) (timelapseTSSeekStart, error) {
	index, err := getTimelapseTSIndex(path, maxSize)
	if err != nil {
		return timelapseTSSeekStart{}, err
	}
	seekStart, ok := index.seekStart(targetPTS)
	if !ok {
		return timelapseTSSeekStart{}, errors.New("no video PES packet found")
	}
	return seekStart, nil
}

func findTimelapseTSStableEnd(path string, maxSize int64) (int64, error) {
	if stableOffset, ok := readTimelapseStableOffset(path, maxSize); ok {
		return stableOffset, nil
	}

	index, err := getTimelapseTSIndex(path, maxSize)
	if err != nil {
		return 0, err
	}
	if index.StableEnd <= 0 {
		return 0, errors.New("not enough stable video PES packets")
	}
	return index.StableEnd, nil
}

func filterTimelapseFramesByStableOffset(frames []timelapseFrame, stableOffset int64) []timelapseFrame {
	stableFrames := make([]timelapseFrame, 0, len(frames))
	for _, frame := range frames {
		if frame.TSOffset != nil && *frame.TSOffset >= 0 && *frame.TSOffset < stableOffset {
			stableFrames = append(stableFrames, frame)
		}
	}
	return stableFrames
}

func stableTimelapseFrames(date string, frames []timelapseFrame, now time.Time) []timelapseFrame {
	stableFrames := append([]timelapseFrame(nil), frames...)
	if !isCurrentTimelapseDate(date, now) {
		return stableFrames
	}

	videoPath, err := timelapseVideoPath(date)
	if err == nil {
		if stat, statErr := os.Stat(videoPath); statErr == nil && !stat.IsDir() {
			if stableOffset, ok := readTimelapseStableOffset(videoPath, stat.Size()); ok {
				return filterTimelapseFramesByStableOffset(stableFrames, stableOffset)
			}
		}
	}

	if len(stableFrames) <= 1 {
		return nil
	}
	return stableFrames[:len(stableFrames)-1]
}

func buildFramesResponse(date string, frames []timelapseFrame) proto.GetScreenTimelapseFramesRsp {
	rsp := proto.GetScreenTimelapseFramesRsp{
		Date:       date,
		FrameCount: len(frames),
		Frames:     make([]proto.ScreenTimelapseFrame, 0, len(frames)),
	}
	if len(frames) > 0 {
		last := frames[len(frames)-1]
		rsp.DurationMs = timelapsePTSMilliseconds(last.PTS + last.Duration)
	}
	for _, frame := range frames {
		rsp.Frames = append(rsp.Frames, proto.ScreenTimelapseFrame{
			PTS:         frame.PTS,
			CurrentTime: frame.CurrentTime,
			CapturedAt:  frame.CapturedAtRaw,
		})
	}
	return rsp
}

func buildDay(date string, frames []timelapseFrame, filesize int64, now time.Time) proto.ScreenTimelapseDay {
	day := proto.ScreenTimelapseDay{
		Date:       date,
		VideoURL:   "/api/ai/screen-timelapse/video?date=" + date,
		FrameCount: len(frames),
		Filesize:   filesize,
		Deletable:  isDeletableTimelapseDate(date, now),
	}
	if len(frames) == 0 {
		return day
	}
	first := frames[0]
	last := frames[len(frames)-1]
	day.StartTime = first.CapturedAtRaw
	day.EndTime = last.CapturedAtRaw
	day.DurationMs = timelapsePTSMilliseconds(last.PTS + last.Duration)
	day.Width = first.Width
	day.Height = first.Height
	return day
}

func timelapseDirectorySize(path string) (int64, error) {
	stat, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !stat.IsDir() {
		return 0, errors.New("timelapse day path is not a directory")
	}

	var size int64
	err = filepath.WalkDir(path, func(_ string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		size += info.Size()
		return nil
	})
	return size, err
}

func clampRecent(recent int) int {
	if recent <= 0 {
		return 30
	}
	if recent > 90 {
		return 90
	}
	return recent
}

func loadTimelapseDays(recent int) ([]proto.ScreenTimelapseDay, error) {
	recent = clampRecent(recent)
	root := timelapseRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var dates []string
	now := screenTimelapseCache.now()
	for _, entry := range entries {
		if entry.IsDir() && isPlayableTimelapseDate(entry.Name(), now) {
			dates = append(dates, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	if len(dates) > recent {
		dates = dates[:recent]
	}

	type dayResult struct {
		day proto.ScreenTimelapseDay
		ok  bool
	}
	results := make([]dayResult, len(dates))
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for index, date := range dates {
		index := index
		date := date
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			videoPath, _ := timelapseVideoPath(date)
			stat, statErr := os.Stat(videoPath)
			if statErr != nil || stat.IsDir() {
				return
			}
			frames, frameErr := getTimelapseFrames(date)
			frames = stableTimelapseFrames(date, frames, now)
			if frameErr != nil || len(frames) == 0 {
				return
			}
			filesize := stat.Size()
			if isCurrentTimelapseDate(date, now) {
				stableSize, sizeErr := findTimelapseTSStableEnd(videoPath, stat.Size())
				if sizeErr != nil || stableSize <= 0 {
					return
				}
				filesize = stableSize
			}
			day := buildDay(date, frames, filesize, now)
			results[index] = dayResult{day: day, ok: true}
		}()
	}
	wg.Wait()

	days := make([]proto.ScreenTimelapseDay, 0, len(dates))
	for _, result := range results {
		if result.ok {
			days = append(days, result.day)
		}
	}
	return days, nil
}

func getTimelapseDays(recent int) ([]proto.ScreenTimelapseDay, error) {
	return getTimelapseDaysWithCache(screenTimelapseCache, recent, loadTimelapseDays)
}

func getTimelapseDaysWithCache(cache *timelapseCache, recent int, loader func(int) ([]proto.ScreenTimelapseDay, error)) ([]proto.ScreenTimelapseDay, error) {
	recent = clampRecent(recent)
	for {
		if days, ok := cache.getDays(recent); ok {
			return days, nil
		}
		generation := cache.currentGeneration()
		days, err := loader(recent)
		if err != nil {
			return nil, err
		}
		if cache.setDaysAtGeneration(recent, days, generation) {
			return days, nil
		}
	}
}

func loadTimelapseFrames(date string) ([]timelapseFrame, error) {
	indexPath, err := timelapseIndexPath(date)
	if err != nil {
		return nil, err
	}
	frames, err := parseTimelapseIndex(indexPath)
	if err != nil {
		return nil, err
	}
	return frames, nil
}

func getTimelapseFrames(date string) ([]timelapseFrame, error) {
	return getTimelapseFramesWithCache(screenTimelapseCache, date, loadTimelapseFrames)
}

func getTimelapseFramesWithCache(cache *timelapseCache, date string, loader func(string) ([]timelapseFrame, error)) ([]timelapseFrame, error) {
	for {
		if frames, ok := cache.getFrames(date); ok {
			return frames, nil
		}
		generation := cache.currentGeneration()
		frames, err := loader(date)
		if err != nil {
			return nil, err
		}
		if cache.setFramesAtGeneration(date, frames, generation) {
			return frames, nil
		}
	}
}

func (s *Service) GetScreenTimelapseDays(c *gin.Context) {
	var req proto.GetScreenTimelapseDaysReq
	var rsp proto.Response
	if err := c.ShouldBindQuery(&req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if !requireScreenTimelapsePlaybackAuthorization(c, &rsp) {
		return
	}

	days, err := getTimelapseDays(req.Recent)
	if err != nil {
		rsp.ErrRsp(c, -2, "get screen timelapse days failed")
		return
	}
	rsp.OkRspWithData(c, proto.GetScreenTimelapseDaysRsp{Days: days})
}

func (s *Service) DeleteScreenTimelapseDay(c *gin.Context) {
	var req proto.DeleteScreenTimelapseDayReq
	var rsp proto.Response
	if err := proto.ParseQueryRequest(c, &req); err != nil || !isValidTimelapseDate(req.Date) {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if !requireScreenTimelapsePlaybackAuthorization(c, &rsp) {
		return
	}

	now := screenTimelapseCache.now()
	deletedBytes, err := deleteScreenTimelapseDay(timelapseRoot(), req.Date, now)
	if errors.Is(err, errTimelapseDayNotDeletable) {
		rsp.ErrRsp(c, -2, "screen timelapse day is not deletable")
		return
	}
	if errors.Is(err, errTimelapseDayBusy) {
		rsp.ErrRsp(c, -5, "screen timelapse day is busy")
		return
	}
	if err != nil {
		rsp.ErrRsp(c, -6, "delete screen timelapse day failed")
		return
	}

	rsp.OkRspWithData(c, proto.DeleteScreenTimelapseDayRsp{
		Date:         req.Date,
		DeletedBytes: deletedBytes,
	})
}

func (s *Service) GetScreenTimelapseFrames(c *gin.Context) {
	var req proto.GetScreenTimelapseDateReq
	var rsp proto.Response
	if err := proto.ParseQueryRequest(c, &req); err != nil || !isValidTimelapseDate(req.Date) {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if !requireScreenTimelapsePlaybackAuthorization(c, &rsp) {
		return
	}

	frames, err := getTimelapseFrames(req.Date)
	if err != nil {
		rsp.ErrRsp(c, -2, "get screen timelapse frames failed")
		return
	}
	frames = stableTimelapseFrames(req.Date, frames, screenTimelapseCache.now())
	rsp.OkRspWithData(c, buildFramesResponse(req.Date, frames))
}

func (s *Service) GetScreenTimelapseVideo(c *gin.Context) {
	if !screenTimelapsePlaybackAuthorized(c) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	date := c.Query("date")
	if !isValidTimelapseDate(date) {
		c.Status(http.StatusBadRequest)
		return
	}
	if !isPlayableTimelapseDate(date, screenTimelapseCache.now()) {
		c.Status(http.StatusConflict)
		return
	}
	releaseLease, ok := screenTimelapsePlaybackLeases.acquire(date)
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	defer releaseLease()

	videoPath, _ := timelapseVideoPath(date)
	stat, err := os.Stat(videoPath)
	if err != nil || stat.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}

	streamSize := stat.Size()
	isCurrentDate := isCurrentTimelapseDate(date, screenTimelapseCache.now())
	if isCurrentDate {
		stableSize, err := findTimelapseTSStableEnd(videoPath, stat.Size())
		if err != nil || stableSize <= 0 {
			c.Status(http.StatusConflict)
			return
		}
		streamSize = stableSize
	}

	c.Header("Content-Type", "video/mp2t")
	startRaw := strings.TrimSpace(c.Query("start"))
	if startRaw != "" {
		startSeconds, err := strconv.ParseFloat(startRaw, 64)
		if err != nil || startSeconds < 0 {
			c.Status(http.StatusBadRequest)
			return
		}
		if startSeconds > 0 {
			serveTimelapseVideoFromStart(c, date, videoPath, streamSize, timelapseSecondsToPTS(startSeconds))
			return
		}
	}

	// Even when the file currently ends at the stable offset, it may grow before
	// c.File stats it again. Keep current recordings bounded to this stable view.
	if isCurrentDate {
		serveTimelapseVideoPrefix(c, videoPath, streamSize)
		return
	}

	c.File(videoPath)
}

func serveTimelapseVideoPrefix(c *gin.Context, videoPath string, contentLength int64) {
	file, err := os.Open(videoPath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()

	c.Header("Content-Type", "video/mp2t")
	// Range requests must not read past the committed prefix while recording.
	http.ServeContent(c.Writer, c.Request, filepath.Base(videoPath), time.Time{},
		io.NewSectionReader(file, 0, contentLength))
}

func serveTimelapseVideoFromStart(c *gin.Context, date string, videoPath string, filesize int64, targetPTS int64) {
	seekStart, ok := findTimelapseTSSeekStartFromIndex(date, filesize, targetPTS)
	if !ok {
		serveTimelapseVideoPrefix(c, videoPath, filesize)
		return
	}
	if seekStart.Offset >= filesize {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	file, err := os.Open(videoPath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()

	c.Header("Content-Type", "video/mp2t")
	// Sidecar offsets already include the frame's TS headers. Expose this suffix
	// as a separate resource so resumed byte ranges are relative to its start.
	http.ServeContent(c.Writer, c.Request, filepath.Base(videoPath), time.Time{},
		io.NewSectionReader(file, seekStart.Offset, filesize-seekStart.Offset))
}

func (s *Service) SearchScreenTimelapse(c *gin.Context) {
	var req proto.SearchScreenTimelapseReq
	var rsp proto.Response
	if err := c.ShouldBindQuery(&req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if !requireScreenTimelapsePlaybackAuthorization(c, &rsp) {
		return
	}
	query, ok := normalizeSearchQuery(req.Query)
	if !ok {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	limit := normalizeSearchLimit(req.Limit)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	searchRsp, err := runMemfabSearch(ctx, query, limit)
	if err != nil {
		rsp.ErrRsp(c, -2, "search screen timelapse failed")
		return
	}

	results := make([]proto.ScreenTimelapseSearchResult, 0, len(searchRsp.Results))
	skippedMissingFileCount := searchRsp.SkippedMissingFileCount
	fileChecks := defaultSearchResultFileChecks()
	for _, raw := range searchRsp.Results {
		result, skipped := enrichSearchResult(raw, query, fileChecks)
		if skipped.missingFile {
			skippedMissingFileCount++
		}
		results = append(results, result)
	}

	rsp.OkRspWithData(c, proto.SearchScreenTimelapseRsp{
		Query:                   query,
		Count:                   len(results),
		Results:                 results,
		SkippedMissingFileCount: skippedMissingFileCount,
	})
}
