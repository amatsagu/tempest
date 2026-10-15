package showcase

import (
	"context"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/route"
)

type showcaseCommand struct{}

func newShowcaseCommand() *showcaseCommand { return &showcaseCommand{} }

func (c *showcaseCommand) Definition() discord.Command {
	return discord.Command{
		Name:        "showcase",
		Description: "Render a Components V2 message.",
	}
}

// handle builds one container holding every V2 layout piece.
func (c *showcaseCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	container := builder.Container(0x5865F2,
		builder.TextDisplay("## Components V2"),
		builder.TextDisplay("Everything below lives inside a single container."),
		builder.Separator(true),
		builder.SectionWithThumbnail(
			builder.Thumbnail("https://cdn.discordapp.com/embed/avatars/0.png", "Tempest"),
			builder.TextDisplay("A section pairs text with a thumbnail or a button."),
		),
		builder.MediaGallery(
			builder.MediaGalleryItem("https://cdn.discordapp.com/embed/avatars/1.png"),
		),
		builder.ActionRow(builder.Button(acknowledgeID, "Acknowledge", discord.ButtonPrimary)),
	)

	return itx.Reply(discord.ResponseMessageData{
		Flags:      discord.MessageFlagIsComponentsV2,
		Components: discord.Components{container},
	}, false)
}

// handleAcknowledge updates the V2 message in place.
func (c *showcaseCommand) handleAcknowledge(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
	return itx.Update(discord.ResponseMessageData{
		Flags:      discord.MessageFlagIsComponentsV2,
		Components: discord.Components{builder.TextDisplay("Acknowledged.")},
	})
}
