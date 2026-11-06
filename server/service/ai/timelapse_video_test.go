package ai

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

const timelapseVideoTestDate = "2026-06-01"

func timelapseVideoFixture(t *testing.T) (string, []byte) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	oldCache := screenTimelapseCache
	screenTimelapseCache = newTimelapseCache()
	t.Cleanup(func() { screenTimelapseCache = oldCache })

	data := make([]byte, 6*mpegTSPacketSize)
	for i := range data {
		data[i] = byte(i*37 + i/mpegTSPacketSize*17)
	}
	videoPath := filepath.Join(t.TempDir(), timelapseVideoTestDate+".ts")
	if err := os.WriteFile(videoPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	var frames []timelapseFrame
	for i := 0; i < 6; i++ {
		frames = append(frames, timelapseFrame{
			PTS:      int64(i * 3000),
			TSOffset: ptrInt64(int64(i * mpegTSPacketSize)),
		})
	}
	screenTimelapseCache.setFrames(timelapseVideoTestDate, frames)
	return videoPath, data
}

func TestTimelapseVideoRanges(t *testing.T) {
	videoPath, data := timelapseVideoFixture(t)
	screenTimelapseCache.setFrames("2026-06-02", nil)
	stableSize := int64(4 * mpegTSPacketSize)

	for _, source := range []struct {
		name  string
		serve gin.HandlerFunc
		want  []byte
	}{
		{
			name: "stable prefix",
			serve: func(c *gin.Context) {
				serveTimelapseVideoPrefix(c, videoPath, stableSize)
			},
			want: data[:stableSize],
		},
		{
			name: "from indexed frame",
			serve: func(c *gin.Context) {
				serveTimelapseVideoFromStart(c, timelapseVideoTestDate, videoPath, int64(len(data)), 3000)
			},
			want: data[mpegTSPacketSize:],
		},
		{
			name: "from indexed frame within stable prefix",
			serve: func(c *gin.Context) {
				serveTimelapseVideoFromStart(c, timelapseVideoTestDate, videoPath, stableSize, 3000)
			},
			want: data[mpegTSPacketSize:stableSize],
		},
		{
			name: "first frame falls back to prefix",
			serve: func(c *gin.Context) {
				serveTimelapseVideoFromStart(c, timelapseVideoTestDate, videoPath, stableSize, 1)
			},
			want: data[:stableSize],
		},
		{
			name: "missing index falls back to prefix",
			serve: func(c *gin.Context) {
				serveTimelapseVideoFromStart(c, "2026-06-02", videoPath, stableSize, 3000)
			},
			want: data[:stableSize],
		},
	} {
		t.Run(source.name, func(t *testing.T) {
			size := len(source.want)
			for _, request := range []struct {
				name       string
				byteRange  string
				status     int
				start, end int
			}{
				{name: "whole resource", status: http.StatusOK, end: size},
				{name: "first packet", byteRange: "bytes=0-187", status: http.StatusPartialContent, end: 188},
				{name: "resume inside packet", byteRange: "bytes=189-", status: http.StatusPartialContent, start: 189, end: size},
				{name: "across packet boundary", byteRange: "bytes=175-200", status: http.StatusPartialContent, start: 175, end: 201},
				{name: "suffix", byteRange: "bytes=-100", status: http.StatusPartialContent, start: size - 100, end: size},
				{name: "clamp to resource end", byteRange: "bytes=188-9999999", status: http.StatusPartialContent, start: 188, end: size},
				{name: "past resource end", byteRange: fmt.Sprintf("bytes=%d-", size), status: http.StatusRequestedRangeNotSatisfiable},
			} {
				t.Run(request.name, func(t *testing.T) {
					c, recorder := newTestContext(http.MethodGet, "/video", "")
					if request.byteRange != "" {
						c.Request.Header.Set("Range", request.byteRange)
					}
					source.serve(c)
					if recorder.Code != request.status {
						t.Fatalf("status = %d, want %d", recorder.Code, request.status)
					}
					wantRange := ""
					if request.status == http.StatusPartialContent {
						wantRange = fmt.Sprintf("bytes %d-%d/%d", request.start, request.end-1, size)
					} else if request.status == http.StatusRequestedRangeNotSatisfiable {
						wantRange = fmt.Sprintf("bytes */%d", size)
					}
					if got := recorder.Header().Get("Content-Range"); got != wantRange {
						t.Fatalf("Content-Range = %q, want %q", got, wantRange)
					}
					if request.status == http.StatusRequestedRangeNotSatisfiable {
						return
					}
					for name, want := range map[string]string{
						"Accept-Ranges":  "bytes",
						"Content-Type":   "video/mp2t",
						"Content-Length": strconv.Itoa(request.end - request.start),
					} {
						if got := recorder.Header().Get(name); got != want {
							t.Errorf("%s = %q, want %q", name, got, want)
						}
					}
					if !bytes.Equal(recorder.Body.Bytes(), source.want[request.start:request.end]) {
						t.Fatal("response does not match the requested bytes of the playable resource")
					}
				})
			}
		})
	}
}

func TestTimelapseVideoResumeAfterRecordingGrows(t *testing.T) {
	for _, fromMiddle := range []bool{false, true} {
		t.Run(fmt.Sprintf("fromMiddle=%t", fromMiddle), func(t *testing.T) {
			videoPath, data := timelapseVideoFixture(t)
			serve := func(c *gin.Context, stableSize int64) {
				if fromMiddle {
					serveTimelapseVideoFromStart(c, timelapseVideoTestDate, videoPath, stableSize, 3000)
				} else {
					serveTimelapseVideoPrefix(c, videoPath, stableSize)
				}
			}

			c, first := newTestContext(http.MethodGet, "/video", "")
			c.Request.Header.Set("Range", "bytes=0-199")
			serve(c, 3*mpegTSPacketSize)
			if first.Code != http.StatusPartialContent || first.Body.Len() != 200 {
				t.Fatalf("initial range: status=%d bytes=%d", first.Code, first.Body.Len())
			}

			file, err := os.OpenFile(videoPath, os.O_APPEND|os.O_WRONLY, 0)
			if err != nil {
				t.Fatal(err)
			}
			_, writeErr := file.Write(bytes.Repeat([]byte{0xff}, mpegTSPacketSize))
			closeErr := file.Close()
			if writeErr != nil || closeErr != nil {
				t.Fatalf("append recording: write=%v close=%v", writeErr, closeErr)
			}

			c, resumed := newTestContext(http.MethodGet, "/video", "")
			c.Request.Header.Set("Range", "bytes=200-")
			serve(c, 5*mpegTSPacketSize)
			if resumed.Code != http.StatusPartialContent {
				t.Fatalf("resume status = %d", resumed.Code)
			}
			start := 0
			if fromMiddle {
				start = mpegTSPacketSize
			}
			combined := append(first.Body.Bytes(), resumed.Body.Bytes()...)
			if !bytes.Equal(combined, data[start:5*mpegTSPacketSize]) {
				t.Fatal("resuming duplicated or skipped bytes, or included the uncommitted tail")
			}
		})
	}
}
