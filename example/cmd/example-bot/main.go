package main

import (
	"example-bot/internal/command"
	"example-bot/internal/logger"
	"example-bot/internal/middleware"
	_ "net/http/pprof"
	"os"

	tempest "github.com/Amatsagu/Tempest"
	godotenv "github.com/joho/godotenv"
)

func main() {
	logger.InitLogger()

	logger.Info.Println("Loading environmental variables...")
	if err := godotenv.Load(".env"); err != nil {
		logger.Error.Panicln(err)
	}

	logger.Info.Println("Creating new Tempest client...")
	client := tempest.NewClient(tempest.ClientOptions{
		PublicKey: os.Getenv("DISCORD_PUBLIC_KEY"),
		Rest:      tempest.NewRest(os.Getenv("DISCORD_BOT_TOKEN")),
		CommandMiddleware: func(itx tempest.CommandInteraction) bool {
			res := middleware.Cooldown(itx)
			if res != nil {
				itx.SendReply(*res, false)
				return false
			}

			return true
		},
	})

	addr := os.Getenv("DISCORD_APP_ADDRESS")
	testServerID, err := tempest.StringToSnowflake(os.Getenv("DISCORD_TEST_SERVER_ID")) // Register example commands only to this guild.
	if err != nil {
		logger.Error.Panicln(err)
	}

	logger.Info.Println("Registering commands & static components...")
	client.RegisterCommand(command.Add)
	client.RegisterCommand(command.AutoComplete)
	client.RegisterCommand(command.Avatar)
	client.RegisterCommand(command.Dynamic)
	client.RegisterCommand(command.FetchMember)
	client.RegisterCommand(command.FetchUser)
	client.RegisterCommand(command.Modal)
	client.RegisterCommand(command.Static)
	client.RegisterCommand(command.MemoryUsage)
	client.RegisterComponent([]string{"button-hello"}, command.HelloStatic)
	client.RegisterModal("my-modal", command.HelloModal)

	err = client.SyncCommands([]tempest.Snowflake{testServerID}, nil, false)
	if err != nil {
		logger.Error.Panicln(err)
	}

	logger.Info.Printf("Serving application at: %s/discord/callback", addr)
	if err := client.ListenAndServe("/discord/callback", addr); err != nil {
		// Will happen in situation where normal std/http would panic so most likely never.
		logger.Error.Panicln(err)
	}
}
