package gateway

import "encoding/json/jsontext"

// Opcode is a gateway opcode.
// https://docs.discord.com/developers/events/gateway#gateway-opcodes
type Opcode uint8

const (
	OpDispatch                Opcode = 0
	OpHeartbeat               Opcode = 1
	OpIdentify                Opcode = 2
	OpPresenceUpdate          Opcode = 3
	OpVoiceStateUpdate        Opcode = 4
	OpResume                  Opcode = 6
	OpReconnect               Opcode = 7
	OpRequestGuildMembers     Opcode = 8
	OpInvalidSession          Opcode = 9
	OpHello                   Opcode = 10
	OpHeartbeatACK            Opcode = 11
	OpRequestSoundboardSounds Opcode = 31
	OpRequestChannelInfo      Opcode = 43
)

// Intent is a gateway intent bitflag.
// https://docs.discord.com/developers/events/gateway#gateway-intents
type Intent uint32

const (
	IntentGuilds                 Intent = 1 << 0
	IntentGuildMembers           Intent = 1 << 1
	IntentGuildModeration        Intent = 1 << 2
	IntentGuildEmojisAndStickers Intent = 1 << 3
	IntentGuildIntegrations      Intent = 1 << 4
	IntentGuildWebhooks          Intent = 1 << 5
	IntentGuildInvites           Intent = 1 << 6
	IntentGuildVoiceStates       Intent = 1 << 7
	IntentGuildPresences         Intent = 1 << 8
	IntentGuildMessages          Intent = 1 << 9
	IntentGuildMessageReactions  Intent = 1 << 10
	IntentGuildMessageTyping     Intent = 1 << 11
	IntentDirectMessages         Intent = 1 << 12
	IntentDirectMessageReactions Intent = 1 << 13
	IntentDirectMessageTyping    Intent = 1 << 14
	IntentMessageContent         Intent = 1 << 15
	IntentGuildScheduledEvents   Intent = 1 << 16
	IntentAutoModerationConfig   Intent = 1 << 20
	IntentAutoModerationExec     Intent = 1 << 21
	IntentGuildMessagePolls      Intent = 1 << 24
	IntentDirectMessagePolls     Intent = 1 << 25
)

// CloseCode is a gateway close code.
// https://docs.discord.com/developers/events/gateway#gateway-close-event-codes
type CloseCode int

const (
	CloseUnknownError         CloseCode = 4000
	CloseUnknownOpcode        CloseCode = 4001
	CloseDecodeError          CloseCode = 4002
	CloseNotAuthenticated     CloseCode = 4003
	CloseAuthenticationFailed CloseCode = 4004
	CloseAlreadyAuthenticated CloseCode = 4005
	CloseInvalidSeq           CloseCode = 4007
	CloseRateLimited          CloseCode = 4008
	CloseSessionTimedOut      CloseCode = 4009
	CloseInvalidShard         CloseCode = 4010
	CloseShardingRequired     CloseCode = 4011
	CloseInvalidAPIVersion    CloseCode = 4012
	CloseInvalidIntents       CloseCode = 4013
	CloseDisallowedIntents    CloseCode = 4014
)

// Event names used for dispatch routing.
const (
	EventReady       = "READY"
	EventResumed     = "RESUMED"
	EventInteraction = "INTERACTION_CREATE"

	// Monetization events. They arrive as ordinary dispatch events, so subscribe
	// with Client.Event; no intent gates them.
	EventEntitlementCreate = "ENTITLEMENT_CREATE"
	EventEntitlementUpdate = "ENTITLEMENT_UPDATE"
	EventEntitlementDelete = "ENTITLEMENT_DELETE"

	EventSubscriptionCreate = "SUBSCRIPTION_CREATE"
	EventSubscriptionUpdate = "SUBSCRIPTION_UPDATE"
	EventSubscriptionDelete = "SUBSCRIPTION_DELETE"
)

// payload is the wire envelope for every gateway message.
type payload struct {
	Seq       *int64         `json:"s,omitzero"`
	EventName string         `json:"t,omitzero"`
	Data      jsontext.Value `json:"d,omitzero"`
	Op        Opcode         `json:"op"`
}

type helloData struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

type identifyData struct {
	Presence   *Presence     `json:"presence,omitzero"`
	Properties identifyProps `json:"properties"`
	Token      string        `json:"token"`
	Shard      [2]int        `json:"shard,omitzero"`
	Intents    uint32        `json:"intents"`
}

type identifyProps struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

// Status is a bot presence status.
// https://docs.discord.com/developers/events/gateway-events#update-presence-status-types
type Status string

const (
	StatusOnline    Status = "online"
	StatusIdle      Status = "idle"
	StatusDND       Status = "dnd"
	StatusInvisible Status = "invisible"
	StatusOffline   Status = "offline"
)

// ActivityType is the kind of activity shown in a presence.
// https://docs.discord.com/developers/events/gateway-events#activity-object-activity-types
type ActivityType uint8

const (
	ActivityPlaying ActivityType = iota
	ActivityStreaming
	ActivityListening
	ActivityWatching
	ActivityCustom
	ActivityCompeting
)

// Activity is one line shown in a bot's presence.
type Activity struct {
	Name  string       `json:"name"`
	State string       `json:"state,omitzero"`
	URL   string       `json:"url,omitzero"`
	Type  ActivityType `json:"type,omitzero"`
}

// Presence is the bot status and activities broadcast to Discord.
type Presence struct {
	Since      *int64     `json:"since,omitzero"`
	Status     Status     `json:"status,omitzero"`
	Activities []Activity `json:"activities,omitzero"`
	AFK        bool       `json:"afk,omitzero"`
}

type resumeData struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Seq       int64  `json:"seq"`
}

type readyData struct {
	SessionID        string           `json:"session_id"`
	ResumeGatewayURL string           `json:"resume_gateway_url"`
	Guilds           []jsontext.Value `json:"guilds,omitzero"`
	Shard            [2]int           `json:"shard,omitzero"`
}
