package stats

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

const accent = 0x5865F2

// statsCommand holds the client (for REST/gateway latency) and the counters.
type statsCommand struct {
	client *tempest.Client
	stats  *Stats
}

func newStatsCommand(client *tempest.Client, stats *Stats) *statsCommand {
	return &statsCommand{client: client, stats: stats}
}

func (c *statsCommand) Definition() discord.Command {
	return discord.Command{
		Name:        "stats",
		Description: "Show runtime stats: uptime, interactions, memory, and latency.",
	}
}

// handle renders the panel as one Components V2 container.
func (c *statsCommand) handle(ctx context.Context, itx *discord.CommandInteraction) error {
	snapshot := c.stats.Snapshot()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	container := builder.Container(accent,
		builder.SectionWithThumbnail(
			builder.Thumbnail("https://cdn.discordapp.com/embed/avatars/0.png", "Tempest"),
			builder.TextDisplay("## Tempest stats"),
			builder.TextDisplay("A live view of this bot's runtime."),
		),
		builder.Separator(true),
		builder.TextDisplay(uptimeSection(snapshot.Uptime)),
		builder.Separator(true),
		builder.TextDisplay(interactionsSection(snapshot)),
		builder.Separator(true),
		builder.TextDisplay(memorySection(&mem)),
		builder.Separator(true),
		builder.TextDisplay(c.latencySection(ctx)),
	)

	return itx.Reply(discord.ResponseMessageData{
		Flags:      discord.MessageFlagIsComponentsV2,
		Components: discord.Components{container},
	}, false)
}

func uptimeSection(uptime time.Duration) string {
	return "### Uptime\nSince start: **" + formatDuration(uptime) + "**"
}

func interactionsSection(s Snapshot) string {
	return fmt.Sprintf(
		"### Interactions since start\n"+
			"Total **%d**\n"+
			"Commands **%d**\n"+
			"Autocomplete **%d**\n"+
			"Components **%d**\n"+
			"Modals **%d**\n"+
			"Errors **%d**",
		s.Total, s.Commands, s.Autocomplete, s.Components, s.Modals, s.Errors,
	)
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

// latencySection times one REST call and reads the shard heartbeat, degrading to a
// sentence when either is unavailable.
func (c *statsCommand) latencySection(ctx context.Context) string {
	rest := "no REST client (no token configured)"
	if client := c.client.REST(); client != nil {
		start := time.Now()
		if _, err := client.GatewayBot(ctx); err != nil {
			rest = "unavailable: " + err.Error()
		} else {
			rest = time.Since(start).Round(time.Millisecond).String()
		}
	}

	gateway := "HTTP mode, no gateway connection"
	if gw := c.client.Gateway(); gw != nil {
		if d, ok := gw.ShardLatency(0); ok {
			gateway = fmt.Sprintf("%s (shard 0 of %d)", d.Round(time.Millisecond), gw.ShardCount())
		} else {
			gateway = fmt.Sprintf("not measured yet (shard count %d)", gw.ShardCount())
		}
	}

	return fmt.Sprintf("### Latency\nDiscord API (REST) **%s**\nGateway **%s**", rest, gateway)
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
