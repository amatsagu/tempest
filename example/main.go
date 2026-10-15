// Command tempest-example is a small but complete Discord bot built on the
// Tempest runtime. It is a tour of the common things a bot does: slash, user,
// and message commands, embeds, buttons, select menus, modals, Components V2,
// and both static and dynamic handler registration.
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

	// With HTTP_ADDR set the bot serves the interaction endpoint over HTTP and
	// needs DISCORD_PUBLIC_KEY. Otherwise it connects over the gateway.
	httpAddr := os.Getenv("HTTP_ADDR")

	// Created before the client so the lifecycle hooks below and the /stats
	// command share one counter set.
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

		// Lifecycle hooks. They run on the interaction path and cost nothing
		// when left nil. OnRawInteraction fires for every interaction except
		// PING, as soon as the route is known, before any handler runs. Return
		// false to veto an interaction: its handler is skipped and it is left
		// unacknowledged, exactly like Tempest's pre-command hook. An
		// interaction with no matching route is dropped the same way, except
		// with DEBUG set, where it gets an ephemeral hint explaining how to
		// register a handler. So a veto is only needed to block a matched route.
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
		// OnInteractionError runs once for every handler that returns an error.
		// It counts the failure for /stats, logs it and tells the user
		// generically through Reply, so the message is the first response or an
		// edit when the handler already answered. The typed itx lets the hook
		// answer the right interaction.
		OnInteractionError: func(err error, info tempest.InteractionInfo, itx any) {
			// The rotate modal answers itself with a multipart upload; its
			// sentinel is not a failure, so it is neither counted nor reported.
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

		// Subscribe to a couple of extra gateway events, and observe every raw
		// dispatch before the runtime parses anything.
		ExtraEvents: events.Names,
		OnRawEvent: func(shard int, event string, data jsontext.Value) {
			slog.Debug("raw dispatch", "shard", shard, "event", event, "bytes", len(data))
		},
	})
	if err != nil {
		return err
	}

	events.Register(client)

	// Every feature registers its handlers and hands back the command
	// definitions it wants announced to Discord. A registration failure stops
	// startup with a clear error instead of running with a missing handler.
	registerFeatures := []func(*tempest.Client) ([]discord.Command, error){
		commands.Register,
		contextmenu.Register,
		components.Register,
		showcase.Register,
		image.Register,
		tags.Register,
		calc.Register,
		// /stats needs the counter set created above, so it is wrapped rather
		// than referenced directly.
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
		// A guild ID here (instead of the zero value) syncs to one guild and
		// updates instantly, which is nicer while iterating locally.
		if err := client.SyncCommands(ctx, discord.GuildID{}, definitions); err != nil {
			return fmt.Errorf("sync commands: %w", err)
		}
		slog.Info("synced command definitions", "count", len(definitions))
	}

	slog.Info("starting bot", "http", httpAddr != "", "gateway", httpAddr == "")
	return client.Run(ctx)
}
