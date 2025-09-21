package command

import (
	"log"
	"qord/api"
)

var V2Component api.Command = api.Command{
	Name:        "v2-component",
	Description: "Sends example message made of new v2 components.",
	SlashCommandHandler: func(itx *api.CommandInteraction) {
		err := itx.SendReply(api.ResponseMessageData{
			Flags: api.IS_COMPONENTS_V2_MESSAGE_FLAG,
			Components: []api.LayoutComponent{
				api.SectionComponent{
					Type: api.SECTION_COMPONENT_TYPE,
					Components: []api.TextDisplayComponent{
						{
							Type:    api.TEXT_DISPLAY_COMPONENT_TYPE,
							Content: "## Welcome Components v2!",
						},
						{
							Type:    api.TEXT_DISPLAY_COMPONENT_TYPE,
							Content: "We hope you're excited that Tempest powered Discord apps/bots can finally support new fancy components! This reply works as real proof of what's now possible.",
						},
						{
							Type:    api.TEXT_DISPLAY_COMPONENT_TYPE,
							Content: "-# Since Tempest v1.3.0",
						},
					},
					Accessory: api.ThumbnailComponent{
						Type: api.THUMBNAIL_COMPONENT_TYPE,
						Media: api.UnfurledMediaItem{
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
