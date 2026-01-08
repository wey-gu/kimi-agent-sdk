package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	kimi "github.com/MoonshotAI/kimi-agent-sdk/go"
	"github.com/MoonshotAI/kimi-agent-sdk/go/wire"
)

func getMockKimiPath(t *testing.T) string {
	t.Helper()

	// Get the directory of this test file
	_, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// The mock_kimi binary should be in testdata
	mockPath := filepath.Join("testdata", "mock_kimi")
	if _, err := os.Stat(mockPath); os.IsNotExist(err) {
		t.Skipf("mock_kimi not found at %s, run 'go build -o testdata/mock_kimi testdata/mock_kimi.go' first", mockPath)
	}

	absPath, err := filepath.Abs(mockPath)
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}
	return absPath
}

func TestIntegration_NewSession_MockCLI(t *testing.T) {
	mockPath := getMockKimiPath(t)

	session, err := kimi.NewSession(
		kimi.WithExecutable(mockPath),
	)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer session.Close()
}

func TestIntegration_RoundTrip_SimpleMessage(t *testing.T) {
	mockPath := getMockKimiPath(t)

	session, err := kimi.NewSession(
		kimi.WithExecutable(mockPath),
	)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer session.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	turn, err := session.RoundTrip(ctx, wire.NewStringUserInput("test input"))
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	// Consume steps
	var messages []wire.Message
	for step := range turn.Steps {
		for msg := range step.Messages {
			messages = append(messages, msg)
		}
	}

	// Verify we received expected messages
	if len(messages) == 0 {
		t.Fatal("expected at least one message")
	}

	// Find ContentPart message
	foundContent := false
	for _, msg := range messages {
		if cp, ok := msg.(wire.ContentPart); ok {
			if cp.Type == wire.ContentPartTypeText && cp.Text == "Hello from mock kimi!" {
				foundContent = true
				break
			}
		}
	}
	if !foundContent {
		t.Error("expected to find ContentPart with 'Hello from mock kimi!'")
	}

	result := turn.Result()
	if result.Status != wire.PromptResultStatusFinished {
		t.Errorf("expected status finished, got %s", result.Status)
	}
}

func TestIntegration_Turn_Steps_Channel(t *testing.T) {
	mockPath := getMockKimiPath(t)

	session, err := kimi.NewSession(
		kimi.WithExecutable(mockPath),
	)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer session.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	turn, err := session.RoundTrip(ctx, wire.NewStringUserInput("test"))
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	// Verify Steps channel receives at least one step
	stepCount := 0
	for step := range turn.Steps {
		stepCount++
		// Drain messages
		for range step.Messages {
		}
	}

	if stepCount == 0 {
		t.Error("expected at least one step")
	}
}

func TestIntegration_StatusUpdate_Usage(t *testing.T) {
	mockPath := getMockKimiPath(t)

	session, err := kimi.NewSession(
		kimi.WithExecutable(mockPath),
	)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer session.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	turn, err := session.RoundTrip(ctx, wire.NewStringUserInput("test"))
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	// Consume all steps
	for step := range turn.Steps {
		for range step.Messages {
		}
	}

	// Check usage was updated
	usage := turn.Usage()
	if usage.Tokens.InputOther != 100 {
		t.Errorf("expected InputOther=100, got %d", usage.Tokens.InputOther)
	}
	if usage.Tokens.Output != 50 {
		t.Errorf("expected Output=50, got %d", usage.Tokens.Output)
	}
}

func TestIntegration_Session_Close(t *testing.T) {
	mockPath := getMockKimiPath(t)

	session, err := kimi.NewSession(
		kimi.WithExecutable(mockPath),
	)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	err = session.Close()
	if err != nil {
		t.Errorf("Close: %v", err)
	}
}
