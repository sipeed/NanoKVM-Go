package updateverify

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type ErrorKind string

const (
	ErrorInvalid ErrorKind = "invalid"
	ErrorPolicy  ErrorKind = "policy"
	ErrorIO      ErrorKind = "io"
)

type VerifyError struct {
	Kind ErrorKind
	Err  error
}

func (e *VerifyError) Error() string { return e.Err.Error() }
func (e *VerifyError) Unwrap() error { return e.Err }

func Kind(err error) ErrorKind {
	var verifyErr *VerifyError
	if errors.As(err, &verifyErr) {
		return verifyErr.Kind
	}
	return ErrorIO
}

// TransportMetadata describes an artifact discovered through untrusted update
// metadata. VerifyTransportArtifact only checks transfer integrity; install
// authorization comes from the signatures embedded in artifact metadata.
type TransportMetadata struct {
	Name   string `json:"name"`
	SHA512 string `json:"sha512"`
	Size   int64  `json:"size"`
}

type Key struct {
	KeyID     string `json:"keyId"`
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"publicKey"`
}

type KeyRingFile struct {
	Schema int   `json:"schema"`
	Keys   []Key `json:"keys"`
}

type KeyRing map[string]ed25519.PublicKey

func PublicKeyID(publicKey ed25519.PublicKey) string {
	fingerprint := sha256.Sum256(publicKey)
	return "sha256:" + hex.EncodeToString(fingerprint[:])
}

func ParseKeyRing(data []byte) (KeyRing, error) {
	var file KeyRingFile
	if err := strictUnmarshal(data, &file); err != nil {
		return nil, invalid("invalid key ring", err)
	}
	if file.Schema != 1 {
		return nil, invalid("unsupported key ring schema", nil)
	}

	ring := make(KeyRing, len(file.Keys))
	for _, key := range file.Keys {
		if key.Algorithm != "ed25519" {
			return nil, invalid("unsupported key algorithm", nil)
		}
		publicKey, err := base64.StdEncoding.Strict().DecodeString(key.PublicKey)
		if err != nil || len(publicKey) != ed25519.PublicKeySize {
			return nil, invalid("invalid public key", err)
		}
		computedID := PublicKeyID(ed25519.PublicKey(publicKey))
		if key.KeyID != computedID {
			return nil, invalid("public key fingerprint mismatch", nil)
		}
		if _, exists := ring[key.KeyID]; exists {
			return nil, invalid("duplicate key id", nil)
		}
		ring[key.KeyID] = append(ed25519.PublicKey(nil), publicKey...)
	}
	return ring, nil
}

func LoadKeyRing(path string) (KeyRing, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, ioError("read key ring", err)
	}
	return ParseKeyRing(data)
}

func VerifyTransportArtifact(path string, metadata TransportMetadata) error {
	if filepath.Base(path) != metadata.Name {
		return policyError("artifact name does not match metadata", nil)
	}
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return ioError("inspect artifact", err)
	}
	if !pathInfo.Mode().IsRegular() {
		return policyError("artifact is not a regular file", nil)
	}
	file, err := os.Open(path)
	if err != nil {
		return ioError("open artifact", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return ioError("stat artifact", err)
	}
	if !info.Mode().IsRegular() {
		return policyError("artifact is not a regular file", nil)
	}
	if info.Size() != metadata.Size {
		return policyError("artifact size does not match metadata", nil)
	}

	hasher := sha512.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return ioError("hash artifact", err)
	}
	actual := base64.StdEncoding.EncodeToString(hasher.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(actual), []byte(metadata.SHA512)) != 1 {
		return policyError("artifact digest does not match metadata", nil)
	}
	return nil
}

func strictUnmarshal(data []byte, target any) error {
	if err := rejectDuplicateFields(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func rejectDuplicateFields(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]struct{}{}
			for decoder.More() {
				nameToken, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := nameToken.(string)
				if !ok {
					return fmt.Errorf("object key is not a string")
				}
				if _, duplicate := seen[name]; duplicate {
					return fmt.Errorf("duplicate field %q", name)
				}
				seen[name] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return fmt.Errorf("unexpected JSON delimiter")
		}
	}
	if err := walk(); err != nil {
		return err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing token %v", token)
		}
		return err
	}
	return nil
}

func invalid(message string, err error) error {
	return typedError(ErrorInvalid, message, err)
}

func policyError(message string, err error) error {
	return typedError(ErrorPolicy, message, err)
}

func ioError(message string, err error) error {
	return typedError(ErrorIO, message, err)
}

func typedError(kind ErrorKind, message string, err error) error {
	if err != nil {
		return &VerifyError{Kind: kind, Err: fmt.Errorf("%s: %w", message, err)}
	}
	return &VerifyError{Kind: kind, Err: errors.New(message)}
}
