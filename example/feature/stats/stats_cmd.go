package stats

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

const accent = 0x5865F2

// statsCommand holds the client (for REST/gateway latency) and the counters.
type statsCommand struct {
	client  *tempest.BaseClient
	gateway *tempest.GatewayClient
	stats   *Stats
}

func newStatsCommand(client *tempest.BaseClient, gateway *tempest.GatewayClient, stats *Stats) *statsCommand {
	return &statsCommand{client: client, gateway: gateway, stats: stats}
}

func (c *statsCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "stats",
		Description:         "Show runtime stats: uptime, commands, memory, and latency.",
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

// handle renders the panel as one Components V2 container.
func (c *statsCommand) handle(itx *tempest.CommandInteraction) error {
	snapshot := c.stats.Snapshot()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	container := util.Container(accent,
		util.SectionWithThumbnail(
			util.Thumbnail("https://cdn.discordapp.com/embed/avatars/0.png", "Tempest"),
			util.TextDisplay("## Tempest stats"),
			util.TextDisplay("A live view of this bot's runtime."),
		),
		util.Separator(true),
		util.TextDisplay(uptimeSection(snapshot.Uptime)),
		util.Separator(true),
		util.TextDisplay(fmt.Sprintf("### Commands since start\nTotal **%d**", snapshot.Commands)),
		util.Separator(true),
		util.TextDisplay(memorySection(&mem)),
		util.Separator(true),
		util.TextDisplay(c.latencySection()),
	)

	return itx.SendReply(tempest.ResponseMessageData{
		Flags:      tempest.IS_COMPONENTS_V2_MESSAGE_FLAG,
		Components: []tempest.MessageComponent{container},
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

// latencySection times one authenticated REST call and reads the shard heartbeats.
func (c *statsCommand) latencySection() string {
	var restText string
	start := time.Now()
	if err := c.client.Rest.Request(http.MethodGet, "/gateway/bot", nil, nil); err != nil {
		restText = "unavailable: " + err.Error()
	} else {
		restText = time.Since(start).Round(time.Millisecond).String()
	}

	gatewayText := "HTTP mode, no gateway connection"
	if c.gateway != nil {
		states, pings := c.gateway.Gateway.AllShardDetails()

		online := 0
		worst := time.Duration(0)
		for i, state := range states {
			if state != tempest.ONLINE_SHARD_STATE {
				continue
			}

			online++
			if pings[i] > worst {
				worst = pings[i]
			}
		}

		gatewayText = fmt.Sprintf("%d/%d online, worst heartbeat %s", online, len(states), worst.Round(time.Millisecond))
	}

	return fmt.Sprintf("### Latency\nDiscord API (REST) **%s**\nGateway **%s**", restText, gatewayText)
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
