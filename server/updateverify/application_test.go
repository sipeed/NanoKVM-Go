package updateverify

import (
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type fixtureDebInspector struct {
	controls map[string]debControl
	err      error
	hook     func(*os.File) error
}

func (inspector fixtureDebInspector) Inspect(file *os.File) (debControl, error) {
	if inspector.hook != nil {
		if err := inspector.hook(file); err != nil {
			return debControl{}, err
		}
	}
	if inspector.err != nil {
		return debControl{}, inspector.err
	}
	control, ok := inspector.controls[filepath.Base(file.Name())]
	if !ok {
		return debControl{}, invalid("fixture has no Debian control metadata", nil)
	}
	return control, nil
}

func TestVerifyDebPackageAcceptsStrictEmbeddedMetadata(t *testing.T) {
	dir := t.TempDir()
	debPath, metadataPath, _, keys := writeSignedDebFixture(
		t, dir, "kvmcomm", "1.2.3", applicationArchitecture, []byte("test-only deb bytes"),
	)
	inspector := fixtureInspectorFor("kvmcomm", "1.2.3", applicationArchitecture)
	verified, err := verifyDebPackage(
		debPath,
		metadataPath,
		keys,
		DebPolicy{
			ExpectedPackage:      "kvmcomm",
			ExpectedVersion:      "1.2.3",
			ExpectedArchitecture: applicationArchitecture,
		},
		inspector,
	)
	if err != nil {
		t.Fatalf("verify Debian package: %v", err)
	}
	if verified.Package != "kvmcomm" || verified.Version != "1.2.3" || verified.Architecture != applicationArchitecture {
		t.Fatalf("verified package = %#v", verified)
	}
	if verified.Path != debPath || verified.MetadataPath != metadataPath {
		t.Fatalf("verified paths = %#v", verified)
	}
}

func TestVerifyDebPackageRejectsMalformedMetadataAndTampering(t *testing.T) {
	tests := []struct {
		name   string
		kind   ErrorKind
		mutate func(t *testing.T, debPath, metadataPath string, metadata DebMetadata)
	}{
		{
			name: "missing signature",
			kind: ErrorSignature,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				writeJSONMap(t, metadataPath, map[string]any{
					"version": metadata.Version, "name": metadata.Name, "sha512": metadata.SHA512,
					"size": metadata.Size, "package": metadata.Package, "architecture": metadata.Architecture,
				})
			},
		},
		{
			name: "null signature",
			kind: ErrorSignature,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				metadata.Signature = nil
				writeJSONFile(t, metadataPath, metadata)
			},
		},
		{
			name: "invalid signature Base64",
			kind: ErrorSignature,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				metadata.Signature.Value = "***"
				writeJSONFile(t, metadataPath, metadata)
			},
		},
		{
			name: "unknown key",
			kind: ErrorSignature,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				metadata.Signature.KeyID = "sha256:" + strings.Repeat("0", 64)
				writeJSONFile(t, metadataPath, metadata)
			},
		},
		{
			name: "invalid UTF-8",
			kind: ErrorInvalid,
			mutate: func(t *testing.T, _, metadataPath string, _ DebMetadata) {
				if err := os.WriteFile(metadataPath, []byte{'{', 0xff, '}'}, 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "unknown field",
			kind: ErrorInvalid,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				raw := marshalJSON(t, metadata)
				raw = append(raw[:len(raw)-1], []byte(`,"extra":true}`)...)
				writeRawMetadata(t, metadataPath, raw)
			},
		},
		{
			name: "duplicate nested field",
			kind: ErrorInvalid,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				raw := marshalJSON(t, metadata)
				needle := []byte(`"schema":1`)
				raw = []byte(strings.Replace(string(raw), string(needle), `"schema":1,"schema":1`, 1))
				writeRawMetadata(t, metadataPath, raw)
			},
		},
		{
			name: "trailing JSON value",
			kind: ErrorInvalid,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				writeRawMetadata(t, metadataPath, append(marshalJSON(t, metadata), []byte("\n{}")...))
			},
		},
		{
			name: "wrong JSON type",
			kind: ErrorInvalid,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				writeJSONMap(t, metadataPath, map[string]any{
					"version": metadata.Version, "name": metadata.Name, "sha512": metadata.SHA512,
					"size": "1", "package": metadata.Package, "architecture": metadata.Architecture,
					"signature": metadata.Signature,
				})
			},
		},
		{
			name: "non-canonical version",
			kind: ErrorInvalid,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				metadata.Version = "01.2.3"
				writeJSONFile(t, metadataPath, metadata)
			},
		},
		{
			name: "different metadata version",
			kind: ErrorPolicy,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				metadata.Version = "1.2.4"
				writeJSONFile(t, metadataPath, metadata)
			},
		},
		{
			name: "different metadata package",
			kind: ErrorPolicy,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				metadata.Package = "nanokvm-go"
				writeJSONFile(t, metadataPath, metadata)
			},
		},
		{
			name: "different metadata architecture",
			kind: ErrorPolicy,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				metadata.Architecture = "amd64"
				writeJSONFile(t, metadataPath, metadata)
			},
		},
		{
			name: "wrong metadata name",
			kind: ErrorPolicy,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				metadata.Name = "other_1.2.3_armhf.deb"
				writeJSONFile(t, metadataPath, metadata)
			},
		},
		{
			name: "wrong size",
			kind: ErrorPolicy,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				metadata.Size++
				writeJSONFile(t, metadataPath, metadata)
			},
		},
		{
			name: "wrong digest",
			kind: ErrorPolicy,
			mutate: func(t *testing.T, _, metadataPath string, metadata DebMetadata) {
				metadata.SHA512 = base64.StdEncoding.EncodeToString(make([]byte, sha512.Size))
				writeJSONFile(t, metadataPath, metadata)
			},
		},
		{
			name: "tampered Debian package",
			kind: ErrorSignature,
			mutate: func(t *testing.T, debPath, _ string, _ DebMetadata) {
				content, err := os.ReadFile(debPath)
				if err != nil {
					t.Fatal(err)
				}
				content[len(content)-1] ^= 1
				if err := os.WriteFile(debPath, content, 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			debPath, metadataPath, metadata, keys := writeSignedDebFixture(
				t, dir, "kvmcomm", "1.2.3", applicationArchitecture, []byte("test-only deb bytes"),
			)
			test.mutate(t, debPath, metadataPath, metadata)
			_, err := verifyDebPackage(
				debPath,
				metadataPath,
				keys,
				DebPolicy{ExpectedPackage: "kvmcomm", ExpectedVersion: "1.2.3", ExpectedArchitecture: applicationArchitecture},
				fixtureInspectorFor("kvmcomm", "1.2.3", applicationArchitecture),
			)
			assertKind(t, err, test.kind)
		})
	}
}

func TestVerifyDebPackageRejectsUnsafeFileBoundaries(t *testing.T) {
	for _, target := range []string{"deb", "metadata"} {
		for _, kind := range []string{"symlink", "fifo", "directory"} {
			t.Run(target+"/"+kind, func(t *testing.T) {
				dir := t.TempDir()
				debPath, metadataPath, _, keys := writeSignedDebFixture(
					t, dir, "kvmcomm", "1.2.3", applicationArchitecture, []byte("test-only deb bytes"),
				)
				path := debPath
				if target == "metadata" {
					path = metadataPath
				}
				original := path + ".original"
				if err := os.Rename(path, original); err != nil {
					t.Fatal(err)
				}
				switch kind {
				case "symlink":
					if err := os.Symlink(original, path); err != nil {
						t.Fatal(err)
					}
				case "fifo":
					if err := unix.Mkfifo(path, 0o600); err != nil {
						t.Fatal(err)
					}
				case "directory":
					if err := os.Mkdir(path, 0o700); err != nil {
						t.Fatal(err)
					}
				}
				_, err := verifyDebPackage(
					debPath,
					metadataPath,
					keys,
					DebPolicy{ExpectedPackage: "kvmcomm", ExpectedVersion: "1.2.3", ExpectedArchitecture: applicationArchitecture},
					fixtureInspectorFor("kvmcomm", "1.2.3", applicationArchitecture),
				)
				assertKind(t, err, ErrorPolicy)
			})
		}
	}
}

func TestVerifyDebPackageChecksControlPolicyAndPostCryptoMutation(t *testing.T) {
	tests := []struct {
		name      string
		inspector fixtureDebInspector
		kind      ErrorKind
	}{
		{name: "wrong Package", inspector: fixtureInspectorFor("other", "1.2.3", applicationArchitecture), kind: ErrorPolicy},
		{name: "wrong Version", inspector: fixtureInspectorFor("kvmcomm", "1.2.4", applicationArchitecture), kind: ErrorPolicy},
		{name: "wrong Architecture", inspector: fixtureInspectorFor("kvmcomm", "1.2.3", "amd64"), kind: ErrorPolicy},
		{name: "invalid Debian archive", inspector: fixtureDebInspector{err: invalid("invalid Debian archive", nil)}, kind: ErrorInvalid},
		{
			name: "mutation after signature verification",
			inspector: fixtureDebInspector{
				controls: fixtureInspectorFor("kvmcomm", "1.2.3", applicationArchitecture).controls,
				hook: func(file *os.File) error {
					// 判据是 stat identity，尺寸变化是确定性可检出的通道。
					// 等长改写 + 同一 tick 属已接受不可见边界（见 .trellis/spec/backend/
					// device-and-system-boundaries.md 的 "Scenario: updateverify 构件稳定性判定边界"），
					// 不在本用例覆盖。
					return os.WriteFile(file.Name(), []byte("mutated deb bytes with a different length"), 0o600)
				},
			},
			kind: ErrorPolicy,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			debPath, metadataPath, _, keys := writeSignedDebFixture(
				t, dir, "kvmcomm", "1.2.3", applicationArchitecture, []byte("test-only deb bytes"),
			)
			_, err := verifyDebPackage(
				debPath,
				metadataPath,
				keys,
				DebPolicy{ExpectedPackage: "kvmcomm", ExpectedVersion: "1.2.3", ExpectedArchitecture: applicationArchitecture},
				test.inspector,
			)
			assertKind(t, err, test.kind)
		})
	}
}

func TestVerifyApplicationBundleExactSetOrderAndVersionPolicy(t *testing.T) {
	dir, keys, inspector := writeSignedApplicationBundle(t, "1.2.3")
	verified, err := verifyApplicationBundle(
		dir,
		keys,
		ApplicationPolicy{RequestedVersion: "1.2.3", CurrentVersion: "2.0.0"},
		inspector,
	)
	if err != nil {
		t.Fatalf("verify lower-version bundle: %v", err)
	}
	if verified.Version != "1.2.3" || len(verified.Packages) != 2 {
		t.Fatalf("verified bundle = %#v", verified)
	}
	if verified.Packages[0].Package != "kvmcomm" || verified.Packages[1].Package != "nanokvm-go" {
		t.Fatalf("package order = %q, %q", verified.Packages[0].Package, verified.Packages[1].Package)
	}

	_, err = verifyApplicationBundle(dir, keys, ApplicationPolicy{CurrentVersion: "1.2.3"}, inspector)
	assertKind(t, err, ErrorPolicy)
	if _, err := verifyApplicationBundle(
		dir,
		keys,
		ApplicationPolicy{CurrentVersion: "1.2.3", AllowSameVersion: true},
		inspector,
	); err != nil {
		t.Fatalf("verify explicitly allowed same version: %v", err)
	}
	_, err = verifyApplicationBundle(dir, keys, ApplicationPolicy{RequestedVersion: "1.2.4"}, inspector)
	assertKind(t, err, ErrorPolicy)
}

func TestVerifyApplicationBundleSignatureVerificationPolicy(t *testing.T) {
	const version = "1.2.3"
	packageNames := []string{"kvmcomm", "nanokvm-go"}

	t.Run("default accepts valid Ed25519 signatures", func(t *testing.T) {
		dir, keys, inspector := writeSignedApplicationBundle(t, version)
		verified, err := verifyApplicationBundle(dir, keys, ApplicationPolicy{}, inspector)
		if err != nil {
			t.Fatalf("verify signed application bundle: %v", err)
		}
		if len(verified.Packages) != len(packageNames) {
			t.Fatalf("verified package count = %d, want %d", len(verified.Packages), len(packageNames))
		}
	})

	for _, packageName := range packageNames {
		packageName := packageName
		t.Run("default rejects missing signature for "+packageName, func(t *testing.T) {
			dir, keys, inspector := writeSignedApplicationBundle(t, version)
			mutateApplicationDebMetadata(t, dir, packageName, version, func(metadata *DebMetadata) {
				metadata.Signature = nil
			})

			_, err := verifyApplicationBundle(dir, keys, ApplicationPolicy{}, inspector)
			assertKind(t, err, ErrorSignature)
		})

		t.Run("default rejects incorrect signature for "+packageName, func(t *testing.T) {
			dir, keys, inspector := writeSignedApplicationBundle(t, version)
			mutateApplicationDebMetadata(t, dir, packageName, version, func(metadata *DebMetadata) {
				metadata.Signature.Value = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
			})

			_, err := verifyApplicationBundle(dir, keys, ApplicationPolicy{}, inspector)
			assertKind(t, err, ErrorSignature)
		})
	}

	t.Run("skip permits missing and incorrect signatures for both packages", func(t *testing.T) {
		dir, _, inspector := writeSignedApplicationBundle(t, version)
		mutateApplicationDebMetadata(t, dir, "kvmcomm", version, func(metadata *DebMetadata) {
			metadata.Signature = nil
		})
		mutateApplicationDebMetadata(t, dir, "nanokvm-go", version, func(metadata *DebMetadata) {
			metadata.Signature.Value = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
		})

		verified, err := verifyApplicationBundle(
			dir,
			nil,
			ApplicationPolicy{SkipSignatureVerification: true},
			inspector,
		)
		if err != nil {
			t.Fatalf("verify application bundle without signatures: %v", err)
		}
		if len(verified.Packages) != len(packageNames) ||
			verified.Packages[0].Package != packageNames[0] ||
			verified.Packages[1].Package != packageNames[1] {
			t.Fatalf("verified packages = %#v", verified.Packages)
		}
	})
}

func TestVerifyApplicationBundleSkipSignatureStillEnforcesIntegrityIdentityAndStructure(t *testing.T) {
	const version = "1.2.3"
	packageNames := []string{"kvmcomm", "nanokvm-go"}
	tests := []struct {
		name   string
		mutate func(*DebMetadata)
	}{
		{
			name: "SHA-512",
			mutate: func(metadata *DebMetadata) {
				metadata.SHA512 = base64.StdEncoding.EncodeToString(make([]byte, sha512.Size))
			},
		},
		{
			name: "size",
			mutate: func(metadata *DebMetadata) {
				metadata.Size++
			},
		},
		{
			name: "package identity",
			mutate: func(metadata *DebMetadata) {
				if metadata.Package == "kvmcomm" {
					metadata.Package = "nanokvm-go"
				} else {
					metadata.Package = "kvmcomm"
				}
			},
		},
	}

	for _, packageName := range packageNames {
		packageName := packageName
		for _, test := range tests {
			test := test
			t.Run(packageName+" rejects wrong "+test.name, func(t *testing.T) {
				dir, _, inspector := writeSignedApplicationBundle(t, version)
				mutateApplicationDebMetadata(t, dir, packageName, version, func(metadata *DebMetadata) {
					metadata.Signature = nil
					test.mutate(metadata)
				})

				_, err := verifyApplicationBundle(
					dir,
					nil,
					ApplicationPolicy{SkipSignatureVerification: true},
					inspector,
				)
				assertKind(t, err, ErrorPolicy)
			})
		}
	}

	t.Run("rejects an unexpected bundle entry", func(t *testing.T) {
		dir, _, inspector := writeSignedApplicationBundle(t, version)
		if err := os.WriteFile(filepath.Join(dir, "unexpected"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}

		_, err := verifyApplicationBundle(
			dir,
			nil,
			ApplicationPolicy{SkipSignatureVerification: true},
			inspector,
		)
		assertKind(t, err, ErrorPolicy)
	})
}

func TestVerifyApplicationBundleRejectsMissingExtraAndUnsafeEntries(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, dir string)
	}{
		{
			name: "missing metadata",
			mutate: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.Remove(filepath.Join(dir, "kvmcomm_1.2.3.json")); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "extra file",
			mutate: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "extra"), []byte("x"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "expected file is symlink",
			mutate: func(t *testing.T, dir string) {
				t.Helper()
				path := filepath.Join(dir, "kvmcomm_1.2.3.json")
				target := path + ".target"
				if err := os.Rename(path, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(target); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir, keys, inspector := writeSignedApplicationBundle(t, "1.2.3")
			test.mutate(t, dir)
			_, err := verifyApplicationBundle(dir, keys, ApplicationPolicy{}, inspector)
			assertKind(t, err, ErrorPolicy)
		})
	}
}

func TestVerifyApplicationBundleRejectsNonCanonicalDirectoryVersion(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"bundle_1.2.3", "nanokvm-go_v1.2.3", "nanokvm-go_01.2.3"} {
		dir := filepath.Join(root, name)
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		_, err := verifyApplicationBundle(dir, nil, ApplicationPolicy{}, fixtureDebInspector{})
		assertKind(t, err, ErrorPolicy)
	}
}

func TestVerifyApplicationBundleRejectsSignedCrossPackageMixes(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*DebMetadata)
		control debControl
	}{
		{
			name: "different package version",
			mutate: func(metadata *DebMetadata) {
				metadata.Version = "1.2.4"
				metadata.Name = debPackageName("nanokvm-go", "1.2.4", applicationArchitecture)
			},
			control: debControl{Package: "nanokvm-go", Version: "1.2.4", Architecture: applicationArchitecture},
		},
		{
			name: "different architecture",
			mutate: func(metadata *DebMetadata) {
				metadata.Architecture = "amd64"
				metadata.Name = debPackageName("nanokvm-go", "1.2.3", "amd64")
			},
			control: debControl{Package: "nanokvm-go", Version: "1.2.3", Architecture: "amd64"},
		},
		{
			name: "duplicate package identity",
			mutate: func(metadata *DebMetadata) {
				metadata.Package = "kvmcomm"
				metadata.Name = debPackageName("kvmcomm", "1.2.3", applicationArchitecture)
			},
			control: debControl{Package: "kvmcomm", Version: "1.2.3", Architecture: applicationArchitecture},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir, keys, inspector := writeSignedApplicationBundle(t, "1.2.3")
			metadataPath := filepath.Join(dir, debMetadataName("nanokvm-go", "1.2.3"))
			raw, err := os.ReadFile(metadataPath)
			if err != nil {
				t.Fatal(err)
			}
			var metadata DebMetadata
			if err := json.Unmarshal(raw, &metadata); err != nil {
				t.Fatal(err)
			}
			test.mutate(&metadata)
			writeJSONFile(t, metadataPath, metadata)
			inspector.controls[debPackageName("nanokvm-go", "1.2.3", applicationArchitecture)] = test.control

			_, err = verifyApplicationBundle(dir, keys, ApplicationPolicy{}, inspector)
			assertKind(t, err, ErrorPolicy)
		})
	}
}

func TestVerifyDebPackageWithRealDPKGAndSameFD(t *testing.T) {
	dir := t.TempDir()
	debPath := buildRealTestDeb(t, dir, "kvmcomm", "1.2.3", applicationArchitecture)
	metadataPath, keys := signExistingDeb(t, debPath, "kvmcomm", "1.2.3", applicationArchitecture)
	verified, err := VerifyDebPackage(
		debPath,
		metadataPath,
		keys,
		DebPolicy{ExpectedPackage: "kvmcomm", ExpectedVersion: "1.2.3", ExpectedArchitecture: applicationArchitecture},
	)
	if err != nil {
		t.Fatalf("verify real Debian package: %v", err)
	}
	if verified.Package != "kvmcomm" {
		t.Fatalf("verified Package = %q", verified.Package)
	}

	snapshot, err := openArtifactSnapshot(debPath, "Debian package", 0, maxDebPackageBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = snapshot.Close() }()
	if err := os.Remove(debPath); err != nil {
		t.Fatal(err)
	}
	control, err := (dpkgDebControlInspector{}).Inspect(snapshot.File())
	if err != nil {
		t.Fatalf("inspect deleted pathname through inherited fd: %v", err)
	}
	if control.Package != "kvmcomm" || control.Version != "1.2.3" || control.Architecture != applicationArchitecture {
		t.Fatalf("control = %#v", control)
	}
}

func TestVerifyDebPackageRealDPKGRejectsSignedNonDeb(t *testing.T) {
	dir := t.TempDir()
	debPath, metadataPath, _, keys := writeSignedDebFixture(
		t, dir, "kvmcomm", "1.2.3", applicationArchitecture, []byte("signed but not a Debian archive"),
	)
	_, err := VerifyDebPackage(
		debPath,
		metadataPath,
		keys,
		DebPolicy{ExpectedPackage: "kvmcomm", ExpectedVersion: "1.2.3", ExpectedArchitecture: applicationArchitecture},
	)
	assertKind(t, err, ErrorInvalid)
}

func TestVerifyApplicationBundleWithRealDPKG(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, applicationBundlePrefix+"1.2.3")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	var keys KeyRing
	for _, packageName := range []string{"kvmcomm", "nanokvm-go"} {
		debPath := buildRealTestDeb(t, dir, packageName, "1.2.3", applicationArchitecture)
		_, packageKeys := signExistingDeb(t, debPath, packageName, "1.2.3", applicationArchitecture)
		keys = packageKeys
	}
	verified, err := VerifyApplicationBundle(
		dir,
		keys,
		ApplicationPolicy{RequestedVersion: "1.2.3", CurrentVersion: "2.0.0"},
	)
	if err != nil {
		t.Fatalf("verify real application bundle: %v", err)
	}
	if verified.Version != "1.2.3" || len(verified.Packages) != 2 {
		t.Fatalf("verified bundle = %#v", verified)
	}
}

func writeSignedApplicationBundle(t *testing.T, version string) (string, KeyRing, fixtureDebInspector) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, applicationBundlePrefix+version)
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	controls := make(map[string]debControl, 2)
	var keys KeyRing
	for _, packageName := range []string{"kvmcomm", "nanokvm-go"} {
		debPath, _, _, packageKeys := writeSignedDebFixture(
			t,
			dir,
			packageName,
			version,
			applicationArchitecture,
			[]byte(packageName+"-test-only-deb"),
		)
		keys = packageKeys
		controls[filepath.Base(debPath)] = debControl{Package: packageName, Version: version, Architecture: applicationArchitecture}
	}
	return dir, keys, fixtureDebInspector{controls: controls}
}

func mutateApplicationDebMetadata(
	t *testing.T,
	dir, packageName, version string,
	mutate func(*DebMetadata),
) {
	t.Helper()
	metadataPath := filepath.Join(dir, debMetadataName(packageName, version))
	raw, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	var metadata DebMetadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		t.Fatal(err)
	}
	mutate(&metadata)
	writeJSONFile(t, metadataPath, metadata)
}

func writeSignedDebFixture(
	t *testing.T,
	dir, packageName, version, architecture string,
	content []byte,
) (string, string, DebMetadata, KeyRing) {
	t.Helper()
	_, privateKey, keyID, keys := testOnlyEd25519Key(t)
	debPath := filepath.Join(dir, debPackageName(packageName, version, architecture))
	if err := os.WriteFile(debPath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	// Back-date the fixture so a same-size in-place mutation during
	// verification is guaranteed to move mtime even when the coarse-grained
	// file timestamp clock has not ticked since the file was created.
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(debPath, past, past); err != nil {
		t.Fatal(err)
	}
	digest := sha512.Sum512(content)
	metadata := DebMetadata{
		Version:      version,
		Name:         filepath.Base(debPath),
		SHA512:       base64.StdEncoding.EncodeToString(digest[:]),
		Size:         int64(len(content)),
		Package:      packageName,
		Architecture: architecture,
		Signature: &Signature{
			Schema:    1,
			Algorithm: "ed25519",
			KeyID:     keyID,
			Value:     base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, content)),
		},
	}
	metadataPath := filepath.Join(dir, debMetadataName(packageName, version))
	writeJSONFile(t, metadataPath, metadata)
	return debPath, metadataPath, metadata, keys
}

func fixtureInspectorFor(packageName, version, architecture string) fixtureDebInspector {
	name := debPackageName("kvmcomm", "1.2.3", applicationArchitecture)
	return fixtureDebInspector{controls: map[string]debControl{
		name: {Package: packageName, Version: version, Architecture: architecture},
	}}
}

func buildRealTestDeb(t *testing.T, dir, packageName, version, architecture string) string {
	t.Helper()
	root := filepath.Join(dir, packageName+"-root")
	controlDir := filepath.Join(root, "DEBIAN")
	if err := os.MkdirAll(controlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	control := fmt.Sprintf(
		"Package: %s\nVersion: %s\nArchitecture: %s\nMaintainer: NanoKVM Test <test.invalid>\nDescription: test-only package\n",
		packageName,
		version,
		architecture,
	)
	if err := os.WriteFile(filepath.Join(controlDir, "control"), []byte(control), 0o600); err != nil {
		t.Fatal(err)
	}
	payload := filepath.Join(root, "usr", "share", "nanokvm-test", packageName)
	if err := os.MkdirAll(filepath.Dir(payload), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(payload, []byte("test-only\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	debPath := filepath.Join(dir, debPackageName(packageName, version, architecture))
	command := exec.Command(dpkgDebPath, "--build", "--root-owner-group", root, debPath)
	command.Env = []string{"LANG=C", "LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build test-only Debian package: %v: %s", err, output)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("remove test-only Debian package root: %v", err)
	}
	return debPath
}

func signExistingDeb(t *testing.T, debPath, packageName, version, architecture string) (string, KeyRing) {
	t.Helper()
	content, err := os.ReadFile(debPath)
	if err != nil {
		t.Fatal(err)
	}
	_, metadataPath, _, keys := writeSignedDebFixture(
		t,
		filepath.Dir(debPath),
		packageName,
		version,
		architecture,
		content,
	)
	return metadataPath, keys
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	writeRawMetadata(t, path, append(marshalJSON(t, value), '\n'))
}

func writeJSONMap(t *testing.T, path string, value map[string]any) {
	t.Helper()
	writeJSONFile(t, path, value)
}

func marshalJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeRawMetadata(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
