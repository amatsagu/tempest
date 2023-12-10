package command

import (
	"fmt"
	"runtime"
	"time"

	tempest "github.com/Amatsagu/Tempest"
	_ "github.com/joho/godotenv"
)

var startedAt = time.Now()

var MemoryUsage tempest.Command = tempest.Command{
	Name:        "memory-usage",
	Description: "Displays basic runtime memory usage statistics.",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		reply := fmt.Sprintf(`
Current memory usage: **%.2fMB**
Finished GC cycles: **%d**
Uptime: **%s**
Ping (to Discord API): **%dms**
`,
			mb(m.Alloc),
			m.NumGC,
			time.Since(startedAt).String(),
			itx.Client.Ping().Milliseconds(),
		)

		itx.SendLinearReply(reply, false)
	},
}

func mb(value uint64) float64 {
	return float64(value) / 1024.0 / 1024.0
}
