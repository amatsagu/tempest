package commands

import tempest "github.com/Amatsagu/Tempest"

var Avatar tempest.Command = tempest.Command{
	Name:        "avatar",
	Description: "Sends member's avatar!",
	Options: []tempest.Option{
		{
			Name:        "user",
			Description: "User to take avatar from.",
			Type:        tempest.OPTION_USER,
			Required:    true,
		},
	},
	SlashCommandHandler: func(itx tempest.CommandInteraction) {
		raw, _ := itx.GetOptionValue("user")

		user, err := itx.Client.FetchUser(tempest.StringToSnowflake(raw.(string)))
		if err != nil {
			itx.SendLinearReply(err.Error(), false) // Received id may potentially be fake (be a non existing snowflake).
		}

		avatar := user.FetchAvatarUrl()
		itx.SendReply(tempest.ResponseData{
			Embeds: []*tempest.Embed{
				{
					Title: user.Tag() + " avatar",
					Url:   avatar,
					Image: &tempest.EmbedImage{
						Url: avatar,
					},
				},
			},
		}, false)
	},
}
