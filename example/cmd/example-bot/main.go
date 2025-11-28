package main

import (
	"context"
	"example-bot/internal/command"
	"log"
	"net/http"

	// _ "net/http/pprof"
	"os"

	tempest "github.com/amatsagu/tempest"
	godotenv "github.com/joho/godotenv"
)

func loadCommands(client *tempest.BaseClient) {
	// Warning!
	// Please make sure you've registered all slash commands & static components before starting bot/app.
	// Client's registry after starting is used as readonly cache so it skips using mutex for performance reasons.
	// You shouldn't update registry after launch.

	client.RegisterCommand(command.Add)
	client.RegisterCommand(command.AutoComplete)
	client.RegisterCommand(command.Avatar)
	client.RegisterCommand(command.Defer)
	client.RegisterCommand(command.Dynamic)
	client.RegisterCommand(command.Fetch)
	client.RegisterSubCommand(command.FetchMember, "fetch")
	client.RegisterSubCommand(command.FetchUser, "fetch")
	client.RegisterCommand(command.MemoryUsage)
	client.RegisterCommand(command.Modal)
	client.RegisterCommand(command.SendFile)
	client.RegisterCommand(command.Static)
	client.RegisterCommand(command.Swap)
	client.RegisterCommand(command.V2Component)
	client.RegisterComponent([]string{"button-hello"}, command.HelloStatic)
	client.RegisterModal("my-modal", command.HelloModal)

	testServerID, err := tempest.StringToSnowflake(os.Getenv("DISCORD_TEST_SERVER_ID")) // Register example commands only to this guild.
	if err != nil {
		log.Fatalln("failed to parse env variable to snowflake", err)
	}

	err = client.SyncCommandsWithDiscord([]tempest.Snowflake{testServerID}, nil, false)
	if err != nil {
		log.Fatalln("failed to sync local commands storage with Discord API", err)
	}
}

// Example implementation how to run bot/app over gateway for lower latency.
func startGateway(ctx context.Context, trace bool) error {
	client := tempest.NewGatewayClient(tempest.GatewayClientOptions{
		Trace: trace,
		BaseClientOptions: tempest.BaseClientOptions{
			Token: os.Getenv("DISCORD_BOT_TOKEN"),
		},
	})

	loadCommands(&client.BaseClient)

	// Spawn recommended amount of shards for your bot/app.
	// You may specify your own intents and then listen to custom events to handle them yourself.
	return client.Gateway.Start(ctx, 0, 0)
}

// Example implementation how to run app as reverse http server for easy scalling.
func startHTTP(addr string, trace bool) error {
	client := tempest.NewHTTPClient(tempest.HTTPClientOptions{
		Trace: trace,
		BaseClientOptions: tempest.BaseClientOptions{
			Token: os.Getenv("DISCORD_BOT_TOKEN"),
		},
		PublicKey: os.Getenv("DISCORD_PUBLIC_KEY"),
	})

	loadCommands(&client.BaseClient)

	http.HandleFunc("POST /", client.DiscordRequestHandler)
	return http.ListenAndServe(addr, nil)
}

func main() {
	log.Println("Loading environmental variables...")
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalln("failed to load env variables", err)
	}

	log.Println("Creating new Tempest client...")

	// Use proper context in real code, this is just an example!
	// if err := startGateway(context.Background(), true); err != nil {
	// 	log.Panicln(err)
	// }

	// Use http(s) reverse server for easy scalling, low maintenance costs, etc. (requires public ip and proper domain setup).
	if err := startHTTP(os.Getenv("APP_ADDRESS"), true); err != nil {
		log.Panicln(err)
	}
}
