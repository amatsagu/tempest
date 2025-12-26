package command

import (
	"encoding/json"
	"log"

	tempest "github.com/amatsagu/tempest"
)

var FetchUser tempest.Command = tempest.Command{
	Name:        "user",
	Description: "Tries to grab and display user data. It only works in guild channels.",
	Options: []tempest.CommandOption{
		{
			Name:        "target",
			Description: "Mention some user",
			Type:        tempest.USER_OPTION_TYPE,
		},
	},
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		target := *itx.Member.User
		rawTargetID, available := itx.GetOptionValue("target")
		if available {
			targetID, err := tempest.StringToSnowflake(rawTargetID.(string))
			if err != nil {
				itx.SendLinearReply("Provited target user ID is invalid.", false)
				return
			}

			target, err = itx.BaseClient().FetchUser(targetID)
			if err != nil {
				log.Println("failed to fetch user data", err)
				itx.SendLinearReply("Failed to fetch user data.", false)
				return
			}
		}

		res, err := json.MarshalIndent(target, "", "    ")
		if err != nil {
			log.Println("failed to parse user data", err)
			itx.SendLinearReply("Failed to parse received user (json) data.", false)
			return
		}

		itx.SendLinearReply("```json\n"+string(res)+"\n```", false)
	},
}
