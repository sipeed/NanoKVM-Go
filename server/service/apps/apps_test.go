package apps

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func makeZIP(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func validManifest() string {
	return `{"app_id":"com.sipeed.voice_bridge","name":"Voice Bridge","creator":"Sipeed","create_time":"2026-07-30","version":"1.0.0","env":{"API_KEY":{"required":true,"secret":true,"description":"Service key"},"LANG":{"default":"zh"}}}`
}

func TestInspectArchiveAcceptsOneAppDirectory(t *testing.T) {
	archive, err := inspectArchive(makeZIP(t, map[string]string{
		"voice-bridge/app.json": validManifest(),
		"voice-bridge/main.py":  "print('ok')\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if archive.Prefix != "voice-bridge/" || archive.Manifest.Name != "Voice Bridge" {
		t.Fatalf("unexpected archive: %#v", archive)
	}
	if got := archiveID(archive, "upload.zip"); got != "voice-bridge" {
		t.Fatalf("unexpected id %q", got)
	}
}

func TestInspectArchiveRejectsTraversalAndMultipleManifests(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"traversal": {
			"app/app.json": validManifest(), "app/main.py": "", "../escape": "x",
		},
		"multiple": {
			"one/app.json": validManifest(), "two/app.json": validManifest(),
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := inspectArchive(makeZIP(t, files)); err == nil {
				t.Fatal("expected invalid archive")
			}
		})
	}
}

func TestInspectRepositoryArchiveFindsTopLevelApps(t *testing.T) {
	catalog, archives, err := inspectRepositoryArchive(makeZIP(t, map[string]string{
		"NanoKVM-Go-Apps-main/README.md":                     "docs",
		"NanoKVM-Go-Apps-main/voice-bridge/app.json":         validManifest(),
		"NanoKVM-Go-Apps-main/voice-bridge/main.py":          "print('voice')\n",
		"NanoKVM-Go-Apps-main/conways-game-of-life/app.json": `{"app_id":"com.sipeed.conways_game_of_life","name":"Conway","creator":"Sipeed","create_time":"2026-07-30","version":"1.0.0"}`,
		"NanoKVM-Go-Apps-main/conways-game-of-life/main.py":  "print('life')\n",
		"NanoKVM-Go-Apps-main/_utils/not-an-app/app.json":    validManifest(),
		"NanoKVM-Go-Apps-main/_utils/not-an-app/main.py":     "print('nested')\n",
		"NanoKVM-Go-Apps-main/missing-entrypoint/app.json":   validManifest(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Apps) != 2 || archives["voice-bridge"] == nil || archives["conways-game-of-life"] == nil {
		t.Fatalf("unexpected repository catalog: %#v", catalog)
	}
	if archives["voice-bridge"].Prefix != "NanoKVM-Go-Apps-main/voice-bridge/" {
		t.Fatalf("unexpected App prefix %q", archives["voice-bridge"].Prefix)
	}
}

func TestInspectArchiveRequiresPackagedDirectoryAndMatchingAppID(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"files at ZIP root": {
			"app.json": validManifest(), "main.py": "print('no')\n",
		},
		"package does not match directory": {
			"different-name/app.json": validManifest(),
			"different-name/main.py":  "print('no')\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := inspectArchive(makeZIP(t, files)); err == nil {
				t.Fatal("expected archive to be rejected")
			}
		})
	}
}

func TestGitHubArchiveURL(t *testing.T) {
	got, ok := githubArchiveURL("https://github.com/sipeed/NanoKVM-Go-Apps.git")
	if !ok || got != "https://codeload.github.com/sipeed/NanoKVM-Go-Apps/zip/HEAD" {
		t.Fatalf("unexpected GitHub archive URL: %q, %v", got, ok)
	}
	for _, invalid := range []string{
		"http://github.com/sipeed/NanoKVM-Go-Apps",
		"https://github.com/sipeed/NanoKVM-Go-Apps/tree/main",
		"https://example.com/sipeed/NanoKVM-Go-Apps",
	} {
		if _, ok := githubArchiveURL(invalid); ok {
			t.Fatalf("accepted invalid GitHub repository URL %q", invalid)
		}
	}
}

func TestApplyEnvRequiresValuesAndKeepsDefaults(t *testing.T) {
	manifest, err := parseManifest([]byte(validManifest()))
	if err != nil {
		t.Fatal(err)
	}
	if err := applyEnv(&manifest, nil); err == nil {
		t.Fatal("expected missing API_KEY")
	}
	if err := applyEnv(&manifest, map[string]string{"API_KEY": "secret"}); err != nil {
		t.Fatal(err)
	}
	if manifest.Env["API_KEY"].Value != "secret" || manifest.Env["LANG"].Value != "zh" {
		t.Fatalf("unexpected env: %#v", manifest.Env)
	}
	if publicManifest(manifest).Env["API_KEY"].Value != "" {
		t.Fatal("secret value must be hidden")
	}
}

func TestUpdateEnvKeepsConfiguredSecretWhenFormLeavesItEmpty(t *testing.T) {
	manifest, err := parseManifest([]byte(validManifest()))
	if err != nil {
		t.Fatal(err)
	}
	if err := applyEnv(&manifest, map[string]string{"API_KEY": "secret"}); err != nil {
		t.Fatal(err)
	}
	if err := updateEnv(&manifest, map[string]string{"API_KEY": "", "LANG": "en"}); err != nil {
		t.Fatal(err)
	}
	if manifest.Env["API_KEY"].Value != "secret" {
		t.Fatal("empty secret edit must preserve its configured value")
	}
	if manifest.Env["LANG"].Value != "en" {
		t.Fatal("non-secret value was not updated")
	}
	public := publicManifest(manifest)
	if !public.Env["API_KEY"].Configured || public.Env["API_KEY"].Value != "" {
		t.Fatal("public manifest must expose configured state without the secret")
	}
}

func TestInstallListAndExportRedactsSecrets(t *testing.T) {
	service := newService(t.TempDir())
	archive, err := inspectArchive(makeZIP(t, map[string]string{
		"voice-bridge/app.json": validManifest(),
		"voice-bridge/main.py":  "print('ok')\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if err := applyEnv(&archive.Manifest, map[string]string{"API_KEY": "secret"}); err != nil {
		t.Fatal(err)
	}
	if err := service.installArchive("voice-bridge", archive); err != nil {
		t.Fatal(err)
	}
	apps, err := service.list()
	if err != nil || len(apps) != 1 || apps[0].Directory != "voice-bridge" {
		t.Fatalf("unexpected Apps: %#v, %v", apps, err)
	}

	deployed, err := readManifest(filepath.Join(service.root, "voice-bridge", "app.json"))
	if err != nil || deployed.Env["API_KEY"].Value != "secret" {
		t.Fatalf("deployed value missing: %#v, %v", deployed, err)
	}
	var output bytes.Buffer
	if err := writeArchive(&output, filepath.Join(service.root, "voice-bridge"), "voice-bridge", deployed); err != nil {
		t.Fatal(err)
	}
	exported, err := inspectArchive(output.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if exported.Manifest.Env["API_KEY"].Value != "" {
		t.Fatal("exported secret was not redacted")
	}
	if _, err := os.Stat(filepath.Join(service.root, "voice-bridge", "main.py")); err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(apps); err != nil {
		t.Fatal(err)
	}
}

func TestInstallRejectsDuplicateAppIDInDifferentDirectory(t *testing.T) {
	service := newService(t.TempDir())
	first, err := inspectArchive(makeZIP(t, map[string]string{
		"voice-bridge/app.json": validManifest(),
		"voice-bridge/main.py":  "print('first')\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if err := applyEnv(&first.Manifest, map[string]string{"API_KEY": "secret"}); err != nil {
		t.Fatal(err)
	}
	if err := service.installArchive("voice-bridge", first); err != nil {
		t.Fatal(err)
	}

	duplicateManifest := strings.Replace(validManifest(), "com.sipeed.voice_bridge", "com.sipeed.other_name", 1)
	second, err := inspectArchive(makeZIP(t, map[string]string{
		"other-name/app.json": duplicateManifest,
		"other-name/main.py":  "print('second')\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	second.Manifest.AppID = "com.sipeed.voice_bridge"
	if err := service.installArchive("other-name", second); err == nil {
		t.Fatal("expected duplicate app_id to be rejected")
	}
}

func TestBuiltinSipeedServerIsAlwaysPresentAndDeduplicated(t *testing.T) {
	service := newService(t.TempDir())
	servers, err := service.readServers()
	if err != nil || len(servers) != 1 || !servers[0].Builtin ||
		servers[0].Name != "Sipeed" || servers[0].URL != "https://github.com/sipeed/NanoKVM-Go-Apps" {
		t.Fatalf("unexpected built-in servers: %#v, %v", servers, err)
	}

	stored := []Server{
		{ID: "legacy-sipeed", Name: "Old Sipeed", URL: "https://github.com/sipeed/NanoKVM-Go-Apps/", Official: true},
		{ID: "community", Name: "Community", URL: "https://example.com/apps"},
	}
	data, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(service.root, "_servers.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	servers, err = service.readServers()
	if err != nil || len(servers) != 2 || servers[0].ID != "sipeed" ||
		!servers[0].Builtin || servers[1].ID != "community" || servers[1].Builtin {
		t.Fatalf("unexpected merged servers: %#v, %v", servers, err)
	}
}

func TestBuiltinSipeedServerSurvivesInvalidCustomConfiguration(t *testing.T) {
	service := newService(t.TempDir())
	config := filepath.Join(service.root, "_servers.json")
	for _, data := range [][]byte{
		{},
		[]byte(`{"invalid"`),
		[]byte(`[{"id":"INVALID ID","name":"Broken","url":"not-a-url"}]`),
	} {
		if err := os.WriteFile(config, data, 0o600); err != nil {
			t.Fatal(err)
		}
		servers, err := service.readServers()
		if err != nil || len(servers) != 1 || servers[0] != builtinSipeedServer {
			t.Fatalf("built-in server lost for invalid config %q: %#v, %v", data, servers, err)
		}
	}
}

func TestInstallRunsPreAndPostScriptsWithAppEnvironment(t *testing.T) {
	service := newService(t.TempDir())
	manifest := strings.Replace(validManifest(), `"version":"1.0.0"`,
		`"version":"1.0.0","pre_script":"scripts/pre.sh","post_script":"scripts/post.sh"`, 1)
	archive, err := inspectArchive(makeZIP(t, map[string]string{
		"voice-bridge/app.json":        manifest,
		"voice-bridge/main.py":         "print('ok')\n",
		"voice-bridge/scripts/pre.sh":  "#!/bin/bash\nset -eu\nprintf '%s:%s' \"$NANOKVM_APP_ID\" \"$LANG\" | tee pre.txt\n",
		"voice-bridge/scripts/post.sh": "#!/bin/bash\nset -eu\ntest -f pre.txt\nprintf '%s' \"$NANOKVM_APP_PHASE\" > post.txt\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if err := applyEnv(&archive.Manifest, map[string]string{"API_KEY": "secret", "LANG": "en"}); err != nil {
		t.Fatal(err)
	}
	var installLog bytes.Buffer
	if err := service.installArchiveWithLog("voice-bridge", archive, &installLog); err != nil {
		t.Fatal(err)
	}
	if text := installLog.String(); !strings.Contains(text, "com.sipeed.voice_bridge:en") ||
		!strings.Contains(text, "Installation completed successfully.") {
		t.Fatalf("missing streamed install output: %q", text)
	}
	dir := filepath.Join(service.root, "voice-bridge")
	pre, err := os.ReadFile(filepath.Join(dir, "pre.txt"))
	if err != nil || string(pre) != "com.sipeed.voice_bridge:en" {
		t.Fatalf("unexpected pre_script output %q: %v", pre, err)
	}
	post, err := os.ReadFile(filepath.Join(dir, "post.txt"))
	if err != nil || string(post) != "post_script" {
		t.Fatalf("unexpected post_script output %q: %v", post, err)
	}
}

func TestInstallScriptFailureDoesNotInstallApp(t *testing.T) {
	service := newService(t.TempDir())
	manifest := strings.Replace(validManifest(), `"version":"1.0.0"`,
		`"version":"1.0.0","pre_script":"pre.sh"`, 1)
	archive, err := inspectArchive(makeZIP(t, map[string]string{
		"voice-bridge/app.json": manifest,
		"voice-bridge/main.py":  "print('ok')\n",
		"voice-bridge/pre.sh":   "#!/bin/bash\necho expected failure\nexit 7\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if err := applyEnv(&archive.Manifest, map[string]string{"API_KEY": "secret"}); err != nil {
		t.Fatal(err)
	}
	err = service.installArchive("voice-bridge", archive)
	if err == nil || !strings.Contains(err.Error(), "expected failure") {
		t.Fatalf("unexpected pre_script error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(service.root, "voice-bridge")); !os.IsNotExist(err) {
		t.Fatalf("failed App must not be installed: %v", err)
	}
}

func TestInstallArchiveFailsClosedWhenRandomIdentifierUnavailable(t *testing.T) {
	service := newService(t.TempDir())
	archive, err := inspectArchive(makeZIP(t, map[string]string{
		"voice-bridge/app.json": validManifest(),
		"voice-bridge/main.py":  "print('ok')\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	original := readAppInstallRandom
	readAppInstallRandom = func([]byte) (int, error) { return 0, errors.New("entropy unavailable") }
	t.Cleanup(func() { readAppInstallRandom = original })

	if err := service.installArchive("voice-bridge", archive); err == nil || !strings.Contains(err.Error(), "identifier") {
		t.Fatalf("expected random identifier error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(service.root, "voice-bridge")); !os.IsNotExist(err) {
		t.Fatalf("App was installed without a secure identifier: %v", err)
	}
}

func TestInstallScriptCancellationKillsProcessGroup(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "pre.sh")
	if err := os.WriteFile(script, []byte("#!/bin/bash\nsleep 30\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	started := time.Now()
	err := runInstallScript(ctx, dir, "pre.sh", "pre_script", Manifest{}, nil)
	if err == nil || !strings.Contains(err.Error(), "installation cancelled") {
		t.Fatalf("unexpected cancellation error: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("cancelled script took too long to stop: %s", elapsed)
	}
}

func TestManifestRejectsUnsafeInstallScriptPaths(t *testing.T) {
	for _, script := range []string{"/tmp/install.sh", "../install.sh", "scripts/../install.sh", `scripts\\install.sh`} {
		manifest := strings.Replace(validManifest(), `"version":"1.0.0"`,
			`"version":"1.0.0","pre_script":`+fmt.Sprintf("%q", script), 1)
		if _, err := parseManifest([]byte(manifest)); err == nil {
			t.Fatalf("accepted unsafe pre_script %q", script)
		}
	}
}
