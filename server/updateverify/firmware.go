package updateverify

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/blake2b"
)

const (
	firmwareMetadataMaxBytes       = int64(64 << 10)
	firmwareArchiveMaxEntries      = int64(4096)
	firmwareArchiveMaxFileBytes    = int64(256 << 20)
	firmwareArchiveMaxExpanded     = int64(512 << 20)
	firmwareArchiveMaxEnvelope     = firmwareArchiveMaxExpanded + int64(8<<20)
	firmwareArchiveMaxPathBytes    = 256
	firmwareArchiveMaxComponent    = 255
	firmwareVersionFileMaxBytes    = int64(4096)
	firmwareB2SumFileMaxBytes      = int64(1 << 20)
	firmwareTarBlockBytes          = 512
	firmwareVersionPath            = "overlay/boot/ver"
	firmwareB2SumPath              = "b2sum.txt"
	firmwareSignatureAlgorithm     = "ed25519"
	firmwareArchiveGzipMagicFirst  = byte(0x1f)
	firmwareArchiveGzipMagicSecond = byte(0x8b)
)

// FirmwarePolicy binds unsigned discovery information to the signed Firmware
// archive. Downgrades are intentionally allowed; only an ordinary same-version
// installation is rejected unless the caller explicitly opts in.
type FirmwarePolicy struct {
	RequestedVersion string
	CurrentVersion   string
	AllowSameVersion bool
}

// FirmwareMetadata is the strict external metadata document for one final
// Firmware archive. Signature authenticates the exact compressed archive bytes,
// not this JSON document.
type FirmwareMetadata struct {
	Version string `json:"version"`
	Name    string `json:"name"`
	SHA512  string `json:"sha512"`
	Size    int64  `json:"size"`
	// UpdateUboot is an unsigned external policy hint. Nil means false.
	UpdateUboot *bool      `json:"updateUboot,omitempty"`
	Signature   *Signature `json:"signature"`
}

// VerifiedFirmwareArtifact contains the authenticated archive identity plus the
// strictly typed external update policy. UpdateUboot is deliberately unsigned:
// the existing signature authenticates the archive bytes, not the metadata.
type VerifiedFirmwareArtifact struct {
	Version     string
	Name        string
	Size        int64
	SHA512      string
	KeyID       string
	UpdateUboot bool
}

type firmwareMemberKind uint8

const (
	firmwareMemberDirectory firmwareMemberKind = iota + 1
	firmwareMemberFile
)

type firmwareArchiveLimits struct {
	maxEntries  int64
	maxFile     int64
	maxExpanded int64
	maxEnvelope int64
}

var defaultFirmwareArchiveLimits = firmwareArchiveLimits{
	maxEntries:  firmwareArchiveMaxEntries,
	maxFile:     firmwareArchiveMaxFileBytes,
	maxExpanded: firmwareArchiveMaxExpanded,
	maxEnvelope: firmwareArchiveMaxEnvelope,
}

type firmwareVersionToken struct {
	prefix  byte
	version string
}

// VerifyFirmwareArtifact authenticates and structurally validates one gzip
// Firmware archive. The current device protocol deliberately does not accept
// xz, PAX, GNU long-name records, multiple gzip members, or unsigned fallback.
func VerifyFirmwareArtifact(
	archivePath, metadataPath string,
	keys KeyRing,
	policy FirmwarePolicy,
) (VerifiedFirmwareArtifact, error) {
	initialArchiveIdentity, err := requireRegularPath(archivePath, "Firmware archive")
	if err != nil {
		return VerifiedFirmwareArtifact{}, err
	}
	rawMetadata, err := readBoundedRegularFile(
		metadataPath,
		"Firmware metadata",
		firmwareMetadataMaxBytes,
	)
	if err != nil {
		return VerifiedFirmwareArtifact{}, err
	}

	metadata, expectedDigest, err := parseFirmwareMetadata(rawMetadata)
	if err != nil {
		return VerifiedFirmwareArtifact{}, err
	}
	if filepath.Base(metadataPath) != "firmware_"+metadata.Version+".json" {
		return VerifiedFirmwareArtifact{}, policyError(
			"Firmware metadata filename does not match its version",
			nil,
		)
	}
	preparedSignature, err := prepareEmbeddedSignature(metadata.Signature, keys)
	if err != nil {
		return VerifiedFirmwareArtifact{}, err
	}
	if filepath.Base(archivePath) != metadata.Name {
		return VerifiedFirmwareArtifact{}, policyError(
			"Firmware archive filename does not match metadata",
			nil,
		)
	}

	snapshot, err := openArtifactSnapshot(
		archivePath,
		"Firmware archive",
		metadata.Size,
		0,
	)
	if err != nil {
		return VerifiedFirmwareArtifact{}, err
	}
	defer func() { _ = snapshot.Close() }()
	if snapshot.identity != initialArchiveIdentity {
		return VerifiedFirmwareArtifact{}, policyError(
			"Firmware archive changed while it was opened",
			nil,
		)
	}

	artifact := snapshot.Bytes()
	if err := preparedSignature.verify(artifact); err != nil {
		return VerifiedFirmwareArtifact{}, err
	}
	actualDigest := sha512.Sum512(artifact)
	if subtle.ConstantTimeCompare(actualDigest[:], expectedDigest) != 1 {
		return VerifiedFirmwareArtifact{}, invalid(
			"Firmware archive SHA-512 does not match metadata",
			nil,
		)
	}
	if err := validateFirmwareGzipArchive(artifact, metadata.Version); err != nil {
		return VerifiedFirmwareArtifact{}, invalid("invalid Firmware archive", err)
	}
	if err := enforceFirmwarePolicy(metadata.Version, policy); err != nil {
		return VerifiedFirmwareArtifact{}, err
	}
	if err := snapshot.VerifyStable(); err != nil {
		return VerifiedFirmwareArtifact{}, err
	}
	if err := snapshot.Close(); err != nil {
		return VerifiedFirmwareArtifact{}, err
	}

	return VerifiedFirmwareArtifact{
		Version:     metadata.Version,
		Name:        metadata.Name,
		Size:        metadata.Size,
		SHA512:      metadata.SHA512,
		KeyID:       metadata.Signature.KeyID,
		UpdateUboot: metadata.UpdateUboot != nil && *metadata.UpdateUboot,
	}, nil
}

func parseFirmwareMetadata(data []byte) (FirmwareMetadata, []byte, error) {
	if !utf8.Valid(data) {
		return FirmwareMetadata{}, nil, invalid("Firmware metadata is not UTF-8", nil)
	}
	var metadata FirmwareMetadata
	if err := strictUnmarshal(data, &metadata); err != nil {
		return FirmwareMetadata{}, nil, invalid("invalid Firmware metadata", err)
	}
	if err := validateFirmwareUpdateUboot(data); err != nil {
		return FirmwareMetadata{}, nil, invalid("invalid Firmware updateUboot policy", err)
	}
	if err := validateCanonicalFirmwareVersion(metadata.Version); err != nil {
		return FirmwareMetadata{}, nil, invalid("invalid Firmware metadata version", err)
	}
	expectedName := "axera_firmware_v" + metadata.Version + ".tar.gz"
	if metadata.Name != expectedName {
		return FirmwareMetadata{}, nil, policyError(
			"unsupported Firmware archive name; only .tar.gz is accepted",
			nil,
		)
	}
	if metadata.Size <= 0 {
		return FirmwareMetadata{}, nil, policyError("invalid Firmware archive size", nil)
	}
	digest, err := decodeCanonicalBase64(metadata.SHA512, sha512.Size)
	if err != nil {
		return FirmwareMetadata{}, nil, invalid("invalid Firmware SHA-512", err)
	}
	return metadata, digest, nil
}

func validateFirmwareUpdateUboot(data []byte) error {
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		return err
	}
	for name := range document {
		if name != "updateUboot" && strings.EqualFold(name, "updateUboot") {
			return fmt.Errorf("field name must be exactly updateUboot")
		}
	}
	raw, present := document["updateUboot"]
	if !present {
		return nil
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("must be a boolean")
	}
	return nil
}

func enforceFirmwarePolicy(version string, policy FirmwarePolicy) error {
	if policy.RequestedVersion != "" {
		if err := validateCanonicalFirmwareVersion(policy.RequestedVersion); err != nil {
			return policyError("invalid requested Firmware version", err)
		}
		if version != policy.RequestedVersion {
			return policyError(
				"Firmware metadata version does not match requested version",
				nil,
			)
		}
	}
	if policy.CurrentVersion != "" {
		if err := validateCanonicalFirmwareVersion(policy.CurrentVersion); err != nil {
			return policyError("invalid current Firmware version", err)
		}
		if version == policy.CurrentVersion && !policy.AllowSameVersion {
			return policyError("same-version Firmware installation is not allowed", nil)
		}
	}
	return nil
}

func validateCanonicalFirmwareVersion(version string) error {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return fmt.Errorf("version must contain three decimal components")
	}
	for _, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return fmt.Errorf("version component is not canonical")
		}
		for index := range len(part) {
			if part[index] < '0' || part[index] > '9' {
				return fmt.Errorf("version component is not decimal")
			}
		}
		if _, err := strconv.ParseUint(part, 10, 64); err != nil {
			return fmt.Errorf("version component exceeds uint64: %w", err)
		}
	}
	return nil
}

func validateFirmwareGzipArchive(artifact []byte, expectedVersion string) error {
	return validateFirmwareGzipArchiveWithLimits(
		artifact,
		expectedVersion,
		defaultFirmwareArchiveLimits,
	)
}

func validateFirmwareGzipArchiveWithLimits(
	artifact []byte,
	expectedVersion string,
	limits firmwareArchiveLimits,
) error {
	if limits.maxEntries <= 0 || limits.maxFile <= 0 ||
		limits.maxExpanded <= 0 || limits.maxEnvelope <= 0 {
		return fmt.Errorf("Firmware archive verification limits are invalid")
	}
	if len(artifact) < 2 ||
		artifact[0] != firmwareArchiveGzipMagicFirst ||
		artifact[1] != firmwareArchiveGzipMagicSecond {
		return fmt.Errorf("Firmware archive is not gzip data")
	}

	compressed := bytes.NewReader(artifact)
	reader, err := gzip.NewReader(compressed)
	if err != nil {
		return fmt.Errorf("open Firmware gzip stream: %w", err)
	}
	reader.Multistream(false)

	limited := &io.LimitedReader{R: reader, N: limits.maxEnvelope + 1}
	validationErr := validateFirmwareTar(limited, expectedVersion, limits)
	if limited.N == 0 {
		validationErr = fmt.Errorf(
			"Firmware archive decompressed envelope exceeds %d bytes",
			limits.maxEnvelope,
		)
	}
	closeErr := reader.Close()
	if validationErr != nil {
		return validationErr
	}
	if closeErr != nil {
		return fmt.Errorf("close Firmware gzip stream: %w", closeErr)
	}
	if compressed.Len() != 0 {
		return fmt.Errorf("Firmware gzip archive contains trailing bytes or multiple streams")
	}
	return nil
}

func validateFirmwareTar(
	reader io.Reader,
	expectedVersion string,
	limits firmwareArchiveLimits,
) error {
	seen := make(map[string]struct{})
	kinds := make(map[string]firmwareMemberKind)
	actualChecksums := make(map[string][blake2b.Size]byte)
	var versionData []byte
	var b2sumData []byte
	var entries int64
	var expanded int64

	for {
		block, err := readFirmwareTarBlock(reader, "header")
		if err != nil {
			return err
		}
		if isZeroFirmwareTarBlock(block) {
			endBlock, err := readFirmwareTarBlock(reader, "end-of-archive marker")
			if err != nil {
				return err
			}
			if !isZeroFirmwareTarBlock(endBlock) {
				return fmt.Errorf("Firmware tar archive has an invalid end marker")
			}
			if err := validateFirmwareTarEndPadding(reader); err != nil {
				return err
			}
			break
		}

		entries++
		if entries > limits.maxEntries {
			return fmt.Errorf("Firmware archive entry count exceeds %d", limits.maxEntries)
		}
		header, kind, err := parseDirectFirmwareTarHeader(block)
		if err != nil {
			return err
		}
		name, err := validateFirmwareMemberName(
			header.Name,
			kind == firmwareMemberDirectory,
		)
		if err != nil {
			return err
		}
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("duplicate Firmware archive member path %q", name)
		}
		seen[name] = struct{}{}

		if header.Size < 0 || header.Size > limits.maxFile {
			return fmt.Errorf("Firmware archive member %q exceeds the file size limit", name)
		}
		if header.Size > limits.maxExpanded-expanded {
			return fmt.Errorf("Firmware archive expanded size exceeds %d", limits.maxExpanded)
		}
		expanded += header.Size

		if kind == firmwareMemberDirectory {
			if header.Size != 0 {
				return fmt.Errorf("Firmware archive directory %q has non-zero size", name)
			}
			kinds[name] = kind
			continue
		}
		kinds[name] = kind

		switch name {
		case firmwareVersionPath:
			versionData, err = readFirmwareTarMember(
				reader,
				header.Size,
				firmwareVersionFileMaxBytes,
				firmwareVersionPath,
			)
			if err == nil {
				actualChecksums[name] = blake2b.Sum512(versionData)
			}
		case firmwareB2SumPath:
			b2sumData, err = readFirmwareTarMember(
				reader,
				header.Size,
				firmwareB2SumFileMaxBytes,
				firmwareB2SumPath,
			)
		default:
			actualChecksums[name], err = hashFirmwareTarMember(reader, header.Size)
		}
		if err != nil {
			return err
		}
		if err := readFirmwareTarMemberPadding(reader, header.Size); err != nil {
			return err
		}
	}

	for name, kind := range kinds {
		if (name == "firmware" || name == "overlay") && kind != firmwareMemberDirectory {
			return fmt.Errorf("Firmware archive root %q must be a directory", name)
		}
		parts := strings.Split(name, "/")
		for index := 1; index < len(parts); index++ {
			parent := strings.Join(parts[:index], "/")
			if kinds[parent] == firmwareMemberFile {
				return fmt.Errorf(
					"Firmware archive file %q conflicts with descendant %q",
					parent,
					name,
				)
			}
		}
	}
	if versionData == nil {
		return fmt.Errorf("Firmware archive is missing %s", firmwareVersionPath)
	}
	if b2sumData == nil {
		return fmt.Errorf("Firmware archive is missing %s", firmwareB2SumPath)
	}
	if err := validateFirmwareInternalVersion(versionData, expectedVersion); err != nil {
		return err
	}
	declaredChecksums, err := parseFirmwareB2Sum(b2sumData)
	if err != nil {
		return err
	}
	if len(declaredChecksums) != len(actualChecksums) {
		return fmt.Errorf("Firmware b2sum file set does not match archive files")
	}
	for name, actual := range actualChecksums {
		declared, ok := declaredChecksums[name]
		if !ok {
			return fmt.Errorf("Firmware b2sum is missing archive member %q", name)
		}
		if subtle.ConstantTimeCompare(actual[:], declared[:]) != 1 {
			return fmt.Errorf("Firmware b2sum checksum does not match %q", name)
		}
	}
	return nil
}

func readFirmwareTarBlock(reader io.Reader, label string) ([]byte, error) {
	block := make([]byte, firmwareTarBlockBytes)
	if _, err := io.ReadFull(reader, block); err != nil {
		return nil, fmt.Errorf("Firmware tar archive has a truncated %s: %w", label, err)
	}
	return block, nil
}

func isZeroFirmwareTarBlock(block []byte) bool {
	for _, value := range block {
		if value != 0 {
			return false
		}
	}
	return true
}

func parseDirectFirmwareTarHeader(block []byte) (*tar.Header, firmwareMemberKind, error) {
	if len(block) != firmwareTarBlockBytes {
		return nil, 0, fmt.Errorf("invalid Firmware tar header size")
	}
	var kind firmwareMemberKind
	switch block[156] {
	case tar.TypeReg, tar.TypeRegA:
		kind = firmwareMemberFile
	case tar.TypeDir:
		kind = firmwareMemberDirectory
	case tar.TypeXHeader, tar.TypeXGlobalHeader, tar.TypeGNULongName,
		tar.TypeGNULongLink, tar.TypeGNUSparse:
		return nil, 0, fmt.Errorf(
			"Firmware tar extended PAX/GNU metadata is not allowed",
		)
	default:
		return nil, 0, fmt.Errorf("unsupported Firmware tar member type %d", block[156])
	}

	reader := tar.NewReader(bytes.NewReader(block))
	header, err := reader.Next()
	if err != nil {
		return nil, 0, fmt.Errorf("invalid Firmware tar header: %w", err)
	}
	if header == nil {
		return nil, 0, fmt.Errorf("invalid empty Firmware tar header")
	}
	return header, kind, nil
}

func validateFirmwareMemberName(name string, directory bool) (string, error) {
	if name == "" || len(name) > firmwareArchiveMaxPathBytes ||
		!utf8.ValidString(name) || strings.HasPrefix(name, "/") ||
		strings.ContainsAny(name, "\\\x00") {
		return "", fmt.Errorf("unsafe Firmware archive member path %q", name)
	}
	for index := range len(name) {
		if name[index] < 32 || name[index] == 127 {
			return "", fmt.Errorf("unsafe Firmware archive member path %q", name)
		}
	}

	canonical := name
	if directory && strings.HasSuffix(canonical, "/") {
		canonical = strings.TrimSuffix(canonical, "/")
	}
	if canonical == "" || path.IsAbs(canonical) || path.Clean(canonical) != canonical {
		return "", fmt.Errorf("non-canonical Firmware archive member path %q", name)
	}
	parts := strings.Split(canonical, "/")
	for _, component := range parts {
		if component == "" || component == "." || component == ".." ||
			len(component) > firmwareArchiveMaxComponent {
			return "", fmt.Errorf("non-canonical Firmware archive member path %q", name)
		}
	}
	if canonical != firmwareB2SumPath && parts[0] != "firmware" && parts[0] != "overlay" {
		return "", fmt.Errorf("unsupported Firmware archive top-level path %q", name)
	}
	return canonical, nil
}

func readFirmwareTarMember(
	reader io.Reader,
	size, limit int64,
	label string,
) ([]byte, error) {
	if size > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", label, limit)
	}
	data := make([]byte, int(size))
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, fmt.Errorf("Firmware tar member %s is truncated: %w", label, err)
	}
	return data, nil
}

func hashFirmwareTarMember(reader io.Reader, size int64) ([blake2b.Size]byte, error) {
	var result [blake2b.Size]byte
	hasher, err := blake2b.New512(nil)
	if err != nil {
		return result, fmt.Errorf("initialize Firmware b2sum: %w", err)
	}
	written, err := io.CopyN(hasher, reader, size)
	if err != nil {
		return result, fmt.Errorf("Firmware tar member is truncated: %w", err)
	}
	if written != size {
		return result, io.ErrUnexpectedEOF
	}
	copy(result[:], hasher.Sum(nil))
	return result, nil
}

func readFirmwareTarMemberPadding(reader io.Reader, size int64) error {
	paddingSize := (-size) & (firmwareTarBlockBytes - 1)
	if paddingSize == 0 {
		return nil
	}
	padding := make([]byte, int(paddingSize))
	if _, err := io.ReadFull(reader, padding); err != nil {
		return fmt.Errorf("Firmware tar member padding is truncated: %w", err)
	}
	if !isZeroFirmwareTarBlock(padding) {
		return fmt.Errorf("Firmware tar member has non-zero padding")
	}
	return nil
}

func validateFirmwareTarEndPadding(reader io.Reader) error {
	buffer := make([]byte, 1<<20)
	var total int64
	for {
		read, err := reader.Read(buffer)
		if read > 0 {
			total += int64(read)
			for _, value := range buffer[:read] {
				if value != 0 {
					return fmt.Errorf("Firmware tar contains non-zero data after its end marker")
				}
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read Firmware tar end padding: %w", err)
		}
	}
	if total%firmwareTarBlockBytes != 0 {
		return fmt.Errorf("Firmware tar end padding is not 512-byte aligned")
	}
	return nil
}

func validateFirmwareInternalVersion(data []byte, expectedVersion string) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("%s is not UTF-8", firmwareVersionPath)
	}
	tokens := findFirmwareVersionTokens(string(data))
	if len(tokens) != 1 {
		return fmt.Errorf("%s must contain exactly one canonical vX.Y.Z token", firmwareVersionPath)
	}
	if tokens[0].prefix != 'v' {
		return fmt.Errorf("%s version token must use a lowercase v", firmwareVersionPath)
	}
	if tokens[0].version != expectedVersion {
		return fmt.Errorf(
			"Firmware internal version %s does not match %s",
			tokens[0].version,
			expectedVersion,
		)
	}
	return nil
}

func findFirmwareVersionTokens(value string) []firmwareVersionToken {
	var tokens []firmwareVersionToken
	for index := 0; index < len(value); index++ {
		if value[index] != 'v' && value[index] != 'V' {
			continue
		}
		if index > 0 && isASCIIAlphaNumeric(value[index-1]) {
			continue
		}
		end, ok := scanFirmwareVersion(value, index+1)
		if !ok {
			continue
		}
		if end < len(value) && (isASCIIAlphaNumeric(value[end]) ||
			strings.ContainsRune(".+-", rune(value[end]))) {
			continue
		}
		tokens = append(tokens, firmwareVersionToken{
			prefix:  value[index],
			version: value[index+1 : end],
		})
	}
	return tokens
}

func scanFirmwareVersion(value string, offset int) (int, bool) {
	position := offset
	for component := 0; component < 3; component++ {
		start := position
		for position < len(value) && value[position] >= '0' && value[position] <= '9' {
			position++
		}
		if position == start || (position-start > 1 && value[start] == '0') {
			return 0, false
		}
		if component < 2 {
			if position >= len(value) || value[position] != '.' {
				return 0, false
			}
			position++
		}
	}
	return position, true
}

func isASCIIAlphaNumeric(value byte) bool {
	return value >= '0' && value <= '9' ||
		value >= 'A' && value <= 'Z' ||
		value >= 'a' && value <= 'z'
}

func parseFirmwareB2Sum(data []byte) (map[string][blake2b.Size]byte, error) {
	if len(data) == 0 || data[len(data)-1] != '\n' {
		return nil, fmt.Errorf("%s must be non-empty and end with LF", firmwareB2SumPath)
	}
	for _, value := range data {
		if value >= 0x80 {
			return nil, fmt.Errorf("%s must be ASCII", firmwareB2SumPath)
		}
	}

	checksums := make(map[string][blake2b.Size]byte)
	for _, line := range strings.Split(string(data[:len(data)-1]), "\n") {
		if len(line) < 131 || line[128:130] != "  " {
			return nil, fmt.Errorf("%s contains an invalid checksum line", firmwareB2SumPath)
		}
		for index := 0; index < 128; index++ {
			if !((line[index] >= '0' && line[index] <= '9') ||
				(line[index] >= 'a' && line[index] <= 'f')) {
				return nil, fmt.Errorf("%s checksum must use lowercase hexadecimal", firmwareB2SumPath)
			}
		}
		name, err := validateFirmwareMemberName(line[130:], false)
		if err != nil {
			return nil, err
		}
		if name == firmwareB2SumPath {
			return nil, fmt.Errorf("%s must not checksum itself", firmwareB2SumPath)
		}
		if _, duplicate := checksums[name]; duplicate {
			return nil, fmt.Errorf("%s contains duplicate path %q", firmwareB2SumPath, name)
		}
		decoded, err := hex.DecodeString(line[:128])
		if err != nil {
			return nil, fmt.Errorf("decode %s checksum: %w", firmwareB2SumPath, err)
		}
		var digest [blake2b.Size]byte
		copy(digest[:], decoded)
		checksums[name] = digest
	}
	return checksums, nil
}
