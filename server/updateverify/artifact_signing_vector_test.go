package updateverify

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
)

type artifactSigningFixture struct {
	Schema    int                     `json:"schema"`
	TestOnly  bool                    `json:"testOnly"`
	Source    string                  `json:"source"`
	PublicKey string                  `json:"publicKey"`
	KeyID     string                  `json:"keyId"`
	Vectors   []artifactSigningVector `json:"vectors"`
}

type artifactSigningVector struct {
	Kind      string                    `json:"kind"`
	Name      string                    `json:"name"`
	Artifact  string                    `json:"artifact"`
	Signature embeddedArtifactSignature `json:"signature"`
}

type embeddedArtifactSignature struct {
	Schema    int    `json:"schema"`
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"keyId"`
	Value     string `json:"value"`
}

func TestPythonArtifactSigningVectors(t *testing.T) {
	data, err := os.ReadFile("testdata/artifact_signing_vectors_v1.json")
	if err != nil {
		t.Fatal(err)
	}

	var fixture artifactSigningFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Schema != 1 || !fixture.TestOnly || fixture.Source == "" {
		t.Fatalf("invalid test fixture header: %#v", fixture)
	}

	publicKeyBytes := decodeStrictBase64(t, fixture.PublicKey)
	if len(publicKeyBytes) != ed25519.PublicKeySize {
		t.Fatalf("public key length = %d, want %d", len(publicKeyBytes), ed25519.PublicKeySize)
	}
	publicKey := ed25519.PublicKey(publicKeyBytes)
	if got := PublicKeyID(publicKey); got != fixture.KeyID {
		t.Fatalf("public key ID = %q, want %q", got, fixture.KeyID)
	}

	wrongPublicKey := append(ed25519.PublicKey(nil), publicKey...)
	wrongPublicKey[0] ^= 1
	seenKinds := make(map[string]bool)
	for _, vector := range fixture.Vectors {
		vector := vector
		t.Run(vector.Kind+"/"+vector.Name, func(t *testing.T) {
			switch vector.Kind {
			case "deb", "firmware":
				seenKinds[vector.Kind] = true
			default:
				t.Fatalf("unexpected vector kind %q", vector.Kind)
			}
			if vector.Name == "" {
				t.Fatal("vector name is empty")
			}
			if vector.Signature.Schema != 1 || vector.Signature.Algorithm != "ed25519" {
				t.Fatalf("invalid embedded signature schema: %#v", vector.Signature)
			}
			if vector.Signature.KeyID != fixture.KeyID {
				t.Fatalf("signature key ID = %q, want %q", vector.Signature.KeyID, fixture.KeyID)
			}

			artifact := decodeStrictBase64(t, vector.Artifact)
			if len(artifact) == 0 {
				t.Fatal("artifact is empty")
			}
			signature := decodeStrictBase64(t, vector.Signature.Value)
			if len(signature) != ed25519.SignatureSize {
				t.Fatalf("signature length = %d, want %d", len(signature), ed25519.SignatureSize)
			}
			if !ed25519.Verify(publicKey, artifact, signature) {
				t.Fatal("fixed Python artifact signature did not verify")
			}

			tampered := append([]byte(nil), artifact...)
			tampered[len(tampered)-1] ^= 1
			if ed25519.Verify(publicKey, tampered, signature) {
				t.Fatal("signature verified after artifact tampering")
			}
			if ed25519.Verify(wrongPublicKey, artifact, signature) {
				t.Fatal("signature verified with the wrong public key")
			}
			if ed25519.Verify(publicKey, artifact, signature[:ed25519.SignatureSize-1]) {
				t.Fatal("63-byte signature verified")
			}
			longSignature := append(append([]byte(nil), signature...), 0)
			if ed25519.Verify(publicKey, artifact, longSignature) {
				t.Fatal("65-byte signature verified")
			}
		})
	}
	if !seenKinds["deb"] || !seenKinds["firmware"] || len(seenKinds) != 2 {
		t.Fatalf("vector kinds = %#v, want deb and firmware", seenKinds)
	}
}

func decodeStrictBase64(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := base64.StdEncoding.Strict().DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}
