package vm

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/hid"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	usbVidFile          = "/boot/usb.vid"
	usbPidFile          = "/boot/usb.pid"
	usbBcdDeviceFile    = "/boot/usb.bcddevice"
	usbSerialnumberFile = "/boot/usb.serialnumber"
	usbManufacturerFile = "/boot/usb.manufacturer"
	usbProductFile      = "/boot/usb.product"

	usbIdentityStringMaxLen = 64
)

var (
	usbIdHexBody     = regexp.MustCompile(`(?i)^[0-9a-f]{1,4}$`)
	usbIdentityFiles = []string{
		usbVidFile,
		usbPidFile,
		usbBcdDeviceFile,
		usbSerialnumberFile,
		usbManufacturerFile,
		usbProductFile,
	}
)

func (s *Service) GetUsbId(c *gin.Context) {
	var rsp proto.Response

	data := &proto.GetUsbIdRsp{}

	vid, vidOK := readUsbIdFile(usbVidFile)
	pid, pidOK := readUsbIdFile(usbPidFile)
	bcdDevice, bcdOK := readUsbIdFile(usbBcdDeviceFile)
	serialnumber, serialOK := readUsbIdFile(usbSerialnumberFile)
	manufacturer, mfgOK := readUsbIdFile(usbManufacturerFile)
	product, productOK := readUsbIdFile(usbProductFile)

	if vidOK || pidOK || bcdOK || serialOK || mfgOK || productOK {
		data.Enabled = true
		data.Vid = vid
		data.Pid = pid
		data.BcdDevice = bcdDevice
		data.Serialnumber = serialnumber
		data.Manufacturer = manufacturer
		data.Product = product
	}

	rsp.OkRspWithData(c, data)
	log.Debugf("get usb id: %v", data)
}

func (s *Service) SetUsbId(c *gin.Context) {
	var req proto.SetUsbIdReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	hid.LockUsbGadgetMutation()
	defer hid.UnlockUsbGadgetMutation()

	if req.Enabled {
		vid, err := normalizeUsbIdHex(req.Vid)
		if err != nil {
			rsp.ErrRsp(c, -2, "invalid vid")
			return
		}
		pid, err := normalizeUsbIdHex(req.Pid)
		if err != nil {
			rsp.ErrRsp(c, -2, "invalid pid")
			return
		}

		var bcdDevice string
		if strings.TrimSpace(req.BcdDevice) != "" {
			bcdDevice, err = normalizeUsbIdHex(req.BcdDevice)
			if err != nil {
				rsp.ErrRsp(c, -2, "invalid bcdDevice")
				return
			}
		}

		serialnumber, err := normalizeUsbIdentityString(req.Serialnumber)
		if err != nil {
			rsp.ErrRsp(c, -2, "invalid serialnumber")
			return
		}
		manufacturer, err := normalizeUsbIdentityString(req.Manufacturer)
		if err != nil {
			rsp.ErrRsp(c, -2, "invalid manufacturer")
			return
		}
		product, err := normalizeUsbIdentityString(req.Product)
		if err != nil {
			rsp.ErrRsp(c, -2, "invalid product")
			return
		}

		writes := []struct {
			path  string
			value string
		}{
			{usbVidFile, vid},
			{usbPidFile, pid},
		}
		optionals := []struct {
			path  string
			value string
		}{
			{usbBcdDeviceFile, bcdDevice},
			{usbSerialnumberFile, serialnumber},
			{usbManufacturerFile, manufacturer},
			{usbProductFile, product},
		}

		var written []string
		rollback := func() {
			for _, path := range written {
				_ = os.Remove(path)
			}
		}

		for _, item := range writes {
			if err := os.WriteFile(item.path, []byte(item.value+"\n"), 0644); err != nil {
				log.Errorf("failed to write %s: %v", item.path, err)
				rollback()
				rsp.ErrRsp(c, -3, "write usb id failed")
				return
			}
			written = append(written, item.path)
		}

		for _, item := range optionals {
			if item.value == "" {
				if err := os.Remove(item.path); err != nil && !os.IsNotExist(err) {
					log.Errorf("failed to remove %s: %v", item.path, err)
					rollback()
					rsp.ErrRsp(c, -3, "write usb id failed")
					return
				}
				continue
			}
			if err := os.WriteFile(item.path, []byte(item.value+"\n"), 0644); err != nil {
				log.Errorf("failed to write %s: %v", item.path, err)
				rollback()
				rsp.ErrRsp(c, -3, "write usb id failed")
				return
			}
			written = append(written, item.path)
		}
	} else {
		for _, path := range usbIdentityFiles {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				log.Errorf("failed to remove %s: %v", path, err)
				rsp.ErrRsp(c, -4, "disable usb id failed")
				return
			}
		}
	}

	if err := restartUsbGadget(); err != nil {
		log.Errorf("failed to restart usb gadget after set usb id: %v", err)
		rsp.ErrRsp(c, -5, "restart usb failed")
		return
	}

	rsp.OkRsp(c)
	log.Debugf(
		"set usb id: enabled=%v vid=%q pid=%q bcdDevice=%q serialnumber=%q manufacturer=%q product=%q",
		req.Enabled, req.Vid, req.Pid, req.BcdDevice, req.Serialnumber, req.Manufacturer, req.Product,
	)
}

func readUsbIdFile(path string) (string, bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(content)), true
}

// normalizeUsbIdHex accepts forms like "0x359F", "359f", "0X1" and returns "0xXXXX".
func normalizeUsbIdHex(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("empty")
	}
	s = strings.TrimPrefix(strings.ToLower(s), "0x")
	if !usbIdHexBody.MatchString(s) {
		return "", fmt.Errorf("invalid hex")
	}
	n, err := strconv.ParseUint(s, 16, 16)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("0x%04X", n), nil
}

// normalizeUsbIdentityString trims and validates optional USB string descriptors.
// Empty input means "use script default" (caller removes the override file).
func normalizeUsbIdentityString(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", nil
	}
	if utf8.RuneCountInString(s) > usbIdentityStringMaxLen {
		return "", fmt.Errorf("too long")
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7F {
			return "", fmt.Errorf("invalid character")
		}
	}
	return s, nil
}

func restartUsbGadget() error {
	h := hid.GetHid()
	if err := h.ResetInputRouting("USB gadget rebuild"); err != nil {
		log.Warnf("failed to release routed input before USB gadget rebuild: %s", err)
	}
	h.Lock()
	h.CloseNoLock()
	defer func() {
		h.OpenNoLock()
		h.Unlock()
	}()

	if err := exec.Command("bash", scriptUsbDev, "restart").Run(); err != nil {
		return err
	}

	// Allow gadget rebind to settle before HID reopen path is heavily used.
	time.Sleep(time.Second)
	return nil
}
