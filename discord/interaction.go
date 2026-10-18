package discord

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
	responder                    Responder
	runtime                      Runtime
	Token                        string                 `json:"token"`
	Locale                       Locale                 `json:"locale,omitzero"`
	GuildLocale                  Locale                 `json:"guild_locale,omitzero"`
	Entitlements                 []Entitlement          `json:"entitlements,omitzero"`
	Channel                      Channel                `json:"channel,omitzero"`
	ID                           Snowflake              `json:"id"`
	ApplicationID                AppID                  `json:"application_id"`
	GuildID                      GuildID                `json:"guild_id,omitzero"`
	ChannelID                    ChannelID              `json:"channel_id,omitzero"`
	AttachmentSizeLimit          uint64                 `json:"attachment_size_limit,omitzero"`
	AppPermissions               Permissions            `json:"app_permissions,string,omitzero"`
	Context                      InteractionContextType `json:"context,omitzero"`
	ShardID                      uint16                 `json:"-"`
	Type                         InteractionType        `json:"type"`
	Version                      uint8                  `json:"version,omitzero"`
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type CommandInteraction struct {
	Data CommandInteractionData `json:"data"`
	Interaction
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type ComponentInteraction struct {
	Data ComponentInteractionData `json:"data"`
	Interaction
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type ModalInteraction struct {
	Data ModalInteractionData `json:"data"`
	Interaction
}
