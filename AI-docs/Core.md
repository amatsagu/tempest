# Core Setup

## Clients
### BaseClient
```go
type BaseClient struct {
	ApplicationID Snowflake
	Rest          *Rest
}
```

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
type HTTPClient struct {
	*BaseClient
	PublicKey ed25519.PublicKey
}

type HTTPClientOptions struct {
	BaseClientOptions
	PublicKey string
	Trace     bool
}

func NewHTTPClient(opt HTTPClientOptions) *HTTPClient
func (client *HTTPClient) DiscordRequestHandler(w http.ResponseWriter, r *http.Request)
```

## Gateway Client
```go
type GatewayClient struct {
	*BaseClient
	Gateway *ShardManager
}

type GatewayClientOptions struct {
	BaseClientOptions
	Trace              bool
	CustomEventHandler func(shardID uint16, packet EventPacket)
}

func NewGatewayClient(opt GatewayClientOptions) *GatewayClient
```

## ShardManager
```go
func NewShardManager(token string, trace bool, eventHandler func(shardID uint16, packet EventPacket), logger *log.Logger) *ShardManager
func (m *ShardManager) Start(ctx context.Context, intents uint32, forcedShardCount uint16, readyCallback func()) error
func (m *ShardManager) Stop()
func (m *ShardManager) Status() map[uint16]ShardState
```

## Logging
- Tracing via internal `tracef`.
- Custom `*log.Logger` in `BaseClientOptions`.
- Fallback: `Trace: true` + nil/Discard `Logger` -> `os.Stdout`.
