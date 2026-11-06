package ai

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v4/mem"
	log "github.com/sirupsen/logrus"
)

const (
	aiServiceMCP             = "mcp"
	aiServiceMemoryFabric    = "memoryFabric"
	aiServiceScreenTimelapse = "screenTimelapse"

	memoryTier256MB = "256MB"
	memoryTier512MB = "512MB"

	memoryTier512MinMiB = 200

	chipTypePath    = "/proc/ax_proc/chip_type"
	chipTypeEnv     = "NANOKVM_CHIP_TYPE"
	chipTypeAX620QF = "AX620QF_CHIP"
	chipTypeAX620QE = "AX620QE_CHIP"

	errAIServiceUnsupportedMsg = "service unsupported on this device"
)

var (
	errAIServiceUnsupported = errors.New(errAIServiceUnsupportedMsg)
	readVisibleMemoryMiB    = getVisibleMemoryMiB
	readChipType            = getChipType
)

type aiServiceRule struct {
	id            string
	requires512MB bool
	minMemoryMiB  int
}

var aiServiceRules = []aiServiceRule{
	{id: aiServiceMCP},
	{id: aiServiceMemoryFabric, requires512MB: true, minMemoryMiB: memoryTier512MinMiB},
	{id: aiServiceScreenTimelapse, requires512MB: true, minMemoryMiB: memoryTier512MinMiB},
}

func getVisibleMemoryMiB() (int, error) {
	info, err := mem.VirtualMemory()
	if err != nil {
		return 0, fmt.Errorf("get memory info failed: %w", err)
	}

	return int(info.Total / 1024 / 1024), nil
}

func getChipType() string {
	if override := strings.TrimSpace(os.Getenv(chipTypeEnv)); override != "" {
		log.Debugf("chip type overridden via %s: %s", chipTypeEnv, override)
		return override
	}

	data, err := os.ReadFile(chipTypePath)
	if err != nil {
		log.Debugf("failed to read chip type: %v", err)
		return ""
	}

	return strings.TrimSpace(string(data))
}

func buildAIServices(memoryMiB int, memoryTier string) proto.GetAIServicesRsp {
	services := make([]proto.AIServiceCapability, 0, len(aiServiceRules))
	for _, rule := range aiServiceRules {
		services = append(services, proto.AIServiceCapability{
			ID:           rule.id,
			Supported:    isAIServiceSupportedForTier(rule, memoryTier),
			MinMemoryMiB: rule.minMemoryMiB,
		})
	}

	return proto.GetAIServicesRsp{
		MemoryMiB:  memoryMiB,
		MemoryTier: memoryTier,
		Services:   services,
	}
}

func memoryTierForChipType(chipType string) string {
	switch strings.TrimSpace(chipType) {
	case chipTypeAX620QE:
		return memoryTier512MB
	case chipTypeAX620QF:
		return memoryTier256MB
	default:
		return memoryTier256MB
	}
}

func isAIServiceSupportedForTier(rule aiServiceRule, memoryTier string) bool {
	return !rule.requires512MB || memoryTier == memoryTier512MB
}

func (s *Service) GetServices(c *gin.Context) {
	var rsp proto.Response

	memoryMiB, err := readVisibleMemoryMiB()
	if err != nil {
		log.Errorf("failed to get AI service capabilities: %v", err)
		rsp.ErrRsp(c, -1, "get AI services failed")
		return
	}

	chipType := readChipType()
	memoryTier := memoryTierForChipType(chipType)

	rsp.OkRspWithData(c, buildAIServices(memoryMiB, memoryTier))
}
