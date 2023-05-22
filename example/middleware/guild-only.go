package middleware

import (
	tempest "github.com/Amatsagu/Tempest"
)

// Stops any commands executed outside server (obviously not required, just an example).
func GuildOnly(itx tempest.CommandInteraction) *tempest.ResponseMessageData {
	if itx.GuildID == 0 {
		return &tempest.ResponseMessageData{
			Content: "This command is not allowed to be used in DM channel.",
		}
	}
	return nil
}
