package transport

import (
	"github.com/MoonshotAI/kimi-agent-sdk/go/wire"
)

//go:generate go tool defc generate -T Transport -o transport_impl.go
//go:generate go tool mockgen -source=transport.go -destination=transport_mock.go -package=transport
type Transport interface {
	Prompt(params *wire.PromptParams) (*wire.PromptResult, error)
	Cancel(params *wire.CancelParams) (*wire.CancelResult, error)
	Event(event *wire.EventParams) (*wire.EventResult, error)
	Request(request *wire.RequestParams) (*wire.RequestResult, error)
}
