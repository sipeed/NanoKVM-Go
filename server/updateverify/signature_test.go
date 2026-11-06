package updateverify

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

const rfc8032TestSeedHex = "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"

func testOnlyEd25519Key(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, string, KeyRing) {
	t.Helper()
	seed, err := hex.DecodeString(rfc8032TestSeedHex)
	if err != nil {
		t.Fatal(err)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := append(ed25519.PublicKey(nil), privateKey.Public().(ed25519.PublicKey)...)
	keyID := PublicKeyID(publicKey)
	return publicKey, privateKey, keyID, KeyRing{keyID: publicKey}
}

func TestVerifyEmbeddedSignatureAcceptsRFC8032TestKey(t *testing.T) {
	_, privateKey, keyID, keys := testOnlyEd25519Key(t)
	artifact := []byte("NanoKVM test-only signed artifact")
	signature := &Signature{
		Schema:    1,
		Algorithm: "ed25519",
		KeyID:     keyID,
		Value:     base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, artifact)),
	}
	if err := verifyEmbeddedSignature(signature, artifact, keys); err != nil {
		t.Fatalf("verify embedded signature: %v", err)
	}
}

func TestVerifyEmbeddedSignatureFailsClosed(t *testing.T) {
	_, privateKey, keyID, keys := testOnlyEd25519Key(t)
	artifact := []byte("artifact")
	validValue := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, artifact))
	valid := Signature{Schema: 1, Algorithm: "ed25519", KeyID: keyID, Value: validValue}

	tests := []struct {
		name      string
		signature *Signature
		artifact  []byte
		keys      KeyRing
		kind      ErrorKind
	}{
		{name: "missing", signature: nil, artifact: artifact, keys: keys, kind: ErrorSignature},
		{name: "schema", signature: signatureCopy(valid, func(value *Signature) { value.Schema = 2 }), artifact: artifact, keys: keys, kind: ErrorInvalid},
		{name: "algorithm", signature: signatureCopy(valid, func(value *Signature) { value.Algorithm = "Ed25519" }), artifact: artifact, keys: keys, kind: ErrorInvalid},
		{name: "key id syntax", signature: signatureCopy(valid, func(value *Signature) { value.KeyID = "sha256:ABC" }), artifact: artifact, keys: keys, kind: ErrorInvalid},
		{name: "unknown key", signature: &valid, artifact: artifact, keys: KeyRing{}, kind: ErrorSignature},
		{name: "bad base64", signature: signatureCopy(valid, func(value *Signature) { value.Value = "***" }), artifact: artifact, keys: keys, kind: ErrorSignature},
		{name: "base64 newline", signature: signatureCopy(valid, func(value *Signature) { value.Value = value.Value[:8] + "\n" + value.Value[8:] }), artifact: artifact, keys: keys, kind: ErrorSignature},
		{name: "63 bytes", signature: signatureCopy(valid, func(value *Signature) { value.Value = base64.StdEncoding.EncodeToString(make([]byte, 63)) }), artifact: artifact, keys: keys, kind: ErrorSignature},
		{name: "65 bytes", signature: signatureCopy(valid, func(value *Signature) { value.Value = base64.StdEncoding.EncodeToString(make([]byte, 65)) }), artifact: artifact, keys: keys, kind: ErrorSignature},
		{name: "tampered artifact", signature: &valid, artifact: []byte("Artifact"), keys: keys, kind: ErrorSignature},
		{name: "bad public key length", signature: &valid, artifact: artifact, keys: KeyRing{keyID: ed25519.PublicKey{1}}, kind: ErrorSignature},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertKind(t, verifyEmbeddedSignature(test.signature, test.artifact, test.keys), test.kind)
		})
	}
}

func TestDecodeCanonicalBase64RejectsIgnoredWhitespace(t *testing.T) {
	canonical := base64.StdEncoding.EncodeToString(make([]byte, 64))
	if _, err := decodeCanonicalBase64(canonical, 64); err != nil {
		t.Fatalf("decode canonical Base64: %v", err)
	}
	for _, value := range []string{
		canonical[:4] + "\r" + canonical[4:],
		canonical[:4] + "\n" + canonical[4:],
		canonical + "=",
	} {
		if _, err := decodeCanonicalBase64(value, 64); err == nil {
			t.Fatalf("decodeCanonicalBase64(%q) succeeded", value)
		}
	}
}

func signatureCopy(value Signature, mutate func(*Signature)) *Signature {
	copy := value
	mutate(&copy)
	return &copy
}
