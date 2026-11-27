// Command tempest-example is a small but complete Discord bot built on Tempest's
// HTTP interactions API.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/feature/calc"
	"github.com/amatsagu/tempest-example/feature/commands"
	"github.com/amatsagu/tempest-example/feature/components"
	"github.com/amatsagu/tempest-example/feature/contextmenu"
	"github.com/amatsagu/tempest-example/feature/image"
	"github.com/amatsagu/tempest-example/feature/premium"
	"github.com/amatsagu/tempest-example/feature/showcase"
	"github.com/amatsagu/tempest-example/feature/stats"
	"github.com/amatsagu/tempest-example/feature/tags"
	"github.com/amatsagu/tempest-example/util"
)

func main() {
	if err := run(); err != nil {
		util.Logger.Printf("example bot stopped: %v", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load(".env")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		return errors.New("DISCORD_BOT_TOKEN is required")
	}

	publicKey := os.Getenv("DISCORD_PUBLIC_KEY")
	if publicKey == "" {
		return errors.New("DISCORD_PUBLIC_KEY is required")
	}

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = os.Getenv("APP_ADDRESS")
	}
	if httpAddr == "" {
		httpAddr = "127.0.0.1:8080"
	}

	trace := os.Getenv("DEBUG") != ""

	// Created before the client so the command hook and /stats share one counter.
	botStats := stats.NewStats()

	baseOptions := tempest.BaseClientOptions{
		Context:                    ctx,
		Token:                      token,
		DefaultInteractionContexts: []discord.InteractionContextType{discord.GuildContextType},
		PreCommandHook:             botStats.PreCommand,
	}
	if trace {
		// Tempest logs through a *log.Logger. Reuse the example's logger only
		// when tracing is enabled; otherwise the client keeps its discard logger.
		baseOptions.Logger = util.Logger
	}

	httpClient := tempest.NewHTTPClient(tempest.HTTPClientOptions{
		Trace:             trace,
		PublicKey:         publicKey,
		BaseClientOptions: baseOptions,
	})
	base := httpClient.BaseClient

	// Every feature registers its handlers and returns its command definitions; a
	// failure stops startup instead of running with a missing handler.
	registerFeatures := []func(*tempest.BaseClient) ([]tempest.Command, error){
		commands.Register,
		contextmenu.Register,
		components.Register,
		showcase.Register,
		image.Register,
		tags.Register,
		calc.Register,
		premium.Register,
		func(client *tempest.BaseClient) ([]tempest.Command, error) {
			return stats.Register(client, botStats)
		},
	}

	var definitions []tempest.Command
	for _, register := range registerFeatures {
		featureDefinitions, err := register(base)
		if err != nil {
			return err
		}
		definitions = append(definitions, featureDefinitions...)
	}

	// A guild ID instead of the zero value would sync to one guild for faster
	// iteration. The zero-value slice syncs globally.
	var guildIDs []discord.GuildID
	if rawGuildID := os.Getenv("DISCORD_TEST_SERVER_ID"); rawGuildID != "" {
		guildID, err := discord.StringToSnowflake(rawGuildID)
		if err != nil {
			return fmt.Errorf("parse DISCORD_TEST_SERVER_ID: %w", err)
		}
		guildIDs = []discord.GuildID{{Snowflake: guildID}}
	}

	if err := base.SyncCommandsWithDiscord(guildIDs, nil, false); err != nil {
		return fmt.Errorf("sync commands: %w", err)
	}
	util.Logger.Printf("synced %d command definitions across %d guild target(s)", len(definitions), len(guildIDs))

	return runHTTP(ctx, httpAddr, httpClient)
}

func runHTTP(ctx context.Context, addr string, client *tempest.HTTPClient) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", client.DiscordRequestHandler)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		util.Logger.Printf("starting bot over HTTP on %s", addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
