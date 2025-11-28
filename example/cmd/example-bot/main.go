package main

import (
	"context"
	"example-bot/internal/command"
	"log"

	// _ "net/http/pprof"
	"os"

	tempest "github.com/amatsagu/tempest"
	godotenv "github.com/joho/godotenv"
)

func main() {
	log.Println("Loading environmental variables...")
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalln("failed to load env variables", err)
	}

	log.Println("Creating new Tempest client...")
	client := tempest.NewGatewayClient(tempest.GatewayClientOptions{
		Trace: true,
		ClientOptions: tempest.ClientOptions{
			Token: os.Getenv("DISCORD_BOT_TOKEN"),
		},
		CustomEventHandler: func(shardID uint16, packet tempest.EventPacket) {
			log.Printf("Received custom gateway packet on shard ID = %d, event = %s\n", shardID, packet.Event)

			raw, _ := packet.Data.MarshalJSON()

			log.Printf("Event's data: %s", string(raw))
		},
	})

	// client := tempest.NewHTTPClient(tempest.HTTPClientOptions{
	// 	Trace: true,
	// 	ClientOptions: tempest.ClientOptions{
	// 		Token: os.Getenv("DISCORD_BOT_TOKEN"),
	// 	},
	// 	PublicKey: "9733f1741174cac2f4ab688ab7f47b3f5eb0576c93ba53f4f01e9099cdb2d8fa",
	// })

	testServerID, err := tempest.StringToSnowflake(os.Getenv("DISCORD_TEST_SERVER_ID")) // Register example commands only to this guild.
	if err != nil {
		log.Fatalln("failed to parse env variable to snowflake", err)
	}

	// Warning!
	// Please make sure you've registered all slash commands & static components before starting http server.
	// Client's registry after starting is used as readonly cache so it skips using mutex for performance reasons.
	// You shouldn't update registry after http server launches.
	log.Println("Registering commands & static components...")
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

	err = client.SyncCommandsWithDiscord([]tempest.Snowflake{testServerID}, nil, false)
	if err != nil {
		log.Fatalln("failed to sync local commands storage with Discord API", err)
	}

	if err := client.Gateway.Start(context.Background(), 0, 32); err != nil {
		log.Fatalln("something went terribly wrong", err)
	}

	// addr := "0.0.0.0:80"
	// http.HandleFunc("POST /", client.DiscordRequestHandler)

	// log.Printf("Serving application at: %s/discord/callback\n", addr)
	// if err := http.ListenAndServe(addr, nil); err != nil {
	// 	log.Fatalln("something went terribly wrong", err)
	// }
}
