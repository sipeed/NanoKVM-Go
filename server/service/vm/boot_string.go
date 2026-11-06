package vm

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/hid"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	diskInquiryFile      = "/boot/disk.inquiry_string"
	diskInquiryCdromFile = "/boot/disk.inquiry_string_cdrom"
	uac2FunctionNameFile = "/boot/uac2.function_name"

	diskInquiryMaxLen      = 28
	uac2FunctionNameMaxLen = 64
)

type bootStringNormalizer func(string) (string, error)

func (s *Service) GetDiskInquiry(c *gin.Context) {
	getBootString(c, diskInquiryFile)
}

func (s *Service) SetDiskInquiry(c *gin.Context) {
	setBootString(c, diskInquiryFile, "disk inquiry", normalizeDiskInquiryString)
}

func (s *Service) GetDiskInquiryCdrom(c *gin.Context) {
	getBootString(c, diskInquiryCdromFile)
}

func (s *Service) SetDiskInquiryCdrom(c *gin.Context) {
	setBootString(c, diskInquiryCdromFile, "disk inquiry cdrom", normalizeDiskInquiryString)
}

func (s *Service) GetUac2FunctionName(c *gin.Context) {
	getBootString(c, uac2FunctionNameFile)
}

func (s *Service) SetUac2FunctionName(c *gin.Context) {
	setBootString(c, uac2FunctionNameFile, "uac2 function name", normalizeUac2FunctionName)
}

func getBootString(c *gin.Context, path string) {
	var rsp proto.Response

	data := &proto.GetBootStringRsp{}
	value, ok := readUsbIdFile(path)
	if ok {
		data.Enabled = true
		data.Value = value
	}

	rsp.OkRspWithData(c, data)
	log.Debugf("get boot string %s: enabled=%v value=%q", path, data.Enabled, data.Value)
}

func setBootString(c *gin.Context, path, label string, normalize bootStringNormalizer) {
	var req proto.SetBootStringReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	hid.LockUsbGadgetMutation()
	defer hid.UnlockUsbGadgetMutation()

	if req.Enabled {
		value, err := normalize(req.Value)
		if err != nil {
			rsp.ErrRsp(c, -2, "invalid value")
			return
		}
		if err := os.WriteFile(path, []byte(value+"\n"), 0644); err != nil {
			log.Errorf("failed to write %s: %v", path, err)
			rsp.ErrRsp(c, -3, "write "+label+" failed")
			return
		}
	} else {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Errorf("failed to remove %s: %v", path, err)
			rsp.ErrRsp(c, -4, "disable "+label+" failed")
			return
		}
	}

	if err := restartUsbGadget(); err != nil {
		log.Errorf("failed to restart usb gadget after set %s: %v", label, err)
		rsp.ErrRsp(c, -5, "restart usb failed")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("set boot string %s: enabled=%v value=%q", path, req.Enabled, req.Value)
}

// normalizeDiskInquiryString validates the 28-byte SCSI INQUIRY identity
// (8-byte vendor, 16-byte product, and 4-byte revision fields).
func normalizeDiskInquiryString(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("empty")
	}
	if len(s) > diskInquiryMaxLen {
		return "", fmt.Errorf("too long")
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7E {
			return "", fmt.Errorf("invalid character")
		}
	}
	return s, nil
}

func normalizeUac2FunctionName(raw string) (string, error) {
	return normalizeBootString(raw, uac2FunctionNameMaxLen)
}

// normalizeBootString trims and validates a non-empty boot override string.
func normalizeBootString(raw string, maxLen int) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("empty")
	}
	if utf8.RuneCountInString(s) > maxLen {
		return "", fmt.Errorf("too long")
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7F {
			return "", fmt.Errorf("invalid character")
		}
	}
	return s, nil
}
