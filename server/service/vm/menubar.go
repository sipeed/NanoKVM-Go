package vm

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	MenuBarConfigFile = "/etc/kvm/menubar"
	menuBarPowerItem  = "power"
)

var defaultMenuBarDisabledItems = []string{menuBarPowerItem, "mouseJiggler"}

var (
	menuBarMutex          sync.RWMutex
	menuBarUpdateMutex    sync.Mutex
	menuBarReadFile       = os.ReadFile
	menuBarWriteFile      = writeMenuBarFileAtomically
	fingerRobotInitialize = InitializeFingerRobotController
	fingerRobotDisable    = DisableFingerRobotController
)

func (s *Service) GetMenuBarConfig(c *gin.Context) {
	var rsp proto.Response

	config, err := readMenuBarConfig()
	if err != nil {
		rsp.ErrRsp(c, -1, "get menu bar config failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetMenuBarConfigRsp{
		DisabledItems: config.DisabledItems,
	})

	log.Debugf("get menu bar config successful")
}

func (s *Service) SetMenuBarConfig(c *gin.Context) {
	var req proto.SetMenuBarConfigReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if req.DisabledItems == nil {
		rsp.ErrRsp(c, -1, "invalid arguments: disabledItems is required")
		return
	}

	menuBarUpdateMutex.Lock()
	defer menuBarUpdateMutex.Unlock()

	current, persisted, err := readMenuBarConfigWithState()
	stateKnown := err == nil && persisted && current != nil && current.DisabledItems != nil
	if !stateKnown {
		// Missing, unreadable, and structurally incomplete settings all fail
		// closed until this explicit update synchronizes the controller and
		// replaces the persisted state.
		current = defaultMenuBarConfig()
	}
	wasEnabled := menuBarFingerRobotEnabled(current)

	updated := &proto.MenuBarConfig{
		// Preserve an explicit empty array as an enabled setting. Persisting []
		// instead of null also keeps future startup checks fail-closed for
		// missing or malformed fields.
		DisabledItems: append([]string{}, req.DisabledItems...),
	}
	isEnabled := menuBarFingerRobotEnabled(updated)
	needsTransition := !stateKnown || wasEnabled != isEnabled
	if needsTransition {
		if err := setFingerRobotControllerEnabled(isEnabled); err != nil {
			log.Errorf("failed to set finger robot enabled=%t: %v", isEnabled, err)
			rollbackEnabled := stateKnown && wasEnabled
			if rollbackErr := setFingerRobotControllerEnabled(rollbackEnabled); rollbackErr != nil {
				log.Errorf("failed to roll back finger robot enabled=%t after transition error: %v", rollbackEnabled, rollbackErr)
			}
			rsp.ErrRsp(c, -3, "update finger robot state failed")
			return
		}
	}

	if err := writeMenuBarConfig(updated); err != nil {
		if needsTransition {
			rollbackEnabled := stateKnown && wasEnabled
			if rollbackErr := setFingerRobotControllerEnabled(rollbackEnabled); rollbackErr != nil {
				log.Errorf("failed to roll back finger robot enabled=%t: %v", rollbackEnabled, rollbackErr)
			}
		}
		rsp.ErrRsp(c, -2, "write config failed")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("set menu bar config: %+v", req)
}

func readMenuBarConfig() (*proto.MenuBarConfig, error) {
	config, _, err := readMenuBarConfigWithState()
	return config, err
}

// readMenuBarConfigWithState distinguishes a real persisted configuration
// from the synthetic disabled default returned for a missing file. Startup and
// GET use that default, while Set treats a missing file as unknown and
// idempotently synchronizes the controller to the user's explicit choice.
func readMenuBarConfigWithState() (*proto.MenuBarConfig, bool, error) {
	menuBarMutex.RLock()
	defer menuBarMutex.RUnlock()

	data, err := menuBarReadFile(MenuBarConfigFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultMenuBarConfig(), false, nil
		}

		log.Errorf("failed to read %s: %v", MenuBarConfigFile, err)
		return nil, false, err
	}

	var config proto.MenuBarConfig
	if err := json.Unmarshal(data, &config); err != nil {
		log.Errorf("failed to unmarshal menu bar: %v", err)
		return nil, false, err
	}

	return &config, true, nil
}

func defaultMenuBarConfig() *proto.MenuBarConfig {
	return &proto.MenuBarConfig{
		DisabledItems: append([]string(nil), defaultMenuBarDisabledItems...),
	}
}

// IsFingerRobotSettingEnabled reports the persisted settings-switch state. A
// missing menu-bar file uses the default disabled configuration; malformed or
// unreadable files fail closed so server startup never initializes PWM by
// accident.
func IsFingerRobotSettingEnabled() bool {
	config, err := readMenuBarConfig()
	if err != nil {
		return false
	}
	return menuBarFingerRobotEnabled(config)
}

func menuBarFingerRobotEnabled(config *proto.MenuBarConfig) bool {
	return config != nil && config.DisabledItems != nil && !slices.Contains(config.DisabledItems, menuBarPowerItem)
}

func setFingerRobotControllerEnabled(enabled bool) error {
	if enabled {
		err := fingerRobotInitialize()
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrFingerRobotInvalid) || errors.Is(err, ErrFingerRobotBusy) ||
			errors.Is(err, ErrFingerRobotClosed) || errors.Is(err, ErrFingerRobotUnavailable) {
			return err
		}
		// Enabling is a persisted user choice, not proof that the servo is
		// already connected. Keep the setting enabled while the controller's
		// background monitor waits for PWM feedback or retries hardware setup.
		log.Warnf("finger robot enabled but currently unavailable; background retry remains active: %v", err)
		return nil
	}
	return fingerRobotDisable()
}

func writeMenuBarConfig(config *proto.MenuBarConfig) error {
	menuBarMutex.Lock()
	defer menuBarMutex.Unlock()

	data, err := json.Marshal(config)
	if err != nil {
		log.Errorf("failed to marshal menu bar: %v", err)
		return err
	}

	if err := menuBarWriteFile(MenuBarConfigFile, data, 0o644); err != nil {
		log.Errorf("failed to write %s: %v", MenuBarConfigFile, err)
		return err
	}

	log.Debugf("write to %s success", MenuBarConfigFile)
	return nil
}

// writeMenuBarFileAtomically guarantees that an error leaves the old target
// intact. SetMenuBarConfig relies on that boundary when it rolls the hardware
// lifecycle back after persistence fails.
func writeMenuBarFileAtomically(path string, data []byte, mode os.FileMode) (err error) {
	dir := filepath.Dir(path)
	temporary, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		if closeErr := temporary.Close(); err == nil && closeErr != nil && !committed {
			err = closeErr
		}
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err = temporary.Chmod(mode); err != nil {
		return err
	}
	if _, err = temporary.Write(data); err != nil {
		return err
	}
	if err = temporary.Sync(); err != nil {
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	if err = os.Rename(temporaryPath, path); err != nil {
		return err
	}
	committed = true

	// The rename is already the commit boundary. A directory fsync failure must
	// not be reported as an uncommitted write, or the caller could roll hardware
	// back to a state that no longer matches the file.
	if directory, openErr := os.Open(dir); openErr == nil {
		if syncErr := directory.Sync(); syncErr != nil {
			log.Warnf("failed to sync menu-bar directory %s: %v", dir, syncErr)
		}
		_ = directory.Close()
	} else {
		log.Warnf("failed to open menu-bar directory %s for sync: %v", dir, openErr)
	}

	return nil
}
