package kimi

import (
	"encoding/json"
	"testing"
)

func TestProviderType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		got      ProviderType
		expected string
	}{
		{"Kimi", ProviderTypeKimi, "kimi"},
		{"OpenAILegacy", ProviderTypeOpenAILegacy, "openai_legacy"},
		{"OpenAIResponses", ProviderTypeOpenAIResponses, "openai_responses"},
		{"Anthropic", ProviderTypeAnthropic, "anthropic"},
		{"GoogleGenAI", ProviderTypeGoogleGenAI, "google_genai"},
		{"Gemini", ProviderTypeGemini, "gemini"},
		{"VertexAI", ProviderTypeVertexAI, "vertexai"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.got)
			}
		})
	}
}

func TestModelCapability_Constants(t *testing.T) {
	tests := []struct {
		name     string
		got      ModelCapability
		expected string
	}{
		{"ImageIn", ModelCapabilityImageIn, "image_in"},
		{"VideoIn", ModelCapabilityVideoIn, "video_in"},
		{"Thinking", ModelCapabilityThinking, "thinking"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.got)
			}
		})
	}
}

func TestLLMProvider_JSONRoundTrip(t *testing.T) {
	original := LLMProvider{
		Type:    ProviderTypeKimi,
		BaseURL: "https://api.moonshot.cn",
		APIKey:  "test-key",
		Env: map[string]string{
			"ENV_VAR": "value",
		},
		CustomHeaders: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed LLMProvider
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Type != original.Type {
		t.Errorf("Type mismatch: expected %s, got %s", original.Type, parsed.Type)
	}
	if parsed.BaseURL != original.BaseURL {
		t.Errorf("BaseURL mismatch: expected %s, got %s", original.BaseURL, parsed.BaseURL)
	}
	if parsed.APIKey != original.APIKey {
		t.Errorf("APIKey mismatch: expected %s, got %s", original.APIKey, parsed.APIKey)
	}
	if parsed.Env["ENV_VAR"] != "value" {
		t.Errorf("Env mismatch: expected value, got %s", parsed.Env["ENV_VAR"])
	}
	if parsed.CustomHeaders["X-Custom-Header"] != "custom-value" {
		t.Errorf("CustomHeaders mismatch")
	}
}

func TestLLMModel_JSONRoundTrip(t *testing.T) {
	original := LLMModel{
		Provider:       "kimi",
		Model:          "moonshot-v1-8k",
		MaxContextSize: 8192,
		Capabilities: map[ModelCapability]bool{
			ModelCapabilityImageIn:  true,
			ModelCapabilityThinking: true,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed LLMModel
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Provider != original.Provider {
		t.Errorf("Provider mismatch: expected %s, got %s", original.Provider, parsed.Provider)
	}
	if parsed.Model != original.Model {
		t.Errorf("Model mismatch: expected %s, got %s", original.Model, parsed.Model)
	}
	if parsed.MaxContextSize != original.MaxContextSize {
		t.Errorf("MaxContextSize mismatch: expected %d, got %d", original.MaxContextSize, parsed.MaxContextSize)
	}
	if !parsed.Capabilities[ModelCapabilityImageIn] {
		t.Errorf("Capabilities[image_in] should be true")
	}
	if !parsed.Capabilities[ModelCapabilityThinking] {
		t.Errorf("Capabilities[thinking] should be true")
	}
}

func TestConfig_JSONRoundTrip(t *testing.T) {
	original := Config{
		DefaultModel: "moonshot-v1-8k",
		Models: map[string]LLMModel{
			"default": {
				Provider:       "kimi",
				Model:          "moonshot-v1-8k",
				MaxContextSize: 8192,
			},
		},
		Providers: map[string]LLMProvider{
			"kimi": {
				Type:    ProviderTypeKimi,
				BaseURL: "https://api.moonshot.cn",
			},
		},
		LoopControl: LoopControl{
			MaxStepsPerRun:    10,
			MaxRetriesPerStep: 3,
		},
		Services: Services{
			MoonshotSearch: &MoonshotSearchConfig{
				BaseURL: "https://search.moonshot.cn",
			},
		},
		MCP: MCPConfig{
			Client: MCPClientConfig{
				ToolCallTimeoutMS: 30000,
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed Config
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.DefaultModel != original.DefaultModel {
		t.Errorf("DefaultModel mismatch")
	}
	if len(parsed.Models) != 1 {
		t.Errorf("Models count mismatch")
	}
	if len(parsed.Providers) != 1 {
		t.Errorf("Providers count mismatch")
	}
	if parsed.LoopControl.MaxStepsPerRun != 10 {
		t.Errorf("LoopControl.MaxStepsPerRun mismatch")
	}
	if parsed.Services.MoonshotSearch == nil {
		t.Errorf("Services.MoonshotSearch should not be nil")
	}
	if parsed.MCP.Client.ToolCallTimeoutMS != 30000 {
		t.Errorf("MCP.Client.ToolCallTimeoutMS mismatch")
	}
}

func TestConfig_EmptyFields(t *testing.T) {
	// Test that empty/nil fields serialize correctly
	original := Config{
		DefaultModel: "test",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed Config
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.DefaultModel != "test" {
		t.Errorf("DefaultModel mismatch")
	}
	if len(parsed.Models) != 0 {
		t.Errorf("Models should be empty")
	}
	if len(parsed.Providers) != 0 {
		t.Errorf("Providers should be empty")
	}
}
