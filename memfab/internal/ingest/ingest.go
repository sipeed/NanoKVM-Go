package ingest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"memfab/internal/capture"
	"memfab/internal/config"
	ocReset "memfab/internal/reset"
	"memfab/internal/store"
)

type StoredObservation struct {
	Path    string `json:"path"`
	Stem    string `json:"file_stem"`
	Indexed bool   `json:"indexed"`
}

func StoreObservation(db *store.DB, o capture.Observation) (StoredObservation, error) {
	if err := o.Normalize(time.Now); err != nil {
		return StoredObservation{}, err
	}
	path, err := capture.WriteAtomic(o)
	if err != nil {
		return StoredObservation{}, err
	}
	stem := capture.StemFromPath(path)
	if err := db.InsertObservation(stem, o); err != nil {
		return StoredObservation{}, err
	}
	return StoredObservation{Path: path, Stem: stem, Indexed: true}, nil
}

type response struct {
	OK     bool                `json:"ok"`
	Stored []StoredObservation `json:"stored,omitempty"`
	Reset  *ocReset.Stats      `json:"reset,omitempty"`
	Error  string              `json:"error,omitempty"`
}

func Handler(db *store.DB, cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	mux.HandleFunc("/v1/observations", observationHandler(db, cfg.Ingest))
	mux.HandleFunc("/v1/reset", resetHandler(db, cfg))
	return mux
}

func observationHandler(db *store.DB, cfg config.IngestConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{OK: false, Error: "method not allowed"})
			return
		}
		if !authorized(w, r, cfg) {
			return
		}
		max := cfg.MaxBodyBytes
		if max <= 0 {
			max = 1 << 20
		}
		r.Body = http.MaxBytesReader(w, r.Body, max)
		defer r.Body.Close()

		var raw json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			writeJSON(w, http.StatusBadRequest, response{OK: false, Error: err.Error()})
			return
		}

		var observations []capture.Observation
		if len(raw) > 0 && raw[0] == '[' {
			if err := json.Unmarshal(raw, &observations); err != nil {
				writeJSON(w, http.StatusBadRequest, response{OK: false, Error: err.Error()})
				return
			}
		} else {
			var o capture.Observation
			if err := json.Unmarshal(raw, &o); err != nil {
				writeJSON(w, http.StatusBadRequest, response{OK: false, Error: err.Error()})
				return
			}
			observations = []capture.Observation{o}
		}

		maxBatch := cfg.MaxBatchSize
		if maxBatch <= 0 {
			maxBatch = 16
		}
		if len(observations) == 0 || len(observations) > maxBatch {
			writeJSON(w, http.StatusBadRequest, response{OK: false, Error: fmt.Sprintf("batch size must be 1..%d", maxBatch)})
			return
		}

		stored := make([]StoredObservation, 0, len(observations))
		for _, o := range observations {
			out, err := StoreObservation(db, o)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, response{OK: false, Error: err.Error()})
				return
			}
			stored = append(stored, out)
		}
		writeJSON(w, http.StatusOK, response{OK: true, Stored: stored})
	}
}

func resetHandler(db *store.DB, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{OK: false, Error: "method not allowed"})
			return
		}
		if !authorized(w, r, cfg.Ingest) {
			return
		}
		stats, err := ocReset.Run(db)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, response{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, response{OK: true, Reset: &stats})
	}
}

func authorized(w http.ResponseWriter, r *http.Request, cfg config.IngestConfig) bool {
	if !cfg.RequireToken {
		return true
	}
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if got == "" || got != cfg.Token {
		writeJSON(w, http.StatusUnauthorized, response{OK: false, Error: "unauthorized"})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
