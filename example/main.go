// Command tempest-example is a small but complete Discord bot built on Tempest.
package main

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/gateway"

	"github.com/amatsagu/tempest-example/events"
	"github.com/amatsagu/tempest-example/feature/calc"
	"github.com/amatsagu/tempest-example/feature/commands"
	"github.com/amatsagu/tempest-example/feature/components"
	"github.com/amatsagu/tempest-example/feature/contextmenu"
	"github.com/amatsagu/tempest-example/feature/image"
	"github.com/amatsagu/tempest-example/feature/showcase"
	"github.com/amatsagu/tempest-example/feature/stats"
	"github.com/amatsagu/tempest-example/feature/tags"
)

func main() {
	if err := run(); err != nil {
		slog.Error("example bot stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// With HTTP_ADDR set the bot serves the interaction endpoint over HTTP and needs
	// DISCORD_PUBLIC_KEY; otherwise it connects over the gateway.
	httpAddr := os.Getenv("HTTP_ADDR")

	// Created before the client so the hooks and /stats share one counter set.
	botStats := stats.NewStats()

	client, err := tempest.New(tempest.Config{
		Token:       os.Getenv("DISCORD_BOT_TOKEN"),
		PublicKey:   os.Getenv("DISCORD_PUBLIC_KEY"),
		HTTPAddr:    httpAddr,
		Gateway:     httpAddr == "",
		Intents:     uint32(gateway.IntentGuilds),
		Compression: gateway.CompressionZstd,
		Logf:        log.Printf,
		Debug:       os.Getenv("DEBUG") != "",

		// Lifecycle hooks; nil hooks cost nothing. Return false from OnRawInteraction
		// to veto a matched route (skipped and left unacknowledged).
		OnRawInteraction: func(info tempest.InteractionInfo) bool {
			slog.Debug("interaction",
				"kind", info.Kind,
				"handled", info.Handled,
				"route", info.Route.Pattern,
				"exact", info.Route.Exact,
			)
			return true
		},
		OnCompletedInteraction: func(info tempest.InteractionInfo) {
			// Feed the /stats panel, then keep the debug trace.
			botStats.Completed(info)
			slog.Debug("interaction completed", "route", info.Route.Pattern, "response", info.Response, "took", info.Duration)
		},
		// OnInteractionError counts failures for /stats, logs them, and tells the user.
		OnInteractionError: func(err error, info tempest.InteractionInfo, itx any) {
			// A file reply reports success through ErrAnswered, not as a failure.
			if errors.Is(err, image.ErrAnswered) {
				return
			}

			botStats.Failed(err, info, itx)
			slog.Error("interaction failed", "route", info.Route.Pattern, "id", info.ID, "err", err)

			message := discord.ResponseMessageData{Content: "Something went wrong."}
			var replyErr error
			switch v := itx.(type) {
			case *discord.CommandInteraction:
				replyErr = v.Reply(message, true)
			case *discord.ComponentInteraction:
				replyErr = v.Reply(message, true)
			case *discord.ModalInteraction:
				replyErr = v.Reply(message, true)
			}

			if replyErr != nil {
				slog.Error("could not report interaction error", "err", replyErr)
			}
		},

		// Subscribe to extra gateway events and observe raw dispatches.
		ExtraEvents: events.Names,
		OnRawEvent: func(shard int, event string, data jsontext.Value) {
			slog.Debug("raw dispatch", "shard", shard, "event", event, "bytes", len(data))
		},
	})

	if err != nil {
		return err
	}

	events.Register(client)

	// Every feature registers its handlers and returns its command definitions; a
	// failure stops startup instead of running with a missing handler.
	registerFeatures := []func(*tempest.Client) ([]discord.Command, error){
		commands.Register,
		contextmenu.Register,
		components.Register,
		showcase.Register,
		image.Register,
		tags.Register,
		calc.Register,
		// /stats needs the counter set created above, so it is wrapped.
		func(client *tempest.Client) ([]discord.Command, error) {
			return stats.Register(client, botStats)
		},
	}

	var definitions []discord.Command
	for _, register := range registerFeatures {
		featureDefinitions, err := register(client)
		if err != nil {
			return err
		}
		definitions = append(definitions, featureDefinitions...)
	}

	if client.REST() != nil {
		// A guild ID instead of the zero value would sync to one guild for faster iteration.
		if err := client.SyncCommands(ctx, discord.GuildID{}, definitions); err != nil {
			return fmt.Errorf("sync commands: %w", err)
		}
		slog.Info("synced command definitions", "count", len(definitions))
	}

	slog.Info("starting bot", "http", httpAddr != "", "gateway", httpAddr == "")
	return client.Run(ctx)
}
