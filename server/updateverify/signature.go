package updateverify

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"regexp"
)

const ErrorSignature ErrorKind = "signature"

const embeddedSignatureSchema = 1

var embeddedKeyIDPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// Signature is the fail-closed signature object embedded in artifact metadata.
type Signature struct {
	Schema    int    `json:"schema"`
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"keyId"`
	Value     string `json:"value"`
}

type preparedEmbeddedSignature struct {
	publicKey ed25519.PublicKey
	value     []byte
}

func prepareEmbeddedSignature(signature *Signature, keys KeyRing) (preparedEmbeddedSignature, error) {
	if signature == nil {
		return preparedEmbeddedSignature{}, signatureError("embedded signature is missing", nil)
	}
	if signature.Schema != embeddedSignatureSchema {
		return preparedEmbeddedSignature{}, invalid("unsupported embedded signature schema", nil)
	}
	if signature.Algorithm != "ed25519" {
		return preparedEmbeddedSignature{}, invalid("unsupported embedded signature algorithm", nil)
	}
	if !embeddedKeyIDPattern.MatchString(signature.KeyID) {
		return preparedEmbeddedSignature{}, invalid("invalid embedded signature key id", nil)
	}

	publicKey, ok := keys[signature.KeyID]
	if !ok {
		return preparedEmbeddedSignature{}, signatureError("unknown embedded signing key", nil)
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return preparedEmbeddedSignature{}, signatureError("invalid embedded signing public key", nil)
	}

	value, err := decodeCanonicalBase64(signature.Value, ed25519.SignatureSize)
	if err != nil {
		return preparedEmbeddedSignature{}, signatureError("invalid embedded Ed25519 signature", err)
	}
	return preparedEmbeddedSignature{
		publicKey: append(ed25519.PublicKey(nil), publicKey...),
		value:     value,
	}, nil
}

func (signature preparedEmbeddedSignature) verify(artifact []byte) error {
	if !ed25519.Verify(signature.publicKey, artifact, signature.value) {
		return signatureError("artifact signature verification failed", nil)
	}
	return nil
}

func verifyEmbeddedSignature(signature *Signature, artifact []byte, keys KeyRing) error {
	prepared, err := prepareEmbeddedSignature(signature, keys)
	if err != nil {
		return err
	}
	return prepared.verify(artifact)
}

// decodeCanonicalBase64 is stricter than base64.Encoding.Strict: Go's decoder
// deliberately ignores CR and LF. Re-encoding rejects whitespace and every
// other non-canonical spelling accepted by the decoder.
func decodeCanonicalBase64(value string, expectedSize int) ([]byte, error) {
	decoded, err := base64.StdEncoding.Strict().DecodeString(value)
	if err != nil {
		return nil, err
	}
	if len(decoded) != expectedSize {
		return nil, fmt.Errorf("decoded length is %d, want %d", len(decoded), expectedSize)
	}
	if base64.StdEncoding.EncodeToString(decoded) != value {
		return nil, fmt.Errorf("base64 value is not canonical")
	}
	return decoded, nil
}

func signatureError(message string, err error) error {
	return typedError(ErrorSignature, message, err)
}
