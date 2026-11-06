package capture

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"memfab/internal/paths"
)

const SchemaVersion = 1
const ObservationType = "ocr_observation"

type Source struct {
	Kind          string `json:"kind,omitempty"`
	Device        string `json:"device,omitempty"`
	Engine        string `json:"engine,omitempty"`
	InputTextFile string `json:"input_text_file,omitempty"`
}

type Screen struct {
	Width       *int   `json:"width,omitempty"`
	Height      *int   `json:"height,omitempty"`
	ImageSHA256 string `json:"image_sha256,omitempty"`
}

type Context struct {
	AppName string `json:"app_name,omitempty"`
	Title   string `json:"title,omitempty"`
	URL     string `json:"url,omitempty"`
}

type Event struct {
	Type   string `json:"type,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type OCRBlock struct {
	ID         int     `json:"id"`
	Text       string  `json:"text"`
	BBox       []int   `json:"bbox,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

type OCR struct {
	Language   string     `json:"language,omitempty"`
	Text       string     `json:"text,omitempty"`
	RawText    string     `json:"raw_text,omitempty"`
	Confidence *float64   `json:"confidence,omitempty"`
	BlockCount int        `json:"block_count,omitempty"`
	Blocks     []OCRBlock `json:"blocks,omitempty"`
}

type Observation struct {
	SchemaVersion int      `json:"schema_version"`
	Type          string   `json:"type"`
	CapturedAt    string   `json:"captured_at"`
	CaptureID     string   `json:"capture_id"`
	Source        Source   `json:"source,omitempty"`
	Screen        Screen   `json:"screen,omitempty"`
	Context       Context  `json:"context,omitempty"`
	Event         Event    `json:"event,omitempty"`
	OCR           OCR      `json:"ocr"`
	Tags          []string `json:"tags,omitempty"`
}

func (o *Observation) Normalize(now func() time.Time) error {
	if now == nil {
		now = time.Now
	}
	if o.SchemaVersion == 0 {
		o.SchemaVersion = SchemaVersion
	}
	if o.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema_version %d", o.SchemaVersion)
	}
	if strings.TrimSpace(o.Type) == "" {
		o.Type = ObservationType
	}
	if o.Type != ObservationType {
		return fmt.Errorf("unsupported observation type %q", o.Type)
	}
	o.CaptureID = strings.TrimSpace(o.CaptureID)
	if o.CaptureID == "" {
		return errors.New("capture_id is required")
	}
	if strings.TrimSpace(o.CapturedAt) == "" {
		o.CapturedAt = now().Local().Format(time.RFC3339)
	}
	if _, err := ParseTimestamp(o.CapturedAt); err != nil {
		return fmt.Errorf("invalid captured_at: %w", err)
	}
	if o.Source.Kind == "" {
		o.Source.Kind = "screen_ocr"
	}
	if o.Event.Type == "" {
		o.Event.Type = "ocr_capture"
	}
	if o.OCR.Text == "" && len(o.OCR.Blocks) > 0 {
		parts := make([]string, 0, len(o.OCR.Blocks))
		for _, b := range o.OCR.Blocks {
			if strings.TrimSpace(b.Text) != "" {
				parts = append(parts, b.Text)
			}
		}
		o.OCR.Text = strings.Join(parts, "\n")
	}
	if o.OCR.BlockCount == 0 && len(o.OCR.Blocks) > 0 {
		o.OCR.BlockCount = len(o.OCR.Blocks)
	}
	if strings.TrimSpace(o.OCR.Text) == "" {
		return errors.New("ocr.text is required")
	}
	return nil
}

func ParseTimestamp(ts string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, ts)
}

func SafeFilename(ts string) (string, error) {
	if _, err := ParseTimestamp(ts); err != nil {
		return "", err
	}
	stem := strings.ReplaceAll(ts, ":", "-")
	stem = strings.ReplaceAll(stem, "+", "p")
	stem = strings.ReplaceAll(stem, ".", "_")
	if idx := strings.LastIndex(stem, "-"); idx > len("2006-01-02T15-04-05") {
		stem = stem[:idx] + "m" + stem[idx+1:]
	}
	return stem, nil
}

func ParseStem(stem string) (time.Time, error) {
	candidate := strings.ReplaceAll(stem, "_", ".")
	if strings.HasSuffix(candidate, "Z") {
		return ParseTimestamp(strings.ReplaceAll(strings.TrimSuffix(candidate, "Z"), "-", ":") + "Z")
	}
	idx := strings.LastIndex(candidate, "p")
	sign := "+"
	if idx < 0 {
		idx = strings.LastIndex(candidate, "m")
		sign = "-"
	}
	if idx < 0 {
		return time.Time{}, errors.New("malformed observation stem")
	}
	datePart := candidate[:idx]
	if len(datePart) >= len("2006-01-02T15-04-05") {
		datePart = datePart[:13] + ":" + datePart[14:16] + ":" + datePart[17:]
	}
	zonePart := strings.ReplaceAll(candidate[idx+1:], "-", ":")
	return ParseTimestamp(datePart + sign + zonePart)
}

func ObservationPath(o Observation) (string, error) {
	stem, err := SafeFilename(o.CapturedAt)
	if err != nil {
		return "", err
	}
	return filepath.Join(paths.ObservationBufferDir(), stem+".json"), nil
}

func WriteAtomic(o Observation) (string, error) {
	if err := paths.EnsureObservationBufferDir(); err != nil {
		return "", err
	}
	path, err := ObservationPath(o)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), paths.DirectoryMode); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".observation-*.tmp")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	enc := json.NewEncoder(tmp)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(o); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return "", err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return "", err
	}
	return path, nil
}

func ReadFile(path string) (Observation, error) {
	var o Observation
	data, err := os.ReadFile(path)
	if err != nil {
		return o, err
	}
	if err := json.Unmarshal(data, &o); err != nil {
		return o, err
	}
	return o, o.Normalize(time.Now)
}

func StemFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
