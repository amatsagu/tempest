package contextmenu

import (
	"context"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

type userInfoCommand struct{}

func newUserInfoCommand() *userInfoCommand { return &userInfoCommand{} }

func (c *userInfoCommand) Definition() discord.Command {
	return discord.Command{
		Type: discord.CommandUser,
		Name: "User info",
	}
}

// handle reads the resolved target user.
func (c *userInfoCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	target := discord.UserID{Snowflake: itx.Data.TargetID}

	var user discord.User
	if itx.Data.Resolved != nil {
		user = itx.Data.Resolved.Users[target]
	}

	card := builder.Embed(user.EffectiveName(), "@"+user.Username)
	card.Fields = []discord.EmbedField{
		builder.EmbedField("ID", user.ID.String(), true),
		builder.EmbedField("Bot", yesNo(user.Bot), true),
	}

	return itx.Reply(discord.ResponseMessageData{Embeds: []discord.Embed{card}}, true)
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
