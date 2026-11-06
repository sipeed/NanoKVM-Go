package main

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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/updateverify"
	"golang.org/x/crypto/blake2b"
)

const cliRFC8032TestSeedHex = "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"

func TestRunBundleVerifiesAndOutputsTrustedCanonicalValues(t *testing.T) {
	dir, keyRingPath := writeCLITestBundle(t, "1.2.3")
	code, stdout, stderr := runCaptured([]string{
		"bundle",
		"--bundle", dir,
		"--keyring", keyRingPath,
		"--requested-version", "1.2.3",
		"--current-version", "2.0.0",
	})
	if code != exitVerified {
		t.Fatalf("bundle exit = %d, stderr = %s", code, stderr)
	}
	assertSingleLineJSON(t, stdout)
	wantStdout := []byte("{\"mode\":\"bundle\",\"version\":\"1.2.3\"}\n")
	if !bytes.Equal(stdout, wantStdout) {
		t.Fatalf("bundle stdout = %q, want %q", stdout, wantStdout)
	}
	var output verifiedBundleOutput
	if err := json.Unmarshal(stdout, &output); err != nil {
		t.Fatalf("decode bundle output: %v: %s", err, stdout)
	}
	if output != (verifiedBundleOutput{Mode: "bundle", Version: "1.2.3"}) {
		t.Fatalf("bundle output = %#v", output)
	}
	if !strings.Contains(string(stderr), "application bundle verified") {
		t.Fatalf("bundle stderr = %q", stderr)
	}
}

func TestRunRejectsRemovedDetachedMetadataModes(t *testing.T) {
	for _, args := range [][]string{
		{"metadata", "--metadata", "latest.json", "--signature", "latest.json.sig", "--keyring", "keys.json"},
		{"artifact", "--artifact", "update.tar.gz", "--metadata", "latest.json", "--signature", "latest.json.sig", "--keyring", "keys.json"},
	} {
		code, stdout, stderr := runCaptured(args)
		if code != exitPolicy || len(stdout) != 0 {
			t.Fatalf("removed mode %q: exit = %d, stdout = %q", args[0], code, stdout)
		}
		if strings.Contains(string(stderr), "metadata|artifact") || strings.Contains(string(stderr), "--signature") {
			t.Fatalf("usage still advertises removed mode: %s", stderr)
		}
	}
}

func TestRunBundlePolicyForbiddenFlagsAndSignatureExit(t *testing.T) {
	t.Run("same-version policy", func(t *testing.T) {
		dir, keyRingPath := writeCLITestBundle(t, "1.2.3")
		base := []string{"bundle", "--bundle", dir, "--keyring", keyRingPath, "--current-version", "1.2.3"}
		if code, stdout, _ := runCaptured(base); code != exitPolicy || len(stdout) != 0 {
			t.Fatalf("same-version exit = %d", code)
		}
		if code, _, stderr := runCaptured(append(append([]string(nil), base...), "--allow-same-version")); code != exitVerified {
			t.Fatalf("allowed same-version exit = %d, stderr = %s", code, stderr)
		}
	})

	t.Run("requested version", func(t *testing.T) {
		dir, keyRingPath := writeCLITestBundle(t, "1.2.3")
		code, stdout, _ := runCaptured([]string{
			"bundle", "--bundle", dir, "--keyring", keyRingPath, "--requested-version", "1.2.4",
		})
		if code != exitPolicy || len(stdout) != 0 {
			t.Fatalf("requested-version exit = %d", code)
		}
	})

	t.Run("same-version override requires current version", func(t *testing.T) {
		dir, keyRingPath := writeCLITestBundle(t, "1.2.3")
		assertExitWithoutStdout(t, []string{
			"bundle", "--bundle", dir, "--keyring", keyRingPath, "--allow-same-version",
		}, exitPolicy)
	})

	t.Run("forbidden legacy flags", func(t *testing.T) {
		dir, keyRingPath := writeCLITestBundle(t, "1.2.3")
		base := []string{"bundle", "--bundle", dir, "--keyring", keyRingPath}
		for _, extra := range [][]string{
			{"--allow-unsigned"},
			{"--signature", "obsolete.sig"},
			{"--metadata", "cross-mode.json"},
			{"--unknown"},
			{"positional"},
			{"--"},
			{"--bundle=" + dir},
			{"--allow-same-version", "--allow-same-version"},
		} {
			args := append(append([]string(nil), base...), extra...)
			if code, stdout, _ := runCaptured(args); code != exitPolicy || len(stdout) != 0 {
				t.Fatalf("forbidden args %v exit = %d", extra, code)
			}
		}
		for _, args := range [][]string{
			{"bundle", "--bundle", dir},
			{"bundle", "--bundle"},
			{"bundle", "--keyring"},
		} {
			if code, stdout, _ := runCaptured(args); code != exitPolicy || len(stdout) != 0 {
				t.Fatalf("invalid args %v exit = %d", args, code)
			}
		}
		if code, stdout, _ := runCaptured([]string{"bundle", "--bundle", dir}); code != exitPolicy || len(stdout) != 0 {
			t.Fatalf("missing keyring exit = %d", code)
		}
	})

	t.Run("signature failure", func(t *testing.T) {
		dir, keyRingPath := writeCLITestBundle(t, "1.2.3")
		path := filepath.Join(dir, "kvmcomm_1.2.3_armhf.deb")
		mutateFirstByte(t, path)
		code, stdout, _ := runCaptured([]string{"bundle", "--bundle", dir, "--keyring", keyRingPath})
		if code != exitSignature || len(stdout) != 0 {
			t.Fatalf("signature failure exit = %d, want %d", code, exitSignature)
		}
	})
}

func TestRunBundleExitCodeMatrix(t *testing.T) {
	t.Run("missing embedded signature", func(t *testing.T) {
		dir, keyRingPath := writeCLITestBundle(t, "1.2.3")
		metadataPath := filepath.Join(dir, "kvmcomm_1.2.3.json")
		deleteJSONField(t, metadataPath, "signature")
		assertExitWithoutStdout(t, []string{
			"bundle", "--bundle", dir, "--keyring", keyRingPath,
		}, exitSignature)
	})

	t.Run("malformed metadata", func(t *testing.T) {
		dir, keyRingPath := writeCLITestBundle(t, "1.2.3")
		write(t, filepath.Join(dir, "kvmcomm_1.2.3.json"), []byte("{"))
		assertExitWithoutStdout(t, []string{
			"bundle", "--bundle", dir, "--keyring", keyRingPath,
		}, exitInvalid)
	})

	t.Run("malformed keyring", func(t *testing.T) {
		dir, keyRingPath := writeCLITestBundle(t, "1.2.3")
		write(t, keyRingPath, []byte("{"))
		assertExitWithoutStdout(t, []string{
			"bundle", "--bundle", dir, "--keyring", keyRingPath,
		}, exitInvalid)
	})

	t.Run("missing bundle path", func(t *testing.T) {
		dir, keyRingPath := writeCLITestBundle(t, "1.2.3")
		assertExitWithoutStdout(t, []string{
			"bundle", "--bundle", filepath.Join(filepath.Dir(dir), "missing-bundle"),
			"--keyring", keyRingPath,
		}, exitIO)
	})

	t.Run("missing keyring path", func(t *testing.T) {
		dir, _ := writeCLITestBundle(t, "1.2.3")
		assertExitWithoutStdout(t, []string{
			"bundle", "--bundle", dir,
			"--keyring", filepath.Join(filepath.Dir(dir), "missing-keyring.json"),
		}, exitIO)
	})
}

func TestRunFirmwareVerifiesAndOutputsTrustedCanonicalValues(t *testing.T) {
	archivePath, metadataPath, keyRingPath, metadata := writeCLITestFirmware(t, "1.0.2")
	code, stdout, stderr := runCaptured([]string{
		"firmware",
		"--artifact", archivePath,
		"--metadata", metadataPath,
		"--keyring", keyRingPath,
		"--requested-version", "1.0.2",
		"--current-version", "2.0.0",
	})
	if code != exitVerified {
		t.Fatalf("Firmware exit = %d, stderr = %s", code, stderr)
	}
	assertSingleLineJSON(t, stdout)
	wantStdout := []byte("{\"mode\":\"firmware\",\"version\":\"1.0.2\",\"name\":\"axera_firmware_v1.0.2.tar.gz\",\"updateUboot\":false}\n")
	if !bytes.Equal(stdout, wantStdout) {
		t.Fatalf("Firmware stdout = %q, want %q", stdout, wantStdout)
	}
	var output verifiedFirmwareOutput
	if err := json.Unmarshal(stdout, &output); err != nil {
		t.Fatalf("decode Firmware output: %v: %s", err, stdout)
	}
	if output != (verifiedFirmwareOutput{
		Mode:        "firmware",
		Version:     metadata.Version,
		Name:        metadata.Name,
		UpdateUboot: false,
	}) {
		t.Fatalf("Firmware output = %#v", output)
	}
	if !strings.Contains(string(stderr), "Firmware artifact verified") {
		t.Fatalf("Firmware stderr = %q", stderr)
	}
}

func TestRunFirmwareOutputsUnsignedUpdateUbootPolicy(t *testing.T) {
	archivePath, metadataPath, keyRingPath, _ := writeCLITestFirmware(t, "1.0.2")
	args := []string{
		"firmware",
		"--artifact", archivePath,
		"--metadata", metadataPath,
		"--keyring", keyRingPath,
	}

	rawMetadata, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(rawMetadata, &document); err != nil {
		t.Fatal(err)
	}
	document["updateUboot"] = json.RawMessage("true")
	write(t, metadataPath, marshal(t, document))

	code, changedOutput, stderr := runCaptured(args)
	if code != exitVerified {
		t.Fatalf("verify updateUboot=true with unchanged signed archive: exit = %d, stderr = %s", code, stderr)
	}
	want := []byte("{\"mode\":\"firmware\",\"version\":\"1.0.2\",\"name\":\"axera_firmware_v1.0.2.tar.gz\",\"updateUboot\":true}\n")
	if !bytes.Equal(changedOutput, want) {
		t.Fatalf("Firmware stdout = %q, want %q", changedOutput, want)
	}
}

func TestRunFirmwarePolicyForbiddenFlagsAndSignatureExit(t *testing.T) {
	t.Run("same and requested version policy", func(t *testing.T) {
		archivePath, metadataPath, keyRingPath, _ := writeCLITestFirmware(t, "1.0.2")
		base := []string{
			"firmware", "--artifact", archivePath, "--metadata", metadataPath,
			"--keyring", keyRingPath, "--current-version", "1.0.2",
		}
		if code, stdout, _ := runCaptured(base); code != exitPolicy || len(stdout) != 0 {
			t.Fatalf("same-version exit = %d", code)
		}
		if code, _, stderr := runCaptured(append(append([]string(nil), base...), "--allow-same-version")); code != exitVerified {
			t.Fatalf("allowed same-version exit = %d, stderr = %s", code, stderr)
		}
		mismatch := []string{
			"firmware", "--artifact", archivePath, "--metadata", metadataPath,
			"--keyring", keyRingPath, "--requested-version", "1.0.3",
		}
		if code, stdout, _ := runCaptured(mismatch); code != exitPolicy || len(stdout) != 0 {
			t.Fatalf("requested-version exit = %d", code)
		}
	})

	t.Run("same-version override requires current version", func(t *testing.T) {
		archivePath, metadataPath, keyRingPath, _ := writeCLITestFirmware(t, "1.0.2")
		assertExitWithoutStdout(t, []string{
			"firmware", "--artifact", archivePath, "--metadata", metadataPath,
			"--keyring", keyRingPath, "--allow-same-version",
		}, exitPolicy)
	})

	t.Run("forbidden legacy flags", func(t *testing.T) {
		archivePath, metadataPath, keyRingPath, _ := writeCLITestFirmware(t, "1.0.2")
		base := []string{
			"firmware", "--artifact", archivePath, "--metadata", metadataPath, "--keyring", keyRingPath,
		}
		for _, extra := range [][]string{
			{"--allow-unsigned"},
			{"--signature", "obsolete.sig"},
			{"--bundle", "cross-mode"},
			{"--unknown"},
			{"positional"},
			{"--"},
			{"--metadata=" + metadataPath},
			{"--allow-same-version", "--allow-same-version"},
		} {
			args := append(append([]string(nil), base...), extra...)
			if code, stdout, _ := runCaptured(args); code != exitPolicy || len(stdout) != 0 {
				t.Fatalf("forbidden args %v exit = %d", extra, code)
			}
		}
		for _, args := range [][]string{
			{"firmware", "--artifact", archivePath, "--metadata", metadataPath},
			{"firmware", "--artifact"},
			{"firmware", "--metadata"},
			{"firmware", "--keyring"},
		} {
			if code, stdout, _ := runCaptured(args); code != exitPolicy || len(stdout) != 0 {
				t.Fatalf("invalid args %v exit = %d", args, code)
			}
		}
		if code, stdout, _ := runCaptured([]string{"firmware", "--artifact", archivePath, "--metadata", metadataPath}); code != exitPolicy || len(stdout) != 0 {
			t.Fatalf("missing keyring exit = %d", code)
		}
	})

	t.Run("signature failure", func(t *testing.T) {
		archivePath, metadataPath, keyRingPath, _ := writeCLITestFirmware(t, "1.0.2")
		mutateFirstByte(t, archivePath)
		code, stdout, _ := runCaptured([]string{
			"firmware", "--artifact", archivePath, "--metadata", metadataPath, "--keyring", keyRingPath,
		})
		if code != exitSignature || len(stdout) != 0 {
			t.Fatalf("signature failure exit = %d, want %d", code, exitSignature)
		}
	})
}

func TestRunFirmwareExitCodeMatrix(t *testing.T) {
	t.Run("missing embedded signature", func(t *testing.T) {
		archivePath, metadataPath, keyRingPath, _ := writeCLITestFirmware(t, "1.0.2")
		deleteJSONField(t, metadataPath, "signature")
		assertExitWithoutStdout(t, []string{
			"firmware", "--artifact", archivePath, "--metadata", metadataPath,
			"--keyring", keyRingPath,
		}, exitSignature)
	})

	t.Run("malformed metadata", func(t *testing.T) {
		archivePath, metadataPath, keyRingPath, _ := writeCLITestFirmware(t, "1.0.2")
		write(t, metadataPath, []byte("{"))
		assertExitWithoutStdout(t, []string{
			"firmware", "--artifact", archivePath, "--metadata", metadataPath,
			"--keyring", keyRingPath,
		}, exitInvalid)
	})

	t.Run("malformed keyring", func(t *testing.T) {
		archivePath, metadataPath, keyRingPath, _ := writeCLITestFirmware(t, "1.0.2")
		write(t, keyRingPath, []byte("{"))
		assertExitWithoutStdout(t, []string{
			"firmware", "--artifact", archivePath, "--metadata", metadataPath,
			"--keyring", keyRingPath,
		}, exitInvalid)
	})

	t.Run("missing artifact path", func(t *testing.T) {
		archivePath, metadataPath, keyRingPath, _ := writeCLITestFirmware(t, "1.0.2")
		assertExitWithoutStdout(t, []string{
			"firmware", "--artifact", filepath.Join(filepath.Dir(archivePath), "missing-firmware.tar.gz"),
			"--metadata", metadataPath, "--keyring", keyRingPath,
		}, exitIO)
	})

	t.Run("missing keyring path", func(t *testing.T) {
		archivePath, metadataPath, _, _ := writeCLITestFirmware(t, "1.0.2")
		assertExitWithoutStdout(t, []string{
			"firmware", "--artifact", archivePath, "--metadata", metadataPath,
			"--keyring", filepath.Join(filepath.Dir(archivePath), "missing-keyring.json"),
		}, exitIO)
	})
}

func TestRunStrictModesRejectEmptyNonBooleanFlagValues(t *testing.T) {
	type strictFlag struct {
		name     string
		value    string
		required bool
	}
	modes := []struct {
		name  string
		flags []strictFlag
	}{
		{
			name: "bundle",
			flags: []strictFlag{
				{name: "bundle", value: "nanokvm-go_1.2.3", required: true},
				{name: "keyring", value: "keys.json", required: true},
				{name: "current-version", value: "1.2.2"},
				{name: "requested-version", value: "1.2.3"},
			},
		},
		{
			name: "firmware",
			flags: []strictFlag{
				{name: "metadata", value: "firmware_1.0.2.json", required: true},
				{name: "artifact", value: "axera_firmware_v1.0.2.tar.gz", required: true},
				{name: "keyring", value: "keys.json", required: true},
				{name: "current-version", value: "1.0.1"},
				{name: "requested-version", value: "1.0.2"},
			},
		},
	}
	for _, mode := range modes {
		for _, target := range mode.flags {
			for _, inline := range []bool{true, false} {
				form := "separate"
				if inline {
					form = "inline"
				}
				t.Run(mode.name+"/"+target.name+"/"+form, func(t *testing.T) {
					args := []string{mode.name}
					for _, candidate := range mode.flags {
						if candidate.name == target.name {
							if inline {
								args = append(args, "--"+candidate.name+"=")
							} else {
								args = append(args, "--"+candidate.name, "")
							}
							continue
						}
						if candidate.required {
							args = append(args, "--"+candidate.name, candidate.value)
						}
					}
					assertExitWithoutStdout(t, args, exitPolicy)
				})
			}
		}
	}
}

func TestReportMapsSignatureToStableIndependentExitCode(t *testing.T) {
	var stderr bytes.Buffer
	if got := report(&stderr, updateverify.ErrorSignature, "signature failed"); got != exitSignature {
		t.Fatalf("signature exit = %d, want %d", got, exitSignature)
	}
	if exitSignature == exitInvalid || exitSignature == exitPolicy || exitSignature == exitIO {
		t.Fatalf("signature exit %d is not independent", exitSignature)
	}
}

func marshal(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}

func write(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func deleteJSONField(t *testing.T, path, field string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	if _, exists := document[field]; !exists {
		t.Fatalf("JSON field %q does not exist in %s", field, path)
	}
	delete(document, field)
	write(t, path, marshal(t, document))
}

func runCaptured(args []string) (int, []byte, []byte) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithIO(args, &stdout, &stderr)
	return code, stdout.Bytes(), stderr.Bytes()
}

func assertExitWithoutStdout(t *testing.T, args []string, want int) {
	t.Helper()
	code, stdout, stderr := runCaptured(args)
	if code != want || len(stdout) != 0 {
		t.Fatalf("run %q: exit = %d, want %d; stdout = %q; stderr = %q", args, code, want, stdout, stderr)
	}
}

func assertSingleLineJSON(t *testing.T, output []byte) {
	t.Helper()
	if len(output) == 0 || output[len(output)-1] != '\n' || bytes.Count(output, []byte{'\n'}) != 1 {
		t.Fatalf("output is not one JSON line: %q", output)
	}
	if !json.Valid(bytes.TrimSuffix(output, []byte{'\n'})) {
		t.Fatalf("output is not strict JSON: %q", output)
	}
}

func cliTestKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, string) {
	t.Helper()
	seed, err := hex.DecodeString(cliRFC8032TestSeedHex)
	if err != nil {
		t.Fatal(err)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := append(ed25519.PublicKey(nil), privateKey.Public().(ed25519.PublicKey)...)
	return publicKey, privateKey, updateverify.PublicKeyID(publicKey)
}

func writeCLIKeyRing(t *testing.T, dir string) (string, ed25519.PrivateKey, string) {
	t.Helper()
	publicKey, privateKey, keyID := cliTestKey(t)
	path := filepath.Join(dir, "keys.json")
	write(t, path, marshal(t, updateverify.KeyRingFile{
		Schema: 1,
		Keys: []updateverify.Key{{
			KeyID: keyID, Algorithm: "ed25519", PublicKey: base64.StdEncoding.EncodeToString(publicKey),
		}},
	}))
	return path, privateKey, keyID
}

func writeCLITestBundle(t *testing.T, version string) (string, string) {
	t.Helper()
	root := t.TempDir()
	bundleDir := filepath.Join(root, "nanokvm-go_"+version)
	if err := os.Mkdir(bundleDir, 0o700); err != nil {
		t.Fatal(err)
	}
	keyRingPath, privateKey, keyID := writeCLIKeyRing(t, root)
	for _, packageName := range []string{"kvmcomm", "nanokvm-go"} {
		debPath := buildCLITestDeb(t, bundleDir, packageName, version)
		content, err := os.ReadFile(debPath)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha512.Sum512(content)
		metadata := updateverify.DebMetadata{
			Version:      version,
			Name:         filepath.Base(debPath),
			SHA512:       base64.StdEncoding.EncodeToString(digest[:]),
			Size:         int64(len(content)),
			Package:      packageName,
			Architecture: "armhf",
			Signature: &updateverify.Signature{
				Schema:    1,
				Algorithm: "ed25519",
				KeyID:     keyID,
				Value:     base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, content)),
			},
		}
		write(t, filepath.Join(bundleDir, packageName+"_"+version+".json"), marshal(t, metadata))
	}
	return bundleDir, keyRingPath
}

func buildCLITestDeb(t *testing.T, outputDir, packageName, version string) string {
	t.Helper()
	root := filepath.Join(outputDir, "."+packageName+"-root")
	controlDir := filepath.Join(root, "DEBIAN")
	if err := os.MkdirAll(controlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(controlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	control := fmt.Sprintf(
		"Package: %s\nVersion: %s\nArchitecture: armhf\nMaintainer: NanoKVM Test <test@example.invalid>\nDescription: test-only package\n",
		packageName,
		version,
	)
	write(t, filepath.Join(controlDir, "control"), []byte(control))
	payload := filepath.Join(root, "usr", "share", "nanokvm-test", packageName)
	if err := os.MkdirAll(filepath.Dir(payload), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, payload, []byte("test-only\n"))
	debPath := filepath.Join(outputDir, packageName+"_"+version+"_armhf.deb")
	command := exec.Command("/usr/bin/dpkg-deb", "--build", "--root-owner-group", root, debPath)
	command.Env = append(os.Environ(), "LANG=C", "LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build test-only Debian package: %v: %s", err, output)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	return debPath
}

func writeCLITestFirmware(t *testing.T, version string) (string, string, string, updateverify.FirmwareMetadata) {
	t.Helper()
	dir := t.TempDir()
	keyRingPath, privateKey, keyID := writeCLIKeyRing(t, dir)
	versionData := []byte("NanoKVM firmware v" + version + "\n")
	digest := blake2b.Sum512(versionData)
	b2sum := []byte(fmt.Sprintf("%x  overlay/boot/ver\n", digest))
	archive := makeCLIFirmwareArchive(t, versionData, b2sum)
	archiveName := "axera_firmware_v" + version + ".tar.gz"
	archivePath := filepath.Join(dir, archiveName)
	write(t, archivePath, archive)
	sha := sha512.Sum512(archive)
	updateUboot := false
	metadata := updateverify.FirmwareMetadata{
		Version:     version,
		Name:        archiveName,
		SHA512:      base64.StdEncoding.EncodeToString(sha[:]),
		Size:        int64(len(archive)),
		UpdateUboot: &updateUboot,
		Signature: &updateverify.Signature{
			Schema:    1,
			Algorithm: "ed25519",
			KeyID:     keyID,
			Value:     base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, archive)),
		},
	}
	metadataPath := filepath.Join(dir, "firmware_"+version+".json")
	write(t, metadataPath, marshal(t, metadata))
	return archivePath, metadataPath, keyRingPath, metadata
}

func makeCLIFirmwareArchive(t *testing.T, versionData, b2sum []byte) []byte {
	t.Helper()
	var tarBuffer bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuffer)
	entries := []struct {
		name     string
		typeflag byte
		mode     int64
		body     []byte
	}{
		{name: "overlay", typeflag: tar.TypeDir, mode: 0o755},
		{name: "overlay/boot", typeflag: tar.TypeDir, mode: 0o755},
		{name: "overlay/boot/ver", typeflag: tar.TypeReg, mode: 0o644, body: versionData},
		{name: "b2sum.txt", typeflag: tar.TypeReg, mode: 0o644, body: b2sum},
	}
	for _, entry := range entries {
		header := &tar.Header{
			Name: entry.name, Typeflag: entry.typeflag, Mode: entry.mode,
			Size: int64(len(entry.body)), ModTime: time.Unix(0, 0),
			Uid: 0, Gid: 0, Uname: "root", Gname: "root", Format: tar.FormatUSTAR,
		}
		if entry.typeflag == tar.TypeDir {
			header.Size = 0
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if len(entry.body) > 0 {
			if _, err := tarWriter.Write(entry.body); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	var compressed bytes.Buffer
	gzipWriter, err := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter.Header.ModTime = time.Unix(0, 0)
	gzipWriter.Header.OS = 255
	if _, err := gzipWriter.Write(tarBuffer.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return compressed.Bytes()
}

func mutateFirstByte(t *testing.T, path string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	var first [1]byte
	if _, err := file.ReadAt(first[:], 0); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	first[0] ^= 1
	if _, err := file.WriteAt(first[:], 0); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
