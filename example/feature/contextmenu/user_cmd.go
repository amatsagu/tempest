package contextmenu

import (
	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

type userInfoCommand struct{}

func newUserInfoCommand() *userInfoCommand { return &userInfoCommand{} }

func (c *userInfoCommand) Definition() tempest.Command {
	return tempest.Command{
		Type:                tempest.USER_COMMAND_TYPE,
		Name:                "User info",
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

// handle reads the resolved target user.
func (c *userInfoCommand) handle(itx *tempest.CommandInteraction) error {
	target := itx.Data.TargetID
	user := itx.Data.Resolved.ResolveUser(target)

	card := util.Embed(util.EffectiveName(user), "@"+user.Username)
	card.Fields = []tempest.EmbedField{
		util.EmbedField("ID", user.ID.String(), true),
		util.EmbedField("Bot", yesNo(user.Bot), true),
	}

	return itx.SendReply(tempest.ResponseMessageData{Embeds: []tempest.Embed{card}}, true, nil)
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
