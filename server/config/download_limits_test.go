package config

import "testing"

func TestCheckDefaultValueNormalizesDownloadStorageLimits(t *testing.T) {
	original := instance
	t.Cleanup(func() { instance = original })

	cases := []struct {
		name     string
		max      uint64
		reserved uint64
		wantMax  uint64
		wantFree uint64
	}{
		{
			name:     "missing values use safe defaults",
			wantMax:  DefaultDownloadMaxImageBytes,
			wantFree: DefaultDownloadReservedFreeBytes,
		},
		{
			name:     "valid overrides are retained",
			max:      4 * 1024 * 1024,
			reserved: 1024 * 1024,
			wantMax:  4 * 1024 * 1024,
			wantFree: 1024 * 1024,
		},
		{
			name:     "zero values use safe defaults",
			wantMax:  DefaultDownloadMaxImageBytes,
			wantFree: DefaultDownloadReservedFreeBytes,
		},
		{
			name:     "zero max uses a safe default without changing a valid reserve",
			reserved: 1024 * 1024,
			wantMax:  DefaultDownloadMaxImageBytes,
			wantFree: 1024 * 1024,
		},
		{
			name:     "oversized reserve uses a safe default without changing a valid max",
			max:      4 * 1024 * 1024,
			reserved: maxDownloadStorageBytes + 1,
			wantMax:  4 * 1024 * 1024,
			wantFree: DefaultDownloadReservedFreeBytes,
		},
		{
			name:     "values outside signed comparison range use safe defaults",
			max:      maxDownloadStorageBytes + 1,
			reserved: maxDownloadStorageBytes + 1,
			wantMax:  DefaultDownloadMaxImageBytes,
			wantFree: DefaultDownloadReservedFreeBytes,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instance = Config{
				Security: Security{Download: DownloadSecurity{
					MaxImageBytes:     tc.max,
					ReservedFreeBytes: tc.reserved,
				}},
			}
			if err := checkDefaultValue(); err != nil {
				t.Fatalf("checkDefaultValue: %v", err)
			}

			if got := instance.Security.Download.MaxImageBytes; got != tc.wantMax {
				t.Fatalf("MaxImageBytes = %d, want %d", got, tc.wantMax)
			}
			if got := instance.Security.Download.ReservedFreeBytes; got != tc.wantFree {
				t.Fatalf("ReservedFreeBytes = %d, want %d", got, tc.wantFree)
			}
		})
	}
}
