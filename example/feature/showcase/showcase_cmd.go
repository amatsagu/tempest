package showcase

import (
	"context"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

const acknowledgeID = "showcase:ack"

type showcaseCommand struct{}

func newShowcaseCommand() *showcaseCommand { return &showcaseCommand{} }

func (c *showcaseCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "showcase",
		Description:         "Render a Components V2 message.",
		SlashCommandHandler: c.handle,
	}
}

func (c *showcaseCommand) handle(ctx context.Context, itx *tempest.CommandInteraction) error {
	container := builder.Container(0x5865F2,
		builder.TextDisplay("## Components V2"),
		builder.TextDisplay("Everything below lives inside a single container."),
		builder.Separator(true),
		builder.Section(
			builder.Thumbnail("https://cdn.discordapp.com/embed/avatars/0.png", "Tempest"),
			builder.TextDisplay("A section pairs text with a thumbnail or a button."),
		),
		builder.MediaGallery(
			builder.MediaGalleryItem("https://cdn.discordapp.com/embed/avatars/1.png"),
		),
		builder.ActionRow(builder.Button(acknowledgeID, "Acknowledge", discord.PrimaryButtonStyle)),
	)

	return itx.SendReply(ctx, discord.ResponseMessageData{
		Flags:      discord.IsComponentsV2MessageFlag,
		Components: []discord.MessageComponent{container},
	}, false, nil)
}

func (c *showcaseCommand) handleAcknowledge(ctx context.Context, itx *tempest.ComponentInteraction) error {
	return itx.AcknowledgeWithUpdate(ctx, discord.ResponseMessageData{
		Flags:      discord.IsComponentsV2MessageFlag,
		Components: []discord.MessageComponent{builder.TextDisplay("Acknowledged.")},
	})
}
