package main

import (
	"example-bot/internal/command"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"

	tempest "github.com/Amatsagu/Tempest"
	godotenv "github.com/joho/godotenv"
)

func main() {
	slog.Info("Loading environmental variables...")
	if err := godotenv.Load(".env"); err != nil {
		slog.Error("failed to load env variables", err)
	}

	slog.Info("Creating new Tempest client...")
	client := tempest.NewClient(tempest.ClientOptions{
		PublicKey: os.Getenv("DISCORD_PUBLIC_KEY"),
		Rest:      tempest.NewRestClient(os.Getenv("DISCORD_BOT_TOKEN")),
	})

	addr := os.Getenv("DISCORD_APP_ADDRESS")
	testServerID, err := tempest.StringToSnowflake(os.Getenv("DISCORD_TEST_SERVER_ID")) // Register example commands only to this guild.
	if err != nil {
		slog.Error("failed to parse env variable to snowflake", err)
	}

	slog.Info("Registering commands & static components...")
	client.RegisterCommand(command.Add)
	client.RegisterCommand(command.AutoComplete)
	client.RegisterCommand(command.Avatar)
	client.RegisterCommand(command.Dynamic)
	client.RegisterCommand(command.FetchMember)
	client.RegisterCommand(command.FetchUser)
	client.RegisterCommand(command.MemoryUsage)
	client.RegisterCommand(command.Modal)
	client.RegisterCommand(command.Static)
	client.RegisterCommand(command.Swap)
	client.RegisterComponent([]string{"button-hello"}, command.HelloStatic)
	client.RegisterModal("my-modal", command.HelloModal)

	err = client.SyncCommands([]tempest.Snowflake{testServerID}, nil, false)
	if err != nil {
		slog.Error("failed to sync local commands storage with Discord API", err)
	}

	http.HandleFunc("POST /discord/callback", client.HandleDiscordRequest)

	slog.Info(fmt.Sprintf("Serving application at: %s/discord/callback", addr))
	if err := http.ListenAndServe(addr, nil); err != nil {
		slog.Error("something went terribly wrong", err)
	}
}
