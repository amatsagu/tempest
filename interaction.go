package tempest

import "github.com/amatsagu/tempest/discord"

type interactionContext struct {
	Client      *Client                              `json:"-"`
	session     *httpSession                         `json:"-"`
	params      [maxInlinePatternParams]patternParam `json:"-"`
	extraParams []patternParam                       `json:"-"`
	paramCount  uint8                                `json:"-"`
	responded   bool                                 `json:"-"`
	deferred    bool                                 `json:"-"`
}

// Slash or context-menu application command interaction.
type CommandInteraction struct {
	interactionContext
	discord.CommandInteraction
}

// Message component interaction (button, select menu, etc.) with extracted custom ID pattern values.
type ComponentInteraction struct {
	interactionContext
	discord.ComponentInteraction
}

// Submitted modal dialog interaction with extracted custom ID pattern values.
type ModalInteraction struct {
	interactionContext
	discord.ModalInteraction
}

// Unified interaction response payload serialized inline over HTTP or via REST callback.
type Response struct {
	Data  any                  `json:"data,omitempty"`
	Files []File               `json:"-"`
	Type  discord.ResponseType `json:"type"`
}
