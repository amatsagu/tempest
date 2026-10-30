package showcase

import (
	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

const acknowledgeID = "showcase:ack"

type showcaseCommand struct{}

func newShowcaseCommand() *showcaseCommand { return &showcaseCommand{} }

func (c *showcaseCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "showcase",
		Description:         "Render a Components V2 message.",
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

// handle builds one container holding every V2 layout piece.
func (c *showcaseCommand) handle(itx *tempest.CommandInteraction) error {
	container := util.Container(0x5865F2,
		util.TextDisplay("## Components V2"),
		util.TextDisplay("Everything below lives inside a single container."),
		util.Separator(true),
		util.SectionWithThumbnail(
			util.Thumbnail("https://cdn.discordapp.com/embed/avatars/0.png", "Tempest"),
			util.TextDisplay("A section pairs text with a thumbnail or a button."),
		),
		util.MediaGallery(
			util.MediaGalleryItem("https://cdn.discordapp.com/embed/avatars/1.png"),
		),
		util.ActionRow(util.Button(acknowledgeID, "Acknowledge", tempest.PRIMARY_BUTTON_STYLE)),
	)

	return itx.SendReply(tempest.ResponseMessageData{
		Flags:      tempest.IS_COMPONENTS_V2_MESSAGE_FLAG,
		Components: []tempest.MessageComponent{container},
	}, false, nil)
}

// handleAcknowledge updates the V2 message in place.
func (c *showcaseCommand) handleAcknowledge(itx *tempest.ComponentInteraction) error {
	return itx.AcknowledgeWithUpdate(tempest.ResponseMessageData{
		Flags:      tempest.IS_COMPONENTS_V2_MESSAGE_FLAG,
		Components: []tempest.MessageComponent{util.TextDisplay("Acknowledged.")},
	})
}
