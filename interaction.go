package tempest

type InteractionType uint8

const (
	PING_TYPE InteractionType = iota + 1
	APPLICATION_COMMAND_TYPE
	MESSAGE_COMPONENT_TYPE
	APPLICATION_COMMAND_AUTO_COMPLETE_TYPE
	MODAL_SUBMIT_TYPE
)

type ResponseType uint8

const (
	PONG_RESPONSE ResponseType = iota + 1
	ACKNOWLEDGE_RESPONSE
	CHANNEL_MESSAGE_RESPONSE
	CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE
	DEFERRED_CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE
	DEFERRED_UPDATE_MESSAGE_RESPONSE // Only valid for component-based interactions.
	UPDATE_MESSAGE_RESPONSE          // Only valid for component-based interactions.
	AUTOCOMPLETE_RESPONSE
	MODAL_RESPONSE // Not available for MODAL_SUBMIT and PING interactions.
)

type Interaction struct {
	ID              Snowflake        `json:"id"`
	ApplicationID   Snowflake        `json:"application_id"`
	Type            InteractionType  `json:"type"`
	Data            *InteractionData `json:"data,omitempty"`
	GuildID         Snowflake        `json:"guild_id,omitempty"`
	ChannelID       Snowflake        `json:"channel_id,omitempty"`
	Member          *Member          `json:"member,omitempty"`
	User            *User            `json:"user,omitempty"`
	Token           string           `json:"token"`                  // Continuation token for responding to the interaction. It's not the same as bot/app token!
	Version         uint8            `json:"version"`                // Read-only property, always = 1.
	Message         *Message         `json:"message,omitempty"`      // For components, the message they were attached to.
	PermissionFlags uint64           `json:"app_permissions,string"` // Bitwise set of permissions the app or bot has within the channel the interaction was sent from.
	Locale          string           `json:"locale,omitempty"`       // Selected language of the invoking user.
	GuildLocale     string           `json:"guild_locale,omitempty"` // Guild's preferred locale, available if invoked in a guild.

	Client *Client `json:"-"` // Client pointer is required for all "higher" structs methods that inherits Interaction data.
}

type InteractionData struct {
	ID            Snowflake                `json:"id,omitempty"`
	CustomID      string                   `json:"custom_id,omitempty"` // Present only for components.
	Name          string                   `json:"name"`                // In case of commands it'll be name of command. If that's a subcommand then it'll be in format "rootCommand@subCommand".
	Type          CommandType              `json:"type"`
	Options       []*InteractionOption     `json:"options,omitempty"`
	Values        []string                 `json:"values,omitempty"`
	GuildID       Snowflake                `json:"guild_id,omitempty"`
	TargetID      Snowflake                `json:"target_id,omitempty"` // ID of either user or message targeted. Depends whether it was user command or message command.
	ComponentType ComponentType            `json:"component_type,omitempty"`
	Resolved      *InteractionDataResolved `json:"resolved,omitempty"`
}

type InteractionDataResolved struct {
	Users       map[Snowflake]*User           `json:"users,omitempty"`
	Members     map[Snowflake]*Member         `json:"members,omitempty"`
	Roles       map[Snowflake]*Role           `json:"roles,omitempty"`
	Channels    map[Snowflake]*PartialChannel `json:"channels,omitempty"`
	Attachments map[Snowflake]*Attachment     `json:"attachments,omitempty"`
}

type InteractionOption struct {
	Name    string               `json:"name"`
	Value   any                  `json:"value,omitempty"`
	Type    OptionType           `json:"type"`
	Options []*InteractionOption `json:"options,omitempty"`
	Focused bool                 `json:"focused,omitempty"` // Will be set to "true" if this option is the currently focused option for autocomplete.
}

// Similar to Message struct but used only for replying on interactions (mostly commands).
type Response struct {
	Type ResponseType  `json:"type"`
	Data *ResponseData `json:"data,omitempty"`
}

// Similar to Message struct - check: https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-response-object-messages
type ResponseData struct {
	TTS        bool         `json:"tts,omitempty"`
	Content    string       `json:"content,omitempty"`
	Embeds     []*Embed     `json:"embeds,omitempty"`
	Components []*Component `json:"components,omitempty"`
	Flags      uint64       `json:"flags,omitempty"`

	// Skipped never used fields from serialization.
}

// Unique to auto complete interaction.
type ResponseChoice struct {
	Type ResponseType       `json:"type"`
	Data ResponseChoiceData `json:"data,omitempty"`
}

// Unique to auto complete interaction.
type ResponseChoiceData struct {
	Choices []Choice `json:"choices,omitempty"`
}

type CommandInteraction Interaction
type AutoCompleteInteraction Interaction
