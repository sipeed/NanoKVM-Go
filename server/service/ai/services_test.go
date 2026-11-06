package ai

import (
	"testing"

	"NanoKVM-Server/proto"
)

func TestMemoryTierForChipType(t *testing.T) {
	tests := []struct {
		name     string
		chipType string
		want     string
	}{
		{
			name:     "AX620QF is 256MB",
			chipType: chipTypeAX620QF,
			want:     memoryTier256MB,
		},
		{
			name:     "AX620QE is 512MB",
			chipType: chipTypeAX620QE,
			want:     memoryTier512MB,
		},
		{
			name:     "trims chip type whitespace",
			chipType: chipTypeAX620QE + "\n",
			want:     memoryTier512MB,
		},
		{
			name:     "unknown chip type defaults to 256MB",
			chipType: "UNKNOWN_CHIP",
			want:     memoryTier256MB,
		},
		{
			name:     "missing chip type defaults to 256MB",
			chipType: "",
			want:     memoryTier256MB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := memoryTierForChipType(tt.chipType); got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestBuildAIServicesUsesMemoryTierForSupport(t *testing.T) {
	t.Run("256MB tier disables 512MB services even with high visible memory", func(t *testing.T) {
		services := buildAIServices(512, memoryTierForChipType(chipTypeAX620QF))

		if services.MemoryTier != memoryTier256MB {
			t.Fatalf("expected memory tier %s, got %s", memoryTier256MB, services.MemoryTier)
		}
		if !serviceSupported(services.Services, aiServiceMCP) {
			t.Fatalf("expected MCP service to be supported")
		}
		if serviceSupported(services.Services, aiServiceMemoryFabric) {
			t.Fatalf("expected memory fabric service to be unsupported")
		}
		if serviceSupported(services.Services, aiServiceScreenTimelapse) {
			t.Fatalf("expected screen timelapse service to be unsupported")
		}
	})

	t.Run("512MB tier enables 512MB services even with low visible memory", func(t *testing.T) {
		services := buildAIServices(128, memoryTierForChipType(chipTypeAX620QE))

		if services.MemoryTier != memoryTier512MB {
			t.Fatalf("expected memory tier %s, got %s", memoryTier512MB, services.MemoryTier)
		}
		if !serviceSupported(services.Services, aiServiceMemoryFabric) {
			t.Fatalf("expected memory fabric service to be supported")
		}
		if !serviceSupported(services.Services, aiServiceScreenTimelapse) {
			t.Fatalf("expected screen timelapse service to be supported")
		}
	})
}

func serviceSupported(services []proto.AIServiceCapability, id string) bool {
	for _, service := range services {
		if service.ID == id {
			return service.Supported
		}
	}

	return false
}
