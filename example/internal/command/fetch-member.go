package command

import (
	"encoding/json"
	"log/slog"

	tempest "github.com/Amatsagu/Tempest"
)

var FetchMember tempest.Command = tempest.Command{
	Name:        "fetch-member",
	Description: "Tries to grab and display guild member data. It only works in guild channels.",
	Options: []tempest.CommandOption{
		{
			Name:        "target",
			Description: "Mention some member",
			Type:        tempest.USER_OPTION_TYPE,
		},
	},
	AvailableInDM: false,
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		target := *itx.Member
		rawTargetID, available := itx.GetOptionValue("target")
		if available {
			targetID, err := tempest.StringToSnowflake(rawTargetID.(string))
			if err != nil {
				itx.SendLinearReply("Provited target member ID is invalid.", false)
				return
			}

			target, err = itx.Client.FetchMember(itx.GuildID, targetID)
			if err != nil {
				slog.Error("failed to fetch member", err)
				itx.SendLinearReply("Failed to fetch member data.", false)
				return
			}
		}

		res, err := json.MarshalIndent(target, "", "    ")
		if err != nil {
			slog.Error("failed to parse member data", err)
			itx.SendLinearReply("Failed to parse received member (json) data.", false)
			return
		}

		itx.SendLinearReply("```json\n"+string(res)+"\n```", false)
	},
}
