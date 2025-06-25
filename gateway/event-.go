package gateway

import (
	"encoding/json"
	"qord/discord"
)

// In modern discord docs - otherwise known as generic gateway event.
//
// https://discord.com/developers/docs/events/gateway#gateway-events
type EventPacket struct {
	Opcode   Opcode          `json:"op"`
	Sequence uint32          `json:"s,omitempty"`
	Event    string          `json:"t,omitempty"`
	Data     json.RawMessage `json:"d"`
}

// https://discord.com/developers/docs/events/gateway-events#hello
type HelloEvent struct {
	HeartbeatInterval float64 `json:"heartbeat_interval"`
}

// https://discord.com/developers/docs/events/gateway-events#ready
type ReadyEvent struct {
	User             discord.User               `json:"user"`
	Version          uint8                      `json:"v"` // Version of Discord API version
	SessionID        string                     `json:"session_id"`
	ResumeGatewayURL string                     `json:"resume_gateway_url"`
	Guilds           []discord.UnavailableGuild `json:"guilds"`
	// + shard order, same like on identify payload
	// + partial application object (docs)
}
