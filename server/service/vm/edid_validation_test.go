package vm

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateEdidAcceptsBuiltinFiles(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{name: "nanokvm-go-1920x1080-60.bin", width: 1920, height: 1080},
		{name: "nanokvm-go-2560x1440-60.bin", width: 2560, height: 1440},
		{name: "nanokvm-go-3440x1440-60.bin", width: 3440, height: 1440},
		{name: "nanokvm-go-3840x2160-30.bin", width: 3840, height: 2160},
		{name: "nanokvm-go-3840x2160-50.bin", width: 3840, height: 2160},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(
				"../../../debhelper/packages/kvmcomm/rootfs/kvmcomm/edid",
				tt.name,
			)
			data, err := readEdidFile(path)
			if err != nil {
				t.Fatalf("readEdidFile(%q) failed: %v", tt.name, err)
			}
			if len(data) < edidBlockSize || len(data) > edidMaxSize {
				t.Fatalf("readEdidFile(%q) returned %d bytes", tt.name, len(data))
			}
			width, height, ok := preferredEdidResolution(data)
			if !ok || width != tt.width || height != tt.height {
				t.Fatalf(
					"preferred resolution = %dx%d, %t; want %dx%d",
					width,
					height,
					ok,
					tt.width,
					tt.height,
				)
			}
		})
	}
}

func TestResolutionFromEdidName(t *testing.T) {
	width, height, ok := resolutionFromEdidName("nanokvm-go-3440x1440-60")
	if !ok || width != 3440 || height != 1440 {
		t.Fatalf("resolution = %dx%d, %t", width, height, ok)
	}
	if _, _, ok := resolutionFromEdidName("custom-display.bin"); ok {
		t.Fatal("unexpected resolution parsed from custom name")
	}
}

func TestValidateEdidRejectsInvalidLengths(t *testing.T) {
	tests := []struct {
		length int
		reason edidErrorReason
	}{
		{length: 0, reason: edidReasonInvalidLength},
		{length: 127, reason: edidReasonInvalidLength},
		{length: 129, reason: edidReasonInvalidLength},
		{length: 513, reason: edidReasonTooLarge},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d_bytes", tt.length), func(t *testing.T) {
			err := validateEdid(make([]byte, tt.length))
			assertEdidValidationReason(t, err, tt.reason)
		})
	}
}

func TestValidateEdidAcceptsOneToFourBlocks(t *testing.T) {
	for blockCount := 1; blockCount <= 4; blockCount++ {
		t.Run(fmt.Sprintf("%d_bytes", blockCount*edidBlockSize), func(t *testing.T) {
			if err := validateEdid(makeValidationTestEdid(blockCount, 0xf0)); err != nil {
				t.Fatalf("validateEdid rejected %d blocks: %v", blockCount, err)
			}
		})
	}
}

func TestValidateEdidRejectsInvalidHeader(t *testing.T) {
	data := makeValidationTestEdid(1, 0)
	data[0] = 0x01
	updateEdidBlockChecksum(data)

	assertEdidValidationReason(t, validateEdid(data), edidReasonInvalidHeader)
}

func TestValidateEdidRejectsUnsupportedVersion(t *testing.T) {
	tests := []struct {
		name     string
		major    byte
		revision byte
	}{
		{name: "major_2", major: 2, revision: 0},
		{name: "revision_5", major: 1, revision: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := makeValidationTestEdid(1, 0)
			data[18] = tt.major
			data[19] = tt.revision
			updateEdidBlockChecksum(data)

			assertEdidValidationReason(t, validateEdid(data), edidReasonUnsupportedVersion)
		})
	}
}

func TestValidateEdidAcceptsVersionsOnePointZeroThroughFour(t *testing.T) {
	for revision := byte(0); revision <= 4; revision++ {
		t.Run(fmt.Sprintf("1.%d", revision), func(t *testing.T) {
			data := makeValidationTestEdid(1, 0)
			data[19] = revision
			updateEdidBlockChecksum(data)

			if err := validateEdid(data); err != nil {
				t.Fatalf("validateEdid rejected EDID 1.%d: %v", revision, err)
			}
		})
	}
}

func TestValidateEdidRejectsExtensionCountMismatch(t *testing.T) {
	data := makeValidationTestEdid(1, 0)
	data[126] = 1
	updateEdidBlockChecksum(data)

	assertEdidValidationReason(t, validateEdid(data), edidReasonExtensionCountMismatch)
}

func TestValidateEdidRejectsChecksumFailureInEveryBlock(t *testing.T) {
	for blockIndex := 0; blockIndex < 4; blockIndex++ {
		t.Run(fmt.Sprintf("block_%d", blockIndex), func(t *testing.T) {
			data := makeValidationTestEdid(4, 0xf0)
			data[blockIndex*edidBlockSize+20]++

			assertEdidValidationReason(t, validateEdid(data), edidReasonInvalidChecksum)
		})
	}
}

func TestValidateEdidRejectsInvalidCTADTDOffset(t *testing.T) {
	data := makeValidationTestEdid(2, 0x02)
	cta := data[edidBlockSize:]
	cta[2] = 255
	updateEdidBlockChecksum(cta)

	assertEdidValidationReason(t, validateEdid(data), edidReasonInvalidCTAExtension)
}

func TestValidateEdidRejectsCTADataBlockCrossingDTDBoundary(t *testing.T) {
	data := makeValidationTestEdid(2, 0x02)
	cta := data[edidBlockSize:]
	cta[2] = 5
	cta[4] = 0x01 // One-byte payload ends at offset 6, beyond the DTD boundary.
	updateEdidBlockChecksum(cta)

	assertEdidValidationReason(t, validateEdid(data), edidReasonInvalidCTAExtension)
}

func TestValidateEdidRejectsCTAExtendedBlockWithoutTag(t *testing.T) {
	data := makeValidationTestEdid(2, 0x02)
	cta := data[edidBlockSize:]
	cta[2] = 5
	cta[4] = 0xe0 // Extended Data Block with a zero-length payload.
	updateEdidBlockChecksum(cta)

	assertEdidValidationReason(t, validateEdid(data), edidReasonInvalidCTAExtension)
}

func TestValidateEdidAcceptsCTAOffsetBoundaries(t *testing.T) {
	for _, offset := range []byte{0, 4, 127} {
		t.Run(fmt.Sprintf("offset_%d", offset), func(t *testing.T) {
			data := makeValidationTestEdid(2, 0x02)
			cta := data[edidBlockSize:]
			cta[2] = offset
			updateEdidBlockChecksum(cta)

			if err := validateEdid(data); err != nil {
				t.Fatalf("validateEdid rejected CTA DTD offset %d: %v", offset, err)
			}
		})
	}
}

func TestValidateEdidAcceptsUnknownChecksumValidExtension(t *testing.T) {
	data := makeValidationTestEdid(2, 0xf0)

	if err := validateEdid(data); err != nil {
		t.Fatalf("validateEdid rejected unknown extension: %v", err)
	}
}

func TestReadEdidLimitsInputAndValidatesContent(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		want := makeValidationTestEdid(4, 0xf0)
		got, err := readEdid(bytes.NewReader(want))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatal("readEdid returned different data")
		}
	})

	t.Run("too large", func(t *testing.T) {
		reader := &countingValidationTestReader{data: make([]byte, 1024)}
		_, err := readEdid(reader)
		assertEdidValidationReason(t, err, edidReasonTooLarge)
		if reader.read != edidReadLimit {
			t.Fatalf("readEdid consumed %d bytes, want %d", reader.read, edidReadLimit)
		}
	})

	t.Run("invalid header", func(t *testing.T) {
		_, err := readEdid(bytes.NewReader(make([]byte, edidBlockSize)))
		assertEdidValidationReason(t, err, edidReasonInvalidHeader)
	})
}

func TestReadEdidFileLimitsAndValidatesContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.bin")
	if err := os.WriteFile(path, make([]byte, edidReadLimit), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readEdidFile(path)
	assertEdidValidationReason(t, err, edidReasonTooLarge)
}

func TestEdidReasonTraversesWrappedErrors(t *testing.T) {
	err := fmt.Errorf("prepare EDID: %w", newEdidError(edidReasonInvalidHeader, "bad header"))
	if got := edidReason(err); got != edidReasonInvalidHeader {
		t.Fatalf("edidReason() = %q, want %q", got, edidReasonInvalidHeader)
	}
	if got := edidReason(errors.New("plain error")); got != "" {
		t.Fatalf("edidReason() = %q for plain error, want empty", got)
	}
}

func makeValidationTestEdid(blockCount int, extensionTag byte) []byte {
	data := make([]byte, blockCount*edidBlockSize)
	copy(data, []byte{0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00})
	data[18] = 1
	data[19] = 4
	data[126] = byte(blockCount - 1)
	updateEdidBlockChecksum(data[:edidBlockSize])

	for blockIndex := 1; blockIndex < blockCount; blockIndex++ {
		block := data[blockIndex*edidBlockSize : (blockIndex+1)*edidBlockSize]
		block[0] = extensionTag
		updateEdidBlockChecksum(block)
	}
	return data
}

func assertEdidValidationReason(t *testing.T, err error, want edidErrorReason) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %q error", want)
	}
	if got := edidReason(err); got != want {
		t.Fatalf("edidReason(%v) = %q, want %q", err, got, want)
	}
}

type countingValidationTestReader struct {
	data []byte
	read int
}

func (r *countingValidationTestReader) Read(p []byte) (int, error) {
	if r.read == len(r.data) {
		return 0, errors.New("unexpected read past available test data")
	}
	n := copy(p, r.data[r.read:])
	r.read += n
	return n, nil
}
