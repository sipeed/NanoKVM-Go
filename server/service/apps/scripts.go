package apps

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	installScriptTimeout = 15 * time.Minute
	installScriptLogMax  = 64 << 10
)

type limitedLog struct {
	buffer bytes.Buffer
	limit  int
}

func (w *limitedLog) Write(data []byte) (int, error) {
	original := len(data)
	remaining := w.limit - w.buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			data = data[:remaining]
		}
		_, _ = w.buffer.Write(data)
	}
	return original, nil
}

func runInstallScript(ctx context.Context, appDir, script, phase string, manifest Manifest, log io.Writer) error {
	if script == "" {
		return nil
	}
	scriptPath := filepath.Join(appDir, filepath.FromSlash(script))
	info, err := os.Lstat(scriptPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s %q is not a regular file", phase, script)
	}

	ctx, cancel := context.WithTimeout(ctx, installScriptTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "/bin/bash", scriptPath)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return nil
		}
		return syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
	command.Dir = appDir
	command.Env = installScriptEnvironment(appDir, phase, manifest)
	output := &limitedLog{limit: installScriptLogMax}
	if log == nil {
		log = io.Discard
	}
	combined := io.MultiWriter(output, log)
	command.Stdout = combined
	command.Stderr = combined
	if err := command.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			return errors.New("installation cancelled")
		}
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%s %q timed out after %s", phase, script, installScriptTimeout)
		}
		message := strings.TrimSpace(output.buffer.String())
		if message == "" {
			return fmt.Errorf("%s %q failed: %w", phase, script, err)
		}
		return fmt.Errorf("%s %q failed: %w: %s", phase, script, err, message)
	}
	return nil
}

func installScriptEnvironment(appDir, phase string, manifest Manifest) []string {
	values := make(map[string]string)
	for _, entry := range os.Environ() {
		if separator := strings.IndexByte(entry, '='); separator > 0 {
			values[entry[:separator]] = entry[separator+1:]
		}
	}
	for name, spec := range manifest.Env {
		value := spec.Value
		if value == "" {
			value = spec.Default
		}
		values[name] = value
	}
	values["NANOKVM_APP_DIR"] = appDir
	values["NANOKVM_APP_ID"] = manifest.AppID
	values["NANOKVM_APP_PHASE"] = phase

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	environment := make([]string, 0, len(names))
	for _, name := range names {
		environment = append(environment, name+"="+values[name])
	}
	return environment
}
