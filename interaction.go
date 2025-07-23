package tempest

import (
	"encoding/json"
	"net/http"
)

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-interaction-type
type InteractionType uint8

const (
	PING_INTERACTION_TYPE InteractionType = iota + 1
	APPLICATION_COMMAND_INTERACTION_TYPE
	MESSAGE_COMPONENT_INTERACTION_TYPE
	APPLICATION_COMMAND_AUTO_COMPLETE_INTERACTION_TYPE
	MODAL_SUBMIT_INTERACTION_TYPE
)

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-interaction-context-types
type InteractionContextType uint16 // use uint16 instead uint8 to avoid Go's json marshal logic that thinks of it as symbols.

const (
	GUILD_CONTEXT_TYPE InteractionContextType = iota
	BOT_DM_CONTEXT_TYPE
	PRIVATE_CHANNEL_CONTEXT_TYPE
)

// Used only for partial JSON parsing.
type InteractionTypeExtractor struct {
	Type InteractionType `json:"type"`
}

// Represents general interaction.
// Use Command/Component/Modal interaction to read Data field.
//
// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-interaction-structure
type Interaction struct {
	// This struct is purposefuly lacking some large but hardly ever used fields like Message or (partial) Guild.
	// We find that there's no reason in having duplicate of data app/bot already knows or can easily receive & cache as it very slowly changes.
	// If you truly need them - apply changes and open pull request to discuss.

	ID            Snowflake       `json:"id"`
	ApplicationID Snowflake       `json:"application_id"`
	Type          InteractionType `json:"type"`
	Data          json.RawMessage `json:"data"`

	// partial guild struct is skipped

	GuildID Snowflake `json:"guild_id,omitempty"`

	// partial channel struct is skipped

	ChannelID Snowflake `json:"channel_id,omitempty"`
	Member    *Member   `json:"member,omitempty"`
	User      *User     `json:"user,omitempty"`
	Token     string    `json:"token"` // Temporary token used for responding to the interaction. It's not the same as bot token.

	// version is skipped (docs says it's always 1, read-only property)

	PermissionFlags PermissionFlags `json:"app_permissions,string"` // Bitwise set of permissions the app/bot has within the channel the interaction was sent from (guild text channel or DM channel).
	Locale          Language        `json:"locale,omitempty"`       // Selected language of the invoking user.
	GuildLocale     string          `json:"guild_locale,omitempty"` // Guild's preferred locale, available if invoked in a guild.
	Entitlements    []Entitlement   `json:"entitlements,omitzero"`  // For monetized apps, any entitlements for the invoking user, representing access to premium SKUs.

	// authorizing_integration_owners or contexts are pointless as they essentially duplicate data you already have :)
	// attachment_size_limit is also skipped - appears to have no use anywhere

	Client *Client `json:"-"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object
type CommandInteraction struct {
	*Interaction
	Data CommandInteractionData `json:"data"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object
type ComponentInteraction struct {
	*Interaction
	Data ComponentInteractionData `json:"data"`
	w    http.ResponseWriter      `json:"-"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object
type ModalInteraction struct {
	*Interaction
	Data ModalInteractionData `json:"data"`
	w    http.ResponseWriter  `json:"-"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-interaction-data
type CommandInteractionData struct {
	ID       Snowflake                  `json:"id"`
	Name     string                     `json:"name"`
	Type     CommandType                `json:"type"`
	Resolved *InteractionDataResolved   `json:"resolved,omitempty"`
	Options  []CommandInteractionOption `json:"options,omitzero"`
	GuildID  Snowflake                  `json:"guild_id,omitempty"`
	TargetID Snowflake                  `json:"target_id,omitempty"` // ID of either user or message targeted. Depends whether it was user command or message command.
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-application-command-interaction-data-option-structure
type CommandInteractionOption struct {
	Name    string                     `json:"name"`
	Type    OptionType                 `json:"type"`
	Value   any                        `json:"value,omitempty"` // string, float64 (double or integer) or bool
	Options []CommandInteractionOption `json:"options,omitzero"`
	Focused bool                       `json:"focused,omitempty"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-resolved-data-structure
type InteractionDataResolved struct {
	Users       map[Snowflake]User           `json:"users,omitzero"`
	Members     map[Snowflake]Member         `json:"members,omitzero"`
	Roles       map[Snowflake]Role           `json:"roles,omitzero"`
	Channels    map[Snowflake]PartialChannel `json:"channels,omitzero"`
	Messages    map[Snowflake]Message        `json:"messages,omitzero"`
	Attachments map[Snowflake]Attachment     `json:"attachments,omitzero"`
}

// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-option-choice-structure
type CommandOptionChoice struct {
	Name              string              `json:"name"`
	NameLocalizations map[Language]string `json:"name_localizations,omitzero"` // https://discord.com/developers/docs/reference#locales
	Value             any                 `json:"value"`                       // string, float64 (double or integer) or bool
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-message-component-data-structure
type ComponentInteractionData struct {
	CustomID string                   `json:"custom_id"`
	Type     ComponentType            `json:"component_type"`
	Values   []string                 `json:"values,omitzero"` // Values the user selected in a select menu component
	Resolved *InteractionDataResolved `json:"resolved,omitempty"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-modal-submit-data-structure
type ModalInteractionData struct {
	CustomID   string            `json:"custom_id"`
	Components []LayoutComponent `json:"components,omitzero"`
}
