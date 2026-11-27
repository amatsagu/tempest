package discord

import "encoding/json/jsontext"

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-interaction-type
type InteractionType uint8

const (
	PingInteractionType InteractionType = iota + 1
	ApplicationCommandInteractionType
	MessageComponentInteractionType
	ApplicationCommandAutoCompleteInteractionType
	ModalSubmitInteractionType
)

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-interaction-context-types
type InteractionContextType uint16 // use uint16 instead uint8 to avoid Go's json marshal logic that thinks of it as symbols.

const (
	GuildContextType InteractionContextType = iota
	BotDMContextType
	PrivateChannelContextType
)

// Represents general interaction.
// Use Command/Component/Modal interaction to read Data field.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-interaction-structure
type Interaction struct {
	Member                       *Member                                  `json:"member,omitzero"`
	User                         *User                                    `json:"user,omitzero"`
	Locale                       Language                                 `json:"locale,omitzero"`       // Selected language of the invoking user.
	GuildLocale                  string                                   `json:"guild_locale,omitzero"` // Guild's preferred locale, available if invoked in a guild.
	Token                        string                                   `json:"token"`                 // Temporary token used for responding to the interaction. It's not the same as bot token.
	Data                         jsontext.Value                           `json:"data"`
	AuthorizingIntegrationOwners map[ApplicationIntegrationType]Snowflake `json:"authorizing_integration_owners,omitzero"` // Mapping of installation contexts that the interaction was authorized for to related user or guild IDs.
	Entitlements                 []Entitlement                            `json:"entitlements,omitzero"`                   // For monetized apps, any entitlements for the invoking user, representing access to premium SKUs.

	// partial guild struct is skipped

	GuildID GuildID `json:"guild_id,omitzero"`
	// This struct is purposefully lacking some large but hardly ever used fields like Message or (partial) Guild.
	// We find that there's no reason in having duplicate of data app/bot already knows or can easily receive & cache as it very slowly changes.
	// If you truly need them - apply changes and open pull request to discuss.

	ID Snowflake `json:"id"`

	// partial channel struct is skipped

	ChannelID ChannelID `json:"channel_id,omitzero"`

	// version is skipped (docs says it's always 1, read-only property)

	PermissionFlags     PermissionFlags        `json:"app_permissions,string"` // Bitwise set of permissions the app/bot has within the channel the interaction was sent from (guild text channel or DM channel).
	ApplicationID       AppID                  `json:"application_id"`
	AttachmentSizeLimit uint64                 `json:"attachment_size_limit,omitzero"` // Attachment size limit in bytes.
	Context             InteractionContextType `json:"context,omitzero"`               // Context where the interaction was triggered from.
	Type                InteractionType        `json:"type"`
}

// A CommandInteraction represents an interaction received from a user invoking an application command, such as a slash command or a context menu command.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type CommandInteraction struct {
	Data CommandInteractionData `json:"data"`
	Interaction
}

// A ComponentInteraction represents an interaction received from a user interacting with a message component,
// such as a [ButtonComponent] or [SelectComponent].
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type ComponentInteraction struct {
	Data ComponentInteractionData `json:"data"`
	Interaction
}

// A ModalInteraction represents an interaction received from a user submitting a modal.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object
type ModalInteraction struct {
	Data ModalInteractionData `json:"data"`
	Interaction
}

// A CommandInteractionData represents the data received from a user invoking an application command, such as a slash command or a context menu command.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-application-command-data-structure
type CommandInteractionData struct {
	Resolved *InteractionDataResolved   `json:"resolved,omitzero"`  // Data that Discord has resolved for the command, such as user and role objects. Fields will only be available to bots with the requisite permissions.
	Name     string                     `json:"name"`               // The name of the invoked command, not including subcommand names.
	Options  []CommandInteractionOption `json:"options,omitzero"`   // The options and values passed by the user.
	ID       CommandID                  `json:"id"`                 // The ID of the invoked command.
	GuildID  GuildID                    `json:"guild_id,omitzero"`  // The ID of the guild the command was invoked from. Will be null (i.e. 0) if invoked in a DM.
	TargetID Snowflake                  `json:"target_id,omitzero"` // The ID of the user or message targeted by a user or message command from the context menu. Will be empty for slash commands.
	Type     CommandType                `json:"type"`               // The type of comamnd that was invoked.
}

// A CommandInteractionOption represents data about a single option passed to a command.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-application-command-interaction-data-option-structure
type CommandInteractionOption struct {
	Value   any                        `json:"value,omitempty"`  // The value provided by the user; will always be of type string, float64 (double/integer) or bool. Mutually exclusive with options.
	Name    string                     `json:"name"`             // The name of the option.
	Options []CommandInteractionOption `json:"options,omitzero"` // For subcommands or grouped commands, the options passed to the subcommand or group. Mutually exclusive with value.
	Type    OptionType                 `json:"type"`             // The type of option that was provided.
	Focused bool                       `json:"focused,omitzero"` // Whether this option is currently focused by the user during autocomplete. Is exclusively used for autocomplete interactions and will always be false otherwise.
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-resolved-data-structure
type InteractionDataResolved struct {
	Users       map[UserID]User              `json:"users,omitzero"`
	Members     map[UserID]Member            `json:"members,omitzero"`
	Roles       map[RoleID]Role              `json:"roles,omitzero"`
	Channels    map[ChannelID]PartialChannel `json:"channels,omitzero"`
	Messages    map[MessageID]Message        `json:"messages,omitzero"`
	Attachments map[AttachmentID]Attachment  `json:"attachments,omitzero"`
}

// https://docs.discord.com/developers/interactions/application-commands#application-command-object-application-command-option-choice-structure
type CommandOptionChoice struct {
	Value             any                 `json:"value"`                       // string, float64 (double or integer) or bool
	NameLocalizations map[Language]string `json:"name_localizations,omitzero"` // https://docs.discord.com/developers/reference#locales
	Name              string              `json:"name"`
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-message-component-data-structure
type ComponentInteractionData struct {
	Resolved *InteractionDataResolved `json:"resolved,omitzero"`
	// The CustomID of the Component having been interacted with.
	CustomID string        `json:"custom_id"`
	Values   []string      `json:"values,omitzero"` // Values the user selected in a select menu component
	Type     ComponentType `json:"component_type"`  // The type of the component having been interacted with.
}

// ModalInteractionData represents the data received from a user submitting a modal.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-modal-submit-data-structure
type ModalInteractionData struct {
	Resolved *InteractionDataResolved `json:"resolved,omitzero"`
	// The CustomID of the modal having been submitted.
	CustomID   string           `json:"custom_id"`
	Components []ModalComponent `json:"components,omitzero"` // The components that were sent inside the modal, having been filled with user input.
}

// Returns user data either from member or from user (depending if interaction was used in a server).
func (itx *Interaction) BaseUser() *User {
	if itx.Member != nil && itx.Member.User != nil {
		return itx.Member.User
	}
	return itx.User
}

// Returns value of any type. Check second value to check whether option was provided or not (true if yes).
func (itx *CommandInteraction) GetOptionValue(name string) (any, bool) {
	options := itx.Data.Options
	if len(options) == 0 {
		return nil, false
	}

	for _, option := range options {
		if option.Name == name {
			return option.Value, true
		}
	}

	return nil, false
}

// Warning! This method is only for handling auto complete interaction which is a part of command logic.
// Returns option name and its value of triggered option. Option name is always of string type but you'll need to check type of value.
func (itx *CommandInteraction) GetFocusedValue() (string, any) {
	for _, option := range itx.Data.Options {
		if option.Focused {
			return option.Name, option.Value
		}
	}

	panic("auto complete interaction had no option with \"focused\" field. This error should never happen with correctly defined slash command")
}

// Returns pointer to user if present in resolved data. It'll return empty struct if there's no resolved user.
func (r *InteractionDataResolved) ResolveUser(id UserID) User {
	if r == nil {
		return User{}
	}
	return r.Users[id]
}

// Returns member if present in resolved data and binds member.User when resolved.
// It returns an empty struct if there's no resolved member.
func (r *InteractionDataResolved) ResolveMember(id UserID) Member {
	if r == nil {
		return Member{}
	}

	member, available := r.Members[id]
	if available {
		if user, ok := r.Users[id]; ok {
			member.User = &user
		}
		return member
	}

	return Member{}
}

// Returns pointer to guild role if present in resolved data. It'll return empty struct if there's no resolved role.
func (r *InteractionDataResolved) ResolveRole(id RoleID) (Role, bool) {
	if r == nil {
		return Role{}, false
	}
	role, ok := r.Roles[id]
	return role, ok
}

// Returns pointer to partial channel if present in resolved data. It'll return empty struct if there's no resolved partial channel.
func (r *InteractionDataResolved) ResolveChannel(id ChannelID) PartialChannel {
	if r == nil {
		return PartialChannel{}
	}
	return r.Channels[id]
}

// Returns pointer to message if present in resolved data. It'll return empty struct if there's no resolved message.
func (r *InteractionDataResolved) ResolveMessage(id MessageID) Message {
	if r == nil {
		return Message{}
	}
	return r.Messages[id]
}

// Returns pointer to attachment if present in resolved data. It'll return empty struct if there's no resolved attachment.
func (r *InteractionDataResolved) ResolveAttachment(id AttachmentID) Attachment {
	if r == nil {
		return Attachment{}
	}
	return r.Attachments[id]
}

func findLabelComponent[T LabelChildComponent](components []ModalComponent, customID string, match func(T) bool) (T, bool) {
	var zero T
	if customID == "" {
		return zero, false
	}

	for _, row := range components {
		if label, ok := row.(LabelComponent); ok {
			if cmp, ok := label.Component.(T); ok && match(cmp) {
				return cmp, true
			}
		}
	}

	return zero, false
}

// Retrieves the contents of the first [TextInputComponent] inside the modal (at any depth) with the given customID.
//
// If no such component exists, an empty string is returned instead.
func (itx *ModalInteraction) GetInputValue(customID string) string {
	if input, ok := findLabelComponent(itx.Data.Components, customID, func(c TextInputComponent) bool {
		return c.CustomID == customID
	}); ok {
		return input.Value
	}

	return ""
}

// Retrieves the uploaded file attachment IDs of the first [FileUploadComponent] inside the modal (at any depth) with the given customID.
//
// If no such component exists, nil is returned instead.
func (itx *ModalInteraction) GetFileUploadValues(customID string) []AttachmentID {
	if upload, ok := findLabelComponent(itx.Data.Components, customID, func(c FileUploadComponent) bool {
		return c.CustomID == customID
	}); ok {
		return upload.Values
	}

	return nil
}

// Retrieves the resolved [Attachment] objects of the first [FileUploadComponent] inside the modal (at any depth) with the given customID.
//
// If no such component exists or no attachments were resolved, nil is returned instead.
func (itx *ModalInteraction) GetFileUploadAttachments(customID string) []Attachment {
	ids := itx.GetFileUploadValues(customID)
	if len(ids) == 0 || itx.Data.Resolved == nil || len(itx.Data.Resolved.Attachments) == 0 {
		return nil
	}

	attachments := make([]Attachment, 0, len(ids))
	for _, id := range ids {
		if attachment, ok := itx.Data.Resolved.Attachments[id]; ok {
			attachments = append(attachments, attachment)
		}
	}

	if len(attachments) == 0 {
		return nil
	}

	return attachments
}

// Retrieves the selected value of the first [RadioGroupComponent] inside the modal (at any depth) with the given customID.
// The second return value indicates whether a value was selected.
func (itx *ModalInteraction) GetRadioGroupValue(customID string) (string, bool) {
	if radio, ok := findLabelComponent(itx.Data.Components, customID, func(c RadioGroupComponent) bool {
		return c.CustomID == customID
	}); ok && radio.Value != nil {
		return *radio.Value, true
	}

	return "", false
}

// Retrieves the selected values of the first [CheckboxGroupComponent] inside the modal (at any depth) with the given customID.
//
// If no such component exists, nil is returned instead.
func (itx *ModalInteraction) GetCheckboxGroupValues(customID string) []string {
	if group, ok := findLabelComponent(itx.Data.Components, customID, func(c CheckboxGroupComponent) bool {
		return c.CustomID == customID
	}); ok {
		return group.Values
	}

	return nil
}

// Retrieves the state of the first [CheckboxComponent] inside the modal (at any depth) with the given customID.
// The second return value indicates whether the component was found.
func (itx *ModalInteraction) GetCheckboxValue(customID string) (bool, bool) {
	if checkbox, ok := findLabelComponent(itx.Data.Components, customID, func(c CheckboxComponent) bool {
		return c.CustomID == customID
	}); ok {
		return checkbox.Value, true
	}

	return false, false
}

// Retrieves the selected values of the first [StringSelectComponent] inside the modal (at any depth) with the given customID.
//
// If no such component exists, nil is returned instead.
func (itx *ModalInteraction) GetStringSelectValues(customID string) []string {
	if selectMenu, ok := findLabelComponent(itx.Data.Components, customID, func(c StringSelectComponent) bool {
		return c.CustomID == customID
	}); ok {
		return selectMenu.Values
	}

	return nil
}

// Retrieves the selected snowflake IDs of the first [SelectComponent] (user, role, mentionable, or channel select) inside the modal (at any depth) with the given customID.
//
// If no such component exists, nil is returned instead.
func (itx *ModalInteraction) GetSelectValues(customID string) []Snowflake {
	if selectMenu, ok := findLabelComponent(itx.Data.Components, customID, func(c SelectComponent) bool {
		return c.CustomID == customID
	}); ok {
		return selectMenu.Values
	}

	return nil
}
