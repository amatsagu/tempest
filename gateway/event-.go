package gateway

import (
	"encoding/json"
	"qord/discord"
)

type EventName string

const (
	READY_EVENT   EventName = "READY"
	RESUMED_EVENT EventName = "RESUMED"
)

// In modern discord docs - otherwise known as generic gateway event.
//
// https://discord.com/developers/docs/events/gateway#gateway-events
type EventPacket struct {
	Opcode   Opcode          `json:"op"`
	Sequence uint32          `json:"s,omitempty"`
	Event    EventName       `json:"t,omitempty"`
	Data     json.RawMessage `json:"d"`
}

// https://discord.com/developers/docs/events/gateway-events#hello
type HelloEventData struct {
	HeartbeatInterval float64 `json:"heartbeat_interval"`
}

// https://discord.com/developers/docs/events/gateway-events#heartbeat
type HeartbeatEvent struct {
	Opcode   Opcode `json:"op"`
	Sequence uint32 `json:"d"`
}

// https://discord.com/developers/docs/events/gateway-events#identify
type IdentifyEvent struct {
	Opcode Opcode              `json:"op"`
	Data   IdentifyPayloadData `json:"d"`
}

// https://discord.com/developers/docs/events/gateway-events#identify-identify-structure
type IdentifyPayloadData struct {
	Token          string                        `json:"token"`
	Intents        uint32                        `json:"intents"`
	ShardOrder     [2]uint16                     `json:"shard"`           // [currentID, maxCount]
	LargeThreshold uint8                         `json:"large_threshold"` // 50 - 250
	Properties     IdentifyPayloadDataProperties `json:"properties"`
}

// https://discord.com/developers/docs/events/gateway-events#identify-identify-connection-properties
type IdentifyPayloadDataProperties struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

// https://discord.com/developers/docs/events/gateway-events#ready
type ReadyEventData struct {
	User             discord.User               `json:"user"`
	Version          uint8                      `json:"v"` // Version of Discord API version
	SessionID        string                     `json:"session_id"`
	ResumeGatewayURL string                     `json:"resume_gateway_url"`
	Guilds           []discord.UnavailableGuild `json:"guilds"`
	// + shard order, same like on identify payload
	// + partial application object (docs)
}
