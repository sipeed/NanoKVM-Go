package vm

import (
	"errors"

	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Test seams mirror the menubar/input-profile pattern so handler tests stay
// host-safe and never touch the real configuration file.
var (
	getOnboardingCompletedVersion       = config.GetOnboardingCompletedVersion
	setOnboardingCompletedVersion       = config.SetOnboardingCompletedVersion
	getMobileOnboardingCompletedVersion = config.GetMobileOnboardingCompletedVersion
	setMobileOnboardingCompletedVersion = config.SetMobileOnboardingCompletedVersion
)

// GetOnboarding reports the device-level onboarding completion state persisted
// in the server configuration file for both the desktop and mobile guides.
func (s *Service) GetOnboarding(c *gin.Context) {
	var rsp proto.Response

	version := getOnboardingCompletedVersion()
	mobileVersion := getMobileOnboardingCompletedVersion()
	rsp.OkRspWithData(c, &proto.GetOnboardingRsp{
		CompletedVersion:       version,
		MobileCompletedVersion: mobileVersion,
	})
	log.Debugf("get onboarding completed versions: desktop=%q mobile=%q", version, mobileVersion)
}

// SetOnboarding marks an onboarding tour version as completed (or dismissed) on
// this device. The desktop and mobile fields are independent: either or both
// may be supplied, but every provided field is validated before any of them is
// persisted so an invalid request never writes a half update. The value charset
// is owned by the config package.
func (s *Service) SetOnboarding(c *gin.Context) {
	var req proto.SetOnboardingReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	completedProvided := req.CompletedVersion != ""
	mobileProvided := req.MobileCompletedVersion != ""

	if !completedProvided && !mobileProvided {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if completedProvided && !config.IsValidOnboardingVersion(req.CompletedVersion) {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if mobileProvided && !config.IsValidOnboardingVersion(req.MobileCompletedVersion) {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if completedProvided {
		if err := setOnboardingCompletedVersion(req.CompletedVersion); err != nil {
			if errors.Is(err, config.ErrInvalidOnboardingVersion) {
				rsp.ErrRsp(c, -1, "invalid arguments")
				return
			}

			log.Errorf("failed to save onboarding completed version %q: %v", req.CompletedVersion, err)
			rsp.ErrRsp(c, -2, "failed to save onboarding state")
			return
		}
	}

	if mobileProvided {
		if err := setMobileOnboardingCompletedVersion(req.MobileCompletedVersion); err != nil {
			if errors.Is(err, config.ErrInvalidOnboardingVersion) {
				rsp.ErrRsp(c, -1, "invalid arguments")
				return
			}

			log.Errorf("failed to save mobile onboarding completed version %q: %v", req.MobileCompletedVersion, err)
			rsp.ErrRsp(c, -2, "failed to save onboarding state")
			return
		}
	}

	rsp.OkRsp(c)
	log.Debugf("set onboarding completed versions: desktop=%q mobile=%q",
		req.CompletedVersion, req.MobileCompletedVersion)
}
