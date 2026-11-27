package tempest

import (
	"context"

	"github.com/amatsagu/tempest/discord"
)

type interactionContext struct {
	session    *httpSession
	ctx        context.Context // Lifecycle/request context for interaction-driven REST work.
	HTTPClient *HTTPClient     `json:"-"` // Provided when handling HTTP interactions.
	BaseClient *BaseClient     `json:"-"` // Always provided.
	responded  bool
	deferred   bool
}

// A CommandInteraction represents an interaction received from a user invoking an application command, such as a slash command or a context menu command.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type CommandInteraction struct {
	interactionContext
	discord.CommandInteraction
}

// A ComponentInteraction represents an interaction received from a user interacting with a message component,
// such as a [discord.ButtonComponent] or [discord.SelectComponent].
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type ComponentInteraction struct {
	interactionContext
	discord.ComponentInteraction
}

// A ModalInteraction represents an interaction received from a user submitting a modal.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type ModalInteraction struct {
	interactionContext
	discord.ModalInteraction
}

// Unified response type for all interaction replies.
// Data field holds one of three response data types (message/modal/auto-complete).
// We're using "any" type because those types are not compatible with each other and will never be used at the same time.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-response-object
type Response struct {
	Data  any                  `json:"data,omitempty"`
	Files []File               `json:"-"`
	Type  discord.ResponseType `json:"type"`
}
