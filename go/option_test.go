package kimi

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestWithExecutable(t *testing.T) {
	args := []string{"kimi"}
	opt := WithExecutable("/usr/local/bin/kimi")
	result := opt(args)

	if result != nil {
		t.Fatalf("expected nil return, got %v", result)
	}
	if args[0] != "/usr/local/bin/kimi" {
		t.Fatalf("expected executable to be replaced, got %s", args[0])
	}
}

func TestWithConfig(t *testing.T) {
	cfg := &Config{
		DefaultModel: "test-model",
		Models: map[string]LLMModel{
			"test-model": {
				Provider:       "kimi",
				Model:          "test-model",
				MaxContextSize: 8192,
			},
		},
		Providers: map[string]LLMProvider{
			"kimi": {
				Type:    ProviderTypeKimi,
				BaseURL: "https://api.moonshot.cn",
			},
		},
	}

	opt := WithConfig(cfg)
	result := opt(nil)

	if len(result) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(result))
	}
	if result[0] != "--config" {
		t.Fatalf("expected --config flag, got %s", result[0])
	}

	// Verify JSON is valid
	var parsed Config
	if err := json.Unmarshal([]byte(result[1]), &parsed); err != nil {
		t.Fatalf("failed to parse JSON config: %v", err)
	}
	if parsed.DefaultModel != "test-model" {
		t.Fatalf("expected default_model=test-model, got %s", parsed.DefaultModel)
	}
}

func TestWithConfigFile(t *testing.T) {
	opt := WithConfigFile("/path/to/config.toml")
	result := opt(nil)

	expected := []string{"--config-file", "/path/to/config.toml"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestWithModel(t *testing.T) {
	opt := WithModel("moonshot-v1-8k")
	result := opt(nil)

	expected := []string{"--model", "moonshot-v1-8k"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestWithWorkDir(t *testing.T) {
	opt := WithWorkDir("/tmp/workspace")
	result := opt(nil)

	expected := []string{"--work-dir", "/tmp/workspace"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestWithSession(t *testing.T) {
	opt := WithSession("session-123")
	result := opt(nil)

	expected := []string{"--session", "session-123"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestWithMCPConfigFile(t *testing.T) {
	opt := WithMCPConfigFile("/path/to/mcp.json")
	result := opt(nil)

	expected := []string{"--mcp-config-file", "/path/to/mcp.json"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestWithMCPConfig(t *testing.T) {
	cfg := &MCPConfig{
		Client: MCPClientConfig{
			ToolCallTimeoutMS: 30000,
		},
	}

	opt := WithMCPConfig(cfg)
	result := opt(nil)

	if len(result) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(result))
	}
	if result[0] != "--mcp-config" {
		t.Fatalf("expected --mcp-config flag, got %s", result[0])
	}

	// Verify JSON is valid
	var parsed MCPConfig
	if err := json.Unmarshal([]byte(result[1]), &parsed); err != nil {
		t.Fatalf("failed to parse JSON MCP config: %v", err)
	}
	if parsed.Client.ToolCallTimeoutMS != 30000 {
		t.Fatalf("expected tool_call_timeout_ms=30000, got %d", parsed.Client.ToolCallTimeoutMS)
	}
}

func TestWithAutoApprove(t *testing.T) {
	opt := WithAutoApprove()
	result := opt(nil)

	expected := []string{"--auto-approve"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestWithThinking_True(t *testing.T) {
	opt := WithThinking(true)
	result := opt(nil)

	expected := []string{"--thinking"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestWithThinking_False(t *testing.T) {
	opt := WithThinking(false)
	result := opt(nil)

	expected := []string{"--no-thinking"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestWithSkillsDir(t *testing.T) {
	opt := WithSkillsDir("/path/to/skills")
	result := opt(nil)

	expected := []string{"--skills-dir", "/path/to/skills"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestOptions_Chaining(t *testing.T) {
	options := []Option{
		WithExecutable("/custom/kimi"),
		WithModel("moonshot-v1"),
		WithWorkDir("/tmp"),
		WithAutoApprove(),
		WithThinking(true),
	}

	args := []string{"kimi"}
	for _, opt := range options {
		args = append(args, opt(args)...)
	}

	expected := []string{
		"/custom/kimi",
		"--model", "moonshot-v1",
		"--work-dir", "/tmp",
		"--auto-approve",
		"--thinking",
	}

	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("expected %v, got %v", expected, args)
	}
}
