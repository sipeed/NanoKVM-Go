package config

const (
	DefaultDownloadMaxImageBytes     uint64 = 16 * 1024 * 1024 * 1024
	DefaultDownloadReservedFreeBytes uint64 = 256 * 1024 * 1024

	// Download sizes are compared with http.Response.ContentLength and filesystem
	// budgets, both of which use int64. Values outside that range must never turn
	// the download limit into an effectively unbounded value.
	maxDownloadStorageBytes uint64 = 1<<63 - 1
)

func normalizeDownloadStorageBytes(value, fallback uint64) uint64 {
	if value == 0 || value > maxDownloadStorageBytes {
		return fallback
	}
	return value
}
