package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	memoryFabricRoot                  = "/var/lib/memfab"
	memoryFabricConfigCommandTimeout  = 30 * time.Second
	memoryFabricRestartReadyTimeout   = 5 * time.Second
	memoryFabricRestartPollInterval   = 250 * time.Millisecond
	memoryFabricRestartStableChecks   = 2
	maxMemoryFabricConfigRequestBytes = 1 << 20
)

var errMemoryFabricConfigRevision = errors.New("memory fabric config revision conflict")

var (
	memoryFabricConfigUpdateMu   sync.Mutex
	memoryFabricConfigFilePath   = filepath.Join(memoryFabricRoot, "config.toml")
	readMemoryFabricConfigFile   = os.ReadFile
	runMemoryFabricConfigCLI     = defaultRunMemoryFabricConfigCLI
	waitMemoryFabricConfigPoll   = defaultWaitMemoryFabricConfigPoll
	newMemoryFabricConfigRuntime = func() memoryFabricConfigRuntime {
		manager := newMemoryFabricManager()
		return memoryFabricConfigRuntime{
			isRunning: func() (bool, error) {
				return manager.isRunning(manager.unitName)
			},
			restart: func() error {
				return utils.RestartService(manager.unitName)
			},
		}
	}
)

type memoryFabricConfigRuntime struct {
	isRunning func() (bool, error)
	restart   func() error
}

type memoryFabricConfigUpdateResult struct {
	Config  json.RawMessage `json:"config"`
	Changed bool            `json:"changed"`
}

func (s *Service) GetMemoryFabricConfig(c *gin.Context) {
	var rsp proto.Response

	output, err := callMemoryFabricConfigCLI(c.Request.Context(), []string{"get", "--json"}, nil)
	if err != nil {
		log.Errorf("failed to get Memory Fabric config: %v", err)
		rsp.ErrRsp(c, -1, "get memory fabric config failed")
		return
	}
	config, err := parseMemoryFabricPublicConfig(output)
	if err != nil {
		log.Errorf("invalid Memory Fabric config output: %v", err)
		rsp.ErrRsp(c, -1, "get memory fabric config failed")
		return
	}
	rsp.OkRspWithData(c, proto.GetMemoryFabricConfigRsp{Config: config})
}

func (s *Service) SetMemoryFabricConfig(c *gin.Context) {
	var rsp proto.Response

	body, requestedRevision, err := readMemoryFabricConfigRequest(c.Request.Body)
	if err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	memoryFabricConfigUpdateMu.Lock()
	defer memoryFabricConfigUpdateMu.Unlock()

	backup, err := backupMemoryFabricConfig()
	if err != nil {
		log.Errorf("failed to back up Memory Fabric config: %v", err)
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	backupRevision, err := memoryFabricConfigRevisionFromRaw(backup)
	if err != nil {
		log.Errorf("failed to calculate Memory Fabric backup revision: %v", err)
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	if backupRevision != requestedRevision {
		rsp.ErrRsp(c, -3, "config changed; reload and retry")
		return
	}

	output, err := callMemoryFabricConfigCLIInBackground([]string{"update", "--json"}, body)
	if err != nil {
		if errors.Is(err, errMemoryFabricConfigRevision) {
			rsp.ErrRsp(c, -3, "config changed; reload and retry")
			return
		}
		// The command may theoretically fail after atomically committing the file
		// but before returning its JSON result. Without the committed revision, an
		// unconditional restore could overwrite a concurrent writer, so fail closed
		// and leave recovery to a subsequent revision-aware request.
		log.Errorf("failed to update Memory Fabric config; commit state is unknown: %v", err)
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}

	var result memoryFabricConfigUpdateResult
	if err := decodeConfigJSONOutput(output, &result); err != nil || len(result.Config) == 0 {
		log.Errorf("invalid Memory Fabric update output: %v", err)
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	newRevision, err := memoryFabricConfigRevision(result.Config)
	if err != nil {
		log.Errorf("Memory Fabric update output has no usable revision: %v", err)
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	if err := validateMemoryFabricConfigJSONSafety(output); err != nil {
		log.Errorf("unsafe Memory Fabric update output rejected: %v", err)
		if result.Changed {
			logMemoryFabricConfigRollbackError(rollbackMemoryFabricConfig(
				backup, newRevision, memoryFabricConfigRuntime{}, false,
			))
		}
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	config, err := parseMemoryFabricPublicConfig(result.Config)
	if err != nil {
		log.Errorf("invalid Memory Fabric updated config: %v", err)
		if result.Changed {
			logMemoryFabricConfigRollbackError(rollbackMemoryFabricConfig(
				backup, newRevision, memoryFabricConfigRuntime{}, false,
			))
		}
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}

	if !result.Changed {
		rsp.OkRspWithData(c, proto.SetMemoryFabricConfigRsp{Config: config})
		return
	}

	runtime := newMemoryFabricConfigRuntime()
	running, err := runtime.isRunning()
	if err != nil {
		log.Errorf("failed to get memfab.service status after config update: %v", err)
		logMemoryFabricConfigRollbackError(rollbackMemoryFabricConfig(
			backup, newRevision, runtime, false,
		))
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	if !running {
		rsp.OkRspWithData(c, proto.SetMemoryFabricConfigRsp{Config: config, Changed: true})
		return
	}

	if err := runtime.restart(); err != nil {
		log.Errorf("failed to restart memfab.service after config update: %v", err)
		logMemoryFabricConfigRollbackError(rollbackMemoryFabricConfig(
			backup, newRevision, runtime, true,
		))
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	if err := waitForMemoryFabricConfigService(runtime); err != nil {
		log.Errorf("memfab.service did not become stable after config update: %v", err)
		logMemoryFabricConfigRollbackError(rollbackMemoryFabricConfig(
			backup, newRevision, runtime, true,
		))
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	rsp.OkRspWithData(c, proto.SetMemoryFabricConfigRsp{
		Config: config, Changed: true, Restarted: true,
	})
}

func defaultRunMemoryFabricConfigCLI(ctx context.Context, args []string, input []byte) ([]byte, error) {
	commandArgs := append([]string{"config"}, args...)
	cmd := exec.CommandContext(ctx, "memfab", commandArgs...)
	cmd.Env = append(os.Environ(), "MEMFAB_ROOT="+memoryFabricRoot)
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if strings.Contains(message, "config revision conflict") {
			return nil, errMemoryFabricConfigRevision
		}
		return nil, fmt.Errorf("memfab config command failed: %w", err)
	}
	return output, nil
}

func callMemoryFabricConfigCLI(parent context.Context, args []string, input []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, memoryFabricConfigCommandTimeout)
	defer cancel()
	return runMemoryFabricConfigCLI(ctx, args, input)
}

func callMemoryFabricConfigCLIInBackground(args []string, input []byte) ([]byte, error) {
	return callMemoryFabricConfigCLI(context.Background(), args, input)
}

func backupMemoryFabricConfig() ([]byte, error) {
	backup, err := readMemoryFabricConfigFile(memoryFabricConfigFilePath)
	if err == nil {
		return backup, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	// ReadUI creates the default config under memfab's own locking and file
	// ownership rules. Do not duplicate those rules in the web server.
	output, initErr := callMemoryFabricConfigCLIInBackground([]string{"get", "--json"}, nil)
	if initErr == nil {
		if _, validateErr := parseMemoryFabricPublicConfig(output); validateErr != nil {
			return nil, fmt.Errorf("validate initialized config: %w", validateErr)
		}
	}

	backup, readErr := readMemoryFabricConfigFile(memoryFabricConfigFilePath)
	if readErr == nil {
		return backup, nil
	}
	if initErr != nil {
		return nil, fmt.Errorf("initialize config: %w", initErr)
	}
	return nil, readErr
}

func readMemoryFabricConfigRequest(reader io.Reader) ([]byte, string, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maxMemoryFabricConfigRequestBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(body) == 0 || len(body) > maxMemoryFabricConfigRequestBytes || !json.Valid(body) {
		return nil, "", errors.New("invalid config request")
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil || object == nil {
		return nil, "", errors.New("config request must be a JSON object")
	}
	var revision string
	if rawRevision, ok := object["revision"]; !ok || json.Unmarshal(rawRevision, &revision) != nil || revision == "" {
		return nil, "", errors.New("config request revision is required")
	}
	return body, revision, nil
}

func decodeConfigJSONOutput(data []byte, value interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func parseMemoryFabricPublicConfig(data []byte) (proto.MemoryFabricConfig, error) {
	var config proto.MemoryFabricConfig
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return proto.MemoryFabricConfig{}, err
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return proto.MemoryFabricConfig{}, errors.New("multiple JSON values")
		}
		return proto.MemoryFabricConfig{}, err
	}
	if config.Revision == "" {
		return proto.MemoryFabricConfig{}, errors.New("config revision is empty")
	}
	if err := validateMemoryFabricConfigJSONSafety(data); err != nil {
		return proto.MemoryFabricConfig{}, err
	}
	return config, nil
}

func memoryFabricConfigRevision(configJSON []byte) (string, error) {
	var object struct {
		Revision string `json:"revision"`
	}
	if err := decodeConfigJSONOutput(configJSON, &object); err != nil {
		return "", err
	}
	if object.Revision == "" {
		return "", errors.New("config revision is empty")
	}
	return object.Revision, nil
}

func memoryFabricConfigRevisionFromRaw(raw []byte) (string, error) {
	output, err := callMemoryFabricConfigCLIInBackground([]string{"revision", "--raw"}, raw)
	if err != nil {
		return "", err
	}
	revision := strings.TrimSpace(string(output))
	if revision == "" || len(revision) > 256 || strings.IndexFunc(revision, func(r rune) bool {
		return r <= ' ' || r == 0x7f
	}) >= 0 {
		return "", errors.New("invalid config revision output")
	}
	return revision, nil
}

func validateMemoryFabricConfigJSONSafety(data []byte) error {
	var value interface{}
	if err := decodeConfigJSONOutput(data, &value); err != nil {
		return err
	}
	return rejectSensitiveMemoryFabricConfigJSON(value)
}

func rejectSensitiveMemoryFabricConfigJSON(value interface{}) error {
	switch value := value.(type) {
	case map[string]interface{}:
		for key, child := range value {
			normalized := normalizeMemoryFabricConfigJSONKey(key)
			switch normalized {
			case "apikeyconfigured":
				if _, ok := child.(bool); !ok {
					return errors.New("apiKeyConfigured must be boolean in CLI output")
				}
			case "maxtokens", "softlimittokens", "hardlimittokens", "minestimatedtokens":
				if _, ok := child.(float64); !ok {
					return fmt.Errorf("token count field %q is not numeric in CLI output", key)
				}
			case "":
				return errors.New("empty config field is not allowed in CLI output")
			default:
				if strings.HasPrefix(normalized, "apikey") ||
					strings.Contains(normalized, "token") ||
					strings.Contains(normalized, "secret") ||
					strings.Contains(normalized, "password") ||
					normalized == "authorization" {
					return fmt.Errorf("sensitive config field %q is not allowed in CLI output", key)
				}
			}
			if err := rejectSensitiveMemoryFabricConfigJSON(child); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, child := range value {
			if err := rejectSensitiveMemoryFabricConfigJSON(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeMemoryFabricConfigJSONKey(key string) string {
	normalized := strings.ToLower(key)
	return strings.NewReplacer("_", "", "-", "", " ", "").Replace(normalized)
}

func defaultWaitMemoryFabricConfigPoll(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func waitForMemoryFabricConfigService(runtime memoryFabricConfigRuntime) error {
	ctx, cancel := context.WithTimeout(context.Background(), memoryFabricRestartReadyTimeout)
	defer cancel()

	consecutiveActive := 0
	var lastStatusErr error
	for {
		active, err := runtime.isRunning()
		if err == nil && active {
			consecutiveActive++
			if consecutiveActive >= memoryFabricRestartStableChecks {
				return nil
			}
		} else {
			consecutiveActive = 0
			lastStatusErr = err
		}

		if err := waitMemoryFabricConfigPoll(ctx, memoryFabricRestartPollInterval); err != nil {
			if lastStatusErr != nil {
				return fmt.Errorf("service status check failed: %w", lastStatusErr)
			}
			return errors.New("service did not remain active")
		}
	}
}

func rollbackMemoryFabricConfig(
	backup []byte,
	expectedRevision string,
	runtime memoryFabricConfigRuntime,
	restart bool,
) error {
	if expectedRevision == "" {
		return errors.New("refusing config rollback without expected revision")
	}
	args := []string{"restore", "--raw", "--if-revision", expectedRevision}
	if _, err := callMemoryFabricConfigCLIInBackground(args, backup); err != nil {
		return fmt.Errorf("restore previous config: %w", err)
	}
	if restart {
		if err := runtime.restart(); err != nil {
			return fmt.Errorf("restart memfab.service after config rollback: %w", err)
		}
		if err := waitForMemoryFabricConfigService(runtime); err != nil {
			return fmt.Errorf("confirm memfab.service after config rollback: %w", err)
		}
	}
	return nil
}

func logMemoryFabricConfigRollbackError(err error) {
	if err == nil {
		return
	}
	if errors.Is(err, errMemoryFabricConfigRevision) {
		log.Warn("Memory Fabric config rollback skipped because the config changed concurrently")
		return
	}
	log.Errorf("failed to roll back Memory Fabric config: %v", err)
}
