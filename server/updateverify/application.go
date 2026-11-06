package updateverify

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

const (
	applicationArchitecture  = "armhf"
	applicationBundlePrefix  = "nanokvm-go_"
	dpkgDebPath              = "/usr/bin/dpkg-deb"
	dpkgDebFDPath            = "/proc/self/fd/3"
	maxDebMetadataBytes      = int64(64 << 10)
	maxDebPackageBytes       = int64(256 << 20)
	maxDPKGFieldOutputBytes  = 4096
	dpkgDebInspectionTimeout = 15 * time.Second
)

var applicationVersionPattern = regexp.MustCompile(`^(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)$`)

type DebMetadata struct {
	Version      string     `json:"version"`
	Name         string     `json:"name"`
	SHA512       string     `json:"sha512"`
	Size         int64      `json:"size"`
	Package      string     `json:"package"`
	Architecture string     `json:"architecture"`
	Signature    *Signature `json:"signature"`
}

type DebPolicy struct {
	ExpectedPackage           string
	ExpectedVersion           string
	ExpectedArchitecture      string
	SkipSignatureVerification bool
}

type VerifiedDebPackage struct {
	Path         string
	MetadataPath string
	Package      string
	Version      string
	Architecture string

	debIdentity      artifactIdentity
	metadataIdentity artifactIdentity
}

type ApplicationPolicy struct {
	RequestedVersion          string
	CurrentVersion            string
	AllowSameVersion          bool
	SkipSignatureVerification bool
}

type VerifiedApplicationBundle struct {
	Version string
	// Packages is always ordered for installation: kvmcomm, then nanokvm-go.
	Packages []VerifiedDebPackage
}

type debControl struct {
	Package      string
	Version      string
	Architecture string
}

type debControlInspector interface {
	Inspect(*os.File) (debControl, error)
}

type dpkgDebControlInspector struct{}

func VerifyDebPackage(debPath, metadataPath string, keys KeyRing, policy DebPolicy) (VerifiedDebPackage, error) {
	return verifyDebPackage(debPath, metadataPath, keys, policy, dpkgDebControlInspector{})
}

func verifyDebPackage(
	debPath, metadataPath string,
	keys KeyRing,
	policy DebPolicy,
	inspector debControlInspector,
) (VerifiedDebPackage, error) {
	if err := validateDebPolicy(policy); err != nil {
		return VerifiedDebPackage{}, err
	}
	expectedDebName := debPackageName(policy.ExpectedPackage, policy.ExpectedVersion, policy.ExpectedArchitecture)
	expectedMetadataName := debMetadataName(policy.ExpectedPackage, policy.ExpectedVersion)
	if filepath.Base(debPath) != expectedDebName {
		return VerifiedDebPackage{}, policyError("Debian package filename does not match policy", nil)
	}
	if filepath.Base(metadataPath) != expectedMetadataName {
		return VerifiedDebPackage{}, policyError("Debian package metadata filename does not match policy", nil)
	}

	initialDebIdentity, err := requireRegularPath(debPath, "Debian package")
	if err != nil {
		return VerifiedDebPackage{}, err
	}
	initialMetadataIdentity, err := requireRegularPath(metadataPath, "Debian package metadata")
	if err != nil {
		return VerifiedDebPackage{}, err
	}

	metadataSnapshot, err := openArtifactSnapshot(
		metadataPath,
		"Debian package metadata",
		0,
		maxDebMetadataBytes,
	)
	if err != nil {
		return VerifiedDebPackage{}, err
	}
	if metadataSnapshot.identity != initialMetadataIdentity {
		_ = metadataSnapshot.Close()
		return VerifiedDebPackage{}, policyError("Debian package metadata changed while it was opened", nil)
	}
	rawMetadata := append([]byte(nil), metadataSnapshot.Bytes()...)
	if err := metadataSnapshot.VerifyStable(); err != nil {
		_ = metadataSnapshot.Close()
		return VerifiedDebPackage{}, err
	}
	if err := metadataSnapshot.Close(); err != nil {
		return VerifiedDebPackage{}, err
	}

	metadata, expectedDigest, preparedSignature, err := parseDebMetadata(
		rawMetadata, keys, policy, policy.SkipSignatureVerification,
	)
	if err != nil {
		return VerifiedDebPackage{}, err
	}

	debSnapshot, err := openArtifactSnapshot(
		debPath,
		"Debian package",
		metadata.Size,
		maxDebPackageBytes,
	)
	if err != nil {
		return VerifiedDebPackage{}, err
	}
	defer func() { _ = debSnapshot.Close() }()
	if debSnapshot.identity != initialDebIdentity {
		return VerifiedDebPackage{}, policyError("Debian package changed while it was opened", nil)
	}

	artifact := debSnapshot.Bytes()
	if !policy.SkipSignatureVerification {
		if err := preparedSignature.verify(artifact); err != nil {
			return VerifiedDebPackage{}, err
		}
	}
	actualDigest := sha512.Sum512(artifact)
	if subtle.ConstantTimeCompare(actualDigest[:], expectedDigest) != 1 {
		return VerifiedDebPackage{}, policyError("Debian package digest does not match metadata", nil)
	}
	if err := debSnapshot.VerifyStable(); err != nil {
		return VerifiedDebPackage{}, err
	}

	control, err := inspector.Inspect(debSnapshot.File())
	if err != nil {
		return VerifiedDebPackage{}, err
	}
	if err := debSnapshot.VerifyStable(); err != nil {
		return VerifiedDebPackage{}, err
	}
	if control.Package != metadata.Package || control.Package != policy.ExpectedPackage {
		return VerifiedDebPackage{}, policyError("Debian control Package does not match metadata", nil)
	}
	if control.Version != metadata.Version || control.Version != policy.ExpectedVersion {
		return VerifiedDebPackage{}, policyError("Debian control Version does not match metadata", nil)
	}
	if control.Architecture != metadata.Architecture || control.Architecture != policy.ExpectedArchitecture {
		return VerifiedDebPackage{}, policyError("Debian control Architecture does not match metadata", nil)
	}
	if metadata.Name != debPackageName(control.Package, control.Version, control.Architecture) {
		return VerifiedDebPackage{}, policyError("Debian package name is not derived from verified control metadata", nil)
	}

	verified := VerifiedDebPackage{
		Path:             debPath,
		MetadataPath:     metadataPath,
		Package:          control.Package,
		Version:          control.Version,
		Architecture:     control.Architecture,
		debIdentity:      debSnapshot.identity,
		metadataIdentity: initialMetadataIdentity,
	}
	if err := verified.verifyPathsStable(); err != nil {
		return VerifiedDebPackage{}, err
	}
	return verified, nil
}

func VerifyApplicationBundle(dir string, keys KeyRing, policy ApplicationPolicy) (VerifiedApplicationBundle, error) {
	return verifyApplicationBundle(dir, keys, policy, dpkgDebControlInspector{})
}

func verifyApplicationBundle(
	dir string,
	keys KeyRing,
	policy ApplicationPolicy,
	inspector debControlInspector,
) (VerifiedApplicationBundle, error) {
	directoryStat, err := lstatArtifact(dir)
	if err != nil {
		return VerifiedApplicationBundle{}, ioError("inspect application bundle", err)
	}
	if directoryStat.Mode&unix.S_IFMT != unix.S_IFDIR {
		return VerifiedApplicationBundle{}, policyError("application bundle is not a directory", nil)
	}
	directoryIdentity := identityFromStat(directoryStat)

	version, err := applicationBundleVersion(dir)
	if err != nil {
		return VerifiedApplicationBundle{}, err
	}
	if err := validateApplicationPolicy(version, policy); err != nil {
		return VerifiedApplicationBundle{}, err
	}

	expectedNames := map[string]struct{}{}
	for _, packageName := range []string{"kvmcomm", "nanokvm-go"} {
		expectedNames[debPackageName(packageName, version, applicationArchitecture)] = struct{}{}
		expectedNames[debMetadataName(packageName, version)] = struct{}{}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return VerifiedApplicationBundle{}, ioError("read application bundle", err)
	}
	if len(entries) != len(expectedNames) {
		return VerifiedApplicationBundle{}, policyError("application bundle file set is not exact", nil)
	}
	for _, entry := range entries {
		if _, ok := expectedNames[entry.Name()]; !ok {
			return VerifiedApplicationBundle{}, policyError("application bundle contains an unexpected entry", nil)
		}
	}

	verifiedPackages := make([]VerifiedDebPackage, 0, 2)
	for _, packageName := range []string{"kvmcomm", "nanokvm-go"} {
		debPath := filepath.Join(dir, debPackageName(packageName, version, applicationArchitecture))
		metadataPath := filepath.Join(dir, debMetadataName(packageName, version))
		verified, err := verifyDebPackage(
			debPath,
			metadataPath,
			keys,
			DebPolicy{
				ExpectedPackage:           packageName,
				ExpectedVersion:           version,
				ExpectedArchitecture:      applicationArchitecture,
				SkipSignatureVerification: policy.SkipSignatureVerification,
			},
			inspector,
		)
		if err != nil {
			return VerifiedApplicationBundle{}, err
		}
		verifiedPackages = append(verifiedPackages, verified)
	}

	for _, verified := range verifiedPackages {
		if verified.Version != version || verified.Architecture != applicationArchitecture {
			return VerifiedApplicationBundle{}, policyError("verified application packages are inconsistent", nil)
		}
		if err := verified.verifyPathsStable(); err != nil {
			return VerifiedApplicationBundle{}, err
		}
	}
	if err := verifyPathIdentity(dir, directoryIdentity, unix.S_IFDIR, "application bundle"); err != nil {
		return VerifiedApplicationBundle{}, err
	}
	return VerifiedApplicationBundle{Version: version, Packages: verifiedPackages}, nil
}

func parseDebMetadata(
	raw []byte,
	keys KeyRing,
	policy DebPolicy,
	skipSignatureVerification bool,
) (DebMetadata, []byte, preparedEmbeddedSignature, error) {
	if !utf8.Valid(raw) {
		return DebMetadata{}, nil, preparedEmbeddedSignature{}, invalid("Debian package metadata is not valid UTF-8", nil)
	}
	var metadata DebMetadata
	if err := strictUnmarshal(raw, &metadata); err != nil {
		return DebMetadata{}, nil, preparedEmbeddedSignature{}, invalid("invalid Debian package metadata", err)
	}
	if err := validateApplicationVersion(metadata.Version); err != nil {
		return DebMetadata{}, nil, preparedEmbeddedSignature{}, invalid("invalid Debian package metadata version", err)
	}
	if metadata.Package == "" || metadata.Architecture == "" || metadata.Name == "" {
		return DebMetadata{}, nil, preparedEmbeddedSignature{}, invalid("Debian package metadata has an empty required field", nil)
	}
	if metadata.Version != policy.ExpectedVersion {
		return DebMetadata{}, nil, preparedEmbeddedSignature{}, policyError("Debian package metadata version does not match policy", nil)
	}
	if metadata.Package != policy.ExpectedPackage {
		return DebMetadata{}, nil, preparedEmbeddedSignature{}, policyError("Debian package metadata package does not match policy", nil)
	}
	if metadata.Architecture != policy.ExpectedArchitecture {
		return DebMetadata{}, nil, preparedEmbeddedSignature{}, policyError("Debian package metadata architecture does not match policy", nil)
	}
	expectedName := debPackageName(policy.ExpectedPackage, policy.ExpectedVersion, policy.ExpectedArchitecture)
	if metadata.Name != expectedName {
		return DebMetadata{}, nil, preparedEmbeddedSignature{}, policyError("Debian package metadata name does not match policy", nil)
	}
	if metadata.Size <= 0 || metadata.Size > maxDebPackageBytes {
		return DebMetadata{}, nil, preparedEmbeddedSignature{}, invalid("invalid Debian package metadata size", nil)
	}
	digest, err := decodeCanonicalBase64(metadata.SHA512, sha512.Size)
	if err != nil {
		return DebMetadata{}, nil, preparedEmbeddedSignature{}, invalid("invalid Debian package SHA-512", err)
	}
	var preparedSignature preparedEmbeddedSignature
	if !skipSignatureVerification {
		preparedSignature, err = prepareEmbeddedSignature(metadata.Signature, keys)
		if err != nil {
			return DebMetadata{}, nil, preparedEmbeddedSignature{}, err
		}
	}
	return metadata, digest, preparedSignature, nil
}

func validateDebPolicy(policy DebPolicy) error {
	if policy.ExpectedPackage != "kvmcomm" && policy.ExpectedPackage != "nanokvm-go" {
		return policyError("unsupported Debian package policy", nil)
	}
	if err := validateApplicationVersion(policy.ExpectedVersion); err != nil {
		return policyError("invalid expected Debian package version", err)
	}
	if policy.ExpectedArchitecture != applicationArchitecture {
		return policyError("unsupported Debian package architecture policy", nil)
	}
	return nil
}

func validateApplicationPolicy(candidate string, policy ApplicationPolicy) error {
	if policy.RequestedVersion != "" {
		if err := validateApplicationVersion(policy.RequestedVersion); err != nil {
			return policyError("invalid requested application version", err)
		}
		if candidate != policy.RequestedVersion {
			return policyError("application bundle version does not match requested version", nil)
		}
	}
	if policy.CurrentVersion != "" {
		if err := validateApplicationVersion(policy.CurrentVersion); err != nil {
			return policyError("invalid current application version", err)
		}
		if candidate == policy.CurrentVersion && !policy.AllowSameVersion {
			return policyError("same-version application update is not allowed", nil)
		}
	}
	return nil
}

func validateApplicationVersion(version string) error {
	if !applicationVersionPattern.MatchString(version) {
		return fmt.Errorf("version must contain exactly three decimal components")
	}
	for _, component := range strings.Split(version, ".") {
		if _, err := strconv.ParseUint(component, 10, 64); err != nil {
			return fmt.Errorf("invalid version component: %w", err)
		}
	}
	return nil
}

func applicationBundleVersion(dir string) (string, error) {
	name := filepath.Base(filepath.Clean(dir))
	if !strings.HasPrefix(name, applicationBundlePrefix) {
		return "", policyError("invalid application bundle directory name", nil)
	}
	version := strings.TrimPrefix(name, applicationBundlePrefix)
	if err := validateApplicationVersion(version); err != nil {
		return "", policyError("invalid application bundle version", err)
	}
	return version, nil
}

func debPackageName(packageName, version, architecture string) string {
	return fmt.Sprintf("%s_%s_%s.deb", packageName, version, architecture)
}

func debMetadataName(packageName, version string) string {
	return fmt.Sprintf("%s_%s.json", packageName, version)
}

func requireRegularPath(path, label string) (artifactIdentity, error) {
	stat, err := lstatArtifact(path)
	if err != nil {
		return artifactIdentity{}, ioError("inspect "+label, err)
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG {
		return artifactIdentity{}, policyError(label+" is not a regular file", nil)
	}
	return identityFromStat(stat), nil
}

func (verified VerifiedDebPackage) verifyPathsStable() error {
	if err := verifyPathIdentity(verified.Path, verified.debIdentity, unix.S_IFREG, "Debian package"); err != nil {
		return err
	}
	return verifyPathIdentity(verified.MetadataPath, verified.metadataIdentity, unix.S_IFREG, "Debian package metadata")
}

func verifyPathIdentity(path string, expected artifactIdentity, expectedType uint32, label string) error {
	stat, err := lstatArtifact(path)
	if err != nil {
		return policyError(label+" changed while it was verified", err)
	}
	if stat.Mode&unix.S_IFMT != expectedType || identityFromStat(stat) != expected {
		return policyError(label+" changed while it was verified", nil)
	}
	return nil
}

func (dpkgDebControlInspector) Inspect(file *os.File) (debControl, error) {
	packageName, err := inspectDPKGField(file, "Package")
	if err != nil {
		return debControl{}, err
	}
	version, err := inspectDPKGField(file, "Version")
	if err != nil {
		return debControl{}, err
	}
	architecture, err := inspectDPKGField(file, "Architecture")
	if err != nil {
		return debControl{}, err
	}
	return debControl{Package: packageName, Version: version, Architecture: architecture}, nil
}

func inspectDPKGField(file *os.File, field string) (string, error) {
	if file == nil {
		return "", ioError("inspect Debian package control metadata", errors.New("package file is closed"))
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", ioError("rewind Debian package", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), dpkgDebInspectionTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, dpkgDebPath, "-f", dpkgDebFDPath, field)
	command.ExtraFiles = []*os.File{file}
	command.Env = []string{
		"LANG=C",
		"LC_ALL=C",
		"PATH=/usr/sbin:/usr/bin:/sbin:/bin",
	}
	command.Stderr = io.Discard
	stdout, err := command.StdoutPipe()
	if err != nil {
		return "", ioError("capture dpkg-deb output", err)
	}
	if err := command.Start(); err != nil {
		if ctx.Err() != nil || errors.Is(err, os.ErrNotExist) {
			return "", ioError("run dpkg-deb", err)
		}
		return "", ioError("run dpkg-deb", err)
	}
	output, readErr := io.ReadAll(io.LimitReader(stdout, maxDPKGFieldOutputBytes+1))
	if len(output) > maxDPKGFieldOutputBytes {
		cancel()
		_ = command.Wait()
		return "", invalid("dpkg-deb returned an invalid control field", nil)
	}
	waitErr := command.Wait()
	if readErr != nil {
		return "", ioError("read dpkg-deb output", readErr)
	}
	if waitErr != nil {
		if ctx.Err() != nil {
			return "", ioError("run dpkg-deb", waitErr)
		}
		var exitError *exec.ExitError
		if errors.As(waitErr, &exitError) {
			return "", invalid("dpkg-deb rejected the signed Debian package", nil)
		}
		return "", ioError("run dpkg-deb", waitErr)
	}
	if len(output) == 0 {
		return "", invalid("dpkg-deb returned an invalid control field", nil)
	}
	if !bytes.HasSuffix(output, []byte{'\n'}) {
		return "", invalid("dpkg-deb returned an unterminated control field", nil)
	}
	value := string(bytes.TrimSuffix(output, []byte{'\n'}))
	if value == "" || strings.ContainsAny(value, "\r\n\x00") {
		return "", invalid("dpkg-deb returned an invalid control field", nil)
	}
	return value, nil
}
