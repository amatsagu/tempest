package gateway

import "github.com/coder/websocket"

// https://discord.com/developers/docs/topics/opcodes-and-status-codes#gateway-gateway-opcodes
type Opcode uint8

const (
	DISPATCH_OPCODE Opcode = iota
	HEARTBEAT_OPCODE
	IDENTIFY_OPCODE
	PRESENCE_UPDATE_OPCODE
	VOICE_STATUS_UPDATE_OPCODE
	_
	RESUME_OPCODE
	RECONNECT_OPCODE
	REQUEST_GUILD_MEMBERS_OPCODE
	INVALID_SESSION_OPCODE
	HELLO_OPCODE
	HEARTBEAT_ACK_OPCODE
	REQUEST_SOUNDBOARD_SOUNDS_OPCODE Opcode = 31
)

// https://discord.com/developers/docs/topics/opcodes-and-status-codes#gateway-gateway-close-event-codes
type ExitCode websocket.StatusCode

const (
	UNKNOWN_ERROR         ExitCode = 4000 + iota // We're not sure what went wrong. Try reconnecting?
	UNKNOWN_OPCODE                               // You sent an invalid Gateway opcode or an invalid payload for an opcode. Don't do that!
	DECODE_ERROR                                 // You sent an invalid payload to Discord. Don't do that!
	NOT_AUTHENTICATED                            // You sent a payload prior to identifying, or the session was invalidated.
	AUTHENTICATION_FAILED                        // The account token sent with your identify payload is incorrect.
	ALREADY_AUTHENTICATED                        // You sent more than one identify payload. Don't do that!
	_                                            //
	INVALID_SEQ                                  // The sequence sent when resuming the session was invalid. Reconnect and start a new session.
	RATE_LIMITED                                 // You're sending payloads too quickly. You'll be disconnected.
	SESSION_TIMED_OUT                            // Your session timed out. Reconnect and start a new one.
	INVALID_SHARD                                // You sent us an invalid shard when identifying.
	SHARDING_REQUIRED                            // You must shard your connection in order to connect.
	INVALID_API_VERSION                          // You sent an invalid version for the gateway.
	INVALID_INTENT                               // You sent an invalid intent for a Gateway Intent.
	DISALLOWED_INTENT                            // You sent a disallowed intent. Not enabled or not approved.
)
