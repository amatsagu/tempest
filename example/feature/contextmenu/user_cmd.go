package contextmenu

import (
	"context"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

type userInfoCommand struct{}

func newUserInfoCommand() *userInfoCommand { return &userInfoCommand{} }

func (c *userInfoCommand) Definition() tempest.Command {
	return tempest.Command{
		Type:                discord.UserCommandType,
		Name:                "User info",
		SlashCommandHandler: c.handle,
	}
}

func (c *userInfoCommand) handle(ctx context.Context, itx *tempest.CommandInteraction) error {
	target := discord.UserID{Snowflake: itx.Data.TargetID}
	user := itx.Data.Resolved.ResolveUser(target)

	card := util.Embed(util.EffectiveName(user), "@"+user.Username)
	card.Fields = []discord.EmbedField{
		util.EmbedField("ID", user.ID.String(), true),
		util.EmbedField("Bot", yesNo(user.Bot), true),
	}

	return itx.SendReply(ctx, discord.ResponseMessageData{Embeds: []discord.Embed{card}}, true, nil)
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
