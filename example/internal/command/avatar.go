package command

import (
	tempest "github.com/Amatsagu/Tempest"
)

var Avatar tempest.Command = tempest.Command{
	Type: tempest.USER_COMMAND_TYPE,
	Name: "avatar",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		user := itx.ResolveUser(itx.Data.TargetID)

		avatar := user.AvatarURL()
		itx.SendReply(tempest.ResponseMessageData{
			Embeds: []tempest.Embed{
				{
					Title: user.Username + "'s avatar",
					URL:   avatar,
					Image: &tempest.EmbedImage{
						URL: avatar,
					},
				},
			},
		}, false, nil)
	},
}
