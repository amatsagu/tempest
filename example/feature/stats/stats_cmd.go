package stats

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

const accent = 0x5865F2

type statsCommand struct {
	client *tempest.Client
	stats  *Stats
}

func newStatsCommand(client *tempest.Client, stats *Stats) *statsCommand {
	return &statsCommand{client: client, stats: stats}
}

func (c *statsCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "stats",
		Description:         "Show runtime stats: uptime, commands, memory, and latency.",
		SlashCommandHandler: c.handle,
	}
}

func (c *statsCommand) handle(ctx context.Context, itx *tempest.CommandInteraction) error {
	snapshot := c.stats.Snapshot()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	container := builder.Container(accent,
		builder.Section(
			builder.Thumbnail("https://cdn.discordapp.com/embed/avatars/0.png", "Tempest"),
			builder.TextDisplay("## Tempest stats"),
			builder.TextDisplay("A live view of this bot's runtime."),
		),
		builder.Separator(true),
		builder.TextDisplay(uptimeSection(snapshot.Uptime)),
		builder.Separator(true),
		builder.TextDisplay(fmt.Sprintf(
			"### Hook Metrics\nCommands: **%d started / %d done**\nComponents: **%d started / %d done**\nModals: **%d started / %d done**\nHandler Errors: **%d**",
			snapshot.CommandsStarted, snapshot.CommandsCompleted,
			snapshot.ComponentsStarted, snapshot.ComponentsCompleted,
			snapshot.ModalsStarted, snapshot.ModalsCompleted,
			snapshot.Errors,
		)),
		builder.Separator(true),
		builder.TextDisplay(memorySection(&mem)),
		builder.Separator(true),
		builder.TextDisplay(c.latencySection(ctx)),
	)

	return itx.SendReply(ctx, discord.ResponseMessageData{
		Flags:      discord.IsComponentsV2MessageFlag,
		Components: []discord.MessageComponent{container},
	}, false, nil)
}

func uptimeSection(uptime time.Duration) string {
	return "### Uptime\nSince start: **" + formatDuration(uptime) + "**"
}

func memorySection(m *runtime.MemStats) string {
	return fmt.Sprintf(
		"### Memory\n"+
			"HeapAlloc **%s** - live heap objects\n"+
			"HeapInuse **%s** - heap spans in use\n"+
			"HeapIdle **%s** - idle heap spans\n"+
			"HeapReleased **%s** - returned to the OS\n"+
			"TotalAlloc **%s** - cumulative allocation\n"+
			"Sys **%s** - memory obtained from the OS\n"+
			"NumGC **%d** - completed GC cycles\n"+
			"GC pause total **%s**\n"+
			"GOMAXPROCS **%d**\n"+
			"goroutines **%d**",
		formatBytes(m.HeapAlloc),
		formatBytes(m.HeapInuse),
		formatBytes(m.HeapIdle),
		formatBytes(m.HeapReleased),
		formatBytes(m.TotalAlloc),
		formatBytes(m.Sys),
		m.NumGC,
		time.Duration(m.PauseTotalNs).Round(time.Millisecond),
		runtime.GOMAXPROCS(0),
		runtime.NumGoroutine(),
	)
}

func (c *statsCommand) latencySection(ctx context.Context) string {
	var restText string
	start := time.Now()
	if err := c.client.Rest.Request(ctx, http.MethodGet, "/users/@me", nil, nil); err != nil {
		restText = "unavailable: " + err.Error()
	} else {
		restText = time.Since(start).Round(time.Millisecond).String()
	}

	return fmt.Sprintf("### Latency\nDiscord API (REST) **%s**", restText)
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	seconds := d / time.Second

	return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
}

func formatBytes(b uint64) string {
	return fmt.Sprintf("%.1f MiB", float64(b)/(1<<20))
}
