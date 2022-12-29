package main

import (
	"example-bot/commands"
	"example-bot/other"
	"fmt"
	"log"
	"os"
	"time"

	tempest "github.com/Amatsagu/Tempest"
	godotenv "github.com/joho/godotenv"
)

func ensureValue(key string) string {
	if value, available := os.LookupEnv(key); available {
		return value
	}

	other.FormatError(fmt.Errorf("failed to obtain environmental value using \"%s\" key", key))
	os.Exit(0)
	return "" // never reaches
}

func main() {
	if err := godotenv.Load(".env"); err != nil {
		other.FormatError(err)
		os.Exit(1)
	}

	client := tempest.CreateClient(tempest.ClientOptions{
		ApplicationID: tempest.StringToSnowflake(ensureValue("DISCORD_APP_ID")),
		PublicKey:     ensureValue("DISCORD_PUBLIC_KEY"),
		Token:         "Bot " + ensureValue("DISCORD_BOT_TOKEN"),
		PreCommandExecutionHandler: func(itx tempest.CommandInteraction) *tempest.ResponseData {
			commands.CommandCounter++
			log.Printf("%s (%d) uses %s slash command (that's %d executed command since app start)\n", itx.Member.User.Tag(), itx.Member.User.ID, itx.Data.Name, commands.CommandCounter)
			return nil
		},
		Cooldowns: &tempest.ClientCooldownOptions{
			Duration:  time.Second * 3,
			Ephemeral: true,
			CooldownResponse: func(user tempest.User, timeLeft time.Duration) tempest.ResponseData {
				return tempest.ResponseData{
					Content: fmt.Sprintf("You're still on cooldown! Try again in **%.2fs**.", timeLeft.Seconds()),
				}
			},
		},
	})

	addr := fmt.Sprintf("0.0.0.0:%s", ensureValue("DISCORD_APP_PORT"))
	experimentalServerID := tempest.StringToSnowflake(ensureValue("DISCORD_EXPERIMENTAL_SERVER_ID"))

	client.RegisterCommand(commands.Add)
	client.RegisterCommand(commands.Avatar)
	client.RegisterCommand(commands.Hello)
	client.RegisterCommand(commands.ButtonMenu)
	client.RegisterCommand(commands.SelectMenu)
	client.RegisterCommand(commands.Statistics)
	client.SyncCommands([]tempest.Snowflake{experimentalServerID}, nil, false)

	log.Printf("Starting application at %s", addr)
	log.Printf("Latency: %dms", client.Ping().Milliseconds())

	if err := client.ListenAndServe("/", addr); err != nil {
		// Will happen in situation where normal std/http would panic so most likely never.
		other.FormatError(err)
		os.Exit(1)
	}
}
