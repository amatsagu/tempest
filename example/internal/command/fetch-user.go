package command

import (
	"encoding/json"
	"example-bot/internal/logger"

	tempest "github.com/Amatsagu/Tempest"
)

var FetchUser tempest.Command = tempest.Command{
	Name:        "fetch-user",
	Description: "Tries to grab and display user data. It only works in guild channels.",
	Options: []tempest.CommandOption{
		{
			Name:        "target",
			Description: "Mention some user",
			Type:        tempest.USER_OPTION_TYPE,
		},
	},
	AvailableInDM: false,
	SlashCommandHandler: func(itx tempest.CommandInteraction) {
		target := *itx.Member.User
		rawTargetID, available := itx.GetOptionValue("target")
		if available {
			targetID, err := tempest.StringToSnowflake(rawTargetID.(string))
			if err != nil {
				itx.SendLinearReply("Provited target user ID is invalid.", false)
				return
			}

			target, err = itx.Client.FetchUser(targetID)
			if err != nil {
				logger.Warn.Println(err)
				itx.SendLinearReply("Failed to fetch user data.", false)
				return
			}
		}

		res, err := json.MarshalIndent(target, "", "    ")
		if err != nil {
			logger.Error.Println(err)
			itx.SendLinearReply("Failed to parse received user (json) data.", false)
			return
		}

		itx.SendLinearReply("```json\n"+string(res)+"\n```", false)
	},
}
