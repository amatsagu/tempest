# Core Setup

## Clients
1. `HTTPClient`: Webhook-based. AWS Lambda/Vercel compatible.
2. `GatewayClient`: Persistent socket. Real-time events, presence.

### BaseClientOptions
```go
type BaseClientOptions struct {
	Token                      string
	DefaultInteractionContexts []InteractionContextType
	PreCommandHook             func(cmd Command, itx *CommandInteraction) bool
	PostCommandHook            func(cmd Command, itx *CommandInteraction)
	ComponentHandler           func(itx *ComponentInteraction)
	ModalHandler               func(itx *ModalInteraction)
	Logger                     *log.Logger
}
```

## HTTP Client
```go
type HTTPClientOptions struct {
	BaseClientOptions
	PublicKey string
	Trace     bool
}
func NewHTTPClient(opt HTTPClientOptions) *HTTPClient
```

## Gateway Client
```go
type GatewayClientOptions struct {
	BaseClientOptions
	Trace              bool
	CustomEventHandler func(shardID uint16, packet EventPacket)
}
func NewGatewayClient(opt GatewayClientOptions) *GatewayClient
```

## Logging
- Tracing via internal `tracef`.
- Custom `*log.Logger` in `BaseClientOptions`.
- Fallback: `Trace: true` + nil `Logger` -> `os.Stdout`.
