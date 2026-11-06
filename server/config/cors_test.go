package config

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCORSConfigurationIsBackwardCompatible(t *testing.T) {
	var oldConfig Config
	if err := yaml.Unmarshal([]byte("authentication: disable\n"), &oldConfig); err != nil {
		t.Fatalf("unmarshal old configuration: %v", err)
	}
	if oldConfig.Security.CORS.AllowedOrigins != nil {
		t.Fatalf("old configuration origins = %#v, want nil/empty", oldConfig.Security.CORS.AllowedOrigins)
	}

	var configured Config
	input := []byte("security:\n  cors:\n    allowedOrigins:\n      - http://localhost:4173\n      - https://dev.example:4443\n")
	if err := yaml.Unmarshal(input, &configured); err != nil {
		t.Fatalf("unmarshal CORS configuration: %v", err)
	}
	want := []string{"http://localhost:4173", "https://dev.example:4443"}
	if !reflect.DeepEqual(configured.Security.CORS.AllowedOrigins, want) {
		t.Fatalf("origins = %#v, want %#v", configured.Security.CORS.AllowedOrigins, want)
	}
}
