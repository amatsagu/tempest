package tempest

import (
	"encoding/json"
)

type EventName string

const (
	READY_EVENT               EventName = "READY"
	RESUMED_EVENT             EventName = "RESUMED"
	INTERACTION_CREATE_EVENT  EventName = "INTERACTION_CREATE"
	MESSAGE_CREATE_EVENT      EventName = "MESSAGE_CREATE"
	MESSAGE_UPDATE_EVENT      EventName = "MESSAGE_UPDATE"
	MESSAGE_DELETE_EVENT      EventName = "MESSAGE_DELETE"
	MESSAGE_DELETE_BULK_EVENT EventName = "MESSAGE_DELETE_BULK"
	GUILD_CREATE_EVENT        EventName = "GUILD_CREATE"
	GUILD_DELETE_EVENT        EventName = "GUILD_DELETE"
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
	HeartbeatInterval uint32 `json:"heartbeat_interval"`
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
	User             User               `json:"user"`
	Version          uint8              `json:"v"` // Version of Discord API version
	SessionID        string             `json:"session_id"`
	ResumeGatewayURL string             `json:"resume_gateway_url"`
	Guilds           []UnavailableGuild `json:"guilds"`
	// + shard order, same like on identify payload
	// + partial application object (docs)
}

// https://discord.com/developers/docs/events/gateway-events#resume
type ResumeEvent struct {
	Opcode Opcode          `json:"op"`
	Data   ResumeEventData `json:"d"`
}

// https://discord.com/developers/docs/events/gateway-events#resume-resume-structure
type ResumeEventData struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Sequence  uint32 `json:"seq"` // Last sequence number received
}

// https://discord.com/developers/docs/events/gateway#get-gateway-bot
type GatewayBot struct {
	URL               string            `json:"url"`
	ShardCount        uint16            `json:"shards"`
	SessionStartLimit SessionStartLimit `json:"session_start_limit"`
}

// https://discord.com/developers/docs/events/gateway#session-start-limit-object
type SessionStartLimit struct {
	ResetAfter     uint32 `json:"reset_after"`
	Total          uint16 `json:"total"`           // max 1000
	Remaining      uint16 `json:"remaining"`       // max 1000
	MaxConcurrency uint16 `json:"max_concurrency"` // Number of identify requests allowed per 5 seconds.
}

type StatusType string

const (
	ONLINE_STATUS    StatusType = "online"
	DND_STATUS       StatusType = "dnd"
	IDLE_STATUS      StatusType = "idle"
	INVISIBLE_STATUS StatusType = "invisible"
	OFFLINE_STATUS   StatusType = "offline"
)

// https://discord.com/developers/docs/events/gateway-events#activity-object-activity-types
type ActivityType uint8

const (
	PLAYING_ACTIVITY_TYPE   ActivityType = 0
	STREAMING_ACTIVITY_TYPE ActivityType = 1
	LISTENING_ACTIVITY_TYPE ActivityType = 2
	WATCHING_ACTIVITY_TYPE  ActivityType = 3
	CUSTOM_ACTIVITY_TYPE    ActivityType = 4
	COMPETING_ACTIVITY_TYPE ActivityType = 5
)

// https://discord.com/developers/docs/events/gateway-events#activity-object
//
// Activity only in context of Discord Bot Presence via Gateway.
type Activity struct {
	Name string       `json:"name"`
	Type ActivityType `json:"type"`
	URL  string       `json:"url,omitempty"` // Stream URL, only for Streaming type
}

// https://discord.com/developers/docs/events/gateway-events#update-presence
type UpdatePresenceEvent struct {
	Opcode Opcode                  `json:"op"`
	Data   UpdatePresenceEventData `json:"d"`
}

// https://discord.com/developers/docs/events/gateway-events#update-presence-gateway-presence-update-structure
type UpdatePresenceEventData struct {
	Since      *int64     `json:"since"` // Unix time (in milliseconds) of when the client went idle, or null if the client is not idle
	Activities []Activity `json:"activities"`
	Emoji      Emoji      `json:"emoji"`
	AFK        bool       `json:"afk"`
}
