package ai

import (
	"errors"
	"os"
)

const (
	defaultMCPEnableFile             = "/tmp/mcp_enable"
	defaultScreenTimelapseEnableFile = "/tmp/scr_enable"
)

var (
	mcpEnableFilePath             = defaultMCPEnableFile
	screenTimelapseEnableFilePath = defaultScreenTimelapseEnableFile
)

func stateFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func setStateFile(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	return file.Close()
}

func removeStateFile(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func syncMCPStateFile(enabled bool) error {
	if enabled {
		return setStateFile(mcpEnableFilePath)
	}
	return removeStateFile(mcpEnableFilePath)
}

func applyMCPStateFile(cfg Config) (Config, error) {
	enabled, err := stateFileExists(mcpEnableFilePath)
	if err != nil {
		return Config{}, err
	}
	cfg.MCPEnabled = enabled
	return cfg, nil
}

func syncScreenTimelapseStateFile(enabled bool) error {
	if enabled {
		return setStateFile(screenTimelapseEnableFilePath)
	}
	return removeStateFile(screenTimelapseEnableFilePath)
}

func readScreenTimelapseStateFile() (bool, error) {
	return stateFileExists(screenTimelapseEnableFilePath)
}
