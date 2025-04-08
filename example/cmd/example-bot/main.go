package main

import (
	"example-bot/internal/command"
	"log"
	"net/http"
	_ "net/http/pprof"
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
	client := tempest.NewClient(tempest.ClientOptions{
		Token:          os.Getenv("DISCORD_BOT_TOKEN"),
		PublicKey:      os.Getenv("DISCORD_PUBLIC_KEY"),
		JSONBufferSize: 4096,
		DefaultInteractionContexts: []tempest.InteractionContextType{
			tempest.GUILD_CONTEXT_TYPE,
		},
	})

	addr := os.Getenv("DISCORD_APP_ADDRESS")
	testServerID, err := tempest.StringToSnowflake(os.Getenv("DISCORD_TEST_SERVER_ID")) // Register example commands only to this guild.
	if err != nil {
		log.Fatalln("failed to parse env variable to snowflake", err)
	}

	// Warning!
	// Please make sure you've registered all slash commands & static components before starting http server.
	// Client's registry after starting is used as readonly cache so it skips using mutex for performance reasons.
	// You shouldn't update registry after http server launches.
	log.Println("Registering commands & static components...")
	client.CommandRegistry.Register(command.AddSlashCommand{})
	client.CommandRegistry.Register(command.AutoCompleteSlashCommand{})
	client.CommandRegistry.Register(command.AvatarSlashCommand{})
	client.CommandRegistry.Register(command.SwapSlashCommand{})

	err = client.CommandRegistry.SyncWithDiscord(client.Rest, []tempest.Snowflake{testServerID}, nil, false)
	if err != nil {
		log.Fatalln("failed to sync local commands storage with Discord API", err)
	}

	http.HandleFunc("POST /discord/callback", client.DiscordRequestHandler)

	log.Printf("Serving application at: %s/discord/callback\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalln("something went terribly wrong ", err)
	}
}
