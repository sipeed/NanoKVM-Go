package vm

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"
)

const (
	zramDevice       = "/dev/zram0"
	zramSwapUnit     = "dev-zram0.swap"
	zramSetupService = "systemd-zram-setup@zram0.service"
)

type swapState struct {
	enabled     bool
	setupMasked bool
}

var errSwapDisableInsufficientMemory = errors.New("Not enough free memory to disable swap. Free up memory and try again.")

func (s *Service) GetSwap(c *gin.Context) {
	var rsp proto.Response

	rsp.OkRspWithData(c, &proto.GetSwapRsp{
		Enabled: isSwapEnabled(),
	})
}

func (s *Service) SetSwap(c *gin.Context) {
	var req proto.SetSwapReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	enable := *req.Enable
	if enable {
		if err := enableSwap(); err != nil {
			log.Errorf("failed to enable swap: %v", err)
			rsp.ErrRsp(c, -2, swapErrorMessage(true, err))
			return
		}
	} else if err := disableSwap(); err != nil {
		log.Errorf("failed to disable swap: %v", err)
		rsp.ErrRsp(c, -3, swapErrorMessage(false, err))
		return
	}

	rsp.OkRsp(c)
	log.Debugf("set swap: %v", enable)
}

func enableSwap() error {
	previous, err := getSwapState()
	if err != nil {
		return err
	}

	target := swapState{
		enabled:     true,
		setupMasked: false,
	}
	if err := applySwapState(target); err != nil {
		return rollbackSwapState(previous, err)
	}
	if err := verifySwapState(target); err != nil {
		return rollbackSwapState(previous, err)
	}

	return nil
}

func disableSwap() error {
	previous, err := getSwapState()
	if err != nil {
		return err
	}

	target := swapState{
		enabled:     false,
		setupMasked: true,
	}
	if err := applySwapState(target); err != nil {
		return rollbackSwapState(previous, err)
	}
	if err := verifySwapState(target); err != nil {
		return rollbackSwapState(previous, err)
	}

	return nil
}

func applySwapState(state swapState) error {
	current, err := getSwapState()
	if err != nil {
		return err
	}

	if state.enabled {
		if err := setZramSetupMasked(false); err != nil {
			return err
		}
		if !current.enabled {
			if err := utils.StartUnit(zramSwapUnit); err != nil {
				return err
			}
		}
		return setZramSetupMasked(state.setupMasked)
	}

	if current.enabled {
		if err := utils.StopUnit(zramSwapUnit); err != nil {
			if stillEnabled, readErr := readSwapEnabled(); readErr == nil && stillEnabled {
				return fmt.Errorf("%w: %v", errSwapDisableInsufficientMemory, err)
			}
			return err
		}
	}

	return setZramSetupMasked(state.setupMasked)
}

func swapErrorMessage(enable bool, err error) string {
	if errors.Is(err, errSwapDisableInsufficientMemory) {
		return errSwapDisableInsufficientMemory.Error()
	}
	if enable {
		return "Failed to enable swap. Please try again."
	}

	return "Failed to disable swap. Please try again."
}

func rollbackSwapState(previous swapState, cause error) error {
	current, err := getSwapState()
	if err == nil && current == previous {
		return cause
	}
	if err != nil {
		log.Errorf("failed to read swap state before rollback after error %v: %v", cause, err)
	}

	if err := applySwapState(previous); err != nil {
		log.Errorf("failed to rollback swap state after error %v: %v", cause, err)
		return fmt.Errorf("%w; rollback failed: %v", cause, err)
	}
	if err := verifySwapState(previous); err != nil {
		log.Errorf("failed to verify rollback swap state after error %v: %v", cause, err)
		return fmt.Errorf("%w; rollback verification failed: %v", cause, err)
	}

	return cause
}

func getSwapState() (swapState, error) {
	enabled, err := readSwapEnabled()
	if err != nil {
		return swapState{}, err
	}

	setupMasked, err := utils.IsUnitMasked(zramSetupService)
	if err != nil {
		return swapState{}, err
	}

	return swapState{
		enabled:     enabled,
		setupMasked: setupMasked,
	}, nil
}

func verifySwapState(expected swapState) error {
	actual, err := getSwapState()
	if err != nil {
		return err
	}

	if actual != expected {
		return fmt.Errorf(
			"swap state mismatch: expected enabled=%v setupMasked=%v, got enabled=%v setupMasked=%v",
			expected.enabled,
			expected.setupMasked,
			actual.enabled,
			actual.setupMasked,
		)
	}

	return nil
}

func isSwapEnabled() bool {
	enabled, err := readSwapEnabled()
	if err != nil {
		log.Errorf("failed to read swap state: %v", err)
		return false
	}

	return enabled
}

func readSwapEnabled() (bool, error) {
	content, err := os.ReadFile("/proc/swaps")
	if err != nil {
		return false, fmt.Errorf("read /proc/swaps failed: %w", err)
	}

	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == zramDevice {
			return true, nil
		}
	}

	return false, nil
}

func setZramSetupMasked(masked bool) error {
	if masked {
		return utils.MaskUnit(zramSetupService)
	}

	return utils.UnmaskUnit(zramSetupService)
}
