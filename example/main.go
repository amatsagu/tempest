package main

import (
	"example-bot/command"
	"example-bot/logger"
	"example-bot/middleware"
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
	client := tempest.CreateClient(tempest.ClientOptions{
		ApplicationID: tempest.StringToSnowflake(os.Getenv("DISCORD_APP_ID")),
		PublicKey:     os.Getenv("DISCORD_PUBLIC_KEY"),
		Token:         "Bot " + os.Getenv("DISCORD_BOT_TOKEN"),
		CommandMiddleware: func(itx tempest.CommandInteraction) *tempest.ResponseMessageData {
			res := middleware.GuildOnly(itx)
			if res != nil {
				return res
			}

			res = middleware.Cooldown(itx)
			if res != nil {
				return res
			}

			return middleware.Counter(itx)
		},
	})

	addr := os.Getenv("DISCORD_APP_ADDRESS")
	testServerID := tempest.StringToSnowflake(os.Getenv("DISCORD_TEST_SERVER_ID")) // Register example commands only to this guild.

	logger.Info.Println("Registering commands & static components...")
	client.RegisterCommand(command.Add)
	client.RegisterCommand(command.Avatar)
	client.RegisterCommand(command.Dynamic)
	client.RegisterCommand(command.Static)
	client.RegisterCommand(command.Statistics)
	client.RegisterComponent([]string{"button-hello"}, command.HelloStatic)

	err := client.SyncCommands([]tempest.Snowflake{testServerID}, nil, false)
	if err != nil {
		logger.Error.Panicln(err)
	}

	logger.Info.Printf("Serving application at: %s/discord", addr)
	if err := client.ListenAndServe("/discord", addr); err != nil {
		// Will happen in situation where normal std/http would panic so most likely never.
		logger.Error.Panicln(err)
	}
}
