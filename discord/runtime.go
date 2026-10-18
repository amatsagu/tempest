package discord

import (
	"context"
	"net/http"
)

// Runtime is the host runtime an interaction can reach directly, for operations
// beyond answering it: raw REST calls and common message and user helpers. A
// runtime installs one before the handler runs and clears it afterwards.
type Runtime interface {
	// Request performs one raw REST call and returns the response body.
	Request(ctx context.Context, method, path string, payload any, auditReason string) ([]byte, error)
	// RequestHeaders is Request with the response headers, for header-driven
	// endpoints such as pagination.
	RequestHeaders(ctx context.Context, method, path string, payload any, auditReason string) ([]byte, http.Header, error)
	// RequestWithFiles is Request with a multipart body.
	RequestWithFiles(ctx context.Context, method, path string, payload any, files []File, auditReason string) ([]byte, error)
	// SendMessage posts a message to a channel, with files when given.
	SendMessage(ctx context.Context, channelID ChannelID, data ResponseMessageData, files ...File) (*Message, error)
	// FetchUser returns a user by ID.
	FetchUser(ctx context.Context, id UserID) (User, error)
	// FetchMember returns a guild member by user ID.
	FetchMember(ctx context.Context, guildID GuildID, userID UserID) (Member, error)
}

// SetRuntime installs the runtime reachable through Runtime; it is cleared after
// the handler so nothing can be sent once the interaction is over.
func (i *Interaction) SetRuntime(r Runtime) { i.runtime = r }

// Runtime returns the host runtime for operations beyond answering the
// interaction, or nil when the interaction is not running under a runtime.
func (i *Interaction) Runtime() Runtime { return i.runtime }
