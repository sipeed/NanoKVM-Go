package tailscale

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const sipeedManagementMaxRequestBytes = 8 << 10

type sipeedClaimCodeRsp struct {
	Code      string `json:"code"`
	ExpiresAt string `json:"expiresAt"`
	SingleUse bool   `json:"singleUse"`
}

type sipeedPendingRsp struct {
	Pending []sipeedPendingItemRsp `json:"pending"`
}

type sipeedPendingItemRsp struct {
	PendingID string   `json:"pendingId"`
	SAS       []string `json:"sas"`
	CreatedAt string   `json:"createdAt"`
	ClientIP  string   `json:"clientIp,omitempty"`
	ClientUA  string   `json:"clientUa,omitempty"`
}

type sipeedNodeRsp struct {
	NodeID string   `json:"nodeId"`
	Name   string   `json:"name"`
	IPs    []string `json:"ips"`
}

type sipeedDeviceRsp struct {
	NodeID   string   `json:"nodeId"`
	Name     string   `json:"name"`
	IPs      []string `json:"ips"`
	Online   bool     `json:"online"`
	LastSeen string   `json:"lastSeen,omitempty"`
	IsSelf   bool     `json:"isSelf"`
}

type sipeedDevicesRsp struct {
	NetworkID string            `json:"networkId,omitempty"`
	Devices   []sipeedDeviceRsp `json:"devices"`
}

type sipeedRenameRsp struct {
	NodeID string `json:"nodeId"`
	Name   string `json:"name"`
}

type sipeedDeleteRsp struct {
	NodeID  string `json:"nodeId"`
	Removed bool   `json:"removed"`
}

type sipeedLogoutRsp struct {
	CloudCleanupFailed bool `json:"cloudCleanupFailed"`
}

type sipeedChallengeRequest struct {
	Challenge string `json:"challenge,omitempty"`
}

type sipeedClaimCodeRequest struct {
	Challenge string `json:"challenge,omitempty"`
	Refresh   bool   `json:"refresh,omitempty"`
}

type sipeedApproveRequest struct {
	Challenge string `json:"challenge,omitempty"`
	Name      string `json:"name,omitempty"`
}

type sipeedAuthorizeBrowserRequest struct {
	Challenge string `json:"challenge,omitempty"`
	Code      string `json:"code"`
	Name      string `json:"name,omitempty"`
}

type sipeedRenameRequest struct {
	Challenge string `json:"challenge,omitempty"`
	Name      string `json:"name"`
}

func (s *Service) SipeedLogout(c *gin.Context) {
	s.sipeedLogout(c, false)
}

func (s *Service) SipeedRevokeDisclaimer(c *gin.Context) {
	s.sipeedLogout(c, true)
}

func (s *Service) sipeedLogout(c *gin.Context, revokeDisclaimer bool) {
	var rsp proto.Response
	if !s.sipeedModeAvailable() || s.sipeed == nil {
		rsp.ErrRsp(c, -1, "managed tailscale mode is unavailable")
		return
	}
	challenge, err := parseSipeedChallengeBody(c, true)
	if err != nil {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}
	if !requireTailscaleChallengeValue(c, &rsp, authservice.SensitiveActionTailscaleLogout, challenge) {
		return
	}
	release, ok := beginTailscaleMutation(c, &rsp, false)
	if !ok {
		return
	}
	defer release()

	// Revoke before cleanup so a failed teardown cannot silently enable again.
	// If saving fails, leave the connection intact so the user can retry here.
	if revokeDisclaimer {
		if err := s.setSipeedDisclaimerAccepted(false); err != nil {
			rsp.ErrRsp(c, -1, err.Error())
			return
		}
	}
	cloudCleanupFailed, err := s.sipeed.logout(c.Request.Context())
	if err != nil {
		log.Errorf("sipeed logout failed: %s", err)
		rsp.ErrRsp(c, -1, "sipeed logout failed")
		return
	}
	if revokeDisclaimer {
		if err := s.actionClient().Stop(); err != nil {
			log.Errorf("failed to stop tailscale after revoking Sipeed consent: %s", err)
			rsp.ErrRsp(c, -1, "stop failed")
			return
		}
	}
	rsp.OkRspWithData(c, &sipeedLogoutRsp{CloudCleanupFailed: cloudCleanupFailed})
}

func (s *Service) SipeedClaimCode(c *gin.Context) {
	var request sipeedClaimCodeRequest
	var rsp proto.Response
	if err := decodeSipeedJSONBody(c, &request, true); err != nil || !validSipeedChallenge(request.Challenge) {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}
	if !requireTailscaleChallengeValue(c, &rsp, authservice.SensitiveActionTailscaleLogin, request.Challenge) {
		return
	}
	// The mode/identity guard keeps its original precedence over the
	// rate-limit check; the issued code then comes from the manager's shared
	// cache. An explicit renewal replaces the code for every consumer.
	if _, _, ok := s.sipeedNetworkClient(c, &rsp); !ok {
		return
	}
	if !s.checkSipeedRateLimit(c, &rsp) {
		return
	}
	response, err := s.sipeed.getClaimCode(c.Request.Context(), request.Refresh)
	if err != nil {
		switch {
		case errors.Is(err, errSipeedManagedIdentityUnavailable):
			rsp.ErrRsp(c, -1, "managed tailscale identity is unavailable")
		case errors.Is(err, errSipeedManagedPairingUnavailable):
			rsp.ErrRsp(c, -1, "managed pairing service is unavailable")
		default:
			s.writeSipeedPairingError(c, &rsp, err)
		}
		return
	}
	rsp.OkRspWithData(c, &sipeedClaimCodeRsp{
		Code: response.Code, ExpiresAt: response.ExpiresAt, SingleUse: response.SingleUse,
	})
}

func (s *Service) SipeedPending(c *gin.Context) {
	var rsp proto.Response
	client, token, ok := s.sipeedNetworkClient(c, &rsp)
	if !ok {
		return
	}
	waitSeconds := 20
	if value := strings.TrimSpace(c.Query("wait")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 || parsed > 20 {
			rsp.ErrRsp(c, -1, "invalid wait")
			return
		}
		waitSeconds = parsed
	}
	if !s.checkSipeedRateLimit(c, &rsp) {
		return
	}
	invalidateClaim := s.sipeed.claimCodeInvalidator(token)
	response, err := client.Pending(c.Request.Context(), token, waitSeconds)
	if err != nil {
		s.writeSipeedPairingError(c, &rsp, err)
		return
	}
	if len(response.Pending) > 0 {
		invalidateClaim()
	}
	items := make([]sipeedPendingItemRsp, 0, len(response.Pending))
	for _, item := range response.Pending {
		items = append(items, sipeedPendingItemRsp{
			PendingID: item.PendingID,
			SAS:       append([]string(nil), item.SAS...),
			CreatedAt: item.CreatedAt,
			ClientIP:  item.ClientIP,
			ClientUA:  item.ClientUA,
		})
	}
	rsp.OkRspWithData(c, &sipeedPendingRsp{Pending: items})
}

func (s *Service) SipeedApprove(c *gin.Context) {
	var request sipeedApproveRequest
	var rsp proto.Response
	if err := decodeSipeedJSONBody(c, &request, true); err != nil ||
		!validSipeedChallenge(request.Challenge) || !validSipeedDisplayName(request.Name, true) {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}
	if !requireTailscaleChallengeValue(c, &rsp, authservice.SensitiveActionTailscaleLogin, request.Challenge) {
		return
	}
	client, token, ok := s.sipeedNetworkClient(c, &rsp)
	if !ok || !s.checkSipeedRateLimit(c, &rsp) {
		return
	}
	invalidateClaim := s.sipeed.claimCodeInvalidator(token)
	response, err := client.ApprovePending(c.Request.Context(), token, c.Param("id"), strings.TrimSpace(request.Name))
	if err != nil {
		s.writeSipeedPairingError(c, &rsp, err)
		return
	}
	invalidateClaim()
	rsp.OkRspWithData(c, sipeedNodeRsp{NodeID: response.NodeID, Name: response.Name, IPs: response.IPs})
}

func (s *Service) SipeedReject(c *gin.Context) {
	var request sipeedChallengeRequest
	var rsp proto.Response
	if err := decodeSipeedJSONBody(c, &request, true); err != nil || !validSipeedChallenge(request.Challenge) {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}
	if !requireTailscaleChallengeValue(c, &rsp, authservice.SensitiveActionTailscaleLogin, request.Challenge) {
		return
	}
	client, token, ok := s.sipeedNetworkClient(c, &rsp)
	if !ok || !s.checkSipeedRateLimit(c, &rsp) {
		return
	}
	invalidateClaim := s.sipeed.claimCodeInvalidator(token)
	response, err := client.RejectPending(c.Request.Context(), token, c.Param("id"))
	if err != nil {
		s.writeSipeedPairingError(c, &rsp, err)
		return
	}
	invalidateClaim()
	rsp.OkRspWithData(c, map[string]any{"pendingId": response.PendingID, "rejected": response.Rejected})
}

func (s *Service) SipeedAuthorize(c *gin.Context) {
	var request sipeedAuthorizeBrowserRequest
	var rsp proto.Response
	if err := decodeSipeedJSONBody(c, &request, false); err != nil ||
		!validSipeedChallenge(request.Challenge) || !validSipeedDisplayName(request.Name, true) ||
		!validSipeedBackupCode(normalizeSipeedBackupCode(request.Code)) {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}
	if !requireTailscaleChallengeValue(c, &rsp, authservice.SensitiveActionTailscaleLogin, request.Challenge) {
		return
	}
	client, token, ok := s.sipeedNetworkClient(c, &rsp)
	if !ok || !s.checkSipeedRateLimit(c, &rsp) {
		return
	}
	invalidateClaim := s.sipeed.claimCodeInvalidator(token)
	response, err := client.Authorize(c.Request.Context(), token, normalizeSipeedBackupCode(request.Code), "", strings.TrimSpace(request.Name))
	if err != nil {
		s.writeSipeedPairingError(c, &rsp, err)
		return
	}
	invalidateClaim()
	rsp.OkRspWithData(c, sipeedNodeRsp{NodeID: response.NodeID, Name: response.Name, IPs: response.IPs})
}

func (s *Service) SipeedDevices(c *gin.Context) {
	var rsp proto.Response
	client, token, ok := s.sipeedNetworkClient(c, &rsp)
	if !ok || !s.checkSipeedRateLimit(c, &rsp) {
		return
	}
	response, err := client.Devices(c.Request.Context(), token)
	if err != nil {
		s.writeSipeedPairingError(c, &rsp, err)
		return
	}
	devices := make([]sipeedDeviceRsp, 0, len(response.Devices))
	for _, device := range response.Devices {
		devices = append(devices, sipeedDeviceRsp{
			NodeID: device.NodeID, Name: device.Name, IPs: append([]string(nil), device.IPs...),
			Online: device.Online, LastSeen: device.LastSeen, IsSelf: device.IsSelf,
		})
	}
	rsp.OkRspWithData(c, &sipeedDevicesRsp{NetworkID: response.NetworkID, Devices: devices})
}

func (s *Service) SipeedRenameDevice(c *gin.Context) {
	var request sipeedRenameRequest
	var rsp proto.Response
	if err := decodeSipeedJSONBody(c, &request, false); err != nil ||
		!validSipeedChallenge(request.Challenge) || !validSipeedDisplayName(request.Name, false) || strings.TrimSpace(request.Name) == "" {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}
	if !requireTailscaleChallengeValue(c, &rsp, authservice.SensitiveActionTailscaleLogin, request.Challenge) {
		return
	}
	client, token, ok := s.sipeedNetworkClient(c, &rsp)
	if !ok || !s.checkSipeedRateLimit(c, &rsp) {
		return
	}
	response, err := client.RenameDevice(c.Request.Context(), token, c.Param("id"), strings.TrimSpace(request.Name))
	if err != nil {
		s.writeSipeedPairingError(c, &rsp, err)
		return
	}
	rsp.OkRspWithData(c, &sipeedRenameRsp{NodeID: response.NodeID, Name: response.Name})
}

func (s *Service) SipeedRemoveDevice(c *gin.Context) {
	var request sipeedChallengeRequest
	var rsp proto.Response
	if err := decodeSipeedJSONBody(c, &request, true); err != nil || !validSipeedChallenge(request.Challenge) {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}
	if !requireTailscaleChallengeValue(c, &rsp, authservice.SensitiveActionTailscaleLogin, request.Challenge) {
		return
	}
	if !validSipeedNodeID(c.Param("id")) {
		rsp.ErrRsp(c, -1, "invalid device id")
		return
	}
	client, token, ok := s.sipeedNetworkClient(c, &rsp)
	if !ok {
		return
	}
	_, selfNodeID, err := s.sipeed.pairingCredentials()
	if err != nil {
		rsp.ErrRsp(c, -1, "managed tailscale identity is unavailable")
		return
	}
	if selfNodeID != "" && c.Param("id") == selfNodeID {
		rsp.ErrRsp(c, -1, "the current device can only be removed by logout")
		return
	}
	if !s.checkSipeedRateLimit(c, &rsp) {
		return
	}
	if err := client.DeleteDevice(c.Request.Context(), token, c.Param("id")); err != nil {
		s.writeSipeedPairingError(c, &rsp, err)
		return
	}
	rsp.OkRspWithData(c, &sipeedDeleteRsp{NodeID: c.Param("id"), Removed: true})
}

func (s *Service) sipeedNetworkClient(c *gin.Context, rsp *proto.Response) (sipeedNetworkPairingClient, string, bool) {
	if !s.sipeedModeAvailable() || s.sipeed == nil {
		rsp.ErrRsp(c, -1, "managed tailscale mode is unavailable")
		return nil, "", false
	}
	client, ok := s.sipeed.deps.pairing.(sipeedNetworkPairingClient)
	if !ok || client == nil {
		rsp.ErrRsp(c, -1, "managed pairing service is unavailable")
		return nil, "", false
	}
	token, _, err := s.sipeed.pairingCredentials()
	if err != nil {
		rsp.ErrRsp(c, -1, "managed tailscale identity is unavailable")
		return nil, "", false
	}
	return client, token, true
}

func (s *Service) checkSipeedRateLimit(c *gin.Context, rsp *proto.Response) bool {
	remaining := s.sipeed.pairingRetryRemaining()
	if remaining <= 0 {
		return true
	}
	s.writeSipeedPairingError(c, rsp, &sipeedPairingAPIError{
		HTTPStatus: http.StatusTooManyRequests,
		Code:       "rate_limited",
		RetryAfter: remaining,
	})
	return false
}

func (s *Service) writeSipeedPairingError(c *gin.Context, rsp *proto.Response, err error) {
	var apiError *sipeedPairingAPIError
	if errors.As(err, &apiError) {
		switch apiError.HTTPStatus {
		case http.StatusUnauthorized:
			if s.sipeed != nil {
				if invalidateErr := s.sipeed.invalidateIdentity(c.Request.Context()); invalidateErr != nil {
					log.Warnf("clear invalid Sipeed identity failed: %s", invalidateErr)
				}
			}
			rsp.ErrRspWithData(c, -2, "managed tailscale identity expired", s.managedStatusSnapshot())
			return
		case http.StatusTooManyRequests:
			if s.sipeed != nil {
				if saveErr := s.sipeed.setPairingRetryAfter(apiError.RetryAfter); saveErr != nil {
					log.Warnf("save Sipeed retry deadline failed: %s", saveErr)
				}
			}
			rsp.ErrRspWithData(c, -2, "managed tailscale operation is rate limited", s.managedStatusSnapshot())
			return
		}
	}
	rsp.ErrRsp(c, -1, "managed tailscale operation failed")
}

func decodeSipeedJSONBody(c *gin.Context, output any, allowEmpty bool) error {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		if allowEmpty {
			return nil
		}
		return errors.New("request body is required")
	}
	body, err := ioReadLimited(c, sipeedManagementMaxRequestBytes)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		if allowEmpty {
			return nil
		}
		return errors.New("request body is required")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return errors.New("invalid request body")
	}
	return ensureSipeedJSONEOF(decoder)
}

func parseSipeedChallengeBody(c *gin.Context, allowEmpty bool) (string, error) {
	var request sipeedChallengeRequest
	if err := decodeSipeedJSONBody(c, &request, allowEmpty); err != nil {
		return "", err
	}
	if !validSipeedChallenge(request.Challenge) {
		return "", errors.New("invalid challenge")
	}
	return strings.TrimSpace(request.Challenge), nil
}

func validSipeedChallenge(challenge string) bool {
	return len(strings.TrimSpace(challenge)) <= 256
}

func ioReadLimited(c *gin.Context, max int64) ([]byte, error) {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return nil, errors.New("request body is missing")
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, max+1))
	if err != nil || int64(len(body)) > max {
		return nil, errors.New("request body is too large")
	}
	return body, nil
}
