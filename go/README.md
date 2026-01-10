# Kimi Agent SDK for Go

Go SDK for programmatically controlling Kimi Agent sessions via the [kimi-cli](https://github.com/MoonshotAI/kimi-cli).

## Installation

```bash
go get github.com/MoonshotAI/kimi-agent-sdk/go
```

## Prerequisites

- `kimi` CLI installed and available in PATH
- `KIMI_BASE_URL`, `KIMI_API_KEY`, `KIMI_MODEL_NAME` environment variables set, or use `kimi.Option` instead

## Usage

```go
package main

import (
    "context"
    "fmt"

    kimi "github.com/MoonshotAI/kimi-agent-sdk/go"
    "github.com/MoonshotAI/kimi-agent-sdk/go/wire"
)

func main() {
    session, err := kimi.NewSession(
        kimi.WithBaseURL("https://api.moonshot.cn/v1"),
        kimi.WithAPIKey("your-api-key"),
        kimi.WithModel("kimi-latest"),
    )
    if err != nil {
        panic(err)
    }
    defer session.Close()

    turn, err := session.Prompt(context.Background(), wire.NewStringUserInput("Hello!"))
    if err != nil {
        panic(err)
    }

    for step := range turn.Steps {
        for msg := range step.Messages {
            if cp, ok := msg.(wire.ContentPart); ok && cp.Type == wire.ContentPartTypeText {
                fmt.Print(cp.Text)
            }
        }
    }

    // Check for errors that occurred during streaming
    if err := turn.Err(); err != nil {
        panic(err)
    }
}
```

## Turn Methods

After consuming all messages from a turn, you can inspect the turn's final state:

- `turn.Err()` - Returns any error that occurred during streaming
- `turn.Result()` - Returns the `wire.PromptResult` containing the final status
- `turn.Usage()` - Returns token usage information (`Context` and `Tokens`)

## Responding to Requests

For `wire.Request` messages (e.g., `ApprovalRequest`), you **must** call `Respond()`. Failing to do so will block the session indefinitely.

```go
for step := range turn.Steps {
    for msg := range step.Messages {
        if req, ok := msg.(wire.ApprovalRequest); ok {
            // Approve the request
            req.Respond(wire.RequestResponseApprove)
            // Or reject: req.Respond(wire.RequestResponseReject)
        }
    }
}
```

## Important Notes

1. **Sequential Prompts**: Call `Prompt` sequentially. Wait for the previous turn to complete before starting a new one.

2. **Resource Cleanup**: Always use `defer session.Close()` to ensure proper cleanup.

3. **Consume All Messages**: You must consume all messages from `step.Messages` and all steps from `turn.Steps` before starting a new Prompt.

4. **Cancellation**: You can cancel a turn either by canceling the context or by calling `turn.Cancel()` explicitly.
