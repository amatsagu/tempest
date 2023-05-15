package command

import (
	"example-bot/middleware"
	"fmt"
	"runtime"
	"time"

	tempest "github.com/Amatsagu/Tempest"
	_ "github.com/joho/godotenv"
)

var startedAt = time.Now()

var Statistics tempest.Command = tempest.Command{
	Name:        "statistics",
	Description: "Displays basic runtime statistics.",
	SlashCommandHandler: func(itx tempest.CommandInteraction) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		reply := fmt.Sprintf(`
Current memory usage: **%.2fMB**
Finished GC cycles: **%d**
Uptime: **%s**
Ping (to Discord API): **%dms**
Executed commands: **%d**
`,
			mb(m.Alloc),
			m.NumGC,
			time.Since(startedAt).String(),
			itx.Client.Ping().Milliseconds(),
			middleware.ExecutedCommands,
		)

		itx.SendLinearReply(reply, false)
	},
}

func mb(value uint64) float64 {
	return float64(value) / 1024.0 / 1024.0
}
