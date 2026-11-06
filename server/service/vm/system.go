package vm

import (
	"errors"
	"os"
	"os/exec"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const flashingModeFlag = "/boot/rec"

var (
	rebootFunc                            = reboot
	rebootToFlashingFunc                  = func() error { return rebootToFlashing(flashingModeFlag, reboot) }
	nanoKVMRestartActionRequiresChallenge = authservice.NanoKVMRestartActionRequiresChallenge
	verifyNanoKVMRestartActionChallenge   = authservice.VerifySensitiveActionChallenge
)

func (s *Service) Reboot(c *gin.Context) {
	s.restart(c, rebootFunc, "system rebooted")
}

func (s *Service) RebootToFlashing(c *gin.Context) {
	s.restart(c, rebootToFlashingFunc, "system rebooting to flashing mode")
}

func (s *Service) restart(c *gin.Context, restartFunc func() error, successMessage string) {
	var rsp proto.Response

	if nanoKVMRestartActionRequiresChallenge() {
		challenge := authservice.ReadSensitiveChallenge(c)
		if err := verifyNanoKVMRestartActionChallenge(c, challenge, authservice.SensitiveActionNanoKVMRestart); err != nil {
			log.Errorf("verify NanoKVM restart challenge failed: %s", err)
			rsp.ErrRsp(c, -3, "sensitive action verification required")
			return
		}
	}

	if err := restartFunc(); err != nil {
		rsp.ErrRsp(c, -1, "operation failed")
		return
	}

	rsp.OkRsp(c)
	log.Debug(successMessage)
}

func reboot() error {
	log.Println("reboot system...")

	err := exec.Command("reboot").Start()
	if err != nil {
		log.Errorf("failed to reboot: %s", err)
		return err
	}

	return nil
}

func rebootToFlashing(flagPath string, rebootSystem func() error) error {
	created, err := createFlashingModeFlag(flagPath)
	if err != nil {
		log.Errorf("failed to create flashing mode flag: %s", err)
		return err
	}

	if err := rebootSystem(); err != nil {
		if created {
			if removeErr := os.Remove(flagPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				log.Errorf("failed to remove flashing mode flag after reboot failure: %s", removeErr)
			}
		}
		return err
	}

	return nil
}

func createFlashingModeFlag(path string) (bool, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return false, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return false, err
	}

	return true, nil
}
