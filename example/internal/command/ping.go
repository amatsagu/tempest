package command

import (
	"time"

	tempest "github.com/amatsagu/tempest"
)

var Ping tempest.Command = tempest.Command{
	Name:        "ping",
	Description: "Sends test message to check Discord API latency and returns latency value.",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		// When using gateway client, calculate average latency from all active shards.
		if itx.GatewayClient != nil {
			latencies := itx.GatewayClient.Gateway.ShardLatencies()
			var finalLatency time.Duration

			// Add latency value from all inside of all shards.
			for _, dur := range latencies {
				finalLatency += dur
			}

			// Divide by number of shards.
			finalLatency /= time.Duration(len(latencies))

			itx.SendLinearReply("Current, average latency: "+finalLatency.String(), false)
			return
		}

		// Do some simple call to discord api that returns actual data to see our own + discord internal latency.
		startedAt := time.Now()
		_, _ = itx.HTTPClient.FetchUser(itx.HTTPClient.ApplicationID)
		finalLatency := time.Since(startedAt)

		itx.SendLinearReply("Current, network + Discord internal latency: "+finalLatency.String(), false)
	},
}
