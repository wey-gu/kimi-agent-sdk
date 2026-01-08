//go:build ignore
// +build ignore

// mock_kimi is a mock implementation of the kimi CLI for testing purposes.
// Build: go build -o mock_kimi mock_kimi.go
// Usage: ./mock_kimi --wire

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
)

var requestID atomic.Uint64

type Payload struct {
	Version string          `json:"jsonrpc"`
	ID      string          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

type PromptParams struct {
	UserInput json.RawMessage `json:"user_input"`
}

func main() {
	// Check for --wire flag
	hasWire := false
	for _, arg := range os.Args[1:] {
		if arg == "--wire" {
			hasWire = true
		}
	}
	if !hasWire {
		fmt.Fprintln(os.Stderr, "missing --wire flag")
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		var req Payload
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}

		switch req.Method {
		case "prompt":
			handlePrompt(encoder, req.ID)
		case "cancel":
			handleCancel(encoder, req.ID)
		}
	}
}

func handlePrompt(encoder *json.Encoder, reqID string) {
	// Send TurnBegin event
	sendEvent(encoder, "TurnBegin", map[string]any{
		"user_input": "test",
	})

	// Send StepBegin event
	sendEvent(encoder, "StepBegin", map[string]any{
		"n": 1,
	})

	// Send ContentPart event
	sendEvent(encoder, "ContentPart", map[string]any{
		"type": "text",
		"text": "Hello from mock kimi!",
	})

	// Send StatusUpdate event
	sendEvent(encoder, "StatusUpdate", map[string]any{
		"token_usage": map[string]any{
			"input_other":          100,
			"output":               50,
			"input_cache_read":     10,
			"input_cache_creation": 5,
		},
	})

	// Send prompt response
	encoder.Encode(Payload{
		Version: "2.0",
		ID:      reqID,
		Result:  json.RawMessage(`{"status":"finished","steps":1}`),
	})
}

func handleCancel(encoder *json.Encoder, reqID string) {
	encoder.Encode(Payload{
		Version: "2.0",
		ID:      reqID,
		Result:  json.RawMessage(`{}`),
	})
}

func sendEvent(encoder *json.Encoder, eventType string, payload any) {
	payloadJSON, _ := json.Marshal(payload)
	paramsJSON, _ := json.Marshal(map[string]any{
		"type":    eventType,
		"payload": json.RawMessage(payloadJSON),
	})

	id := requestID.Add(1)
	encoder.Encode(Payload{
		Version: "2.0",
		ID:      fmt.Sprintf("evt-%d", id),
		Method:  "event",
		Params:  paramsJSON,
	})
}
