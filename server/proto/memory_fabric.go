package proto

type MemoryFabricModelConfig struct {
	Model            string `json:"model"`
	BaseURL          string `json:"baseUrl"`
	APIKeyConfigured bool   `json:"apiKeyConfigured"`
	MaxTokens        int    `json:"maxTokens"`
	TimeoutSeconds   int    `json:"timeoutSeconds"`
}

type MemoryFabricModelsConfig struct {
	Default MemoryFabricModelConfig `json:"default"`
}

type MemoryFabricConfig struct {
	Revision string                   `json:"revision"`
	Models   MemoryFabricModelsConfig `json:"models"`
}

type GetMemoryFabricConfigRsp struct {
	Config MemoryFabricConfig `json:"config"`
}

type SetMemoryFabricConfigRsp struct {
	Config    MemoryFabricConfig `json:"config"`
	Changed   bool               `json:"changed"`
	Restarted bool               `json:"restarted"`
}
