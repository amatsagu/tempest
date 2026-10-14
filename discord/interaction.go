package discord

import "encoding/json/jsontext"

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-interaction-type
type InteractionType uint8

const (
	InteractionPing InteractionType = iota + 1
	InteractionApplicationCommand
	InteractionMessageComponent
	InteractionApplicationCommandAutocomplete
	InteractionModalSubmit
)

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-interaction-context-types
type InteractionContextType uint16

const (
	InteractionContextGuild InteractionContextType = iota
	InteractionContextBotDM
	InteractionContextPrivateChannel
)

// https://docs.discord.com/developers/resources/application#application-object-application-integration-types
type ApplicationIntegrationType uint8

const (
	ApplicationIntegrationGuildInstall ApplicationIntegrationType = iota
	ApplicationIntegrationUserInstall
)

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-interaction-structure
type Interaction struct {
	AuthorizingIntegrationOwners map[ApplicationIntegrationType]Snowflake `json:"authorizing_integration_owners,omitzero"`
	Member                       *Member                                  `json:"member,omitzero"`
	User                         *User                                    `json:"user,omitzero"`
	Message                      *Message                                 `json:"message,omitzero"`
	Token                        string                                   `json:"token"`
	Data                         jsontext.Value                           `json:"data,omitzero"`
	Entitlements                 []Entitlement                            `json:"entitlements,omitzero"`
	Channel                      Channel                                  `json:"channel,omitzero"`
	Locale                       Locale                                   `json:"locale,omitzero"`
	GuildLocale                  Locale                                   `json:"guild_locale,omitzero"`
	ID                           Snowflake                                `json:"id"`
	ApplicationID                AppID                                    `json:"application_id"`
	GuildID                      GuildID                                  `json:"guild_id,omitzero"`
	ChannelID                    ChannelID                                `json:"channel_id,omitzero"`
	AttachmentSizeLimit          uint64                                   `json:"attachment_size_limit,omitzero"`
	AppPermissions               Permissions                              `json:"app_permissions,string,omitzero"`
	Context                      InteractionContextType                   `json:"context,omitzero"`
	Type                         InteractionType                          `json:"type"`
	Version                      uint8                                    `json:"version,omitzero"`
	ShardID                      uint16                                   `json:"-"`
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type CommandInteraction struct {
	Interaction
	Data CommandInteractionData `json:"data"`
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type ComponentInteraction struct {
	Interaction
	Data ComponentInteractionData `json:"data"`
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type ModalInteraction struct {
	Interaction
	Data ModalInteractionData `json:"data"`
}
