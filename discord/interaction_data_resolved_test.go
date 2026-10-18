package discord

import "testing"

// TestCommandInteractionDataOptionFindsNested proves Option/StringOption reach
// through a subcommand and a subcommand group to find the user's argument.
func TestCommandInteractionDataOptionFindsNested(t *testing.T) {
	cases := []struct {
		name string
		data CommandInteractionData
	}{
		{
			name: "subcommand",
			data: CommandInteractionData{
				Name: "tag",
				Options: []CommandInteractionOption{
					{Name: "add", Type: OptionSubCommand, Options: []CommandInteractionOption{
						{Name: "name", Type: OptionString, Value: "hi"},
					}},
				},
			},
		},
		{
			name: "subcommand group",
			data: CommandInteractionData{
				Name: "tag",
				Options: []CommandInteractionOption{
					{Name: "group", Type: OptionSubCommandGroup, Options: []CommandInteractionOption{
						{Name: "add", Type: OptionSubCommand, Options: []CommandInteractionOption{
							{Name: "name", Type: OptionString, Value: "hi"},
						}},
					}},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opt, ok := tc.data.Option("name")
			if !ok || opt.Name != "name" {
				t.Fatalf("Option(name) = %+v, %v; want the nested option", opt, ok)
			}

			value, ok := tc.data.StringOption("name")
			if !ok || value != "hi" {
				t.Fatalf("StringOption(name) = %q, %v; want hi", value, ok)
			}
		})
	}
}

// TestCommandInteractionDataSubCommand pins the subcommand path and options the
// router flattens registered subcommands with.
func TestCommandInteractionDataSubCommand(t *testing.T) {
	flat := CommandInteractionData{
		Name: "tag",
		Options: []CommandInteractionOption{
			{Name: "add", Type: OptionSubCommand, Options: []CommandInteractionOption{
				{Name: "name", Type: OptionString, Value: "hi"},
			}},
		},
	}

	path, options, ok := flat.SubCommand()
	if !ok || path != "add" || len(options) != 1 || options[0].Name != "name" {
		t.Fatalf("SubCommand() = %q, %+v, %v; want add with the name option", path, options, ok)
	}

	group := CommandInteractionData{
		Name: "tag",
		Options: []CommandInteractionOption{
			{Name: "group", Type: OptionSubCommandGroup, Options: []CommandInteractionOption{
				{Name: "sub", Type: OptionSubCommand, Options: []CommandInteractionOption{
					{Name: "name", Type: OptionString, Value: "hi"},
				}},
			}},
		},
	}

	path, options, ok = group.SubCommand()
	if !ok || path != "group sub" || len(options) != 1 || options[0].Name != "name" {
		t.Fatalf("SubCommand() = %q, %+v, %v; want group sub with the name option", path, options, ok)
	}

	normal := CommandInteractionData{
		Name: "tag",
		Options: []CommandInteractionOption{
			{Name: "reason", Type: OptionString, Value: "hi"},
		},
	}

	if path, options, ok := normal.SubCommand(); ok || path != "" || options != nil {
		t.Fatalf("SubCommand() on a plain option = %q, %+v, %v; want false", path, options, ok)
	}
}

// TestCommandInteractionDataFocused proves Focused reaches the option nested
// under a subcommand that the user is currently typing.
func TestCommandInteractionDataFocused(t *testing.T) {
	data := CommandInteractionData{
		Name: "tag",
		Options: []CommandInteractionOption{
			{Name: "add", Type: OptionSubCommand, Options: []CommandInteractionOption{
				{Name: "name", Type: OptionString, Value: "h", Focused: true},
			}},
		},
	}

	name, value, ok := data.Focused()
	if !ok || name != "name" || value != "h" {
		t.Fatalf("Focused() = %q, %v, %v; want name/h", name, value, ok)
	}

	if name, value, ok := (CommandInteractionData{}).Focused(); ok || name != "" || value != nil {
		t.Fatalf("Focused() on empty data = %q, %v, %v; want false", name, value, ok)
	}
}

// TestInteractionDataResolvedAccessors covers the present/absent contract and
// that Member binds the matching resolved User onto the member.
func TestInteractionDataResolvedAccessors(t *testing.T) {
	resolved := &InteractionDataResolved{
		Users:       map[UserID]User{{Snowflake: 1}: {ID: UserID{Snowflake: 1}, Username: "u"}},
		Members:     map[UserID]Member{{Snowflake: 1}: {Nick: "nick"}},
		Roles:       map[RoleID]Role{{Snowflake: 2}: {ID: RoleID{Snowflake: 2}, Name: "r"}},
		Channels:    map[ChannelID]Channel{{Snowflake: 3}: {ID: ChannelID{Snowflake: 3}, Name: "c"}},
		Messages:    map[MessageID]Message{{Snowflake: 4}: {ID: MessageID{Snowflake: 4}, Content: "m"}},
		Attachments: map[AttachmentID]Attachment{{Snowflake: 5}: {ID: AttachmentID{Snowflake: 5}, Filename: "f"}},
	}

	user, ok := resolved.User(UserID{Snowflake: 1})
	if !ok || user.Username != "u" {
		t.Fatalf("User = %+v, %v; want u", user, ok)
	}

	member, ok := resolved.Member(UserID{Snowflake: 1})
	if !ok || member.Nick != "nick" || member.User == nil || member.User.ID != (UserID{Snowflake: 1}) {
		t.Fatalf("Member = %+v, %v; want nick with its user bound", member, ok)
	}

	role, ok := resolved.Role(RoleID{Snowflake: 2})
	if !ok || role.Name != "r" {
		t.Fatalf("Role = %+v, %v; want r", role, ok)
	}

	channel, ok := resolved.Channel(ChannelID{Snowflake: 3})
	if !ok || channel.Name != "c" {
		t.Fatalf("Channel = %+v, %v; want c", channel, ok)
	}

	message, ok := resolved.Message(MessageID{Snowflake: 4})
	if !ok || message.Content != "m" {
		t.Fatalf("Message = %+v, %v; want m", message, ok)
	}

	attachment, ok := resolved.Attachment(AttachmentID{Snowflake: 5})
	if !ok || attachment.Filename != "f" {
		t.Fatalf("Attachment = %+v, %v; want f", attachment, ok)
	}

	if _, ok := resolved.User(UserID{Snowflake: 99}); ok {
		t.Error("User(absent) = ok true, want false")
	}

	if _, ok := resolved.Member(UserID{Snowflake: 99}); ok {
		t.Error("Member(absent) = ok true, want false")
	}

	if _, ok := resolved.Role(RoleID{Snowflake: 99}); ok {
		t.Error("Role(absent) = ok true, want false")
	}

	if _, ok := resolved.Channel(ChannelID{Snowflake: 99}); ok {
		t.Error("Channel(absent) = ok true, want false")
	}

	if _, ok := resolved.Message(MessageID{Snowflake: 99}); ok {
		t.Error("Message(absent) = ok true, want false")
	}

	if _, ok := resolved.Attachment(AttachmentID{Snowflake: 99}); ok {
		t.Error("Attachment(absent) = ok true, want false")
	}
}

// TestInteractionDataResolvedMemberWithoutUser proves a member entry whose user is
// missing is returned without a zero-valued user bound onto it.
func TestInteractionDataResolvedMemberWithoutUser(t *testing.T) {
	resolved := &InteractionDataResolved{
		Members: map[UserID]Member{{Snowflake: 1}: {Nick: "nick"}},
	}

	member, ok := resolved.Member(UserID{Snowflake: 1})
	if !ok {
		t.Fatal("Member = ok false, want true")
	}

	if member.Nick != "nick" {
		t.Fatalf("Member.Nick = %q, want nick", member.Nick)
	}

	if member.User != nil {
		t.Fatalf("Member.User = %+v, want nil when the resolved user is absent", member.User)
	}
}

// TestInteractionDataResolvedNilAccessors proves a nil resolved block answers
// every accessor with ok false instead of panicking.
func TestInteractionDataResolvedNilAccessors(t *testing.T) {
	var resolved *InteractionDataResolved

	if _, ok := resolved.User(UserID{Snowflake: 1}); ok {
		t.Error("User on nil receiver = ok true, want false")
	}

	if _, ok := resolved.Member(UserID{Snowflake: 1}); ok {
		t.Error("Member on nil receiver = ok true, want false")
	}

	if _, ok := resolved.Role(RoleID{Snowflake: 1}); ok {
		t.Error("Role on nil receiver = ok true, want false")
	}

	if _, ok := resolved.Channel(ChannelID{Snowflake: 1}); ok {
		t.Error("Channel on nil receiver = ok true, want false")
	}

	if _, ok := resolved.Message(MessageID{Snowflake: 1}); ok {
		t.Error("Message on nil receiver = ok true, want false")
	}

	if _, ok := resolved.Attachment(AttachmentID{Snowflake: 1}); ok {
		t.Error("Attachment on nil receiver = ok true, want false")
	}
}
