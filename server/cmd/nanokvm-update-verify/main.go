package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"NanoKVM-Server/updateverify"
)

const (
	exitVerified  = 0
	exitInvalid   = 20
	exitSignature = 25
	exitPolicy    = 30
	exitIO        = 40
)

type verifiedBundleOutput struct {
	Mode    string `json:"mode"`
	Version string `json:"version"`
}

type verifiedFirmwareOutput struct {
	Mode        string `json:"mode"`
	Version     string `json:"version"`
	Name        string `json:"name"`
	UpdateUboot bool   `json:"updateUboot"`
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	return runWithIO(args, os.Stdout, os.Stderr)
}

func runWithIO(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return exitPolicy
	}

	switch args[0] {
	case "bundle":
		return runBundle(args[1:], stdout, stderr)
	case "firmware":
		return runFirmware(args[1:], stdout, stderr)
	default:
		usage(stderr)
		return exitPolicy
	}
}

func runBundle(args []string, stdout, stderr io.Writer) int {
	if err := validateStrictModeArgs(args, map[string]bool{
		"bundle":             false,
		"keyring":            false,
		"current-version":    false,
		"requested-version":  false,
		"allow-same-version": true,
	}); err != nil {
		fmt.Fprintln(stderr, "invalid bundle arguments")
		return exitPolicy
	}
	flags := flag.NewFlagSet("bundle", flag.ContinueOnError)
	flags.SetOutput(stderr)
	bundlePath := flags.String("bundle", "", "path to the extracted application bundle directory")
	keyRingPath := flags.String("keyring", "", "path to pinned public key ring")
	currentVersion := flags.String("current-version", "", "currently installed version")
	requestedVersion := flags.String("requested-version", "", "exact requested version")
	allowSameVersion := flags.Bool("allow-same-version", false, "allow same-version recovery")
	if err := flags.Parse(args); err != nil {
		return exitPolicy
	}
	if flags.NArg() != 0 || *bundlePath == "" || *keyRingPath == "" {
		usage(stderr)
		return exitPolicy
	}
	if *allowSameVersion && *currentVersion == "" {
		fmt.Fprintln(stderr, "--allow-same-version requires --current-version")
		return exitPolicy
	}

	keys, err := updateverify.LoadKeyRing(*keyRingPath)
	if err != nil {
		return report(stderr, updateverify.Kind(err), "unable to load pinned update keys")
	}
	verified, err := updateverify.VerifyApplicationBundle(
		*bundlePath,
		keys,
		updateverify.ApplicationPolicy{
			RequestedVersion: *requestedVersion,
			CurrentVersion:   *currentVersion,
			AllowSameVersion: *allowSameVersion,
		},
	)
	if err != nil {
		return report(stderr, updateverify.Kind(err), "application bundle verification failed")
	}

	output := verifiedBundleOutput{
		Mode:    "bundle",
		Version: verified.Version,
	}
	if err := json.NewEncoder(stdout).Encode(output); err != nil {
		return report(stderr, updateverify.ErrorIO, "unable to write verified bundle result")
	}
	fmt.Fprintln(stderr, "application bundle verified")
	return exitVerified
}

func runFirmware(args []string, stdout, stderr io.Writer) int {
	if err := validateStrictModeArgs(args, map[string]bool{
		"metadata":           false,
		"artifact":           false,
		"keyring":            false,
		"current-version":    false,
		"requested-version":  false,
		"allow-same-version": true,
	}); err != nil {
		fmt.Fprintln(stderr, "invalid Firmware arguments")
		return exitPolicy
	}
	flags := flag.NewFlagSet("firmware", flag.ContinueOnError)
	flags.SetOutput(stderr)
	metadataPath := flags.String("metadata", "", "path to Firmware metadata")
	artifactPath := flags.String("artifact", "", "path to Firmware archive")
	keyRingPath := flags.String("keyring", "", "path to pinned public key ring")
	currentVersion := flags.String("current-version", "", "currently installed version")
	requestedVersion := flags.String("requested-version", "", "exact requested version")
	allowSameVersion := flags.Bool("allow-same-version", false, "allow an explicit same-version Firmware reinstall")
	if err := flags.Parse(args); err != nil {
		return exitPolicy
	}
	if flags.NArg() != 0 || *metadataPath == "" || *artifactPath == "" || *keyRingPath == "" {
		usage(stderr)
		return exitPolicy
	}
	if *allowSameVersion && *currentVersion == "" {
		fmt.Fprintln(stderr, "--allow-same-version requires --current-version")
		return exitPolicy
	}

	keys, err := updateverify.LoadKeyRing(*keyRingPath)
	if err != nil {
		return report(stderr, updateverify.Kind(err), "unable to load pinned update keys")
	}
	verified, err := updateverify.VerifyFirmwareArtifact(
		*artifactPath,
		*metadataPath,
		keys,
		updateverify.FirmwarePolicy{
			RequestedVersion: *requestedVersion,
			CurrentVersion:   *currentVersion,
			AllowSameVersion: *allowSameVersion,
		},
	)
	if err != nil {
		return report(stderr, updateverify.Kind(err), "Firmware artifact verification failed")
	}
	output := verifiedFirmwareOutput{
		Mode:        "firmware",
		Version:     verified.Version,
		Name:        verified.Name,
		UpdateUboot: verified.UpdateUboot,
	}
	if err := json.NewEncoder(stdout).Encode(output); err != nil {
		return report(stderr, updateverify.ErrorIO, "unable to write verified Firmware result")
	}
	fmt.Fprintln(stderr, "Firmware artifact verified")
	return exitVerified
}

func report(stderr io.Writer, kind updateverify.ErrorKind, message string) int {
	fmt.Fprintln(stderr, message)
	switch kind {
	case updateverify.ErrorInvalid:
		return exitInvalid
	case updateverify.ErrorSignature:
		return exitSignature
	case updateverify.ErrorPolicy:
		return exitPolicy
	default:
		return exitIO
	}
}

// validateStrictModeArgs rejects every flag ambiguity before flag.Parse. The
// map value is true only for boolean flags that do not consume the next
// argument.
func validateStrictModeArgs(args []string, allowed map[string]bool) error {
	seen := make(map[string]struct{}, len(allowed))
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if argument == "--" || !strings.HasPrefix(argument, "--") || len(argument) == 2 {
			return errors.New("positional arguments and -- are not allowed")
		}
		nameAndValue := strings.TrimPrefix(argument, "--")
		name, inlineValue, hasInlineValue := strings.Cut(nameAndValue, "=")
		isBoolean, ok := allowed[name]
		if !ok {
			return fmt.Errorf("unknown flag %q", name)
		}
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("duplicate flag %q", name)
		}
		seen[name] = struct{}{}
		if isBoolean {
			continue
		}
		if hasInlineValue {
			if inlineValue == "" {
				return fmt.Errorf("flag %q has an empty value", name)
			}
			continue
		}
		if index+1 >= len(args) || args[index+1] == "" || args[index+1] == "--" || strings.HasPrefix(args[index+1], "-") {
			return fmt.Errorf("flag %q is missing its value", name)
		}
		index++
	}
	return nil
}

func usage(stderr io.Writer) {
	fmt.Fprintln(stderr, "usage:")
	fmt.Fprintln(stderr, "  nanokvm-update-verify bundle --bundle DIR --keyring FILE [options]")
	fmt.Fprintln(stderr, "  nanokvm-update-verify firmware --metadata FILE --artifact FILE --keyring FILE [options]")
}
