package middleware

import (
	"example-bot/logger"

	tempest "github.com/Amatsagu/Tempest"
)

var ExecutedCommands uint32 = 0

// Counts all executed commands.
func Counter(itx tempest.CommandInteraction) *tempest.ResponseMessageData {
	ExecutedCommands++
	logger.Info.Printf("@%s (%d) uses %s slash command (%dth)\n", itx.Member.User.Username, itx.Member.User.ID, itx.Data.Name, ExecutedCommands)
	return nil
}
