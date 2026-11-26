package vm

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"NanoKVM-Server/common"
	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/hid"
	"NanoKVM-Server/service/stream/opus"
	"NanoKVM-Server/storageguard"
	"NanoKVM-Server/usbgadget"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	virtualNetwork     = "/boot/usb.ncm"
	virtualNetworkFlag = "/sys/kernel/config/usb_gadget/g0/configs/c.1/ncm.usb0"

	virtualDisk         = "/boot/usb.disk1"
	virtualDiskFunction = "/sys/kernel/config/usb_gadget/g0/functions/mass_storage.disk1"
	virtualDiskFlag     = virtualDiskFunction + "/lun.0/file"
	mountedImageFlag    = "/sys/kernel/config/usb_gadget/g0/functions/mass_storage.disk0/lun.0/file"

	virtualMic         = "/boot/usb.uac2"
	virtualMicFlag     = "/sys/kernel/config/usb_gadget/g0/configs/c.1/uac2.usb0"
	virtualSpeaker     = "/boot/usb.uac2_speaker"
	virtualSpeakerFlag = "/sys/kernel/config/usb_gadget/g0/functions/uac2.usb0/c_chmask"

	scriptUsbDev = "/opt/scripts/usbdev.sh"

	scriptMountSdCard = "/kvmcomm/scripts/mount.py"
	sdCardFlag        = "/exfat.img"

	virtualDiskMinSizeMiB      int64 = 5 * 1024
	virtualDiskReservedSizeMiB int64 = 5 * 1024
	bytesPerMiB                      = 1024 * 1024

	virtualAudioModeOff    = "off"
	virtualAudioModeMic    = "mic"
	virtualAudioModeDuplex = "duplex"
)

var (
	virtualDeviceUpdateMu    sync.Mutex
	runVirtualDeviceCommands = executeWithHidLock
	resetVirtualAudioCapture = func() {
		common.GetKvmVision().ResetAudioCapture()
	}
	pauseVirtualAudioPlayback = func() func() {
		player := opus.GetAudioInputPlayer()
		player.PausePlayback()
		return player.ResumePlayback
	}
	applyVirtualAudioEdid = applyCurrentEdid
	virtualAudioHidOnly   = func() bool {
		return isFileExists(hid.HidOnlyFlag)
	}
	virtualMicMarkerExists = func() bool {
		return isFileExists(virtualMic)
	}
	virtualSpeakerMarkerExists = func() bool {
		return isFileExists(virtualSpeaker)
	}
	virtualMicRuntimeMounted     = isVirtualMicMounted
	virtualSpeakerRuntimeMounted = isVirtualSpeakerRuntimeMounted
	virtualDiskRuntimeMounted    = isVirtualDiskMounted
	virtualDiskImagePresent      = isSdCardImagePresent
	virtualDiskRuntimePresent    = func() bool { return isFileExists(virtualDiskFunction) }
	virtualDiskBackingState      = isVirtualDiskBackingActive
	virtualDiskMountedImageState = isMountedImageActive
	virtualDiskAvailableMiB      = getVirtualDiskAvailableMiB
	removeVirtualDiskMarker      = removeVirtualDiskMarkerFile
	runVirtualDiskImageManager   = executeVirtualDiskImageManager
)

type VirtualDevice interface {
	IsMounted() bool
	Mount() error
	Unmount() error
}

func (s *Service) GetVirtualDevice(c *gin.Context) {
	var rsp proto.Response

	virtualDeviceUpdateMu.Lock()
	defer virtualDeviceUpdateMu.Unlock()
	hid.LockUsbGadgetMutation()
	defer hid.UnlockUsbGadgetMutation()

	diskMinSizeMiB, diskMaxSizeMiB, err := getVirtualDiskSizeLimits()
	if err != nil {
		log.Warnf("failed to calculate virtual disk size limits: %s", err)
		diskMaxSizeMiB = 0
	}

	var usedEndpoints *int
	if used, err := usbgadget.INEndpoints(usbgadget.ConfigDir); err == nil {
		usedEndpoints = &used
	}
	rsp.OkRspWithData(c, &proto.GetVirtualDeviceRsp{
		IsUSBDisplayEnabled: isFileExists(usbgadget.ConfigDir + "/Loopback.0"),
		USBInEndpointsUsed:  usedEndpoints,
		USBInEndpointsTotal: usbgadget.INEndpointLimit,
		IsWebcamEnabled:     isFileExists("/boot/usb.webcam") && isFileExists("/run/nanokvm-webcam/live.sock"),
		IsNetworkEnabled:    isVirtualNetworkMounted(),
		IsDiskEnabled:       virtualDiskRuntimeMounted(),
		DiskMinSizeMiB:      diskMinSizeMiB,
		DiskMaxSizeMiB:      diskMaxSizeMiB,
		IsMicEnabled:        isVirtualMicMounted(),
		IsSpeakerEnabled:    isVirtualSpeakerMounted(),
		AudioMode:           getVirtualAudioMode(),
		IsSdCardExist:       virtualDiskImagePresent(),
	})

	log.Debug("get virtual device success")
}

func (s *Service) UpdateVirtualDevice(c *gin.Context) {
	var req proto.UpdateVirtualDeviceReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid argument")
		return
	}

	virtualDeviceUpdateMu.Lock()
	defer virtualDeviceUpdateMu.Unlock()
	hid.LockUsbGadgetMutation()
	defer hid.UnlockUsbGadgetMutation()

	if req.Device == "webcam" {
		if req.Mode != "on" && req.Mode != "off" {
			rsp.ErrRsp(c, -1, "Webcam mode must be on or off")
			return
		}
		if err := runVirtualDeviceCommands([]string{scriptUsbDev + " webcam-" + req.Mode}); err != nil {
			log.Warnf("webcam update: %v", err)
			rsp.ErrRsp(c, -3, err.Error())
			return
		}
		rsp.OkRsp(c)
		return
	}
	if isFileExists("/boot/usb.webcam") {
		var requestedLink string
		switch {
		case req.Device == "network" && !isVirtualNetworkMounted():
			requestedLink = "ncm.usb0"
		case req.Device == "audio" && req.Mode != "off" && !isVirtualMicMounted():
			requestedLink = "uac2.usb0"
		case req.Device == "disk" && !isVirtualDiskMounted():
			requestedLink = "mass_storage.disk1"
		}
		if requestedLink != "" && !webcamEndpointsAvailable(usbgadget.ConfigDir, requestedLink) {
			rsp.ErrRsp(c, -3, "USB endpoints are in use. Disable Webcam or another virtual device first.")
			return
		}
	}

	if req.Device == "audio" && virtualAudioHidOnly() {
		rsp.ErrRsp(c, -5, "virtual audio is unavailable in HID-only mode")
		return
	}

	commands, err := resolveDeviceCommands(req.Device, req.Mode, req.DiskSizeMiB)
	if err != nil {
		rsp.ErrRsp(c, -2, err.Error())
		return
	}

	micWasEnabled := false
	speakerWasEnabled := false
	speakerWillBeEnabled := false
	edidWasUpdated := false
	resumePlaybackAfterUpdate := false
	if req.Device == "audio" {
		micWasEnabled = virtualMicMarkerExists()
		speakerWasEnabled = virtualSpeakerMarkerExists()
		speakerWillBeEnabled = req.Mode == virtualAudioModeDuplex
		stateIsCurrent := isVirtualAudioStateCurrent(req.Mode)
		if speakerWasEnabled != speakerWillBeEnabled {
			if err := applyVirtualAudioEdid(!speakerWillBeEnabled); err != nil {
				log.Errorf("failed to update DP audio capability: %s", err)
				if rollbackErr := applyVirtualAudioEdid(!speakerWasEnabled); rollbackErr != nil {
					log.Errorf("failed to restore previous DP audio capability: %s", rollbackErr)
				}
				rsp.ErrRsp(c, -4, "failed to update DP audio capability")
				return
			}
			edidWasUpdated = true
		}
		if stateIsCurrent {
			rsp.OkRsp(c)
			log.Debugf("virtual device %s already uses mode %s", req.Device, req.Mode)
			return
		}

		resumePlayback := pauseVirtualAudioPlayback()
		resumePlaybackAfterUpdate = req.Mode != virtualAudioModeOff
		if resumePlayback != nil {
			defer func() {
				if resumePlaybackAfterUpdate {
					resumePlayback()
				}
			}()
		}
	}

	if err := runVirtualDeviceCommands(commands); err != nil {
		if req.Device == "audio" {
			// The rollback restores the previously captured audio mode. Resume
			// playback only when that known old mode still exposes UAC2.
			resumePlaybackAfterUpdate = micWasEnabled
			rollbackVirtualAudioState(micWasEnabled, speakerWasEnabled, edidWasUpdated)
		}
		if errors.Is(err, storageguard.ErrBusy) {
			rsp.ErrRsp(c, -7, err.Error())
			return
		}
		log.Errorf("failed to execute virtual device commands: %s", err)
		rsp.ErrRsp(c, -3, "failed to mount device")
		return
	}
	if req.Device == "audio" {
		if !isVirtualAudioStateCurrent(req.Mode) {
			log.Errorf("virtual audio mode %s failed postcondition validation", req.Mode)
			resumePlaybackAfterUpdate = micWasEnabled
			rollbackVirtualAudioState(micWasEnabled, speakerWasEnabled, edidWasUpdated)
			rsp.ErrRsp(c, -3, "failed to mount device")
			return
		}
		resetVirtualAudioCapture()
	}

	rsp.OkRsp(c)
	if req.Device == "audio" {
		log.Debugf("update virtual device %s mode %s success", req.Device, req.Mode)
	} else {
		log.Debugf("update virtual device %s success", req.Device)
	}
}

func webcamEndpointsAvailable(configDir, requestedLink string) bool {
	used, err := usbgadget.INEndpoints(configDir, requestedLink)
	return err == nil && used <= usbgadget.INEndpointLimit
}

func (s *Service) RefreshVirtualDevice(c *gin.Context) {
	var req proto.RefreshVirtualDeviceReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil || req.Device != "disk" {
		rsp.ErrRsp(c, -1, "invalid argument")
		return
	}

	virtualDeviceUpdateMu.Lock()
	defer virtualDeviceUpdateMu.Unlock()
	hid.LockUsbGadgetMutation()
	defer hid.UnlockUsbGadgetMutation()
	err := runVirtualDiskImageManager("restart")
	if err != nil {
		log.Errorf("failed to restart sd card image: %v", err)
		rsp.ErrRsp(c, -2, "failed to refresh")
		return
	}

	rsp.OkRsp(c)
	log.Debug("refresh virtual device success")
}

func (s *Service) DisableVirtualDisk(c *gin.Context) {
	var rsp proto.Response

	virtualDeviceUpdateMu.Lock()
	defer virtualDeviceUpdateMu.Unlock()
	hid.LockUsbGadgetMutation()
	defer hid.UnlockUsbGadgetMutation()

	diskActive, err := virtualDiskBackingState()
	if err != nil {
		log.Errorf("failed to inspect virtual disk backing file: %s", err)
		rsp.ErrRsp(c, -1, "failed to inspect virtual disk")
		return
	}
	if diskActive {
		rsp.ErrRsp(c, -2, "disconnect virtual disk first")
		return
	}

	imageMounted, err := virtualDiskMountedImageState()
	if err != nil {
		log.Errorf("failed to inspect mounted image backing file: %s", err)
		rsp.ErrRsp(c, -3, "failed to inspect mounted image")
		return
	}
	if imageMounted {
		rsp.ErrRsp(c, -4, "unmount image first")
		return
	}

	if err := removeVirtualDiskMarker(); err != nil {
		log.Errorf("failed to remove virtual disk marker: %s", err)
		rsp.ErrRsp(c, -5, "failed to disable virtual disk")
		return
	}

	if virtualDiskRuntimePresent() {
		if err := runVirtualDeviceCommands([]string{scriptUsbDev + " restart"}); err != nil {
			log.Errorf("failed to remove virtual disk USB function: %s", err)
			rsp.ErrRsp(c, -5, "failed to disable virtual disk")
			return
		}
	}

	if virtualDiskImagePresent() {
		if err := runVirtualDiskImageManager("remove"); err != nil {
			log.Errorf("failed to remove virtual disk image: %s", err)
			rsp.ErrRsp(c, -6, "virtual disk storage is busy")
			return
		}
		if virtualDiskImagePresent() {
			log.Error("virtual disk image still exists after removal")
			rsp.ErrRsp(c, -6, "failed to remove virtual disk storage")
			return
		}
	}

	rsp.OkRsp(c)
	log.Info("disabled virtual disk storage")
}

func resolveDeviceCommands(device, mode string, diskSizeMiB int64) ([]string, error) {
	switch device {
	case "network":
		return getNetworkCommands(), nil
	case "audio":
		return getAudioCommands(mode)
	case "disk":
		return getDiskCommands(diskSizeMiB)
	default:
		return nil, fmt.Errorf("invalid device: %s", device)
	}
}

func getNetworkCommands() []string {
	cmd := []string{
		scriptUsbDev + " stop",
	}

	if !isVirtualNetworkMounted() {
		cmd = append(cmd, "touch "+virtualNetwork)
	} else {
		cmd = append(cmd, "rm -rf "+virtualNetwork)
	}

	cmd = append(cmd, scriptUsbDev+" start")
	return cmd
}

func getAudioCommands(mode string) ([]string, error) {
	switch mode {
	case virtualAudioModeOff:
		return getAudioStateCommands(false, false), nil
	case virtualAudioModeMic:
		return getAudioStateCommands(true, false), nil
	case virtualAudioModeDuplex:
		return getAudioStateCommands(true, true), nil
	default:
		return nil, fmt.Errorf("invalid audio mode: %s", mode)
	}
}

func getAudioStateCommands(micEnabled, speakerEnabled bool) []string {
	commands := []string{scriptUsbDev + " stop"}
	if micEnabled {
		commands = append(commands, "touch "+virtualMic)
	} else {
		commands = append(commands, "rm -f "+virtualMic)
	}
	if speakerEnabled {
		commands = append(commands, "touch "+virtualSpeaker)
	} else {
		commands = append(commands, "rm -f "+virtualSpeaker)
	}
	return append(commands, scriptUsbDev+" start")
}

func rollbackVirtualAudioState(micEnabled, speakerEnabled, edidWasUpdated bool) {
	if err := runVirtualDeviceCommands(getAudioStateCommands(micEnabled, speakerEnabled)); err != nil {
		log.Errorf("failed to roll back virtual audio gadget: %s", err)
	} else if !isVirtualAudioState(micEnabled, speakerEnabled) {
		log.Error("virtual audio rollback failed postcondition validation")
	}
	resetVirtualAudioCapture()

	if edidWasUpdated {
		if err := applyVirtualAudioEdid(!speakerEnabled); err != nil {
			log.Errorf("failed to restore previous DP audio capability after virtual audio rollback: %s", err)
		}
	}
}

func getDiskCommands(sizeMiB int64) ([]string, error) {
	cmd := []string{
		"rm -f " + virtualDisk,
	}

	if !virtualDiskRuntimeMounted() {
		if err := ensureSdCardImageExists(sizeMiB); err != nil {
			return nil, err
		}
		cmd = append(cmd, "touch "+virtualDisk)
	}

	cmd = append(cmd, scriptUsbDev+" restart")
	return cmd, nil
}

func ensureSdCardImageExists(sizeMiB int64) error {
	if virtualDiskImagePresent() {
		return nil
	}

	if sizeMiB == 0 {
		return errors.New("virtual disk size is required")
	}
	if err := validateVirtualDiskSize(sizeMiB); err != nil {
		return err
	}
	if err := runVirtualDiskImageManager("start", "--size", strconv.FormatInt(sizeMiB, 10)); err != nil {
		log.Errorf("failed to create virtual disk image: %s", err)
		return err
	}

	return nil
}

func getVirtualDiskSizeLimits() (int64, int64, error) {
	availableMiB, err := virtualDiskAvailableMiB()
	if err != nil {
		return virtualDiskMinSizeMiB, 0, err
	}

	maxSizeMiB := availableMiB - virtualDiskReservedSizeMiB
	if maxSizeMiB < virtualDiskMinSizeMiB {
		return virtualDiskMinSizeMiB, maxSizeMiB, fmt.Errorf(
			"insufficient storage: %d MiB available, %d MiB reserved",
			availableMiB,
			virtualDiskReservedSizeMiB,
		)
	}

	return virtualDiskMinSizeMiB, maxSizeMiB, nil
}

func validateVirtualDiskSize(sizeMiB int64) error {
	minSizeMiB, maxSizeMiB, err := getVirtualDiskSizeLimits()
	if err != nil {
		return err
	}
	if sizeMiB < minSizeMiB || sizeMiB > maxSizeMiB {
		return fmt.Errorf(
			"virtual disk size must be between %d and %d MiB",
			minSizeMiB,
			maxSizeMiB,
		)
	}
	return nil
}

func getVirtualDiskAvailableMiB() (int64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs("/", &stat); err != nil {
		return 0, err
	}
	if stat.Bsize <= 0 {
		return 0, errors.New("invalid filesystem block size")
	}

	availableBytes := stat.Bavail * uint64(stat.Bsize)
	return int64(availableBytes / bytesPerMiB), nil
}

func executeVirtualDiskImageManager(args ...string) error {
	cmd := exec.Command(scriptMountSdCard, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"%s %s failed: %w (%s)",
			scriptMountSdCard,
			strings.Join(args, " "),
			err,
			strings.TrimSpace(string(output)),
		)
	}
	return nil
}

func removeVirtualDiskMarkerFile() error {
	err := os.Remove(virtualDisk)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func isVirtualDiskBackingActive() (bool, error) {
	return isBackingFileActive(virtualDiskFlag)
}

func isMountedImageActive() (bool, error) {
	return isBackingFileActive(mountedImageFlag)
}

func isBackingFileActive(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(content)) != "", nil
}

func executeWithHidLock(commands []string) error {
	if len(commands) == 0 {
		return errors.New("no command provided")
	}
	storageLock, err := storageguard.Exclusive()
	if err != nil {
		return err
	}
	defer storageLock.Close()

	h := hid.GetHid()
	if err := h.ResetInputRouting("virtual device rebuild"); err != nil {
		log.Warnf("failed to release routed input before virtual device rebuild: %s", err)
	}
	h.Lock()
	h.CloseNoLock()
	defer func() {
		h.OpenNoLock()
		h.Unlock()
	}()

	command := exec.Command("/bin/sh", "-c", strings.Join(commands, " && "))
	command.Dir = "/"
	storageLock.Attach(command)
	result, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("virtual device update failed: %w (%s)", err, strings.TrimSpace(string(result)))
	}

	return nil
}

func isVirtualDiskMounted() bool {
	if !isFileExists(virtualDiskFlag) {
		return false
	}

	content, err := os.ReadFile(virtualDiskFlag)
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(content)) == sdCardFlag
}

func isVirtualNetworkMounted() bool {
	return isFileExists(virtualNetworkFlag)
}

func isVirtualMicMounted() bool {
	return isFileExists(virtualMicFlag)
}

func getVirtualAudioMode() string {
	return resolveVirtualAudioMode(
		virtualMicMarkerExists(),
		virtualSpeakerMarkerExists(),
	)
}

func isVirtualAudioStateCurrent(mode string) bool {
	switch mode {
	case virtualAudioModeOff:
		return isVirtualAudioState(false, false)
	case virtualAudioModeMic:
		return isVirtualAudioState(true, false)
	case virtualAudioModeDuplex:
		return isVirtualAudioState(true, true)
	default:
		return false
	}
}

func isVirtualAudioState(micEnabled, speakerEnabled bool) bool {
	return virtualMicMarkerExists() == micEnabled &&
		virtualSpeakerMarkerExists() == speakerEnabled &&
		virtualMicRuntimeMounted() == micEnabled &&
		virtualSpeakerRuntimeMounted() == speakerEnabled
}

func resolveVirtualAudioMode(micEnabled, speakerEnabled bool) string {
	if !micEnabled {
		return virtualAudioModeOff
	}
	if speakerEnabled {
		return virtualAudioModeDuplex
	}
	return virtualAudioModeMic
}

func isVirtualSpeakerMounted() bool {
	return isFileExists(virtualSpeaker) && isVirtualSpeakerRuntimeMounted()
}

func isVirtualSpeakerRuntimeMounted() bool {
	if !isVirtualMicMounted() {
		return false
	}

	content, err := os.ReadFile(virtualSpeakerFlag)
	return err == nil && strings.TrimSpace(string(content)) != "0"
}

func isSdCardImagePresent() bool {
	return isFileExists(sdCardFlag)
}

func isFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
