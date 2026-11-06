package vm

import (
	"strings"
	"testing"
)

func TestValidateScriptNameRejectsInvalidNames(t *testing.T) {
	tests := []string{
		"../safe.sh",
		"sub/safe.sh",
		`sub\safe.sh`,
		"safe.sh;id",
		"safe.sh && id",
		"safe.sh`id`",
		"safe.sh$(id)",
		"safe.txt",
		".sh",
		"",
	}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := validateScriptName(name); err == nil {
				t.Fatal("expected invalid script name to be rejected")
			}
		})
	}
}

func TestValidateScriptNameAcceptsBasenameWithAllowedCharacters(t *testing.T) {
	tests := []string{"safe.sh", "safe-test_01.py", "SAFE.02.SH"}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			validated, err := validateScriptName(name)
			if err != nil {
				t.Fatal(err)
			}
			if validated != name {
				t.Fatalf("expected %q, got %q", name, validated)
			}
		})
	}
}

func TestResolveScriptPathRejectsShellMetacharacters(t *testing.T) {
	_, err := resolveScriptPath("safe.sh; touch /tmp/nanokvm-run-script-injection-poc")
	if err == nil {
		t.Fatal("expected shell metacharacters to be rejected")
	}
}

func TestResolveScriptPathRejectsPathTraversal(t *testing.T) {
	_, err := resolveScriptPath("../safe.sh")
	if err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
}

func TestScriptCommandUsesFixedExecutableAndArguments(t *testing.T) {
	cmd, err := scriptCommand("safe.sh", ScriptDirectory+"/safe.sh")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path != "/bin/sh" || len(cmd.Args) != 2 || cmd.Args[1] != ScriptDirectory+"/safe.sh" {
		t.Fatalf("unexpected shell command: path=%q args=%v", cmd.Path, cmd.Args)
	}
	if cmd.Dir != ScriptDirectory {
		t.Fatalf("expected command directory %q, got %q", ScriptDirectory, cmd.Dir)
	}
	if strings.Join(cmd.Env, "\n") != "PATH=/usr/sbin:/usr/bin:/sbin:/bin" {
		t.Fatalf("unexpected command env: %v", cmd.Env)
	}

	cmd, err = scriptCommand("safe.py", ScriptDirectory+"/safe.py")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(cmd.Path, "python3") || len(cmd.Args) != 2 || cmd.Args[1] != ScriptDirectory+"/safe.py" {
		t.Fatalf("unexpected python command: path=%q args=%v", cmd.Path, cmd.Args)
	}
}
