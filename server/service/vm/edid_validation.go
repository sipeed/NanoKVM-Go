package vm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
)

const (
	edidMaxSize   = 4 * edidBlockSize
	edidReadLimit = edidMaxSize + 1
)

type edidErrorReason string

const (
	edidReasonBadRequest                 edidErrorReason = "bad_request"
	edidReasonInvalidFileName            edidErrorReason = "invalid_file_name"
	edidReasonTooLarge                   edidErrorReason = "edid_too_large"
	edidReasonInvalidLength              edidErrorReason = "invalid_edid_length"
	edidReasonInvalidHeader              edidErrorReason = "invalid_edid_header"
	edidReasonUnsupportedVersion         edidErrorReason = "unsupported_edid_version"
	edidReasonExtensionCountMismatch     edidErrorReason = "edid_extension_count_mismatch"
	edidReasonInvalidChecksum            edidErrorReason = "invalid_edid_checksum"
	edidReasonInvalidCTAExtension        edidErrorReason = "invalid_cta_extension"
	edidReasonCurrentCannotBeOverwritten edidErrorReason = "current_edid_cannot_be_overwritten"
	edidReasonSaveFailed                 edidErrorReason = "save_failed"
)

var edidHeader = []byte{0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00}
var edidResolutionPattern = regexp.MustCompile(`(?:^|[^0-9])(\d{3,5})x(\d{3,5})(?:[^0-9]|$)`)

type edidValidationError struct {
	reason edidErrorReason
	detail string
}

func (e *edidValidationError) Error() string {
	return e.detail
}

func newEdidError(reason edidErrorReason, format string, args ...any) error {
	return &edidValidationError{
		reason: reason,
		detail: fmt.Sprintf(format, args...),
	}
}

func edidReason(err error) edidErrorReason {
	var validationErr *edidValidationError
	if errors.As(err, &validationErr) {
		return validationErr.reason
	}
	return ""
}

// readEdid deliberately reads one byte beyond the supported maximum so an
// oversized stream is detected without allowing an unbounded allocation.
func readEdid(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, edidReadLimit))
	if err != nil {
		return nil, err
	}
	if err := validateEdid(data); err != nil {
		return nil, err
	}
	return data, nil
}

func readEdidFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("EDID source is not a regular file: %s", path)
	}

	return readEdid(file)
}

func validateEdid(data []byte) error {
	if len(data) > edidMaxSize {
		return newEdidError(edidReasonTooLarge, "EDID is larger than %d bytes", edidMaxSize)
	}
	if len(data) < edidBlockSize || len(data)%edidBlockSize != 0 {
		return newEdidError(
			edidReasonInvalidLength,
			"EDID length %d is not one to four %d-byte blocks",
			len(data),
			edidBlockSize,
		)
	}
	if !bytes.Equal(data[:len(edidHeader)], edidHeader) {
		return newEdidError(edidReasonInvalidHeader, "invalid EDID header")
	}
	if data[18] != 1 || data[19] > 4 {
		return newEdidError(
			edidReasonUnsupportedVersion,
			"unsupported EDID version %d.%d",
			data[18],
			data[19],
		)
	}

	blockCount := len(data) / edidBlockSize
	if int(data[126])+1 != blockCount {
		return newEdidError(
			edidReasonExtensionCountMismatch,
			"EDID extension count %d does not match %d blocks",
			data[126],
			blockCount,
		)
	}

	for blockIndex := 0; blockIndex < blockCount; blockIndex++ {
		start := blockIndex * edidBlockSize
		var sum byte
		for _, value := range data[start : start+edidBlockSize] {
			sum += value
		}
		if sum != 0 {
			return newEdidError(
				edidReasonInvalidChecksum,
				"invalid EDID checksum in block %d",
				blockIndex,
			)
		}
	}

	for blockIndex := 1; blockIndex < blockCount; blockIndex++ {
		block := data[blockIndex*edidBlockSize : (blockIndex+1)*edidBlockSize]
		if block[0] != 0x02 {
			continue
		}
		if err := validateCTAExtension(block, blockIndex); err != nil {
			return err
		}
	}

	return nil
}

func preferredEdidResolution(data []byte) (int, int, bool) {
	if len(data) < edidBlockSize {
		return 0, 0, false
	}

	for offset := 54; offset+18 <= 126; offset += 18 {
		if width, height, ok := detailedTimingResolution(data[offset : offset+18]); ok {
			return width, height, true
		}
	}

	for blockIndex := 1; blockIndex*edidBlockSize < len(data); blockIndex++ {
		block := data[blockIndex*edidBlockSize : (blockIndex+1)*edidBlockSize]
		if block[0] != 0x02 {
			continue
		}
		offset := int(block[2])
		if offset < 4 {
			continue
		}
		for ; offset+18 <= 127; offset += 18 {
			if width, height, ok := detailedTimingResolution(block[offset : offset+18]); ok {
				return width, height, true
			}
		}
	}

	return 0, 0, false
}

func detailedTimingResolution(descriptor []byte) (int, int, bool) {
	if len(descriptor) < 18 || descriptor[0] == 0 && descriptor[1] == 0 {
		return 0, 0, false
	}
	width := int(descriptor[2]) | int(descriptor[4]&0xf0)<<4
	height := int(descriptor[5]) | int(descriptor[7]&0xf0)<<4
	return width, height, width > 0 && height > 0
}

func resolutionFromEdidName(name string) (int, int, bool) {
	matches := edidResolutionPattern.FindStringSubmatch(name)
	if len(matches) != 3 {
		return 0, 0, false
	}
	width, widthErr := strconv.Atoi(matches[1])
	height, heightErr := strconv.Atoi(matches[2])
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func validateCTAExtension(block []byte, blockIndex int) error {
	dtdOffset := int(block[2])
	if dtdOffset == 0 {
		return nil
	}
	if dtdOffset < 4 || dtdOffset > edidBlockSize-1 {
		return newEdidError(
			edidReasonInvalidCTAExtension,
			"invalid CTA DTD offset %d in block %d",
			dtdOffset,
			blockIndex,
		)
	}

	for offset := 4; offset < dtdOffset; {
		header := block[offset]
		length := int(header & 0x1f)
		end := offset + 1 + length
		if end > dtdOffset {
			return newEdidError(
				edidReasonInvalidCTAExtension,
				"CTA data block at offset %d crosses DTD boundary %d in block %d",
				offset,
				dtdOffset,
				blockIndex,
			)
		}
		if header>>5 == 7 && length == 0 {
			return newEdidError(
				edidReasonInvalidCTAExtension,
				"CTA extended data block at offset %d has no extended tag in block %d",
				offset,
				blockIndex,
			)
		}
		offset = end
	}

	return nil
}
