package application

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const updateSignatureVerificationDisabledFlag = "/etc/kvm/update_signature_verification_disabled"

const updateSignatureVerificationDisabledMarker = "disabled\n"

var (
	updateSignatureVerificationMu       sync.RWMutex
	updateSignatureVerificationFlagPath = updateSignatureVerificationDisabledFlag
	verifyUpdateSignatureChallenge      = authservice.VerifySensitiveActionChallenge
	applicationUpdatePolicyLockPath     = applicationUpdateLockPath
	applicationUpdatePolicyUnitBusy     = applicationInstallUnitBlocksAdmission
)

func (s *Service) GetSignatureVerification(c *gin.Context) {
	var rsp proto.Response

	enabled, err := updateSignatureVerificationEnabled()
	if err != nil {
		log.Errorf("failed to read application update signature-verification setting: %s", err)
		rsp.ErrRsp(c, -1, "get signature verification failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetUpdateSignatureVerificationRsp{Enabled: enabled})
}

func (s *Service) SetSignatureVerification(c *gin.Context) {
	var req proto.SetUpdateSignatureVerificationReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil || req.Enabled == nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if err := verifyUpdateSignatureChallenge(
		c,
		strings.TrimSpace(req.Challenge),
		authservice.SensitiveActionUpdateSignatureVerification,
	); err != nil {
		log.Warnf("failed to authorize application update signature-verification change: %s", err)
		rsp.ErrRsp(c, -3, "sensitive action verification required")
		return
	}
	if err := setUpdateSignatureVerificationEnabled(*req.Enabled); err != nil {
		if errors.Is(err, errApplicationUpdateBusy) {
			rsp.ErrRsp(c, -2, "application update is busy")
			return
		}
		log.Errorf("failed to persist application update signature-verification setting: %s", err)
		rsp.ErrRsp(c, -4, "set signature verification failed")
		return
	}

	log.Warnf("application update signature verification changed: enabled=%t", *req.Enabled)
	rsp.OkRspWithData(c, &proto.GetUpdateSignatureVerificationRsp{Enabled: *req.Enabled})
}

// updateSignatureVerificationEnabled fails closed. A missing marker is the
// secure default for existing devices and after factory reset.
func updateSignatureVerificationEnabled() (bool, error) {
	updateSignatureVerificationMu.RLock()
	defer updateSignatureVerificationMu.RUnlock()

	file, err := os.OpenFile(
		updateSignatureVerificationFlagPath,
		os.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW,
		0,
	)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return true, fmt.Errorf("open signature-verification marker: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return true, fmt.Errorf("inspect signature-verification marker: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return true, fmt.Errorf("signature-verification marker is not a regular file")
	}
	if info.Mode().Perm() != 0o600 || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return true, fmt.Errorf("signature-verification marker has unsafe permissions")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) || stat.Nlink != 1 {
		return true, fmt.Errorf("signature-verification marker has unsafe ownership or link count")
	}
	if info.Size() != int64(len(updateSignatureVerificationDisabledMarker)) {
		return true, fmt.Errorf("signature-verification marker has invalid size")
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(len(updateSignatureVerificationDisabledMarker)+1)))
	if err != nil {
		return true, err
	}
	if string(data) != updateSignatureVerificationDisabledMarker {
		return true, fmt.Errorf("signature-verification marker has invalid contents")
	}
	return false, nil
}

func setUpdateSignatureVerificationEnabled(enabled bool) error {
	// Serialize setting changes with application update admission. This makes
	// the value captured in each worker job immutable for the whole operation.
	lock, err := openAndLockApplicationFile(applicationUpdatePolicyLockPath, true)
	if err != nil {
		return err
	}
	defer unlockApplicationFile(lock)

	busy, err := applicationUpdatePolicyUnitBusy()
	if err != nil {
		return err
	}
	if busy {
		return errApplicationUpdateBusy
	}

	updateSignatureVerificationMu.Lock()
	defer updateSignatureVerificationMu.Unlock()

	if enabled {
		changed := true
		if _, err := os.Lstat(updateSignatureVerificationFlagPath); errors.Is(err, os.ErrNotExist) {
			changed = false
		} else if err != nil {
			return err
		}
		if err := os.Remove(updateSignatureVerificationFlagPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if !changed {
			return nil
		}
		return syncUpdateSignatureVerificationDirectory()
	}

	directory := filepath.Dir(updateSignatureVerificationFlagPath)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(directory, ".update-signature-verification-disabled-*")
	if err != nil {
		return err
	}
	temporaryPath := file.Name()
	keepTemporary := true
	defer func() {
		if keepTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.WriteString(updateSignatureVerificationDisabledMarker); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, updateSignatureVerificationFlagPath); err != nil {
		return err
	}
	keepTemporary = false
	return syncUpdateSignatureVerificationDirectory()
}

func syncUpdateSignatureVerificationDirectory() error {
	directory, err := os.Open(filepath.Dir(updateSignatureVerificationFlagPath))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
