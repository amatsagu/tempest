package command

import (
	"context"
	"log/slog"

	tempest "github.com/Amatsagu/Tempest"
)

var Shutdown tempest.Command = tempest.Command{
	Name:        "shutdown",
	Description: "Gracefully shutdowns app process.",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		err := itx.Client.Close(context.Background())
		if err != nil {
			slog.Error("failed at closing client", err)
			itx.SendLinearReply(err.Error(), false)
			return
		}
	},
}
