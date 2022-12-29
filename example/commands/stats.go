package commands

import (
	"fmt"
	"runtime"
	"time"

	tempest "github.com/Amatsagu/Tempest"
	_ "github.com/joho/godotenv"
)

var CommandCounter uint32 = 0
var startedAt = time.Now()

var Statistics tempest.Command = tempest.Command{
	Name:        "statistics",
	Description: "Displays basic runtime statistics.",
	SlashCommandHandler: func(itx tempest.CommandInteraction) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		reply := fmt.Sprintf(`
Current memory usage: %.2fMB
 => Heap usage: %.2fMB (Allocated: %.2fMB)
 => Stack usage: %.2fMB (Allocated: %.2fMB)

Total system allocated memory: %.2fMB
GC cycles: %d (like 95%% of all sweeps in example/tutorial code is not made by library but GC internal timer)
Uptime: %.2f minute(s)

Ping: %dms
Executed commands: %d`, mb(m.Alloc), mb(m.HeapInuse), mb(m.HeapSys), mb(m.StackInuse), mb(m.StackSys), mb(m.Sys), m.NumGC, time.Since(startedAt).Minutes(), itx.Client.Ping().Milliseconds(), CommandCounter)

		itx.SendLinearReply(reply, false)
	},
}

func mb(value uint64) float64 {
	return float64(value) / 1024.0 / 1024.0
}
