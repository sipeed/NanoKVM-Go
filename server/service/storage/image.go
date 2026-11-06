package storage

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"
	"NanoKVM-Server/service/hid"
	"NanoKVM-Server/storageguard"
)

const (
	imageDirectory = "/data"
	mountDevice    = "/sys/kernel/config/usb_gadget/g0/configs/c.1/mass_storage.disk0/lun.0/file"
	cdromFlag      = "/sys/kernel/config/usb_gadget/g0/configs/c.1/mass_storage.disk0/lun.0/cdrom"
	roFlag         = "/sys/kernel/config/usb_gadget/g0/configs/c.1/mass_storage.disk0/lun.0/ro"
	usbDisk        = "/boot/usb.disk0"
)

func ensureImageDirectory(directory string) error {
	info, err := os.Lstat(directory)
	if errors.Is(err, os.ErrNotExist) {
		if mkdirErr := os.Mkdir(directory, 0o755); mkdirErr != nil && !errors.Is(mkdirErr, os.ErrExist) {
			return fmt.Errorf("create image directory: %w", mkdirErr)
		}
		info, err = os.Lstat(directory)
	}
	if err != nil {
		return fmt.Errorf("inspect image directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("image directory is not a directory")
	}
	return nil
}

var (
	deleteImageActionRequiresChallenge = authservice.DeleteImageActionRequiresChallenge
	verifyDeleteImageActionChallenge   = authservice.VerifySensitiveActionChallenge
	removeImageFile                    = os.Remove
	resolveDeleteImagePath             = safeImageDeletePath
	acquireStorageAccess               = storageguard.Local
)

func (s *Service) GetImages(c *gin.Context) {
	var rsp proto.Response
	var images []string
	release, err := acquireStorageAccess(false)
	if err != nil {
		rsp.ErrRsp(c, -1, err.Error())
		return
	}
	defer release()
	if err := ensureImageDirectory(imageDirectory); err != nil {
		log.Errorf("failed to prepare image directory: %s", err)
		rsp.ErrRsp(c, -1, "image directory unavailable")
		return
	}

	err = filepath.Walk(imageDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			name := strings.ToLower(info.Name())
			if strings.HasSuffix(name, ".iso") || strings.HasSuffix(name, ".img") {
				images = append(images, path)
			}
		}

		return nil
	})

	if err != nil {
		log.Errorf("failed to get images: %s", err)
	}

	rsp.OkRspWithData(c, &proto.GetImagesRsp{
		Files: images,
	})
	log.Debugf("get images success, total %d", len(images))
}

func (s *Service) MountImage(c *gin.Context) {
	var req proto.MountImageReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	// Check the lexical boundary before rebuilding USB; resolve the real file
	// only after disk1 is released and /data has a fresh filesystem cache.
	if req.File != "" && !validImageName(req.File, imageDirectory) {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	hid.LockUsbGadgetMutation()
	defer hid.UnlockUsbGadgetMutation()
	storageLock, err := storageguard.Exclusive()
	if err != nil {
		rsp.ErrRsp(c, -2, err.Error())
		return
	}
	defer storageLock.Close()
	h := hid.GetHid()
	if err := h.ResetInputRouting("virtual media rebuild"); err != nil {
		log.Warnf("failed to release routed input before virtual media rebuild: %s", err)
	}
	h.Lock()
	h.CloseNoLock()
	defer func() {
		h.OpenNoLock()
		h.Unlock()
	}()

	// unmount
	if req.File == "" {
		if err := os.WriteFile(mountDevice, []byte("\n"), 0o666); err != nil {
			log.Errorf("unmount file failed: %s", err)
			rsp.ErrRsp(c, -3, "unmount image failed")
			return
		}
		if err := restartUSB(0, storageLock); err != nil {
			rsp.ErrRsp(c, -3, "unmount image failed")
			return
		}
		time.Sleep(3 * time.Second)
		rsp.OkRsp(c)
		return
	}

	if err := restartUSB(1, storageLock); err != nil {
		rollbackImageMount(storageLock)
		rsp.ErrRsp(c, -2, "mount image failed")
		return
	}
	imagePath, ok := safeImagePath(req.File, []string{imageDirectory})
	if !ok {
		rollbackImageMount(storageLock)
		rsp.ErrRsp(c, -2, "image is unavailable after releasing the virtual disk")
		return
	}

	cdrom := "0"
	readOnly := "0"

	if req.Cdrom {
		cdrom = "1"
		readOnly = "1"
	}

	if req.ReadOnly {
		readOnly = "1"
	}

	// cdrom flag
	if err := os.WriteFile(cdromFlag, []byte(cdrom), 0o666); err != nil {
		log.Errorf("set cdrom flag failed: %s", err)
		rollbackImageMount(storageLock)
		rsp.ErrRsp(c, -4, "set cdrom flag failed")
		return
	}

	// ro flag
	if err := os.WriteFile(roFlag, []byte(readOnly), 0o666); err != nil {
		log.Errorf("set ro flag failed: %s", err)
		rollbackImageMount(storageLock)
		rsp.ErrRsp(c, -5, "set ro flag failed")
		return
	}

	// mount
	if err := os.WriteFile(mountDevice, []byte(imagePath), 0o666); err != nil {
		log.Errorf("mount file %s failed: %s", imagePath, err)
		rollbackImageMount(storageLock)
		rsp.ErrRsp(c, -6, "mount image failed")
		return
	}

	time.Sleep(3 * time.Second)
	rsp.OkRsp(c)
	log.Debugf("mount image %s success", imagePath)
}

func (s *Service) GetMountedImage(c *gin.Context) {
	var rsp proto.Response
	var data proto.GetMountedImageRsp

	file, mounted, err := readMountedImageFile(mountDevice)
	if err != nil {
		rsp.ErrRsp(c, -2, "read failed")
		return
	}
	data.File = file
	data.Mounted = mounted

	if content, err := os.ReadFile(cdromFlag); err == nil {
		data.Cdrom = strings.ReplaceAll(string(content), "\n", "") == "1"
	}

	if content, err := os.ReadFile(roFlag); err == nil {
		data.ReadOnly = strings.ReplaceAll(string(content), "\n", "") == "1"
	}

	rsp.OkRspWithData(c, data)
}

func readMountedImageFile(path string) (string, bool, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	file := strings.TrimSpace(string(content))
	return file, file != "", nil
}

func (s *Service) DeleteImage(c *gin.Context) {
	var req proto.DeleteImageReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if deleteImageActionRequiresChallenge() {
		if err := verifyDeleteImageActionChallenge(c, req.Challenge, authservice.SensitiveActionDeleteImage); err != nil {
			log.Errorf("verify delete image challenge failed: %s", err)
			rsp.ErrRsp(c, -3, "sensitive action verification required")
			return
		}
	}

	release, err := acquireStorageAccess(true)
	if err != nil {
		rsp.ErrRsp(c, -3, err.Error())
		return
	}
	defer release()
	mediumOperationMu.Lock()
	defer mediumOperationMu.Unlock()
	imagePath, ok := resolveDeleteImagePath(req.File)
	if !ok {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	if err := guardMediaDelete(imagePath); err != nil {
		rsp.ErrRsp(c, -3, err.Error())
		return
	}
	if err := removeImageFile(imagePath); err != nil {
		rsp.ErrRsp(c, -3, "remove file failed")
		log.Errorf("failed to remove file %s: %s", imagePath, err)
		return
	}

	rsp.OkRsp(c)
	log.Debugf("delete image %s success", imagePath)
}

func safeImageDeletePath(name string) (string, bool) {
	return safeImagePath(name, []string{imageDirectory})
}

func safeImagePath(name string, allowedDirectories []string) (string, bool) {
	if name == "" {
		return "", false
	}

	cleanPath, err := filepath.Abs(filepath.Clean(name))
	if err != nil {
		return "", false
	}

	ext := strings.ToLower(filepath.Ext(cleanPath))
	if ext != ".iso" && ext != ".img" {
		return "", false
	}

	for _, directory := range allowedDirectories {
		basePath, err := filepath.Abs(filepath.Clean(directory))
		if err != nil || !isPathWithin(basePath, cleanPath) {
			continue
		}

		resolvedPath, err := filepath.EvalSymlinks(cleanPath)
		if err != nil {
			return "", false
		}

		resolvedBase, err := filepath.EvalSymlinks(basePath)
		if err != nil || !isPathWithin(resolvedBase, resolvedPath) {
			return "", false
		}

		info, err := os.Stat(resolvedPath)
		if err != nil || !info.Mode().IsRegular() {
			return "", false
		}

		return cleanPath, true
	}

	return "", false
}

func isPathWithin(basePath string, targetPath string) bool {
	rel, err := filepath.Rel(basePath, targetPath)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func validImageName(name, directory string) bool {
	cleanPath, err := filepath.Abs(filepath.Clean(name))
	if err != nil || !isPathWithin(directory, cleanPath) {
		return false
	}
	ext := strings.ToLower(filepath.Ext(cleanPath))
	return ext == ".iso" || ext == ".img"
}

// rollbackImageMount clears the disk0 marker and rebuilds the gadget after a
// failed image attach. Leaving /boot/usb.disk0 behind would make usbdev.sh
// skip disk1 and silently block the virtual disk from attaching.
func rollbackImageMount(storageLock *storageguard.Lock) {
	if err := restartUSB(0, storageLock); err != nil {
		log.Errorf("failed to roll back image mount: %s", err)
	}
}

func restartUSB(flag int, storageLock *storageguard.Lock) error {
	if flag == 0 {
		if err := os.Remove(usbDisk); err != nil {
			log.Errorf("remove %s failed: %s", usbDisk, err)
			return err
		}
	} else if flag == 1 {
		file, err := os.Create(usbDisk)
		if err != nil {
			log.Errorf("create file failed: %s", err)
			return err
		}
		defer file.Close()
	}

	cmd := usbRestartCommand()
	storageLock.Attach(cmd)
	if err := cmd.Run(); err != nil {
		log.Errorf("failed to run %v: %s", cmd.Args, err)
		return err
	}

	return nil
}

func usbRestartCommand() *exec.Cmd {
	return exec.Command("/bin/bash", "/opt/scripts/usbdev.sh", "restart")
}

// sanitizeFileName cleans up potentially dangerous characters in filenames
func sanitizeFileName(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == ' ' || r == '-' || r == '_' || r == '.':
			return r
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			return r
		default:
			return '_'
		}
	}, name)
}
