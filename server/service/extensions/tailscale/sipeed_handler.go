package tailscale

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
)

const sipeedEnableMaxRequestBytes = 4 << 10

// errManagedModeUnavailable reports that the managed mode is not offered on
// this device (feature manifest) or the manager is not wired.
var errManagedModeUnavailable = errors.New("managed tailscale mode is unavailable")

// enableManaged is the transport-independent core of EnableSipeed: it takes
// the global mutation lock, persists the enrolling state, and launches the
// async enable job. It returns errManagedModeUnavailable, errTailscaleMutationBusy,
// or the prepareEnable failure; on success the background job owns the
// mutation release. The web handler and root-local gate share this job.
func (s *Service) enableManaged(acceptDisclaimer bool) error {
	if !s.sipeedModeAvailable() || s.sipeed == nil {
		return errManagedModeUnavailable
	}
	release, ok := tailscaleMutations.tryBegin(false)
	if !ok {
		return errTailscaleMutationBusy
	}
	// Persist explicit consent before preparing the job or contacting Sipeed.
	// Other entry points can reuse saved consent but cannot grant it implicitly.
	if !s.sipeedDisclaimerAccepted() {
		if !acceptDisclaimer {
			release()
			return errSipeedDisclaimerRequired
		}
		if err := s.setSipeedDisclaimerAccepted(true); err != nil {
			release()
			return err
		}
	}
	if err := s.sipeed.prepareEnable(); err != nil {
		release()
		return err
	}
	go s.sipeed.runEnable(release)
	return nil
}

func (s *Service) EnableSipeed(c *gin.Context) {
	var rsp proto.Response
	if !s.sipeedModeAvailable() || s.sipeed == nil {
		rsp.ErrRsp(c, -1, "managed tailscale mode is unavailable")
		return
	}

	request, err := parseSipeedEnableRequest(c)
	if err != nil {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}
	if !requireTailscaleChallengeValue(
		c,
		&rsp,
		authservice.SensitiveActionTailscaleLogin,
		request.Challenge,
	) {
		return
	}

	if err := s.enableManaged(request.AcceptDisclaimer); err != nil {
		if errors.Is(err, errTailscaleMutationBusy) {
			data, _ := s.getStatus(c.Request.Context())
			rsp.ErrRspWithData(c, -1, "tailscale operation already in progress", data)
			return
		}
		data := s.statusAfterEnablePreparationFailure(c.Request.Context())
		if errors.Is(err, errSipeedDisclaimerRequired) || errors.Is(err, errSipeedDisclaimerSaveFailed) {
			rsp.ErrRspWithData(c, -1, err.Error(), data)
			return
		}
		rsp.ErrRspWithData(c, -1, sipeedEnableFailureMessage(data), data)
		return
	}

	data, err := s.getStatus(c.Request.Context())
	if err != nil {
		data = s.managedStatusSnapshot()
	}
	rsp.OkRspWithData(c, data)
}

// statusAfterEnablePreparationFailure keeps the persisted managed error and
// retry deadline in the response even when the local daemon is unavailable.
// The web client uses these fields to distinguish rate limiting, invalid
// identity, service unavailability, and local state failures.
func (s *Service) statusAfterEnablePreparationFailure(ctx context.Context) *proto.GetTailscaleStatusRsp {
	if data, err := s.getStatus(ctx); err == nil && data != nil {
		return data
	}
	return s.managedStatusSnapshot()
}

func (s *Service) managedStatusSnapshot() *proto.GetTailscaleStatusRsp {
	data := &proto.GetTailscaleStatusRsp{
		AvailableModes:           s.availableModes(),
		ManagedState:             proto.TailscaleManagedDisabled,
		SipeedDisclaimerAccepted: s.sipeedDisclaimerAccepted(),
	}
	// The read path must never append managed details while the manifest closes
	// the mode, even if the manager still holds a record.
	if s.sipeedEnabled() && s.sipeed != nil {
		s.sipeed.appendStatus(data)
	}
	s.setManagedActiveMode(data)
	return data
}

func sipeedEnableFailureMessage(data *proto.GetTailscaleStatusRsp) string {
	if data != nil {
		switch data.ManagedError {
		case "rateLimited":
			return "managed tailscale enable is rate limited"
		case "invalidToken":
			return "managed tailscale identity is invalid"
		case "pairingUnavailable", "pairingUpstreamError", "pairingRejected", "nodeNotFound", "protocolError":
			return "managed tailscale pairing failed"
		case "stateFileInvalid", "stateWriteFailed":
			return "managed tailscale state is unavailable"
		}
	}
	return "managed tailscale enable is unavailable"
}

type sipeedEnableRequest struct {
	Challenge        string `json:"challenge,omitempty"`
	AcceptDisclaimer bool   `json:"acceptDisclaimer,omitempty"`
}

func parseSipeedEnableRequest(c *gin.Context) (sipeedEnableRequest, error) {
	var request sipeedEnableRequest
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return request, nil
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, sipeedEnableMaxRequestBytes+1))
	if err != nil || len(body) > sipeedEnableMaxRequestBytes {
		return request, errors.New("invalid request body")
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return request, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, errors.New("invalid request body")
	}
	if err := ensureSipeedJSONEOF(decoder); err != nil {
		return request, err
	}
	request.Challenge = strings.TrimSpace(request.Challenge)
	if len(request.Challenge) > 256 {
		return request, errors.New("challenge is too long")
	}
	return request, nil
}
