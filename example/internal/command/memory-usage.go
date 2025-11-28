package command

import (
	"fmt"
	"log"
	"runtime"
	"time"

	tempest "github.com/amatsagu/tempest"
	_ "github.com/joho/godotenv"
)

var startedAt = time.Now()

var MemoryUsage tempest.Command = tempest.Command{
	Name:        "memory-usage",
	Description: "Displays basic runtime memory usage statistics.",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		//runtime.GC()

		now := time.Now()
		reply := fmt.Sprintf(`
Current memory usage: **%.2fMB**
Finished GC cycles: **%d**
Goroutines: **%d**
Uptime: **%s**
`,
			mb(m.Alloc),
			m.NumGC,
			runtime.NumGoroutine(),
			time.Since(startedAt).String(),
		)

		itx.SendLinearReply(reply, false)
		log.Println(time.Since(now).String())
	},
}

func mb(value uint64) float64 {
	return float64(value) / 1024.0 / 1024.0
}
