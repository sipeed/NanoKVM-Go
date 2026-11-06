package utils

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	log "github.com/sirupsen/logrus"
)

var (
	systemctlClient    systemdClient
	systemctlClientMu  sync.Mutex
	newSystemctlClient = func() (systemdClient, error) {
		// This context controls the bus lifetime; per-call timeouts are applied to methods.
		return dbus.NewWithContext(context.Background())
	}
)

const systemctlJobTimeout = 45 * time.Second

type systemdClient interface {
	GetUnitPropertiesContext(ctx context.Context, unit string) (map[string]interface{}, error)
	StartUnitContext(ctx context.Context, name string, mode string, ch chan<- string) (int, error)
	StopUnitContext(ctx context.Context, name string, mode string, ch chan<- string) (int, error)
	RestartUnitContext(ctx context.Context, name string, mode string, ch chan<- string) (int, error)
	EnableUnitFilesContext(ctx context.Context, files []string, runtime bool, force bool) (bool, []dbus.EnableUnitFileChange, error)
	DisableUnitFilesContext(ctx context.Context, files []string, runtime bool) ([]dbus.DisableUnitFileChange, error)
	MaskUnitFilesContext(ctx context.Context, files []string, runtime bool, force bool) ([]dbus.MaskUnitFileChange, error)
	UnmaskUnitFilesContext(ctx context.Context, files []string, runtime bool) ([]dbus.UnmaskUnitFileChange, error)
	ListUnitFilesByPatternsContext(ctx context.Context, states []string, patterns []string) ([]dbus.UnitFile, error)
	ReloadContext(ctx context.Context) error
}

func getSystemctlClient() (systemdClient, error) {
	systemctlClientMu.Lock()
	defer systemctlClientMu.Unlock()

	if systemctlClient != nil {
		return systemctlClient, nil
	}

	client, err := newSystemctlClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to systemd bus: %w", err)
	}

	systemctlClient = client
	return systemctlClient, nil
}

func IsServiceRunning(servicename string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), systemctlJobTimeout)
	defer cancel()

	client, err := getSystemctlClient()
	if err != nil {
		return false, err
	}

	properties, err := client.GetUnitPropertiesContext(ctx, servicename)
	if err != nil {
		return false, err
	}

	loadState, exists := properties["LoadState"]
	if !exists {
		return false, fmt.Errorf("LoadState property not found")
	}

	if loadStateStr, ok := loadState.(string); ok && loadStateStr == "not-found" {
		return false, fmt.Errorf("service not found")
	}

	activeState, exists := properties["ActiveState"]
	if !exists {
		return false, fmt.Errorf("ActiveState property not found")
	}

	activeStateStr, ok := activeState.(string)
	if !ok {
		return false, fmt.Errorf("ActiveState is not a string")
	}

	subState, exists := properties["SubState"]
	if !exists {
		return false, fmt.Errorf("SubState property not found")
	}

	subStateStr, ok := subState.(string)
	if !ok {
		return false, fmt.Errorf("SubState is not a string")
	}

	subStateActive := subStateStr == "listening" || subStateStr == "running"
	// is running
	return activeStateStr == "active" && subStateActive, nil
}

func StartUnit(name string) error {
	return runUnitJob(name, "start", func(client systemdClient, ctx context.Context, ch chan<- string) error {
		_, err := client.StartUnitContext(ctx, name, "replace", ch)
		return err
	})
}

func StopUnit(name string) error {
	return runUnitJob(name, "stop", func(client systemdClient, ctx context.Context, ch chan<- string) error {
		_, err := client.StopUnitContext(ctx, name, "replace", ch)
		return err
	})
}

func MaskUnit(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), systemctlJobTimeout)
	defer cancel()

	client, err := getSystemctlClient()
	if err != nil {
		return err
	}

	if _, err := client.MaskUnitFilesContext(ctx, []string{name}, false, true); err != nil {
		return fmt.Errorf("failed to mask unit %s: %w", name, err)
	}

	return reloadSystemdManager(ctx, client)
}

func UnmaskUnit(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), systemctlJobTimeout)
	defer cancel()

	client, err := getSystemctlClient()
	if err != nil {
		return err
	}

	if _, err := client.UnmaskUnitFilesContext(ctx, []string{name}, false); err != nil {
		return fmt.Errorf("failed to unmask unit %s: %w", name, err)
	}

	return reloadSystemdManager(ctx, client)
}

func UnitFileState(name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), systemctlJobTimeout)
	defer cancel()

	client, err := getSystemctlClient()
	if err != nil {
		return "", err
	}

	patterns := []string{name}
	if templateName := unitTemplateName(name); templateName != "" {
		patterns = append(patterns, templateName)
	}

	files, err := client.ListUnitFilesByPatternsContext(ctx, []string{}, patterns)
	if err != nil {
		return "", fmt.Errorf("failed to get unit file state for %s: %w", name, err)
	}

	templateName := unitTemplateName(name)
	templateState := ""
	for _, file := range files {
		fileName := filepath.Base(file.Path)
		if file.Path == name || fileName == name {
			return file.Type, nil
		}
		if templateName != "" && (file.Path == templateName || fileName == templateName) {
			templateState = file.Type
		}
	}

	if templateState != "" {
		return templateState, nil
	}

	return "", fmt.Errorf("unit file %s not found", name)
}

func unitTemplateName(name string) string {
	at := strings.IndexByte(name, '@')
	if at == -1 {
		return ""
	}

	suffix := filepath.Ext(name)
	if suffix == "" {
		return ""
	}

	return name[:at+1] + suffix
}

func IsUnitMasked(name string) (bool, error) {
	state, err := UnitFileState(name)
	if state == "masked" || state == "masked-runtime" {
		return true, nil
	}
	if state == "" {
		if err != nil {
			return false, err
		}
		return false, nil
	}
	if isKnownUnmaskedUnitFileState(state) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return false, fmt.Errorf("unexpected unit file state for %s: %s", name, state)
}

func reloadSystemdManager(ctx context.Context, client systemdClient) error {
	if err := client.ReloadContext(ctx); err != nil {
		return fmt.Errorf("failed to reload systemd manager: %w", err)
	}

	return nil
}

func runUnitJob(name string, action string, run func(systemdClient, context.Context, chan<- string) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), systemctlJobTimeout)
	defer cancel()

	client, err := getSystemctlClient()
	if err != nil {
		return err
	}

	ch := make(chan string, 1)
	if err := run(client, ctx, ch); err != nil {
		return fmt.Errorf("failed to %s unit %s: %w", action, name, err)
	}

	select {
	case result := <-ch:
		if result != "done" {
			return fmt.Errorf("unit %s failed: %s", action, result)
		}
	case <-ctx.Done():
		return fmt.Errorf("unit %s timed out: %w", action, ctx.Err())
	}

	return nil
}

func isKnownUnmaskedUnitFileState(state string) bool {
	switch state {
	case "enabled", "enabled-runtime", "linked", "linked-runtime", "alias", "static", "indirect", "disabled", "generated", "transient":
		return true
	default:
		return false
	}
}

func DaemonReload() error {
	ctx, cancel := context.WithTimeout(context.Background(), systemctlJobTimeout)
	defer cancel()

	client, err := getSystemctlClient()
	if err != nil {
		log.Errorf("Failed to connect systemd D-Bus: %v", err)
		return err
	}

	if err := reloadSystemdManager(ctx, client); err != nil {
		log.Errorf("Failed to execute daemon-reload: %v", err)
		return err
	}

	log.Debugf("Systemd daemon-reload completed successfully.")
	return nil
}

func StartService(name string, enable bool) error {
	if enable {
		if err := enableUnitFile(name); err != nil {
			return err
		}

		if err := RestartService(name); err != nil {
			log.Debugf("restart service failed %v", err)
			return err
		}

		return nil
	}

	return StartUnit(name)
}

func RestartService(serviceName string) error {
	return runUnitJob(serviceName, "restart", func(client systemdClient, ctx context.Context, ch chan<- string) error {
		_, err := client.RestartUnitContext(ctx, serviceName, "replace", ch)
		return err
	})
}

func StopService(name string, disable bool) error {
	if err := StopUnit(name); err != nil {
		return err
	}

	if disable {
		if err := disableUnitFile(name); err != nil {
			return err
		}
	}

	return nil
}

func enableUnitFile(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), systemctlJobTimeout)
	defer cancel()

	client, err := getSystemctlClient()
	if err != nil {
		return err
	}

	if _, _, err := client.EnableUnitFilesContext(ctx, []string{name}, false, true); err != nil {
		return fmt.Errorf("failed to enable service %s: %w", name, err)
	}

	return reloadSystemdManager(ctx, client)
}

func disableUnitFile(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), systemctlJobTimeout)
	defer cancel()

	client, err := getSystemctlClient()
	if err != nil {
		return err
	}

	if _, err := client.DisableUnitFilesContext(ctx, []string{name}, false); err != nil {
		return fmt.Errorf("failed to disable service %s: %w", name, err)
	}

	return reloadSystemdManager(ctx, client)
}
