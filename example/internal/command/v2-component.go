package command

import (
	"log"

	tempest "github.com/amatsagu/tempest"
)

var V2Component tempest.Command = tempest.Command{
	Name:        "v2-component",
	Description: "Sends example message made of new v2 components.",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		err := itx.SendReply(tempest.ResponseMessageData{
			Flags: tempest.IS_COMPONENTS_V2_MESSAGE_FLAG,
			Components: []tempest.MessageComponent{
				tempest.SectionComponent{
					Type: tempest.SECTION_COMPONENT_TYPE,
					Components: []tempest.TextDisplayComponent{
						{
							Type:    tempest.TEXT_DISPLAY_COMPONENT_TYPE,
							Content: "## Welcome Components v2!",
						},
						{
							Type:    tempest.TEXT_DISPLAY_COMPONENT_TYPE,
							Content: "We hope you're excited that Tempest powered Discord apps/bots can finally support new fancy components! This reply works as real proof of what's now possible.",
						},
						{
							Type:    tempest.TEXT_DISPLAY_COMPONENT_TYPE,
							Content: "-# Since Tempest v1.3.0",
						},
					},
					Accessory: tempest.ThumbnailComponent{
						Type: tempest.THUMBNAIL_COMPONENT_TYPE,
						Media: tempest.UnfurledMediaItem{
							URL: "https://raw.githubusercontent.com/amatsagu/tempest/refs/heads/master/.github/tempest-logo.png",
						},
					},
				},
			},
		}, false, nil)

		if err != nil {
			log.Println("Run into a problem when trying to construct new components v2 reply message:", err)
		}
	},
}
