package command

import (
	"log"
	"time"

	tempest "github.com/amatsagu/tempest"
)

var Ping tempest.Command = tempest.Command{
	Name:        "ping",
	Description: "Sends test message to check Discord API latency and returns latency value.",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		// When using gateway client, calculate average latency from all active shards.
		if itx.GatewayClient != nil {
			_, latency, err := itx.GatewayClient.Gateway.ShardDetails(itx.ShardID)
			if err != nil {
				log.Println("failed to fetch current shard details", err)
				itx.SendLinearReply("Failed to fetch shard details.", false)
				return
			}

			itx.SendLinearReply("Current, average latency: "+latency.String(), false)
			return
		}

		// Do some simple call to discord api that returns actual data to see our own + discord internal latency.
		startedAt := time.Now()
		_, _ = itx.HTTPClient.FetchUser(itx.HTTPClient.ApplicationID)
		finalLatency := time.Since(startedAt)

		itx.SendLinearReply("Current, network + Discord internal latency: "+finalLatency.String(), false)
	},
}
