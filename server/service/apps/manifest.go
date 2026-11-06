package apps

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
)

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var appPackagePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z][a-z0-9_]*){2,}$`)

type EnvSpec struct {
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
	Default     string `json:"default,omitempty"`
	Value       string `json:"value,omitempty"`
	Configured  bool   `json:"configured,omitempty"`
}

type Manifest struct {
	AppID      string             `json:"app_id"`
	Name       string             `json:"name"`
	Creator    string             `json:"creator"`
	CreateTime string             `json:"create_time"`
	Version    string             `json:"version"`
	Desc       string             `json:"desc,omitempty"`
	Category   string             `json:"category,omitempty"`
	Icon       string             `json:"icon,omitempty"`
	PreScript  string             `json:"pre_script,omitempty"`
	PostScript string             `json:"post_script,omitempty"`
	Env        map[string]EnvSpec `json:"env,omitempty"`
}

func readManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	return parseManifest(data)
}

func parseManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("invalid app.json: %w", err)
	}
	if !appPackagePattern.MatchString(manifest.AppID) {
		return Manifest{}, errors.New("app.json requires a valid reverse-domain app_id")
	}
	if strings.TrimSpace(manifest.Name) == "" || strings.TrimSpace(manifest.Creator) == "" ||
		strings.TrimSpace(manifest.CreateTime) == "" || strings.TrimSpace(manifest.Version) == "" {
		return Manifest{}, errors.New("app.json requires name, creator, create_time and version")
	}
	for name := range manifest.Env {
		if !envNamePattern.MatchString(name) {
			return Manifest{}, fmt.Errorf("invalid environment variable name %q", name)
		}
		spec := manifest.Env[name]
		if spec.Secret && spec.Default != "" {
			return Manifest{}, fmt.Errorf("secret environment variable %s must not define a default", name)
		}
	}
	for field, script := range map[string]string{
		"pre_script": manifest.PreScript, "post_script": manifest.PostScript,
	} {
		if err := validateScriptPath(script); err != nil {
			return Manifest{}, fmt.Errorf("invalid %s: %w", field, err)
		}
	}
	return manifest, nil
}

func validateScriptPath(script string) error {
	if script == "" {
		return nil
	}
	if strings.ContainsRune(script, '\\') || strings.ContainsRune(script, '\x00') ||
		path.IsAbs(script) || path.Clean(script) != script || script == "." ||
		strings.HasPrefix(script, "../") {
		return errors.New("must be a safe relative App path")
	}
	return nil
}

func validateAppIDForDirectory(manifest Manifest, directory string) error {
	parts := strings.Split(manifest.AppID, ".")
	want := strings.ReplaceAll(directory, "-", "_")
	if len(parts) < 3 || parts[len(parts)-1] != want {
		return fmt.Errorf("app_id %q must end in .%s for directory %q", manifest.AppID, want, directory)
	}
	return nil
}

func clearSecretValues(manifest *Manifest) {
	for name, spec := range manifest.Env {
		if spec.Secret {
			spec.Value = ""
		}
		manifest.Env[name] = spec
	}
}

func applyEnv(manifest *Manifest, values map[string]string) error {
	for name := range values {
		if _, ok := manifest.Env[name]; !ok {
			return fmt.Errorf("unknown environment variable %q", name)
		}
	}
	for name, spec := range manifest.Env {
		if value, ok := values[name]; ok {
			spec.Value = value
		}
		if spec.Value == "" && spec.Default != "" {
			spec.Value = spec.Default
		}
		if spec.Required && strings.TrimSpace(spec.Value) == "" {
			return fmt.Errorf("environment variable %s is required", name)
		}
		manifest.Env[name] = spec
	}
	return nil
}

func publicManifest(manifest Manifest) Manifest {
	for name, spec := range manifest.Env {
		spec.Configured = spec.Value != "" || spec.Default != ""
		if spec.Secret {
			spec.Value = ""
			spec.Default = ""
		}
		manifest.Env[name] = spec
	}
	return manifest
}

func updateEnv(manifest *Manifest, values map[string]string) error {
	for name := range values {
		if _, ok := manifest.Env[name]; !ok {
			return fmt.Errorf("unknown environment variable %q", name)
		}
	}
	for name, spec := range manifest.Env {
		if value, ok := values[name]; ok {
			// An empty secret from the web form means "keep the stored value".
			if !(spec.Secret && value == "" && spec.Value != "") {
				spec.Value = value
			}
		}
		if spec.Required && spec.Value == "" && spec.Default == "" {
			return fmt.Errorf("environment variable %s is required", name)
		}
		spec.Configured = false
		manifest.Env[name] = spec
	}
	return nil
}
