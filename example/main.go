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
	"github.com/amatsagu/tempest-example/feature/patterns"
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

	botStats := stats.NewStats()

	clientCfg := tempest.ClientConfig{
		Context:                    ctx,
		Token:                      token,
		PublicKey:                  publicKey,
		DefaultInteractionContexts: []discord.InteractionContextType{discord.GuildContextType},
		Trace:                      trace,
		Hooks: tempest.ClientHooks{
			PreCommand:  botStats.PreCommand,
			PostCommand: botStats.PostCommand,
			CommandError: func(ctx context.Context, cmd tempest.Command, itx *tempest.CommandInteraction, err error) {
				botStats.RecordCommandError(ctx, cmd, itx, err)
				util.Logger.Printf("command %q failed (interaction %s): %v", itx.Data.Name, itx.ID, err)
				if !itx.Responded() {
					_ = itx.SendLinearReply(ctx, "Command failed: "+err.Error(), true)
				}
			},
			PreComponent:  botStats.PreComponent,
			PostComponent: botStats.PostComponent,
			ComponentError: func(ctx context.Context, itx *tempest.ComponentInteraction, err error) {
				botStats.RecordComponentError(ctx, itx, err)
				util.Logger.Printf("component %q failed (interaction %s): %v", itx.Data.CustomID, itx.ID, err)
				if !itx.Responded() {
					_ = itx.AcknowledgeWithLinearMessage(ctx, "Component failed: "+err.Error(), true)
				}
			},
			PreModal:  botStats.PreModal,
			PostModal: botStats.PostModal,
			ModalError: func(ctx context.Context, itx *tempest.ModalInteraction, err error) {
				botStats.RecordModalError(ctx, itx, err)
				util.Logger.Printf("modal %q failed (interaction %s): %v", itx.Data.CustomID, itx.ID, err)
				if !itx.Responded() {
					_ = itx.AcknowledgeWithLinearMessage(ctx, "Modal failed: "+err.Error(), true)
				}
			},
		},
	}
	if trace {
		clientCfg.TraceLogger = util.Logger
	}

	client := tempest.NewClient(clientCfg)

	registerFeatures := []func(*tempest.Client) ([]tempest.Command, error){
		commands.Register,
		contextmenu.Register,
		components.Register,
		patterns.Register,
		showcase.Register,
		image.Register,
		tags.Register,
		calc.Register,
		premium.Register,
		func(c *tempest.Client) ([]tempest.Command, error) {
			return stats.Register(c, botStats)
		},
	}

	var definitions []tempest.Command
	for _, register := range registerFeatures {
		featureDefinitions, err := register(client)
		if err != nil {
			return err
		}
		definitions = append(definitions, featureDefinitions...)
	}

	var guildIDs []discord.GuildID
	if rawGuildID := os.Getenv("DISCORD_TEST_SERVER_ID"); rawGuildID != "" {
		guildID, err := discord.StringToSnowflake(rawGuildID)
		if err != nil {
			return fmt.Errorf("parse DISCORD_TEST_SERVER_ID: %w", err)
		}
		guildIDs = []discord.GuildID{{Snowflake: guildID}}
	}

	if err := client.SyncCommands(ctx, guildIDs, nil, false); err != nil {
		return fmt.Errorf("sync commands: %w", err)
	}
	util.Logger.Printf("synced %d command definitions across %d guild target(s)", len(definitions), len(guildIDs))

	return runHTTP(ctx, httpAddr, client)
}

func runHTTP(ctx context.Context, addr string, client *tempest.Client) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           client,
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
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = client.Shutdown(shutdownCtx)
			return err
		}
		return client.Shutdown(shutdownCtx)
	}
}
