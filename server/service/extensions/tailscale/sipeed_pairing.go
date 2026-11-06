package tailscale

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	sipeedControlURL             = "https://control.tailnet.sipeed.com"
	sipeedEnrollPath             = "/v1/network/enroll"
	sipeedNetworkPath            = "/v1/network"
	sipeedJoinKeyPath            = "/v1/devices/self/join-key"
	sipeedSelfNodePath           = "/v1/devices/self/node"
	sipeedClaimCodePath          = "/v1/network/claim-code"
	sipeedPendingPath            = "/v1/network/pending"
	sipeedAuthorizePath          = "/v1/devices/authorize"
	sipeedDevicesPath            = "/v1/devices"
	sipeedDevicePathPrefix       = "/v1/devices/"
	sipeedPairingMaxResponseSize = 64 << 10
	// Pending is a long poll. Keep the transport budget above the upstream
	// 20-second wait so a response has time to reach the caller.
	sipeedPairingTimeout = 40 * time.Second
)

type sipeedEnrollRequest struct {
	DeviceID string `json:"device_id"`
	Hostname string `json:"hostname"`
}

type sipeedEnrollResponse struct {
	NetworkID string `json:"network_id"`
	DeviceID  string `json:"device_id"`
	AuthKey   string `json:"auth_key"`
	ExpiresAt string `json:"expires_at"`
}

// sipeedJoinKeyResponse deliberately contains only the short-lived key
// returned by the already-enrolled device endpoint. The key is consumed by
// LocalAPI immediately and is never part of persisted/public state.
type sipeedJoinKeyResponse struct {
	AuthKey   string `json:"auth_key"`
	ExpiresAt string `json:"expires_at"`
}

type sipeedSelfNodeRequest struct {
	IP string `json:"ip"`
}

type sipeedSelfNodeResponse struct {
	NodeID string `json:"node_id"`
	IP     string `json:"ip"`
}

type sipeedDeleteDeviceResponse struct {
	NodeID  string `json:"node_id"`
	Removed bool   `json:"removed"`
}

type sipeedDeleteNetworkResponse struct {
	NetworkID      string `json:"network_id"`
	Destroyed      bool   `json:"destroyed"`
	DevicesRemoved *int   `json:"devices_removed"`
}

type sipeedClaimCodeResponse struct {
	Code      string `json:"code"`
	ExpiresAt string `json:"expires_at"`
	SingleUse bool   `json:"single_use"`
}

type sipeedPendingRequest struct {
	PendingID string   `json:"pending_id"`
	SAS       []string `json:"sas"`
	CreatedAt string   `json:"created_at"`
	ClientIP  string   `json:"client_ip"`
	ClientUA  string   `json:"client_ua"`
}

type sipeedPendingResponse struct {
	Pending []sipeedPendingRequest `json:"pending"`
}

type sipeedApproveResponse struct {
	NodeID string   `json:"node_id"`
	Name   string   `json:"name"`
	IPs    []string `json:"ips"`
}

type sipeedRejectResponse struct {
	PendingID string `json:"pending_id"`
	Rejected  bool   `json:"rejected"`
}

type sipeedAuthorizeRequest struct {
	Code   string `json:"code,omitempty"`
	AuthID string `json:"auth_id,omitempty"`
	Name   string `json:"name,omitempty"`
}

type sipeedAuthorizeResponse struct {
	NodeID string   `json:"node_id"`
	Name   string   `json:"name"`
	IPs    []string `json:"ips"`
}

type sipeedDevice struct {
	NodeID   string   `json:"node_id"`
	Name     string   `json:"name"`
	IPs      []string `json:"ips"`
	Online   bool     `json:"online"`
	LastSeen string   `json:"last_seen"`
	IsSelf   bool     `json:"is_self"`
}

type sipeedDevicesResponse struct {
	NetworkID string         `json:"network_id"`
	Devices   []sipeedDevice `json:"devices"`
}

type sipeedRenameDeviceRequest struct {
	Name string `json:"name"`
}

type sipeedRenameDeviceResponse struct {
	NodeID string `json:"node_id"`
	Name   string `json:"name"`
}

type sipeedPairingAPIError struct {
	HTTPStatus int
	Code       string
	RetryAfter time.Duration
}

func (err *sipeedPairingAPIError) Error() string {
	if err == nil {
		return "pairing service request failed"
	}
	return fmt.Sprintf("pairing service request failed with HTTP %d (%s)", err.HTTPStatus, err.Code)
}

type sipeedPairingClient interface {
	Enroll(ctx context.Context, deviceID, deviceToken, hostname string) (*sipeedEnrollResponse, error)
	JoinKey(ctx context.Context, deviceToken string) (*sipeedJoinKeyResponse, error)
	ReportSelfNode(ctx context.Context, deviceToken, ip string) (string, error)
	DeleteNetwork(ctx context.Context, deviceToken, networkID string) error
	DeleteDevice(ctx context.Context, deviceToken, nodeID string) error
}

// sipeedNetworkPairingClient is kept separate from the enrollment interface so
// existing enable tests can use a small fake while the management handlers use
// the complete device-management contract.
type sipeedNetworkPairingClient interface {
	ClaimCode(ctx context.Context, deviceToken string) (*sipeedClaimCodeResponse, error)
	Pending(ctx context.Context, deviceToken string, waitSeconds int) (*sipeedPendingResponse, error)
	ApprovePending(ctx context.Context, deviceToken, pendingID, name string) (*sipeedApproveResponse, error)
	RejectPending(ctx context.Context, deviceToken, pendingID string) (*sipeedRejectResponse, error)
	Authorize(ctx context.Context, deviceToken, code, authID, name string) (*sipeedAuthorizeResponse, error)
	Devices(ctx context.Context, deviceToken string) (*sipeedDevicesResponse, error)
	RenameDevice(ctx context.Context, deviceToken, nodeID, name string) (*sipeedRenameDeviceResponse, error)
	DeleteNetwork(ctx context.Context, deviceToken, networkID string) error
	DeleteDevice(ctx context.Context, deviceToken, nodeID string) error
}

type httpSipeedPairingClient struct {
	baseURL    string
	httpClient *http.Client
}

// The production pairing client must always satisfy the full device-management
// contract: the LCD gate driver asserts it at construction to fail closed
// instead of discovering a missing method at pairing time.
var _ sipeedNetworkPairingClient = (*httpSipeedPairingClient)(nil)

func newHTTPSipeedPairingClient(baseURL string, httpClient *http.Client) *httpSipeedPairingClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = sipeedControlURL
	}
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: sipeedPairingTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	return &httpSipeedPairingClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (client *httpSipeedPairingClient) Enroll(
	ctx context.Context,
	deviceID string,
	deviceToken string,
	hostname string,
) (*sipeedEnrollResponse, error) {
	request := sipeedEnrollRequest{DeviceID: deviceID, Hostname: hostname}
	var response sipeedEnrollResponse
	if err := client.post(ctx, sipeedEnrollPath, deviceToken, request, &response); err != nil {
		return nil, err
	}
	if response.DeviceID != deviceID || !validSipeedStateString(response.NetworkID, 256, false) ||
		strings.TrimSpace(response.AuthKey) == "" || len(response.AuthKey) > 4096 {
		return nil, errors.New("pairing service returned an invalid enrollment response")
	}
	if _, err := time.Parse(time.RFC3339, response.ExpiresAt); err != nil {
		return nil, errors.New("pairing service returned an invalid enrollment expiry")
	}
	return &response, nil
}

// JoinKey renews a short-lived auth key for an identity that already has a
// network. It intentionally sends no JSON body: unlike enroll, this endpoint
// must not create a new network or consume the enroll rate limit.
func (client *httpSipeedPairingClient) JoinKey(
	ctx context.Context,
	deviceToken string,
) (*sipeedJoinKeyResponse, error) {
	var response sipeedJoinKeyResponse
	if err := client.post(ctx, sipeedJoinKeyPath, deviceToken, nil, &response); err != nil {
		return nil, err
	}
	if strings.TrimSpace(response.AuthKey) == "" || len(response.AuthKey) > 4096 {
		return nil, errors.New("pairing service returned an invalid join key response")
	}
	if _, err := time.Parse(time.RFC3339, response.ExpiresAt); err != nil {
		return nil, errors.New("pairing service returned an invalid join key expiry")
	}
	return &response, nil
}

func (client *httpSipeedPairingClient) ReportSelfNode(
	ctx context.Context,
	deviceToken string,
	ip string,
) (string, error) {
	parsedIP := net.ParseIP(strings.TrimSpace(ip))
	if parsedIP == nil || parsedIP.To4() == nil {
		return "", errors.New("invalid managed IPv4 address")
	}
	canonicalIP := parsedIP.String()
	request := sipeedSelfNodeRequest{IP: canonicalIP}
	var response sipeedSelfNodeResponse
	if err := client.post(ctx, sipeedSelfNodePath, deviceToken, request, &response); err != nil {
		return "", err
	}
	if !validSipeedNodeID(response.NodeID) || response.IP != canonicalIP {
		return "", errors.New("pairing service returned an invalid self-node response")
	}
	return response.NodeID, nil
}

func (client *httpSipeedPairingClient) DeleteDevice(
	ctx context.Context,
	deviceToken string,
	nodeID string,
) error {
	if !validSipeedNodeID(nodeID) {
		return errors.New("invalid managed node id")
	}
	var response sipeedDeleteDeviceResponse
	path := sipeedDevicePathPrefix + url.PathEscape(nodeID)
	if err := client.do(ctx, http.MethodDelete, path, deviceToken, nil, &response); err != nil {
		return err
	}
	if response.NodeID != nodeID || !response.Removed {
		return errors.New("pairing service returned an invalid delete response")
	}
	return nil
}

// DeleteNetwork destroys the entire managed network. A repeated request after
// a successful deletion receives 401 invalid_token because the pairing record
// no longer exists; that response is therefore the same terminal success from
// the device's perspective.
func (client *httpSipeedPairingClient) DeleteNetwork(
	ctx context.Context,
	deviceToken string,
	networkID string,
) error {
	if !validSipeedStateString(networkID, 256, false) {
		return errors.New("invalid managed network id")
	}

	var response sipeedDeleteNetworkResponse
	status, err := client.doWithStatus(
		ctx,
		http.MethodDelete,
		sipeedNetworkPath,
		deviceToken,
		nil,
		&response,
	)
	if err != nil {
		var apiError *sipeedPairingAPIError
		if errors.As(err, &apiError) && apiError.HTTPStatus == http.StatusUnauthorized &&
			apiError.Code == "invalid_token" {
			return nil
		}
		return err
	}
	if status != http.StatusOK {
		return errors.New("pairing service returned an invalid delete-network status")
	}
	if response.NetworkID != networkID || !response.Destroyed ||
		response.DevicesRemoved == nil || *response.DevicesRemoved < 0 {
		return errors.New("pairing service returned an invalid delete-network response")
	}
	return nil
}

func (client *httpSipeedPairingClient) ClaimCode(
	ctx context.Context,
	deviceToken string,
) (*sipeedClaimCodeResponse, error) {
	var response sipeedClaimCodeResponse
	if err := client.post(ctx, sipeedClaimCodePath, deviceToken, nil, &response); err != nil {
		return nil, err
	}
	digits, err := normalizeSipeedClaimCode(response.Code)
	if err != nil || response.SingleUse == false {
		return nil, errors.New("pairing service returned an invalid claim code response")
	}
	if _, err := time.Parse(time.RFC3339, response.ExpiresAt); err != nil {
		return nil, errors.New("pairing service returned an invalid claim code expiry")
	}
	response.Code = formatSipeedClaimCode(digits)
	return &response, nil
}

func (client *httpSipeedPairingClient) Pending(
	ctx context.Context,
	deviceToken string,
	waitSeconds int,
) (*sipeedPendingResponse, error) {
	if waitSeconds < 0 || waitSeconds > 20 {
		return nil, errors.New("invalid pending wait")
	}
	path := sipeedPendingPath
	if waitSeconds > 0 {
		path += "?wait=" + strconv.Itoa(waitSeconds)
	}
	var response sipeedPendingResponse
	if err := client.do(ctx, http.MethodGet, path, deviceToken, nil, &response); err != nil {
		return nil, err
	}
	for index := range response.Pending {
		if err := validateSipeedPending(&response.Pending[index]); err != nil {
			return nil, err
		}
	}
	return &response, nil
}

func (client *httpSipeedPairingClient) ApprovePending(
	ctx context.Context,
	deviceToken string,
	pendingID string,
	name string,
) (*sipeedApproveResponse, error) {
	if !validSipeedPendingID(pendingID) || !validSipeedDisplayName(name, true) {
		return nil, errors.New("invalid pending approval parameters")
	}
	var input any
	if strings.TrimSpace(name) != "" {
		input = map[string]string{"name": strings.TrimSpace(name)}
	}
	var response sipeedApproveResponse
	path := sipeedPendingPath + "/" + url.PathEscape(pendingID) + "/approve"
	if err := client.post(ctx, path, deviceToken, input, &response); err != nil {
		return nil, err
	}
	if err := validateSipeedNodeResponse(response.NodeID, response.IPs); err != nil {
		return nil, err
	}
	if !validSipeedDisplayName(response.Name, true) {
		return nil, errors.New("pairing service returned an invalid approval response")
	}
	return &response, nil
}

func (client *httpSipeedPairingClient) RejectPending(
	ctx context.Context,
	deviceToken string,
	pendingID string,
) (*sipeedRejectResponse, error) {
	if !validSipeedPendingID(pendingID) {
		return nil, errors.New("invalid pending request id")
	}
	var response sipeedRejectResponse
	path := sipeedPendingPath + "/" + url.PathEscape(pendingID) + "/reject"
	if err := client.post(ctx, path, deviceToken, nil, &response); err != nil {
		return nil, err
	}
	if response.PendingID != pendingID || !response.Rejected {
		return nil, errors.New("pairing service returned an invalid rejection response")
	}
	return &response, nil
}

func (client *httpSipeedPairingClient) Authorize(
	ctx context.Context,
	deviceToken string,
	code string,
	authID string,
	name string,
) (*sipeedAuthorizeResponse, error) {
	code = normalizeSipeedBackupCode(code)
	authID = strings.TrimSpace(authID)
	if (code == "" && authID == "") || (code != "" && !validSipeedBackupCode(code)) ||
		(authID != "" && !validSipeedAuthID(authID)) || !validSipeedDisplayName(name, true) {
		return nil, errors.New("invalid authorization parameters")
	}
	input := sipeedAuthorizeRequest{Code: code, AuthID: authID, Name: strings.TrimSpace(name)}
	var response sipeedAuthorizeResponse
	if err := client.post(ctx, sipeedAuthorizePath, deviceToken, input, &response); err != nil {
		return nil, err
	}
	if err := validateSipeedNodeResponse(response.NodeID, response.IPs); err != nil {
		return nil, err
	}
	if !validSipeedDisplayName(response.Name, true) {
		return nil, errors.New("pairing service returned an invalid authorization response")
	}
	return &response, nil
}

func (client *httpSipeedPairingClient) Devices(
	ctx context.Context,
	deviceToken string,
) (*sipeedDevicesResponse, error) {
	var response sipeedDevicesResponse
	if err := client.do(ctx, http.MethodGet, sipeedDevicesPath, deviceToken, nil, &response); err != nil {
		return nil, err
	}
	if response.NetworkID != "" && !validSipeedStateString(response.NetworkID, 256, false) {
		return nil, errors.New("pairing service returned an invalid network id")
	}
	for index := range response.Devices {
		device := &response.Devices[index]
		if err := validateSipeedNodeResponse(device.NodeID, device.IPs); err != nil ||
			!validSipeedDisplayName(device.Name, false) || !validSipeedDisplayName(device.LastSeen, true) {
			return nil, errors.New("pairing service returned an invalid device response")
		}
		if device.LastSeen != "" {
			if _, err := time.Parse(time.RFC3339, device.LastSeen); err != nil {
				return nil, errors.New("pairing service returned an invalid device timestamp")
			}
		}
	}
	return &response, nil
}

func (client *httpSipeedPairingClient) RenameDevice(
	ctx context.Context,
	deviceToken string,
	nodeID string,
	name string,
) (*sipeedRenameDeviceResponse, error) {
	if !validSipeedNodeID(nodeID) || !validSipeedDisplayName(name, false) || strings.TrimSpace(name) == "" {
		return nil, errors.New("invalid device rename parameters")
	}
	var response sipeedRenameDeviceResponse
	path := sipeedDevicePathPrefix + url.PathEscape(nodeID) + "/name"
	if err := client.post(ctx, path, deviceToken, sipeedRenameDeviceRequest{Name: strings.TrimSpace(name)}, &response); err != nil {
		return nil, err
	}
	if response.NodeID != nodeID || strings.TrimSpace(response.Name) == "" ||
		!validSipeedDisplayName(response.Name, false) {
		return nil, errors.New("pairing service returned an invalid rename response")
	}
	return &response, nil
}

func (client *httpSipeedPairingClient) post(
	ctx context.Context,
	path string,
	deviceToken string,
	input any,
	output any,
) error {
	return client.do(ctx, http.MethodPost, path, deviceToken, input, output)
}

func (client *httpSipeedPairingClient) do(
	ctx context.Context,
	method string,
	path string,
	deviceToken string,
	input any,
	output any,
) error {
	_, err := client.doWithStatus(ctx, method, path, deviceToken, input, output)
	return err
}

func (client *httpSipeedPairingClient) doWithStatus(
	ctx context.Context,
	method string,
	path string,
	deviceToken string,
	input any,
	output any,
) (int, error) {
	if client == nil || client.httpClient == nil || !validRawURLCredential(deviceToken, sipeedDeviceTokenBytes) {
		return 0, errors.New("invalid pairing client parameters")
	}
	endpoint, err := url.Parse(client.baseURL + path)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" || endpoint.User != nil {
		return 0, errors.New("invalid pairing service URL")
	}
	var requestBody io.Reader
	if input != nil {
		payload, err := json.Marshal(input)
		if err != nil {
			return 0, errors.New("encode pairing request failed")
		}
		requestBody = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), requestBody)
	if err != nil {
		return 0, errors.New("build pairing request failed")
	}
	request.Header.Set("Authorization", "Bearer "+deviceToken)
	if input != nil {
		request.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	request.Header.Set("Accept", "application/json")

	response, err := client.httpClient.Do(request)
	if err != nil {
		if ctx != nil && ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, errors.New("pairing service is unavailable")
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(response.Body, sipeedPairingMaxResponseSize+1))
	if err != nil || len(body) > sipeedPairingMaxResponseSize {
		return response.StatusCode, errors.New("invalid pairing service response")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return response.StatusCode, decodeSipeedPairingError(response, body)
	}
	if output == nil {
		return response.StatusCode, nil
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return response.StatusCode, errors.New("invalid pairing service response")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(output); err != nil {
		return response.StatusCode, errors.New("invalid pairing service response")
	}
	if err := ensureSipeedJSONEOF(decoder); err != nil {
		return response.StatusCode, errors.New("invalid pairing service response")
	}
	return response.StatusCode, nil
}

func validSipeedNodeID(nodeID string) bool {
	return validSipeedStateString(nodeID, 128, false)
}

func decodeSipeedPairingError(response *http.Response, body []byte) error {
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &envelope)
	code := strings.TrimSpace(envelope.Error.Code)
	if !knownSipeedPairingErrorCode(code) {
		code = ""
	}
	if response.StatusCode == http.StatusConflict {
		// The device contract requires idempotent 200. Treating 409 as success
		// would discard the fresh auth key required for the next LocalAPI start.
		code = "protocol_error"
	}
	if code == "" {
		code = "http_" + strconv.Itoa(response.StatusCode)
	}

	apiError := &sipeedPairingAPIError{HTTPStatus: response.StatusCode, Code: code}
	if response.StatusCode == http.StatusTooManyRequests {
		retryAfter := strings.TrimSpace(response.Header.Get("Retry-After"))
		seconds, err := strconv.Atoi(retryAfter)
		if err != nil || seconds < 1 {
			if deadline, parseErr := http.ParseTime(retryAfter); parseErr == nil {
				seconds = int(math.Ceil(time.Until(deadline).Seconds()))
			}
		}
		if seconds < 1 {
			seconds = 60
		}
		if seconds > 24*60*60 {
			seconds = 24 * 60 * 60
		}
		apiError.RetryAfter = time.Duration(seconds) * time.Second
	}
	return apiError
}

func knownSipeedPairingErrorCode(code string) bool {
	switch code {
	case "invalid_token", "rate_limited", "already_enrolled", "upstream_error", "not_found",
		"invalid_request", "auth_id_expired":
		return true
	default:
		return false
	}
}

func normalizeSipeedClaimCode(value string) (string, error) {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		if r == ' ' || r == '-' || r == '\t' || r == '\n' {
			return -1
		}
		return r
	}, strings.TrimSpace(value))
	if len(digits) != 6 {
		return "", errors.New("invalid claim code")
	}
	for _, digit := range digits {
		if digit < '0' || digit > '9' {
			return "", errors.New("invalid claim code")
		}
	}
	return digits, nil
}

func formatSipeedClaimCode(digits string) string {
	if len(digits) != 6 {
		return digits
	}
	return digits[:3] + " " + digits[3:]
}

func normalizeSipeedBackupCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	return strings.ReplaceAll(value, "-", "")
}

func validSipeedBackupCode(value string) bool {
	if len(value) != 8 {
		return false
	}
	const alphabet = "23456789ABCDEFGHJKMNPQRSTVWXYZ"
	for _, char := range value {
		if !strings.ContainsRune(alphabet, char) {
			return false
		}
	}
	return true
}

func validSipeedAuthID(value string) bool {
	return validSipeedStateString(value, 512, false)
}

func validSipeedPendingID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}

func validSipeedDisplayName(value string, allowEmpty bool) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return allowEmpty
	}
	if len(value) > 255 || strings.ContainsAny(value, "\r\n") {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

func validateSipeedPending(request *sipeedPendingRequest) error {
	if request == nil || !validSipeedPendingID(request.PendingID) || len(request.SAS) != 3 {
		return errors.New("pairing service returned an invalid pending response")
	}
	for _, value := range request.SAS {
		if len(value) != 1 || value[0] < '0' || value[0] > '9' {
			return errors.New("pairing service returned an invalid pending SAS")
		}
	}
	if _, err := time.Parse(time.RFC3339, request.CreatedAt); err != nil {
		return errors.New("pairing service returned an invalid pending timestamp")
	}
	if !validSipeedDisplayName(request.ClientIP, true) || !validSipeedDisplayName(request.ClientUA, true) {
		return errors.New("pairing service returned an invalid pending client")
	}
	if request.ClientIP != "" {
		if net.ParseIP(request.ClientIP) == nil {
			return errors.New("pairing service returned an invalid pending client address")
		}
	}
	return nil
}

func validateSipeedNodeResponse(nodeID string, ips []string) error {
	if !validSipeedNodeID(nodeID) || len(ips) == 0 {
		return errors.New("pairing service returned an invalid node response")
	}
	for _, value := range ips {
		if net.ParseIP(value) == nil {
			return errors.New("pairing service returned an invalid node address")
		}
	}
	return nil
}

func managedErrorFromPairing(err error) (code string, invalidIdentity bool, retryAfter time.Duration) {
	var apiError *sipeedPairingAPIError
	if errors.As(err, &apiError) {
		switch {
		case apiError.HTTPStatus == http.StatusUnauthorized:
			return "invalidToken", true, 0
		case apiError.HTTPStatus == http.StatusTooManyRequests:
			return "rateLimited", false, apiError.RetryAfter
		case apiError.HTTPStatus == http.StatusConflict:
			return "protocolError", false, 0
		case apiError.HTTPStatus == http.StatusBadGateway:
			return "pairingUpstreamError", false, 0
		case apiError.HTTPStatus == http.StatusNotFound:
			return "nodeNotFound", false, 0
		default:
			return "pairingRejected", false, 0
		}
	}
	return "pairingUnavailable", false, 0
}
