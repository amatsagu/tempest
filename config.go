// Package tempest is a low-level Discord application runtime that supports HTTP-only,
// gateway, or both modes.
package tempest

import (
	"encoding/json/jsontext"
	"time"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/gateway"
)

// Config configures a Client. A zero field means the documented default, and fields
// are ordered to minimize struct padding.
type Config struct {
	// Logf is the debug log sink. When nil, logging is disabled.
	Logf func(format string, args ...any)

	// Presence is the initial gateway presence. It is optional.
	Presence *gateway.Presence

	// OnRawEvent receives every gateway dispatch event verbatim, before parsing.
	// Session opcodes are handled internally and never reach it.
	OnRawEvent func(shard int, event string, data jsontext.Value)

	// OnRawInteraction fires for every interaction except PING once the route is
	// known and before the handler runs; return false to drop it unacknowledged.
	OnRawInteraction func(info InteractionInfo) bool

	// OnCompletedInteraction fires after a registry handler runs to completion.
	OnCompletedInteraction func(info InteractionInfo)

	// OnInteractionError is called once per handler that returns an error. itx is
	// the typed interaction pointer, so the hook can answer through Reply, Update,
	// or Defer. With it nil and Debug on, a default hook logs the error and replies
	// generically; with both nil the error is dropped, so set this in production.
	OnInteractionError func(err error, info InteractionInfo, itx any)

	// Token is the bot token, used for the REST API and the gateway.
	Token string

	// PublicKey is the hex-encoded Ed25519 public key Discord signs HTTP
	// interactions with. Required for HTTP applications.
	PublicKey string

	// HTTPAddr, when non-empty, starts the HTTP server for the interaction
	// endpoint, for example "127.0.0.1:8080".
	HTTPAddr string

	// DefaultContexts is applied to commands synced without their own Contexts.
	DefaultContexts []discord.InteractionContextType

	// TimestampWindow rejects a signed interaction whose timestamp is older than
	// this, or in the future by more than this. Zero disables the check, which is
	// the default because Discord retries deliveries and a late retry carries the
	// original timestamp; set it to a few minutes to bound replay.
	TimestampWindow time.Duration

	// ApplicationID is optional; when zero it is derived from Token.
	ApplicationID discord.AppID

	// MaxBodyBytes caps the incoming HTTP interaction body, enforced with
	// http.MaxBytesReader. Zero uses a 1 MiB default.
	MaxBodyBytes int64

	// MaxIdleConnsPerHost tunes the REST connection pool per Discord host. Zero
	// uses a fixed default; nothing here is inferred from the host.
	MaxIdleConnsPerHost int

	// ShardCount forces a gateway shard count. Zero means ask Discord.
	ShardCount int

	// Intents is the gateway intent bitfield.
	Intents uint32

	// Compression enables optional gateway payload compression.
	Compression gateway.Compression

	// Debug enables debug logging; when false the log sites are guarded so
	// nothing is formatted or allocated.
	Debug bool

	// Gateway enables a gateway connection.
	Gateway bool
}

const (
	defaultMaxBodyBytes        = 1 << 20 // 1 MiB
	defaultMaxIdleConnsPerHost = 32
)

// resolved holds the effective settings derived from Config.
type resolved struct {
	MaxBodyBytes     int64
	IdleConnsPerHost int
	ShardCount       int
}

// resolve fills in the config defaults. It only reads Config and has no side
// effects: nothing here touches the Go runtime or the host. Host-adaptive sizing is
// opt-in through TuneRuntime.
func (cfg Config) resolve() resolved {
	body := cfg.MaxBodyBytes
	if body <= 0 {
		body = defaultMaxBodyBytes
	}

	idle := cfg.MaxIdleConnsPerHost
	if idle <= 0 {
		idle = defaultMaxIdleConnsPerHost
	}

	return resolved{
		MaxBodyBytes:     body,
		IdleConnsPerHost: idle,
		ShardCount:       cfg.ShardCount,
	}
}
