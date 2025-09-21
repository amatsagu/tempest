package command

import "github.com/amatsagu/tempest/api"

// This command will only appear when right clicking on user name or avatar in Discord.

var Avatar api.Command = api.Command{
	Type: api.USER_COMMAND_TYPE,
	Name: "avatar",
	SlashCommandHandler: func(itx *api.CommandInteraction) {
		user := itx.ResolveUser(itx.Data.TargetID)

		avatar := user.AvatarURL()
		itx.SendReply(api.ResponseMessageData{
			Embeds: []api.Embed{
				{
					Title: user.Username + "'s avatar",
					URL:   avatar,
					Image: &api.EmbedImage{
						URL: avatar,
					},
				},
			},
		}, false, nil)
	},
}
