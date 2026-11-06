package updateverify

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/sys/unix"
)

const firmwareRFC8032SeedHex = "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"

type firmwareTestTarEntry struct {
	name     string
	typeflag byte
	body     []byte
	linkname string
	format   tar.Format
}

type firmwareSigningVectorFile struct {
	Schema    int    `json:"schema"`
	TestOnly  bool   `json:"testOnly"`
	PublicKey string `json:"publicKey"`
	KeyID     string `json:"keyId"`
	Vectors   []struct {
		Kind      string    `json:"kind"`
		Name      string    `json:"name"`
		Artifact  string    `json:"artifact"`
		Signature Signature `json:"signature"`
	} `json:"vectors"`
}

func firmwareTestDirectory(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate firmware test source")
	}
	parent := filepath.Join(filepath.Dir(source), "..", "..", "build", "test-tmp")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	directory, err := os.MkdirTemp(parent, ".firmware-verifier-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(directory); err != nil {
			t.Errorf("remove firmware test directory: %v", err)
		}
	})
	return directory
}

func firmwareTestKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, string) {
	t.Helper()
	seed, err := hex.DecodeString(firmwareRFC8032SeedHex)
	if err != nil {
		t.Fatal(err)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := append(ed25519.PublicKey(nil), privateKey.Public().(ed25519.PublicKey)...)
	return publicKey, privateKey, PublicKeyID(publicKey)
}

func firmwareTestB2Sum(files map[string][]byte) []byte {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var document strings.Builder
	for _, name := range names {
		digest := blake2b.Sum512(files[name])
		fmt.Fprintf(&document, "%x  %s\n", digest, name)
	}
	return []byte(document.String())
}

func validFirmwareTestEntries(versionText []byte, extra map[string][]byte) []firmwareTestTarEntry {
	files := map[string][]byte{firmwareVersionPath: append([]byte(nil), versionText...)}
	for name, content := range extra {
		files[name] = append([]byte(nil), content...)
	}
	entries := []firmwareTestTarEntry{
		{name: "overlay", typeflag: tar.TypeDir},
		{name: "overlay/boot", typeflag: tar.TypeDir},
	}
	if _, ok := files["firmware/image.bin"]; ok {
		entries = append([]firmwareTestTarEntry{{name: "firmware", typeflag: tar.TypeDir}}, entries...)
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		entries = append(entries, firmwareTestTarEntry{
			name:     name,
			typeflag: tar.TypeReg,
			body:     files[name],
		})
	}
	entries = append(entries, firmwareTestTarEntry{
		name:     firmwareB2SumPath,
		typeflag: tar.TypeReg,
		body:     firmwareTestB2Sum(files),
	})
	return entries
}

func firmwareTestTar(t *testing.T, entries []firmwareTestTarEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	for _, entry := range entries {
		mode := int64(0o644)
		if entry.typeflag == tar.TypeDir {
			mode = 0o755
		}
		header := &tar.Header{
			Name:     entry.name,
			Typeflag: entry.typeflag,
			Mode:     mode,
			Size:     int64(len(entry.body)),
			Linkname: entry.linkname,
			ModTime:  time.Unix(0, 0),
			Uid:      0,
			Gid:      0,
			Uname:    "root",
			Gname:    "root",
			Format:   entry.format,
		}
		if entry.typeflag != tar.TypeReg && entry.typeflag != tar.TypeRegA {
			header.Size = 0
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatalf("write tar header %q: %v", entry.name, err)
		}
		if len(entry.body) > 0 {
			if _, err := writer.Write(entry.body); err != nil {
				t.Fatalf("write tar body %q: %v", entry.name, err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func firmwareTestGzip(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buffer, gzip.BestCompression)
	if err != nil {
		t.Fatal(err)
	}
	writer.Header.ModTime = time.Unix(0, 0)
	writer.Header.OS = 255
	if _, err := writer.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func recompressFirmwareTestArchive(t *testing.T, artifact []byte) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(artifact))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}

	var buffer bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buffer, gzip.BestSpeed)
	if err != nil {
		t.Fatal(err)
	}
	writer.Header.ModTime = time.Unix(1, 0)
	writer.Header.OS = 255
	if _, err := writer.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	recompressed := buffer.Bytes()
	if bytes.Equal(recompressed, artifact) {
		t.Fatal("recompressed Firmware test archive did not change bytes")
	}
	return recompressed
}

func validFirmwareTestArchive(t *testing.T, version string) []byte {
	t.Helper()
	entries := validFirmwareTestEntries(
		[]byte("NanoKVM firmware v"+version+" test archive\n"),
		map[string][]byte{"firmware/image.bin": []byte("signed firmware image\n")},
	)
	return firmwareTestGzip(t, firmwareTestTar(t, entries))
}

func firmwareTestTarEndOffset(t *testing.T, payload []byte) int {
	t.Helper()
	zero := make([]byte, firmwareTarBlockBytes)
	for offset := 0; offset+2*firmwareTarBlockBytes <= len(payload); offset += firmwareTarBlockBytes {
		if bytes.Equal(payload[offset:offset+firmwareTarBlockBytes], zero) &&
			bytes.Equal(payload[offset+firmwareTarBlockBytes:offset+2*firmwareTarBlockBytes], zero) {
			return offset
		}
	}
	t.Fatal("locate tar end marker")
	return 0
}

func firmwareTestReplaceB2Sum(
	t *testing.T,
	entries []firmwareTestTarEntry,
	mutate func([]byte) []byte,
) []firmwareTestTarEntry {
	t.Helper()
	for index := range entries {
		if entries[index].name == firmwareB2SumPath {
			entries[index].body = mutate(append([]byte(nil), entries[index].body...))
			return entries
		}
	}
	t.Fatal("locate b2sum test entry")
	return nil
}

func assertFirmwareArchiveInvalid(
	t *testing.T,
	artifact []byte,
	version, errorMatch string,
) {
	t.Helper()
	err := validateFirmwareGzipArchive(artifact, version)
	if err == nil {
		t.Fatal("expected invalid Firmware archive")
	}
	if errorMatch != "" && !strings.Contains(err.Error(), errorMatch) {
		t.Fatalf("error = %v, want substring %q", err, errorMatch)
	}
}

func firmwareTestMetadata(
	t *testing.T,
	version, name string,
	artifact []byte,
	privateKey ed25519.PrivateKey,
	keyID string,
) FirmwareMetadata {
	t.Helper()
	digest := sha512.Sum512(artifact)
	return FirmwareMetadata{
		Version: version,
		Name:    name,
		SHA512:  base64.StdEncoding.EncodeToString(digest[:]),
		Size:    int64(len(artifact)),
		Signature: &Signature{
			Schema:    1,
			Algorithm: firmwareSignatureAlgorithm,
			KeyID:     keyID,
			Value:     base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, artifact)),
		},
	}
}

func cloneFirmwareTestMetadata(metadata FirmwareMetadata) FirmwareMetadata {
	clone := metadata
	if metadata.Signature != nil {
		signature := *metadata.Signature
		clone.Signature = &signature
	}
	return clone
}

func writeFirmwareTestMetadata(t *testing.T, path string, metadata FirmwareMetadata) {
	t.Helper()
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeFirmwareTestCase(
	t *testing.T,
	directory string,
	metadata FirmwareMetadata,
	artifact []byte,
) (string, string) {
	t.Helper()
	archivePath := filepath.Join(directory, metadata.Name)
	metadataPath := filepath.Join(directory, "firmware_"+metadata.Version+".json")
	if err := os.WriteFile(archivePath, artifact, 0o600); err != nil {
		t.Fatal(err)
	}
	writeFirmwareTestMetadata(t, metadataPath, metadata)
	return archivePath, metadataPath
}

func TestVerifyFirmwareArtifactAppliesUnsignedUpdateUbootPolicy(t *testing.T) {
	directory := firmwareTestDirectory(t)
	publicKey, privateKey, keyID := firmwareTestKey(t)
	artifact := validFirmwareTestArchive(t, "1.0.2")
	metadata := firmwareTestMetadata(
		t,
		"1.0.2",
		"axera_firmware_v1.0.2.tar.gz",
		artifact,
		privateKey,
		keyID,
	)
	archivePath, metadataPath := writeFirmwareTestCase(t, directory, metadata, artifact)
	keys := KeyRing{keyID: publicKey}

	verified, err := VerifyFirmwareArtifact(archivePath, metadataPath, keys, FirmwarePolicy{})
	if err != nil {
		t.Fatalf("verify metadata without updateUboot: %v", err)
	}
	if verified.UpdateUboot {
		t.Fatal("missing updateUboot must default to false")
	}

	// The same archive and embedded signature must verify for both values because
	// updateUboot is intentionally outside the existing archive signature.
	for _, updateUboot := range []bool{false, true} {
		metadata.UpdateUboot = &updateUboot
		writeFirmwareTestMetadata(t, metadataPath, metadata)
		verified, err = VerifyFirmwareArtifact(archivePath, metadataPath, keys, FirmwarePolicy{})
		if err != nil {
			t.Fatalf("verify updateUboot=%t with unchanged signed archive: %v", updateUboot, err)
		}
		if verified.UpdateUboot != updateUboot {
			t.Fatalf("verified UpdateUboot = %t, want %t", verified.UpdateUboot, updateUboot)
		}
	}
}

func TestVerifyFirmwareArtifactFixedPythonVectorAndVersionPolicy(t *testing.T) {
	directory := firmwareTestDirectory(t)
	fixtureData, err := os.ReadFile("testdata/artifact_signing_vectors_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture firmwareSigningVectorFile
	if err := json.Unmarshal(fixtureData, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Schema != 1 || !fixture.TestOnly {
		t.Fatalf("unexpected signing fixture header: %#v", fixture)
	}
	publicKey, err := base64.StdEncoding.Strict().DecodeString(fixture.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if PublicKeyID(ed25519.PublicKey(publicKey)) != fixture.KeyID {
		t.Fatal("fixed vector key ID mismatch")
	}

	var vector *struct {
		Kind      string    `json:"kind"`
		Name      string    `json:"name"`
		Artifact  string    `json:"artifact"`
		Signature Signature `json:"signature"`
	}
	for index := range fixture.Vectors {
		if fixture.Vectors[index].Kind == "firmware" {
			vector = &fixture.Vectors[index]
			break
		}
	}
	if vector == nil {
		t.Fatal("fixed Firmware vector is missing")
	}
	artifact, err := base64.StdEncoding.Strict().DecodeString(vector.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha512.Sum512(artifact)
	metadata := FirmwareMetadata{
		Version:   "1.0.2",
		Name:      vector.Name,
		SHA512:    base64.StdEncoding.EncodeToString(digest[:]),
		Size:      int64(len(artifact)),
		Signature: &vector.Signature,
	}
	archivePath, metadataPath := writeFirmwareTestCase(t, directory, metadata, artifact)
	keys := KeyRing{fixture.KeyID: ed25519.PublicKey(publicKey)}

	verified, err := VerifyFirmwareArtifact(
		archivePath,
		metadataPath,
		keys,
		FirmwarePolicy{RequestedVersion: "1.0.2", CurrentVersion: "1.0.3"},
	)
	if err != nil {
		t.Fatalf("verify fixed Python Firmware vector: %v", err)
	}
	if verified.Version != metadata.Version || verified.Name != metadata.Name ||
		verified.Size != metadata.Size || verified.SHA512 != metadata.SHA512 ||
		verified.KeyID != fixture.KeyID {
		t.Fatalf("verified result = %#v", verified)
	}

	_, err = VerifyFirmwareArtifact(
		archivePath,
		metadataPath,
		keys,
		FirmwarePolicy{CurrentVersion: "1.0.2"},
	)
	assertKind(t, err, ErrorPolicy)
	if _, err := VerifyFirmwareArtifact(
		archivePath,
		metadataPath,
		keys,
		FirmwarePolicy{CurrentVersion: "1.0.2", AllowSameVersion: true},
	); err != nil {
		t.Fatalf("allow explicit same-version Firmware verification: %v", err)
	}
}

func TestVerifyFirmwareArtifactRejectsXZWithoutFallback(t *testing.T) {
	directory := firmwareTestDirectory(t)
	publicKey, privateKey, keyID := firmwareTestKey(t)
	artifact := []byte("\xfd7zXZ\x00signed-test-only-xz-like-bytes")
	metadata := firmwareTestMetadata(
		t,
		"1.0.2",
		"axera_firmware_v1.0.2.tar.xz",
		artifact,
		privateKey,
		keyID,
	)
	archivePath, metadataPath := writeFirmwareTestCase(t, directory, metadata, artifact)
	_, err := VerifyFirmwareArtifact(
		archivePath,
		metadataPath,
		KeyRing{keyID: publicKey},
		FirmwarePolicy{},
	)
	assertKind(t, err, ErrorPolicy)
	if !strings.Contains(err.Error(), "only .tar.gz") {
		t.Fatalf("xz rejection = %v", err)
	}
}

func TestVerifyFirmwareArtifactRejectsSignatureFailures(t *testing.T) {
	root := firmwareTestDirectory(t)
	publicKey, privateKey, keyID := firmwareTestKey(t)
	artifact := validFirmwareTestArchive(t, "1.0.2")
	valid := firmwareTestMetadata(
		t,
		"1.0.2",
		"axera_firmware_v1.0.2.tar.gz",
		artifact,
		privateKey,
		keyID,
	)
	wrongSignature := bytes.Repeat([]byte{0x5a}, ed25519.SignatureSize)

	tests := []struct {
		name       string
		mutate     func(*FirmwareMetadata)
		keys       KeyRing
		expected   ErrorKind
		errorMatch string
	}{
		{
			name: "missing signature",
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Signature = nil
			},
			keys:       KeyRing{keyID: publicKey},
			expected:   ErrorSignature,
			errorMatch: "missing",
		},
		{
			name: "wrong schema",
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Signature.Schema = 2
			},
			keys:     KeyRing{keyID: publicKey},
			expected: ErrorInvalid,
		},
		{
			name: "wrong algorithm",
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Signature.Algorithm = "Ed25519"
			},
			keys:     KeyRing{keyID: publicKey},
			expected: ErrorInvalid,
		},
		{
			name: "malformed key id",
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Signature.KeyID = "sha256:ABC"
			},
			keys:     KeyRing{keyID: publicKey},
			expected: ErrorInvalid,
		},
		{
			name: "unknown key id",
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Signature.KeyID = "sha256:" + strings.Repeat("0", 64)
			},
			keys:       KeyRing{keyID: publicKey},
			expected:   ErrorSignature,
			errorMatch: "unknown",
		},
		{
			name: "invalid base64",
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Signature.Value = "***"
			},
			keys:     KeyRing{keyID: publicKey},
			expected: ErrorSignature,
		},
		{
			name: "base64 newline",
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Signature.Value += "\n"
			},
			keys:     KeyRing{keyID: publicKey},
			expected: ErrorSignature,
		},
		{
			name: "63-byte signature",
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Signature.Value = base64.StdEncoding.EncodeToString(make([]byte, 63))
			},
			keys:     KeyRing{keyID: publicKey},
			expected: ErrorSignature,
		},
		{
			name: "65-byte signature",
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Signature.Value = base64.StdEncoding.EncodeToString(make([]byte, 65))
			},
			keys:     KeyRing{keyID: publicKey},
			expected: ErrorSignature,
		},
		{
			name: "wrong signature",
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Signature.Value = base64.StdEncoding.EncodeToString(wrongSignature)
			},
			keys:       KeyRing{keyID: publicKey},
			expected:   ErrorSignature,
			errorMatch: "verification failed",
		},
		{
			name:   "invalid public key",
			mutate: func(*FirmwareMetadata) {},
			keys: KeyRing{
				keyID: ed25519.PublicKey{1, 2, 3},
			},
			expected: ErrorSignature,
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := filepath.Join(root, fmt.Sprintf("signature-%02d", index))
			if err := os.Mkdir(directory, 0o700); err != nil {
				t.Fatal(err)
			}
			metadata := cloneFirmwareTestMetadata(valid)
			test.mutate(&metadata)
			archivePath, metadataPath := writeFirmwareTestCase(t, directory, metadata, artifact)
			_, err := VerifyFirmwareArtifact(
				archivePath,
				metadataPath,
				test.keys,
				FirmwarePolicy{},
			)
			assertKind(t, err, test.expected)
			if test.errorMatch != "" && !strings.Contains(err.Error(), test.errorMatch) {
				t.Fatalf("error = %v, want substring %q", err, test.errorMatch)
			}
		})
	}
}

func TestVerifyFirmwareArtifactRejectsStrictMetadataFailures(t *testing.T) {
	root := firmwareTestDirectory(t)
	publicKey, privateKey, keyID := firmwareTestKey(t)
	artifact := validFirmwareTestArchive(t, "1.0.2")
	valid := firmwareTestMetadata(
		t,
		"1.0.2",
		"axera_firmware_v1.0.2.tar.gz",
		artifact,
		privateKey,
		keyID,
	)
	validJSON, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	withUpdateUboot := func(value string) []byte {
		t.Helper()
		return bytes.Replace(
			validJSON,
			[]byte(`"signature":`),
			[]byte(`"updateUboot":`+value+`,"signature":`),
			1,
		)
	}

	tests := []struct {
		name     string
		metadata func() []byte
		expected ErrorKind
	}{
		{
			name: "duplicate field",
			metadata: func() []byte {
				return bytes.Replace(
					validJSON,
					[]byte(`"version":"1.0.2"`),
					[]byte(`"version":"1.0.2","version":"1.0.2"`),
					1,
				)
			},
			expected: ErrorInvalid,
		},
		{
			name: "unknown field",
			metadata: func() []byte {
				var document map[string]any
				if err := json.Unmarshal(validJSON, &document); err != nil {
					t.Fatal(err)
				}
				document["extra"] = true
				data, err := json.Marshal(document)
				if err != nil {
					t.Fatal(err)
				}
				return data
			},
			expected: ErrorInvalid,
		},
		{
			name: "duplicate updateUboot field",
			metadata: func() []byte {
				return withUpdateUboot(`false,"updateUboot":true`)
			},
			expected: ErrorInvalid,
		},
		{
			name: "non-canonical updateUboot field name",
			metadata: func() []byte {
				return bytes.Replace(
					validJSON,
					[]byte(`"signature":`),
					[]byte(`"UpdateUboot":true,"signature":`),
					1,
				)
			},
			expected: ErrorInvalid,
		},
		{
			name:     "null updateUboot",
			metadata: func() []byte { return withUpdateUboot(`null`) },
			expected: ErrorInvalid,
		},
		{
			name:     "string updateUboot",
			metadata: func() []byte { return withUpdateUboot(`"false"`) },
			expected: ErrorInvalid,
		},
		{
			name:     "number updateUboot",
			metadata: func() []byte { return withUpdateUboot(`0`) },
			expected: ErrorInvalid,
		},
		{
			name:     "array updateUboot",
			metadata: func() []byte { return withUpdateUboot(`[]`) },
			expected: ErrorInvalid,
		},
		{
			name:     "object updateUboot",
			metadata: func() []byte { return withUpdateUboot(`{}`) },
			expected: ErrorInvalid,
		},
		{
			name: "trailing JSON",
			metadata: func() []byte {
				return append(append([]byte(nil), validJSON...), []byte(`{}`)...)
			},
			expected: ErrorInvalid,
		},
		{
			name: "wrong size type",
			metadata: func() []byte {
				return bytes.Replace(
					validJSON,
					[]byte(fmt.Sprintf(`"size":%d`, valid.Size)),
					[]byte(fmt.Sprintf(`"size":"%d"`, valid.Size)),
					1,
				)
			},
			expected: ErrorInvalid,
		},
		{
			name: "invalid UTF-8",
			metadata: func() []byte {
				return []byte{'{', 0xff, '}'}
			},
			expected: ErrorInvalid,
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := filepath.Join(root, fmt.Sprintf("metadata-%02d", index))
			if err := os.Mkdir(directory, 0o700); err != nil {
				t.Fatal(err)
			}
			archivePath := filepath.Join(directory, valid.Name)
			metadataPath := filepath.Join(directory, "firmware_1.0.2.json")
			if err := os.WriteFile(archivePath, artifact, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(metadataPath, test.metadata(), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := VerifyFirmwareArtifact(
				archivePath,
				metadataPath,
				KeyRing{keyID: publicKey},
				FirmwarePolicy{},
			)
			assertKind(t, err, test.expected)
		})
	}
}

func TestVerifyFirmwareArtifactRejectsIdentityAndPolicyMismatches(t *testing.T) {
	root := firmwareTestDirectory(t)
	publicKey, privateKey, keyID := firmwareTestKey(t)
	artifact := validFirmwareTestArchive(t, "1.0.2")
	valid := firmwareTestMetadata(
		t,
		"1.0.2",
		"axera_firmware_v1.0.2.tar.gz",
		artifact,
		privateKey,
		keyID,
	)
	recompressed := recompressFirmwareTestArchive(t, artifact)

	tests := []struct {
		name       string
		artifact   func() []byte
		mutate     func(*FirmwareMetadata)
		policy     FirmwarePolicy
		expected   ErrorKind
		errorMatch string
	}{
		{
			name:     "leading v metadata version",
			artifact: func() []byte { return artifact },
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Version = "v1.0.2"
			},
			expected: ErrorInvalid,
		},
		{
			name:     "non-canonical metadata version",
			artifact: func() []byte { return artifact },
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Version = "1.00.2"
			},
			expected: ErrorInvalid,
		},
		{
			name:     "overflow metadata version",
			artifact: func() []byte { return artifact },
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Version = "18446744073709551616.0.0"
			},
			expected: ErrorInvalid,
		},
		{
			name:     "wrong archive name",
			artifact: func() []byte { return artifact },
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Name = "axera_firmware_v1.0.3.tar.gz"
			},
			expected: ErrorPolicy,
		},
		{
			name:     "zero size",
			artifact: func() []byte { return artifact },
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Size = 0
			},
			expected: ErrorPolicy,
		},
		{
			name:     "wrong positive size",
			artifact: func() []byte { return artifact },
			mutate: func(metadata *FirmwareMetadata) {
				metadata.Size++
			},
			expected: ErrorPolicy,
		},
		{
			name:     "invalid SHA base64",
			artifact: func() []byte { return artifact },
			mutate: func(metadata *FirmwareMetadata) {
				metadata.SHA512 = "***"
			},
			expected: ErrorInvalid,
		},
		{
			name:     "SHA base64 newline",
			artifact: func() []byte { return artifact },
			mutate: func(metadata *FirmwareMetadata) {
				metadata.SHA512 += "\n"
			},
			expected: ErrorInvalid,
		},
		{
			name:     "wrong SHA",
			artifact: func() []byte { return artifact },
			mutate: func(metadata *FirmwareMetadata) {
				metadata.SHA512 = base64.StdEncoding.EncodeToString(make([]byte, sha512.Size))
			},
			expected:   ErrorInvalid,
			errorMatch: "SHA-512",
		},
		{
			name: "tampered archive",
			artifact: func() []byte {
				changed := append([]byte(nil), artifact...)
				changed[len(changed)/2] ^= 1
				return changed
			},
			mutate:   func(*FirmwareMetadata) {},
			expected: ErrorSignature,
		},
		{
			name: "recompressed archive",
			artifact: func() []byte {
				return recompressed
			},
			mutate: func(metadata *FirmwareMetadata) {
				digest := sha512.Sum512(recompressed)
				metadata.Size = int64(len(recompressed))
				metadata.SHA512 = base64.StdEncoding.EncodeToString(digest[:])
			},
			expected: ErrorSignature,
		},
		{
			name:     "requested version mismatch",
			artifact: func() []byte { return artifact },
			mutate:   func(*FirmwareMetadata) {},
			policy: FirmwarePolicy{
				RequestedVersion: "1.0.3",
			},
			expected: ErrorPolicy,
		},
		{
			name:     "non-canonical requested version",
			artifact: func() []byte { return artifact },
			mutate:   func(*FirmwareMetadata) {},
			policy: FirmwarePolicy{
				RequestedVersion: "v1.0.2",
			},
			expected: ErrorPolicy,
		},
		{
			name:     "non-canonical current version",
			artifact: func() []byte { return artifact },
			mutate:   func(*FirmwareMetadata) {},
			policy: FirmwarePolicy{
				CurrentVersion: "1.00.2",
			},
			expected: ErrorPolicy,
		},
		{
			name: "signed non-gzip bytes",
			artifact: func() []byte {
				return []byte("not a gzip archive")
			},
			mutate: func(metadata *FirmwareMetadata) {
				bad := []byte("not a gzip archive")
				digest := sha512.Sum512(bad)
				metadata.Size = int64(len(bad))
				metadata.SHA512 = base64.StdEncoding.EncodeToString(digest[:])
				metadata.Signature.Value = base64.StdEncoding.EncodeToString(
					ed25519.Sign(privateKey, bad),
				)
			},
			expected: ErrorInvalid,
		},
		{
			name: "signed xz magic with gzip name",
			artifact: func() []byte {
				return []byte("\xfd7zXZ\x00signed-test-only-xz-like-bytes")
			},
			mutate: func(metadata *FirmwareMetadata) {
				bad := []byte("\xfd7zXZ\x00signed-test-only-xz-like-bytes")
				digest := sha512.Sum512(bad)
				metadata.Size = int64(len(bad))
				metadata.SHA512 = base64.StdEncoding.EncodeToString(digest[:])
				metadata.Signature.Value = base64.StdEncoding.EncodeToString(
					ed25519.Sign(privateKey, bad),
				)
			},
			expected: ErrorInvalid,
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := filepath.Join(root, fmt.Sprintf("identity-%02d", index))
			if err := os.Mkdir(directory, 0o700); err != nil {
				t.Fatal(err)
			}
			metadata := cloneFirmwareTestMetadata(valid)
			test.mutate(&metadata)
			candidate := test.artifact()
			archivePath, metadataPath := writeFirmwareTestCase(t, directory, metadata, candidate)
			_, err := VerifyFirmwareArtifact(
				archivePath,
				metadataPath,
				KeyRing{keyID: publicKey},
				test.policy,
			)
			assertKind(t, err, test.expected)
			if test.errorMatch != "" && !strings.Contains(err.Error(), test.errorMatch) {
				t.Fatalf("error = %v, want substring %q", err, test.errorMatch)
			}
		})
	}

	t.Run("internal signed version mismatch", func(t *testing.T) {
		directory := filepath.Join(root, "internal-version")
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		wrongInternal := validFirmwareTestArchive(t, "1.0.3")
		metadata := firmwareTestMetadata(
			t,
			"1.0.2",
			"axera_firmware_v1.0.2.tar.gz",
			wrongInternal,
			privateKey,
			keyID,
		)
		archivePath, metadataPath := writeFirmwareTestCase(t, directory, metadata, wrongInternal)
		_, err := VerifyFirmwareArtifact(
			archivePath,
			metadataPath,
			KeyRing{keyID: publicKey},
			FirmwarePolicy{},
		)
		assertKind(t, err, ErrorInvalid)
		if !strings.Contains(err.Error(), "internal version") {
			t.Fatalf("internal version error = %v", err)
		}
	})

	t.Run("forged metadata and filename version", func(t *testing.T) {
		directory := filepath.Join(root, "forged-metadata-version")
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		metadata := firmwareTestMetadata(
			t,
			"1.0.3",
			"axera_firmware_v1.0.3.tar.gz",
			artifact,
			privateKey,
			keyID,
		)
		archivePath, metadataPath := writeFirmwareTestCase(t, directory, metadata, artifact)
		_, err := VerifyFirmwareArtifact(
			archivePath,
			metadataPath,
			KeyRing{keyID: publicKey},
			FirmwarePolicy{RequestedVersion: "1.0.3"},
		)
		assertKind(t, err, ErrorInvalid)
		if !strings.Contains(err.Error(), "internal version") {
			t.Fatalf("forged metadata version error = %v", err)
		}
	})

	t.Run("signature failure precedes requested policy mismatch", func(t *testing.T) {
		directory := filepath.Join(root, "signature-before-policy")
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		metadata := cloneFirmwareTestMetadata(valid)
		metadata.Signature.Value = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
		archivePath, metadataPath := writeFirmwareTestCase(t, directory, metadata, artifact)
		_, err := VerifyFirmwareArtifact(
			archivePath,
			metadataPath,
			KeyRing{keyID: publicKey},
			FirmwarePolicy{RequestedVersion: "1.0.3"},
		)
		assertKind(t, err, ErrorSignature)
	})
}

func TestVerifyFirmwareArtifactRejectsPathTypeAndFilenameTricks(t *testing.T) {
	root := firmwareTestDirectory(t)
	publicKey, privateKey, keyID := firmwareTestKey(t)
	artifact := validFirmwareTestArchive(t, "1.0.2")
	metadata := firmwareTestMetadata(
		t,
		"1.0.2",
		"axera_firmware_v1.0.2.tar.gz",
		artifact,
		privateKey,
		keyID,
	)
	keys := KeyRing{keyID: publicKey}

	t.Run("metadata filename", func(t *testing.T) {
		directory := filepath.Join(root, "metadata-filename")
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		archivePath := filepath.Join(directory, metadata.Name)
		metadataPath := filepath.Join(directory, "metadata.json")
		if err := os.WriteFile(archivePath, artifact, 0o600); err != nil {
			t.Fatal(err)
		}
		writeFirmwareTestMetadata(t, metadataPath, metadata)
		_, err := VerifyFirmwareArtifact(archivePath, metadataPath, keys, FirmwarePolicy{})
		assertKind(t, err, ErrorPolicy)
	})

	t.Run("archive filename", func(t *testing.T) {
		directory := filepath.Join(root, "archive-filename")
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		archivePath := filepath.Join(directory, "renamed.tar.gz")
		metadataPath := filepath.Join(directory, "firmware_1.0.2.json")
		if err := os.WriteFile(archivePath, artifact, 0o600); err != nil {
			t.Fatal(err)
		}
		writeFirmwareTestMetadata(t, metadataPath, metadata)
		_, err := VerifyFirmwareArtifact(archivePath, metadataPath, keys, FirmwarePolicy{})
		assertKind(t, err, ErrorPolicy)
	})

	types := []struct {
		name     string
		prepare  func(string, string, string)
		expected ErrorKind
	}{
		{
			name: "archive symlink",
			prepare: func(directory, archivePath, metadataPath string) {
				target := filepath.Join(directory, "target.tar.gz")
				if err := os.WriteFile(target, artifact, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, archivePath); err != nil {
					t.Fatal(err)
				}
				writeFirmwareTestMetadata(t, metadataPath, metadata)
			},
			expected: ErrorPolicy,
		},
		{
			name: "archive FIFO",
			prepare: func(_ string, archivePath, metadataPath string) {
				if err := unix.Mkfifo(archivePath, 0o600); err != nil {
					t.Fatal(err)
				}
				writeFirmwareTestMetadata(t, metadataPath, metadata)
			},
			expected: ErrorPolicy,
		},
		{
			name: "archive directory",
			prepare: func(_ string, archivePath, metadataPath string) {
				if err := os.Mkdir(archivePath, 0o700); err != nil {
					t.Fatal(err)
				}
				writeFirmwareTestMetadata(t, metadataPath, metadata)
			},
			expected: ErrorPolicy,
		},
		{
			name: "metadata symlink",
			prepare: func(directory, archivePath, metadataPath string) {
				if err := os.WriteFile(archivePath, artifact, 0o600); err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(directory, "target.json")
				writeFirmwareTestMetadata(t, target, metadata)
				if err := os.Symlink(target, metadataPath); err != nil {
					t.Fatal(err)
				}
			},
			expected: ErrorPolicy,
		},
		{
			name: "metadata FIFO",
			prepare: func(_ string, archivePath, metadataPath string) {
				if err := os.WriteFile(archivePath, artifact, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := unix.Mkfifo(metadataPath, 0o600); err != nil {
					t.Fatal(err)
				}
			},
			expected: ErrorPolicy,
		},
		{
			name: "metadata directory",
			prepare: func(_ string, archivePath, metadataPath string) {
				if err := os.WriteFile(archivePath, artifact, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(metadataPath, 0o700); err != nil {
					t.Fatal(err)
				}
			},
			expected: ErrorPolicy,
		},
	}
	for index, test := range types {
		t.Run(test.name, func(t *testing.T) {
			directory := filepath.Join(root, fmt.Sprintf("type-%02d", index))
			if err := os.Mkdir(directory, 0o700); err != nil {
				t.Fatal(err)
			}
			archivePath := filepath.Join(directory, metadata.Name)
			metadataPath := filepath.Join(directory, "firmware_1.0.2.json")
			test.prepare(directory, archivePath, metadataPath)
			_, err := VerifyFirmwareArtifact(archivePath, metadataPath, keys, FirmwarePolicy{})
			assertKind(t, err, test.expected)
		})
	}

	t.Run("missing files", func(t *testing.T) {
		_, err := VerifyFirmwareArtifact(
			filepath.Join(root, metadata.Name),
			filepath.Join(root, "missing.json"),
			keys,
			FirmwarePolicy{},
		)
		assertKind(t, err, ErrorIO)
	})

	t.Run("missing artifact", func(t *testing.T) {
		directory := filepath.Join(root, "missing-artifact")
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		metadataPath := filepath.Join(directory, "firmware_1.0.2.json")
		writeFirmwareTestMetadata(t, metadataPath, metadata)
		_, err := VerifyFirmwareArtifact(
			filepath.Join(directory, metadata.Name),
			metadataPath,
			keys,
			FirmwarePolicy{},
		)
		assertKind(t, err, ErrorIO)
	})
}

func TestValidateFirmwareArchiveRejectsGzipAndTarFramingTricks(t *testing.T) {
	version := "1.0.2"
	entries := validFirmwareTestEntries(
		[]byte("NanoKVM firmware v1.0.2 test archive\n"),
		map[string][]byte{"firmware/image.bin": []byte("signed firmware image\n")},
	)
	validTar := firmwareTestTar(t, entries)
	validGzip := firmwareTestGzip(t, validTar)
	endOffset := firmwareTestTarEndOffset(t, validTar)

	tests := []struct {
		name       string
		artifact   func() []byte
		errorMatch string
	}{
		{
			name: "gzip trailing bytes",
			artifact: func() []byte {
				return append(append([]byte(nil), validGzip...), 0x01)
			},
			errorMatch: "trailing bytes",
		},
		{
			name: "concatenated gzip members",
			artifact: func() []byte {
				return append(append([]byte(nil), validGzip...), validGzip...)
			},
			errorMatch: "multiple streams",
		},
		{
			name: "truncated gzip",
			artifact: func() []byte {
				return append([]byte(nil), validGzip[:len(validGzip)-4]...)
			},
			errorMatch: "unexpected EOF",
		},
		{
			name: "bad gzip CRC",
			artifact: func() []byte {
				artifact := append([]byte(nil), validGzip...)
				artifact[len(artifact)-8] ^= 1
				return artifact
			},
			errorMatch: "checksum",
		},
		{
			name: "hidden data after tar end",
			artifact: func() []byte {
				payload := append([]byte(nil), validTar...)
				payload = append(payload, make([]byte, firmwareTarBlockBytes)...)
				payload[endOffset+2*firmwareTarBlockBytes] = 1
				return firmwareTestGzip(t, payload)
			},
			errorMatch: "non-zero data",
		},
		{
			name: "second tar after end",
			artifact: func() []byte {
				return firmwareTestGzip(t, append(append([]byte(nil), validTar...), validTar...))
			},
			errorMatch: "non-zero data",
		},
		{
			name: "unaligned zero end padding",
			artifact: func() []byte {
				return firmwareTestGzip(t, append(append([]byte(nil), validTar...), 0))
			},
			errorMatch: "not 512-byte aligned",
		},
		{
			name: "missing second tar end block",
			artifact: func() []byte {
				return firmwareTestGzip(t, validTar[:endOffset+firmwareTarBlockBytes])
			},
			errorMatch: "truncated end-of-archive marker",
		},
		{
			name: "non-zero member padding",
			artifact: func() []byte {
				payload := append([]byte(nil), validTar...)
				body := []byte("signed firmware image\n")
				offset := bytes.Index(payload, body)
				if offset < 0 {
					t.Fatal("locate test member body")
				}
				payload[offset+len(body)] = 1
				return firmwareTestGzip(t, payload)
			},
			errorMatch: "non-zero padding",
		},
		{
			name: "PAX metadata",
			artifact: func() []byte {
				pax := append([]firmwareTestTarEntry(nil), entries...)
				pax = append(pax, firmwareTestTarEntry{
					name:     "firmware/" + strings.Repeat("p", 120),
					typeflag: tar.TypeReg,
					body:     []byte("pax"),
					format:   tar.FormatPAX,
				})
				return firmwareTestGzip(t, firmwareTestTar(t, pax))
			},
			errorMatch: "PAX/GNU",
		},
		{
			name: "GNU LongLink metadata",
			artifact: func() []byte {
				gnu := append([]firmwareTestTarEntry(nil), entries...)
				gnu = append(gnu, firmwareTestTarEntry{
					name:     "firmware/" + strings.Repeat("g", 120),
					typeflag: tar.TypeReg,
					body:     []byte("gnu"),
					format:   tar.FormatGNU,
				})
				return firmwareTestGzip(t, firmwareTestTar(t, gnu))
			},
			errorMatch: "PAX/GNU",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertFirmwareArchiveInvalid(t, test.artifact(), version, test.errorMatch)
		})
	}
}

func TestValidateFirmwareArchiveRejectsUnsafeMembers(t *testing.T) {
	versionData := []byte("NanoKVM firmware v1.0.2 test archive\n")
	base := validFirmwareTestEntries(versionData, nil)
	tests := []struct {
		name       string
		entries    func() []firmwareTestTarEntry
		errorMatch string
	}{
		{
			name: "traversal",
			entries: func() []firmwareTestTarEntry {
				return append(base, firmwareTestTarEntry{name: "firmware/../escape", typeflag: tar.TypeReg})
			},
			errorMatch: "non-canonical",
		},
		{
			name: "backslash",
			entries: func() []firmwareTestTarEntry {
				return append(base, firmwareTestTarEntry{name: `firmware\escape`, typeflag: tar.TypeReg})
			},
			errorMatch: "unsafe",
		},
		{
			name: "duplicate path",
			entries: func() []firmwareTestTarEntry {
				return append(base, firmwareTestTarEntry{name: firmwareVersionPath, typeflag: tar.TypeReg, body: versionData})
			},
			errorMatch: "duplicate",
		},
		{
			name: "file parent conflict",
			entries: func() []firmwareTestTarEntry {
				return []firmwareTestTarEntry{
					{name: "overlay", typeflag: tar.TypeDir},
					{name: "overlay/boot", typeflag: tar.TypeReg},
					{name: firmwareVersionPath, typeflag: tar.TypeReg, body: versionData},
					{name: firmwareB2SumPath, typeflag: tar.TypeReg, body: []byte("invalid\n")},
				}
			},
			errorMatch: "conflicts with descendant",
		},
		{
			name: "symlink",
			entries: func() []firmwareTestTarEntry {
				return append(base, firmwareTestTarEntry{name: "firmware/link", typeflag: tar.TypeSymlink, linkname: "/etc/passwd"})
			},
			errorMatch: "unsupported",
		},
		{
			name: "FIFO",
			entries: func() []firmwareTestTarEntry {
				return append(base, firmwareTestTarEntry{name: "firmware/fifo", typeflag: tar.TypeFifo})
			},
			errorMatch: "unsupported",
		},
		{
			name: "device",
			entries: func() []firmwareTestTarEntry {
				return append(base, firmwareTestTarEntry{name: "firmware/device", typeflag: tar.TypeChar})
			},
			errorMatch: "unsupported",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			artifact := firmwareTestGzip(t, firmwareTestTar(t, test.entries()))
			assertFirmwareArchiveInvalid(t, artifact, "1.0.2", test.errorMatch)
		})
	}
}

func TestValidateFirmwareArchiveRejectsVersionAndB2SumFailures(t *testing.T) {
	version := "1.0.2"
	tests := []struct {
		name       string
		entries    func() []firmwareTestTarEntry
		errorMatch string
	}{
		{
			name: "missing version",
			entries: func() []firmwareTestTarEntry {
				entries := validFirmwareTestEntries([]byte("v1.0.2\n"), nil)
				result := entries[:0]
				for _, entry := range entries {
					if entry.name != firmwareVersionPath {
						result = append(result, entry)
					}
				}
				return result
			},
			errorMatch: "missing overlay/boot/ver",
		},
		{
			name: "missing b2sum",
			entries: func() []firmwareTestTarEntry {
				entries := validFirmwareTestEntries([]byte("v1.0.2\n"), nil)
				return entries[:len(entries)-1]
			},
			errorMatch: "missing b2sum.txt",
		},
		{
			name: "uppercase version prefix",
			entries: func() []firmwareTestTarEntry {
				return validFirmwareTestEntries([]byte("Firmware V1.0.2\n"), nil)
			},
			errorMatch: "lowercase v",
		},
		{
			name: "multiple version tokens",
			entries: func() []firmwareTestTarEntry {
				return validFirmwareTestEntries([]byte("v1.0.2 and v1.0.2\n"), nil)
			},
			errorMatch: "exactly one",
		},
		{
			name: "wrong version token",
			entries: func() []firmwareTestTarEntry {
				return validFirmwareTestEntries([]byte("Firmware v1.0.3\n"), nil)
			},
			errorMatch: "does not match",
		},
		{
			name: "malformed b2sum",
			entries: func() []firmwareTestTarEntry {
				return firmwareTestReplaceB2Sum(t, validFirmwareTestEntries([]byte("v1.0.2\n"), nil), func([]byte) []byte {
					return []byte("malformed\n")
				})
			},
			errorMatch: "invalid checksum line",
		},
		{
			name: "b2sum mismatch",
			entries: func() []firmwareTestTarEntry {
				return firmwareTestReplaceB2Sum(t, validFirmwareTestEntries([]byte("v1.0.2\n"), nil), func(data []byte) []byte {
					data[0] = '0' + (data[0]+1-'0')%10
					return data
				})
			},
			errorMatch: "checksum does not match",
		},
		{
			name: "b2sum missing member",
			entries: func() []firmwareTestTarEntry {
				return firmwareTestReplaceB2Sum(t, validFirmwareTestEntries([]byte("v1.0.2\n"), nil), func([]byte) []byte {
					return []byte(strings.Repeat("0", 128) + "  firmware/missing.bin\n")
				})
			},
			errorMatch: "missing archive member",
		},
		{
			name: "b2sum unknown member",
			entries: func() []firmwareTestTarEntry {
				return firmwareTestReplaceB2Sum(t, validFirmwareTestEntries([]byte("v1.0.2\n"), nil), func(data []byte) []byte {
					return append(data, []byte(strings.Repeat("0", 128)+"  firmware/unknown.bin\n")...)
				})
			},
			errorMatch: "file set",
		},
		{
			name: "duplicate b2sum member",
			entries: func() []firmwareTestTarEntry {
				return firmwareTestReplaceB2Sum(t, validFirmwareTestEntries([]byte("v1.0.2\n"), nil), func(data []byte) []byte {
					return append(data, data...)
				})
			},
			errorMatch: "duplicate path",
		},
		{
			name: "b2sum without final LF",
			entries: func() []firmwareTestTarEntry {
				return firmwareTestReplaceB2Sum(t, validFirmwareTestEntries([]byte("v1.0.2\n"), nil), func(data []byte) []byte {
					return data[:len(data)-1]
				})
			},
			errorMatch: "end with LF",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			artifact := firmwareTestGzip(t, firmwareTestTar(t, test.entries()))
			assertFirmwareArchiveInvalid(t, artifact, version, test.errorMatch)
		})
	}
}

func TestValidateFirmwareArchiveEnforcesResourceLimits(t *testing.T) {
	artifact := validFirmwareTestArchive(t, "1.0.2")
	tests := []struct {
		name       string
		limits     firmwareArchiveLimits
		errorMatch string
	}{
		{
			name: "entry count",
			limits: firmwareArchiveLimits{
				maxEntries: 1, maxFile: firmwareArchiveMaxFileBytes,
				maxExpanded: firmwareArchiveMaxExpanded, maxEnvelope: firmwareArchiveMaxEnvelope,
			},
			errorMatch: "entry count",
		},
		{
			name: "file size",
			limits: firmwareArchiveLimits{
				maxEntries: firmwareArchiveMaxEntries, maxFile: 1,
				maxExpanded: firmwareArchiveMaxExpanded, maxEnvelope: firmwareArchiveMaxEnvelope,
			},
			errorMatch: "file size limit",
		},
		{
			name: "expanded size",
			limits: firmwareArchiveLimits{
				maxEntries: firmwareArchiveMaxEntries, maxFile: firmwareArchiveMaxFileBytes,
				maxExpanded: 1, maxEnvelope: firmwareArchiveMaxEnvelope,
			},
			errorMatch: "expanded size",
		},
		{
			name: "decompressed envelope",
			limits: firmwareArchiveLimits{
				maxEntries: firmwareArchiveMaxEntries, maxFile: firmwareArchiveMaxFileBytes,
				maxExpanded: firmwareArchiveMaxExpanded, maxEnvelope: firmwareTarBlockBytes,
			},
			errorMatch: "decompressed envelope",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateFirmwareGzipArchiveWithLimits(artifact, "1.0.2", test.limits)
			if err == nil || !strings.Contains(err.Error(), test.errorMatch) {
				t.Fatalf("error = %v, want substring %q", err, test.errorMatch)
			}
		})
	}
}
