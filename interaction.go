package tempest

import "net/http"

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-interaction-type
type InteractionType uint8

const (
	PING_INTERACTION_TYPE InteractionType = iota + 1
	APPLICATION_COMMAND_INTERACTION_TYPE
	MESSAGE_COMPONENT_INTERACTION_TYPE
	APPLICATION_COMMAND_AUTO_COMPLETE_INTERACTION_TYPE
	MODAL_SUBMIT_INTERACTION_TYPE
)

// Used only for partial JSON parsing.
type InteractionTypeExtractor struct {
	Type InteractionType `json:"type"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object
type CommandInteraction struct {
	ID              Snowflake              `json:"id"`
	ApplicationID   Snowflake              `json:"application_id"`
	Type            InteractionType        `json:"type"`
	Data            CommandInteractionData `json:"data"`
	GuildID         Snowflake              `json:"guild_id,omitempty"`
	ChannelID       Snowflake              `json:"channel_id,omitempty"`
	Member          *Member                `json:"member,omitempty"`
	User            *User                  `json:"user,omitempty"`
	Token           string                 `json:"token"`                  // Temporary token used for responding to the interaction. It's not the same as bot/app token.
	Version         uint8                  `json:"version"`                // Read-only property, always = 1.
	PermissionFlags uint64                 `json:"app_permissions,string"` // Bitwise set of permissions the app or bot has within the channel the interaction was sent from.
	Locale          string                 `json:"locale,omitempty"`       // Selected language of the invoking user.
	GuildLocale     string                 `json:"guild_locale,omitempty"` // Guild's preferred locale, available if invoked in a guild.

	Client *Client `json:"-"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object
type ComponentInteraction struct {
	ID              Snowflake                `json:"id"`
	ApplicationID   Snowflake                `json:"application_id"`
	Type            InteractionType          `json:"type"`
	Data            ComponentInteractionData `json:"data"`
	GuildID         Snowflake                `json:"guild_id,omitempty"`
	ChannelID       Snowflake                `json:"channel_id,omitempty"`
	Member          *Member                  `json:"member,omitempty"`
	User            *User                    `json:"user,omitempty"`
	Token           string                   `json:"token"`   // Temporary token used for responding to the interaction. It's not the same as bot/app token.
	Version         uint8                    `json:"version"` // Read-only property, always = 1.
	Message         Message                  `json:"message"`
	PermissionFlags uint64                   `json:"app_permissions,string"` // Bitwise set of permissions the app or bot has within the channel the interaction was sent from.
	Locale          string                   `json:"locale,omitempty"`       // Selected language of the invoking user.
	GuildLocale     string                   `json:"guild_locale,omitempty"` // Guild's preferred locale, available if invoked in a guild.

	Client *Client             `json:"-"`
	w      http.ResponseWriter `json:"-"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object
type ModalInteraction struct {
	ID              Snowflake            `json:"id"`
	ApplicationID   Snowflake            `json:"application_id"`
	Type            InteractionType      `json:"type"`
	Data            ModalInteractionData `json:"data"`
	GuildID         Snowflake            `json:"guild_id,omitempty"`
	ChannelID       Snowflake            `json:"channel_id,omitempty"`
	Member          *Member              `json:"member,omitempty"`
	User            *User                `json:"user,omitempty"`
	Token           string               `json:"token"`                  // Temporary token used for responding to the interaction. It's not the same as bot/app token.
	Version         uint8                `json:"version"`                // Read-only property, always = 1.
	PermissionFlags uint64               `json:"app_permissions,string"` // Bitwise set of permissions the app or bot has within the channel the interaction was sent from.
	Locale          string               `json:"locale,omitempty"`       // Selected language of the invoking user.
	GuildLocale     string               `json:"guild_locale,omitempty"` // Guild's preferred locale, available if invoked in a guild.

	Client *Client             `json:"-"`
	w      http.ResponseWriter `json:"-"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-application-command-data-structure
type CommandInteractionData struct {
	ID       Snowflake                   `json:"id,omitempty"`
	Name     string                      `json:"name"`
	Type     CommandType                 `json:"type"`
	Resolved *InteractionDataResolved    `json:"resolved,omitempty"`
	Options  []*CommandInteractionOption `json:"options,omitempty"`
	GuildID  Snowflake                   `json:"guild_id,omitempty"`
	TargetID Snowflake                   `json:"target_id,omitempty"` // ID of either user or message targeted. Depends whether it was user command or message command.
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-application-command-interaction-data-option-structure
type CommandInteractionOption struct {
	Name    string                      `json:"name"`
	Type    OptionType                  `json:"type"`
	Value   any                         `json:"value,omitempty"` // string, float64 (double or integer) or bool
	Options []*CommandInteractionOption `json:"options,omitempty"`
	Focused bool                        `json:"focused,omitempty"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-resolved-data-structure
type InteractionDataResolved struct {
	Users    map[Snowflake]*User           `json:"users,omitempty"`
	Members  map[Snowflake]*Member         `json:"members,omitempty"`
	Roles    map[Snowflake]*Role           `json:"roles,omitempty"`
	Channels map[Snowflake]*PartialChannel `json:"channels,omitempty"`
}

// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-option-choice-structure
type Choice struct {
	Name              string            `json:"name"`
	NameLocalizations map[string]string `json:"name_localizations,omitempty"` // https://discord.com/developers/docs/reference#locales
	Value             any               `json:"value"`                        // string, float64 (double or integer) or bool
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-message-component-data-structure
type ComponentInteractionData struct {
	CustomID string              `json:"custom_id"`
	Type     ComponentType       `json:"component_type"`
	Values   []*SelectMenuOption `json:"values,omitempty"`
}

type ModalInteractionData struct {
	CustomID   string         `json:"custom_id"`
	Components []ComponentRow `json:"components"`
}
