package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	edidSourceDefault = "default"
	edidSourceCustom  = "custom"

	defaultCurrentEdidName = "nanokvm-go-3840x2160-30"
	edidBlockSize          = 128
	edidUploadRequestLimit = 16 * 1024
)

var (
	EdidDir              = "/kvmcomm/edid"
	LT7911Edid           = "/proc/lt7911_info/edid"
	CustomEdidDir        = "/etc/kvm/edid"
	CurrentEdidStateFile = "/etc/kvm/edid/current.json"

	edidMutationMu sync.Mutex

	writeEdidHardware = func(data []byte) error {
		return os.WriteFile(LT7911Edid, data, 0o644)
	}
	saveCurrentEdidStateAction = saveCurrentEdidState
	renameEdidFileAction       = os.Rename
)

type edidState struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	File   string `json:"file"`
}

type edidRollbackCandidate struct {
	data          []byte
	fallbackState *edidState
}

type edidErrorResponse struct {
	Reason edidErrorReason `json:"reason"`
}

type uploadEdidError struct {
	code int
	msg  string
	err  error
}

func (e *uploadEdidError) Error() string {
	return e.err.Error()
}

func (e *uploadEdidError) Unwrap() error {
	return e.err
}

func writeEdidErrorResponse(c *gin.Context, rsp *proto.Response, code int, msg string, reason edidErrorReason) {
	if reason == "" {
		rsp.ErrRsp(c, code, msg)
		return
	}
	rsp.ErrRspWithData(c, code, msg, &edidErrorResponse{Reason: reason})
}

func (s *Service) GetEdid(ctx *gin.Context) {
	var resp proto.Response

	edidMutationMu.Lock()
	defer edidMutationMu.Unlock()

	state, err := loadCurrentEdidState()
	if err != nil {
		log.Errorf("get current EDID failed: %s", err)
		resp.ErrRsp(ctx, -1, "get edid failed")
		return
	}

	width, height := currentEdidResolution(state)
	resp.OkRspWithData(ctx, &proto.GetEdidRsp{Edid: state.Name, Width: width, Height: height})
}

func currentEdidResolution(state edidState) (int, int) {
	data, err := readEdidFile(state.File)
	if err == nil {
		if width, height, ok := preferredEdidResolution(data); ok {
			return width, height
		}
	} else {
		log.Warnf("failed to read current EDID resolution from %s: %s", state.File, err)
	}
	if width, height, ok := resolutionFromEdidName(state.Name); ok {
		return width, height
	}
	return 1920, 1080
}

func (s *Service) SwitchEdid(c *gin.Context) {
	var req proto.SwitchEdidReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		writeEdidErrorResponse(c, &rsp, -1, "invalid arguments", edidReasonBadRequest)
		return
	}

	if req.Edid == "" {
		writeEdidErrorResponse(c, &rsp, -2, "invalid EDID", edidReasonInvalidFileName)
		return
	}

	if !isSafeBuiltinEdidName(req.Edid) && !isSafeCustomEdidName(req.Edid) {
		writeEdidErrorResponse(c, &rsp, -2, "invalid EDID", edidReasonInvalidFileName)
		return
	}

	virtualDeviceUpdateMu.Lock()
	defer virtualDeviceUpdateMu.Unlock()
	edidMutationMu.Lock()
	defer edidMutationMu.Unlock()

	state, err := resolveEdidState(req.Edid)
	if err != nil {
		log.Debugf("unknown edid: %s", req.Edid)
		writeEdidErrorResponse(c, &rsp, -3, "invalid EDID", edidReasonInvalidFileName)
		return
	}

	dpAudioEnabled := !virtualSpeakerMarkerExists()
	effectiveEdid, err := readEffectiveEdid(state.File, dpAudioEnabled)
	if err != nil {
		log.Errorf("failed to prepare EDID %s: %s", req.Edid, err)
		writeEdidErrorResponse(c, &rsp, -4, "failed to switch EDID", edidReason(err))
		return
	}

	previousEdid, err := prepareEdidRollbackCandidate(dpAudioEnabled)
	if err != nil {
		log.Errorf("failed to prepare previous EDID for rollback: %s", err)
		writeEdidErrorResponse(c, &rsp, -4, "failed to get current EDID", edidReason(err))
		return
	}

	if err := writeValidatedEdidHardware(effectiveEdid); err != nil {
		log.Errorf("failed to switch EDID %s: %s", req.Edid, err)
		if rollbackErr := rollbackEdidHardware(previousEdid, "switch write failure"); rollbackErr != nil {
			rsp.ErrRsp(c, -4, "failed to switch EDID and rollback failed")
			return
		}
		rsp.ErrRsp(c, -4, "failed to switch EDID")
		return
	}

	if err := saveCurrentEdidStateAction(state); err != nil {
		log.Errorf("failed to save current EDID %s: %s", req.Edid, err)
		if rollbackErr := rollbackEdidHardware(previousEdid, "current state save failure"); rollbackErr != nil {
			rsp.ErrRsp(c, -5, "failed to save EDID state and rollback failed")
			return
		}
		rsp.ErrRsp(c, -5, "failed to save EDID state")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("switch edid %s", req.Edid)
}

func (s *Service) GetCustomEdidList(c *gin.Context) {
	var rsp proto.Response

	var files []string

	err := filepath.Walk(CustomEdidDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && isBin(info.Name()) {
			files = append(files, info.Name())
		}

		return nil
	})
	if err != nil {
		rsp.ErrRsp(c, -1, "get EDID failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetCustomEdidListRsp{
		EdidList: files,
	})
}

func (s *Service) DeleteEdid(c *gin.Context) {
	var req proto.DeleteEdidReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	file, ok := safeCustomEdidPath(req.Edid)
	if !ok {
		rsp.ErrRsp(c, -2, "invalid EDID")
		return
	}

	edidMutationMu.Lock()
	defer edidMutationMu.Unlock()

	if current, err := loadCurrentEdidState(); err == nil {
		if current.Source == edidSourceCustom && current.Name == req.Edid {
			rsp.ErrRsp(c, -2, "current EDID cannot be deleted")
			return
		}
	} else {
		log.Warnf("failed to read current EDID before delete: %s", err)
	}

	if err := os.Remove(file); err != nil {
		log.Errorf("delete edid %s failed: %s", req.Edid, err)
		rsp.ErrRsp(c, -2, "delete failed")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("delete edid %s success", req.Edid)
}

func (s *Service) UploadEdid(c *gin.Context) {
	var rsp proto.Response

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, edidUploadRequestLimit)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		reason := edidReasonBadRequest
		if errors.As(err, &maxBytesErr) {
			reason = edidReasonTooLarge
		}
		writeEdidErrorResponse(c, &rsp, -1, "bad request", reason)
		return
	}
	defer file.Close()

	target, ok := safeCustomEdidPath(header.Filename)
	if !ok {
		writeEdidErrorResponse(c, &rsp, -2, "invalid EDID", edidReasonInvalidFileName)
		return
	}

	data, err := readEdid(file)
	if err != nil {
		reason := edidReason(err)
		if reason != "" {
			writeEdidErrorResponse(c, &rsp, -2, "invalid EDID", reason)
		} else {
			writeEdidErrorResponse(c, &rsp, -3, "save failed", edidReasonSaveFailed)
		}
		return
	}

	edidMutationMu.Lock()
	defer edidMutationMu.Unlock()

	if err := saveUploadedEdid(target, header.Filename, data); err != nil {
		var uploadErr *uploadEdidError
		if errors.As(err, &uploadErr) {
			writeEdidErrorResponse(c, &rsp, uploadErr.code, uploadErr.msg, edidReason(uploadErr))
		} else {
			writeEdidErrorResponse(c, &rsp, -3, "save failed", edidReasonSaveFailed)
		}
		return
	}

	responseData := &proto.UploadEdidRsp{
		File: header.Filename,
	}
	rsp.OkRspWithData(c, responseData)
	log.Debugf("upload edid file: %s", header.Filename)
}

func saveUploadedEdid(target, name string, data []byte) error {
	if err := validateEdid(data); err != nil {
		return &uploadEdidError{code: -2, msg: "invalid EDID", err: err}
	}
	if err := os.MkdirAll(CustomEdidDir, 0o755); err != nil {
		return newUploadEdidError(-2, "create file failed", edidReasonSaveFailed, "create EDID directory: %v", err)
	}

	current, hasCurrent, err := loadPersistedCurrentEdidState()
	if err != nil {
		return newUploadEdidError(-2, "create file failed", edidReasonSaveFailed, "inspect current EDID: %v", err)
	}
	if hasCurrent && current.Source == edidSourceCustom && current.Name == name {
		return newUploadEdidError(
			-2,
			"current EDID cannot be overwritten",
			edidReasonCurrentCannotBeOverwritten,
			"current EDID cannot be overwritten",
		)
	}

	info, err := os.Lstat(target)
	if err != nil && !os.IsNotExist(err) {
		return newUploadEdidError(-2, "create file failed", edidReasonSaveFailed, "inspect EDID target: %v", err)
	}
	if err == nil {
		if !info.Mode().IsRegular() {
			return newUploadEdidError(-2, "create file failed", edidReasonSaveFailed, "EDID target is not a regular file")
		}
	}

	tmp, err := os.CreateTemp(CustomEdidDir, ".edid-upload-*")
	if err != nil {
		return newUploadEdidError(-2, "create file failed", edidReasonSaveFailed, "create temporary EDID: %v", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return newUploadEdidError(-3, "save failed", edidReasonSaveFailed, "set temporary EDID permissions: %v", err)
	}
	if n, err := tmp.Write(data); err != nil || n != len(data) {
		_ = tmp.Close()
		if err == nil {
			err = io.ErrShortWrite
		}
		return newUploadEdidError(-3, "save failed", edidReasonSaveFailed, "write temporary EDID: %v", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return newUploadEdidError(-3, "save failed", edidReasonSaveFailed, "sync temporary EDID: %v", err)
	}
	if err := tmp.Close(); err != nil {
		return newUploadEdidError(-3, "save failed", edidReasonSaveFailed, "close temporary EDID: %v", err)
	}
	if err := renameEdidFileAction(tmpPath, target); err != nil {
		return newUploadEdidError(-3, "save failed", edidReasonSaveFailed, "commit EDID: %v", err)
	}

	dir, err := os.Open(CustomEdidDir)
	if err != nil {
		log.Warnf("failed to open custom EDID directory for sync: %s", err)
		return nil
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		log.Warnf("failed to sync custom EDID directory: %s", err)
	}
	return nil
}

func newUploadEdidError(code int, msg string, reason edidErrorReason, format string, args ...any) error {
	return &uploadEdidError{
		code: code,
		msg:  msg,
		err:  newEdidError(reason, format, args...),
	}
}

func resolveEdidState(name string) (edidState, error) {
	if isSafeBuiltinEdidName(name) {
		defaultPath := filepath.Join(EdidDir, name+".bin")
		if exists, err := regularFileExists(defaultPath); err != nil {
			return edidState{}, err
		} else if exists {
			return edidState{
				Name:   name,
				Source: edidSourceDefault,
				File:   defaultPath,
			}, nil
		}
	}

	if isSafeCustomEdidName(name) {
		customPath, ok := safeCustomEdidPath(name)
		if !ok {
			return edidState{}, fmt.Errorf("invalid EDID: %s", name)
		}
		if exists, err := regularFileExists(customPath); err != nil {
			return edidState{}, err
		} else if exists {
			return edidState{
				Name:   name,
				Source: edidSourceCustom,
				File:   customPath,
			}, nil
		}
	}

	return edidState{}, fmt.Errorf("unknown EDID: %s", name)
}

func loadCurrentEdidState() (edidState, error) {
	data, err := os.ReadFile(CurrentEdidStateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return createDefaultCurrentEdidState()
		}
		return edidState{}, err
	}

	var state edidState
	if err := json.Unmarshal(data, &state); err != nil {
		return edidState{}, err
	}

	if err := validateEdidState(state); err != nil {
		return edidState{}, err
	}

	return state, nil
}

func loadPersistedCurrentEdidState() (edidState, bool, error) {
	data, err := os.ReadFile(CurrentEdidStateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return edidState{}, false, nil
		}
		return edidState{}, false, err
	}

	var state edidState
	if err := json.Unmarshal(data, &state); err != nil {
		return edidState{}, false, err
	}
	if err := validateEdidStateReference(state); err != nil {
		return edidState{}, false, err
	}
	return state, true, nil
}

func createDefaultCurrentEdidState() (edidState, error) {
	state, err := resolveEdidState(defaultCurrentEdidName)
	if err != nil {
		return edidState{}, err
	}

	if err := saveCurrentEdidState(state); err != nil {
		return edidState{}, err
	}

	return state, nil
}

func saveCurrentEdidState(state edidState) error {
	if err := validateEdidState(state); err != nil {
		return err
	}

	dir := filepath.Dir(CurrentEdidStateFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".current.json.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, CurrentEdidStateFile); err != nil {
		return err
	}

	dirFile, err := os.Open(dir)
	if err != nil {
		log.Warnf("failed to open current EDID state directory for sync: %s", err)
		return nil
	}
	defer dirFile.Close()
	if err := dirFile.Sync(); err != nil {
		log.Warnf("failed to sync current EDID state directory: %s", err)
	}
	return nil
}

func readEffectiveEdid(path string, dpAudioEnabled bool) ([]byte, error) {
	data, err := readEdidFile(path)
	if err != nil {
		return nil, err
	}

	if !dpAudioEnabled {
		return stripEdidAudio(data)
	}
	return data, nil
}

func prepareEdidRollbackCandidate(dpAudioEnabled bool) (edidRollbackCandidate, error) {
	state, hasState, currentErr := loadPersistedCurrentEdidState()
	if currentErr == nil && !hasState {
		state, currentErr = createDefaultCurrentEdidState()
	}
	if currentErr == nil {
		data, err := readEffectiveEdid(state.File, dpAudioEnabled)
		if err == nil {
			return edidRollbackCandidate{data: data}, nil
		}
		currentErr = err
	}

	defaultState, fallbackErr := resolveEdidState(defaultCurrentEdidName)
	if fallbackErr != nil {
		return edidRollbackCandidate{}, fmt.Errorf(
			"current EDID cannot be prepared (%v) and default rollback state is unavailable: %w",
			currentErr,
			fallbackErr,
		)
	}
	fallbackData, fallbackErr := readEffectiveEdid(defaultState.File, dpAudioEnabled)
	if fallbackErr != nil {
		return edidRollbackCandidate{}, fmt.Errorf(
			"current EDID cannot be prepared (%v) and default rollback EDID cannot be prepared: %w",
			currentErr,
			fallbackErr,
		)
	}

	log.Warnf(
		"current EDID %s cannot be prepared (%s); using %s as the rollback candidate",
		state.Name,
		currentErr,
		defaultCurrentEdidName,
	)
	return edidRollbackCandidate{
		data:          fallbackData,
		fallbackState: &defaultState,
	}, nil
}

func writeValidatedEdidHardware(data []byte) error {
	if err := validateEdid(data); err != nil {
		return err
	}
	return writeEdidHardware(data)
}

func rollbackEdidHardware(candidate edidRollbackCandidate, reason string) error {
	if len(candidate.data) == 0 {
		return errors.New("no EDID data available for rollback")
	}
	if err := writeValidatedEdidHardware(candidate.data); err != nil {
		log.Errorf("failed to roll back EDID after %s: %s", reason, err)
		return err
	}
	if candidate.fallbackState != nil {
		if err := saveCurrentEdidStateAction(*candidate.fallbackState); err != nil {
			log.Errorf("failed to save default EDID state after %s rollback: %s", reason, err)
			return err
		}
		log.Warnf("rolled back EDID hardware and current state to %s", candidate.fallbackState.Name)
	}
	return nil
}

func validateEdidState(state edidState) error {
	if err := validateEdidStateReference(state); err != nil {
		return err
	}

	exists, err := regularFileExists(state.File)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("EDID file does not exist: %s", state.File)
	}

	return nil
}

func validateEdidStateReference(state edidState) error {
	if state.Name == "" || state.Source == "" || state.File == "" {
		return errors.New("invalid EDID state")
	}

	expectedFile := ""
	switch state.Source {
	case edidSourceDefault:
		if !isSafeBuiltinEdidName(state.Name) {
			return fmt.Errorf("invalid EDID name: %s", state.Name)
		}
		expectedFile = filepath.Join(EdidDir, state.Name+".bin")
	case edidSourceCustom:
		if !isSafeCustomEdidName(state.Name) {
			return fmt.Errorf("invalid EDID name: %s", state.Name)
		}
		customPath, ok := safeCustomEdidPath(state.Name)
		if !ok {
			return fmt.Errorf("invalid EDID file: %s", state.File)
		}
		expectedFile = customPath
	default:
		return fmt.Errorf("invalid EDID source: %s", state.Source)
	}

	if state.File != expectedFile {
		return fmt.Errorf("invalid EDID file: %s", state.File)
	}
	return nil
}

func regularFileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return info.Mode().IsRegular(), nil
}

// Build 2 replaced the stock profiles and retired the separate name override.
func RestoreLegacyEdid() error {
	done := filepath.Join(CustomEdidDir, ".nomad-2-profiles-applied")
	legacy := filepath.Join(CustomEdidDir, "config.json")
	hasOverride, err := regularFileExists(legacy)
	if err != nil {
		return err
	}
	if !hasOverride {
		if applied, err := regularFileExists(done); err != nil || applied {
			return err
		}
	}
	state, err := loadCurrentEdidState()
	if err != nil {
		return err
	}
	if hasOverride || state.Source == edidSourceDefault {
		if err := applyCurrentEdid(!virtualSpeakerMarkerExists()); err != nil {
			return err
		}
	}
	if hasOverride {
		if err := os.Remove(legacy); err != nil {
			return err
		}
	}
	return os.WriteFile(done, nil, 0644)
}

func applyCurrentEdid(dpAudioEnabled bool) error {
	edidMutationMu.Lock()
	defer edidMutationMu.Unlock()

	state, err := loadCurrentEdidState()
	if err != nil {
		return err
	}
	input, err := readEffectiveEdid(state.File, dpAudioEnabled)
	if err != nil {
		return err
	}
	return writeValidatedEdidHardware(input)
}

func stripEdidAudio(input []byte) ([]byte, error) {
	if err := validateEdid(input); err != nil {
		return nil, err
	}

	extensions := int(input[126])
	output := append([]byte(nil), input...)
	for blockIndex := 1; blockIndex <= extensions; blockIndex++ {
		block := output[blockIndex*128 : (blockIndex+1)*128]
		if block[0] != 0x02 { // CTA-861 extension
			continue
		}

		// Clear CTA Basic Audio support even if there is no data-block collection.
		block[3] &^= 0x40
		dtdOffset := int(block[2])
		if dtdOffset == 0 {
			updateEdidChecksum(block)
			continue
		}
		if dtdOffset < 4 || dtdOffset > 127 {
			return nil, errors.New("invalid CTA data-block offset")
		}

		readPos, writePos := 4, 4
		for readPos < dtdOffset {
			header := block[readPos]
			length := int(header & 0x1f)
			end := readPos + 1 + length
			if end > dtdOffset {
				return nil, errors.New("invalid CTA data block")
			}

			tag := header >> 5
			remove := tag == 1 || tag == 4 // Audio and Speaker Allocation blocks.
			if tag == 7 && length > 0 {
				extendedTag := block[readPos+1]
				remove = extendedTag >= 0x11 && extendedTag <= 0x14
			}
			if !remove {
				copy(block[writePos:], block[readPos:end])
				writePos += end - readPos
			}
			readPos = end
		}

		dtdLength := 127 - dtdOffset
		copy(block[writePos:], block[dtdOffset:127])
		newEnd := writePos + dtdLength
		clear(block[newEnd:127])
		block[2] = byte(writePos)
		updateEdidChecksum(block)
	}

	if err := validateEdid(output); err != nil {
		return nil, err
	}
	return output, nil
}

func updateEdidChecksum(block []byte) {
	var sum byte
	for _, value := range block[:127] {
		sum += value
	}
	block[127] = -sum
}

func isBin(name string) bool {
	nameLower := strings.ToLower(name)
	if strings.HasSuffix(nameLower, ".bin") {
		return true
	}

	return false
}

func safeCustomEdidPath(name string) (string, bool) {
	if !isSafeCustomEdidName(name) {
		return "", false
	}

	basePath, err := filepath.Abs(filepath.Clean(CustomEdidDir))
	if err != nil {
		return "", false
	}
	targetPath := filepath.Join(basePath, name)
	if !isPathWithin(basePath, targetPath) {
		return "", false
	}

	return targetPath, true
}

func isSafeCustomEdidName(name string) bool {
	return isSafePathName(name) && isBin(name)
}

func isSafeBuiltinEdidName(name string) bool {
	return isSafePathName(name) && !strings.Contains(name, ".")
}

func isSafePathName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if filepath.Base(name) != name {
		return false
	}
	if strings.ContainsAny(name, `/\`) {
		return false
	}
	return true
}

func isPathWithin(basePath string, targetPath string) bool {
	rel, err := filepath.Rel(basePath, targetPath)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}
