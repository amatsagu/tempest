package commands

import tempest "github.com/Amatsagu/Tempest"

var Hello tempest.Command = tempest.Command{
	Name:        "hello",
	Description: "Replies with hello message!",
	Options: []tempest.Option{
		{
			Name:        "user",
			Description: "User to greet.",
			Type:        tempest.OPTION_USER,
			// Required:    true,
		},
	},
	SlashCommandHandler: func(itx tempest.CommandInteraction) {
		raw, available := itx.GetOptionValue("user")

		if available {
			user, err := itx.Client.FetchUser(tempest.StringToSnowflake(raw.(string)))
			if err != nil {
				itx.SendLinearReply(err.Error(), false)
			}

			itx.SendLinearReply("Hello "+user.Tag(), false)
		} else {
			itx.SendLinearReply("Hello "+itx.Member.User.Tag(), false)
		}
	},
}
