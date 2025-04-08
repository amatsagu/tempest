package command

import (
	tempest "github.com/amatsagu/tempest"
)

type AvatarSlashCommand struct{}

func (cmd AvatarSlashCommand) Data() tempest.Command {
	return tempest.Command{
		Type: tempest.USER_COMMAND_TYPE,
		Name: "avatar",
	}
}

func (cmd AvatarSlashCommand) AutoCompleteHandler(itx *tempest.CommandInteraction) []tempest.Choice {
	return nil
}

func (cmd AvatarSlashCommand) CommandHandler(itx tempest.CommandInteraction) {
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
}
