package tempest

import "encoding/json"

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-interaction-type
type InteractionType uint8

const (
	PING_INTERACTION_TYPE InteractionType = iota + 1
	APPLICATION_COMMAND_INTERACTION_TYPE
	MESSAGE_COMPONENT_INTERACTION_TYPE
	APPLICATION_COMMAND_AUTO_COMPLETE_INTERACTION_TYPE
	MODAL_SUBMIT_INTERACTION_TYPE
)

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-interaction-context-types
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
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-interaction-structure
type Interaction struct {
	// This struct is purposefully lacking some large but hardly ever used fields like Message or (partial) Guild.
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

	BaseClient    *BaseClient              `json:"-"` // Always provided.
	HTTPClient    *HTTPClient              `json:"-"` // Only provided if using HTTP Client.
	GatewayClient *GatewayClient           `json:"-"` // Only provided if using Gateway Client.
	ShardID       uint16                   `json:"-"` // Only provided if using Gateway Client. Shard ID = 0 is also a valid ID.
	responded     bool                     `json:"-"`
	deferred      bool                     `json:"-"`
	responder     func(res Response) error `json:"-"`
}

// A CommandInteraction represents an interaction received from a user invoking an application command, such as a slash command or a context menu command.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type CommandInteraction struct {
	*Interaction
	Data CommandInteractionData `json:"data"`
}

// A ComponentInteraction represents an interaction received from a user interacting with a message component,
// such as a [ButtonComponent] or [SelectComponent].
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type ComponentInteraction struct {
	*Interaction
	Data ComponentInteractionData `json:"data"`
}

// A ModalInteraction represents an interaction received from a user submitting a modal.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type ModalInteraction struct {
	*Interaction
	Data ModalInteractionData `json:"data"`
}

// A CommandInteractionData represents the data received from a user invoking an application command, such as a slash command or a context menu command.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-interaction-data
type CommandInteractionData struct {
	ID       Snowflake                  `json:"id"`                  // The ID of the invoked command.
	Name     string                     `json:"name"`                // The name of the invoked command, not including subcommand names.
	Type     CommandType                `json:"type"`                // The type of comamnd that was invoked.
	Resolved *InteractionDataResolved   `json:"resolved,omitempty"`  // Data that Discord has resolved for the command, such as user and role objects. Fields will only be available to bots with the requisite permissions.
	Options  []CommandInteractionOption `json:"options,omitzero"`    // The options and values passed by the user.
	GuildID  Snowflake                  `json:"guild_id,omitempty"`  // The ID of the guild the command was invoked from. Will be null (i.e. 0) if invoked in a DM.
	TargetID Snowflake                  `json:"target_id,omitempty"` // The ID of the user or message targeted by a user or message command from the context menu. Will be empty for slash commands.
}

// A CommandInteractionOption represents data about a single option passed to a command.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-application-command-interaction-data-option-structure
type CommandInteractionOption struct {
	Name    string                     `json:"name"`             // The name of the option.
	Type    OptionType                 `json:"type"`             // The type of option that was provided.
	Value   any                        `json:"value,omitempty"`  // The value provided by the user; will always be of type string, float64 (double/integer) or bool. Mutually exclusive with options.
	Options []CommandInteractionOption `json:"options,omitzero"` // For subcommands or grouped commands, the options passed to the subcommand or group. Mutually exclusive with value.
	Focused bool                       `json:"focused"`          // Whether this option is currently focused by the user during autocomplete. Is exclusively used for autocomplete interactions and will always be false otherwise.
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-resolved-data-structure
type InteractionDataResolved struct {
	Users       map[Snowflake]User           `json:"users,omitzero"`
	Members     map[Snowflake]Member         `json:"members,omitzero"`
	Roles       map[Snowflake]Role           `json:"roles,omitzero"`
	Channels    map[Snowflake]PartialChannel `json:"channels,omitzero"`
	Messages    map[Snowflake]Message        `json:"messages,omitzero"`
	Attachments map[Snowflake]Attachment     `json:"attachments,omitzero"`
}

// https://docs.discord.com/developers/interactions/application-commands#application-command-object-application-command-option-choice-structure
type CommandOptionChoice struct {
	Name              string              `json:"name"`
	NameLocalizations map[Language]string `json:"name_localizations,omitzero"` // https://docs.discord.com/developers/reference#locales
	Value             any                 `json:"value"`                       // string, float64 (double or integer) or bool
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-message-component-data-structure
type ComponentInteractionData struct {
	// The CustomID of the Component having been interacted with.
	CustomID string                   `json:"custom_id"`
	Type     ComponentType            `json:"component_type"`  // The type of the component having been interacted with.
	Values   []string                 `json:"values,omitzero"` // Values the user selected in a select menu component
	Resolved *InteractionDataResolved `json:"resolved,omitempty"`
}

// ModalInteractionData represents the data received from a user submitting a modal.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-modal-submit-data-structure
type ModalInteractionData struct {
	// The CustomID of the modal having been submitted.
	CustomID   string           `json:"custom_id"`
	Components []ModalComponent `json:"components,omitzero"` // The components that were sent inside the modal, having been filled with user input.
}

// Unified response type for all interaction replies.
// Data field holds one of three response data types (message/modal/auto-complete).
// We're using "any" type because those types are not compatible with each other and will never be used at the same time.
type Response struct {
	Type  ResponseType `json:"type"`
	Data  any          `json:"data,omitempty"`
	Files []File       `json:"-"`
}
