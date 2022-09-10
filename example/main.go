package main

import (
	"example-bot/commands"
	"fmt"
	"log"
	"time"

	tempest "github.com/Amatsagu/Tempest"
)

// ==[ CREDENTIALS (FAKE EXAMPLE) ]====================================================================

const AppId tempest.Snowflake = 1003423309444165121
const PublicKey string = "168b4f26de412c4fcaaf7166b58cc234f0746e54b791551de3f90e716624761a"
const Token string = "Bot MTAwMzQyMzMwOTQ0NDE2NTczMw.GrvRPb.jcjBaT74BHU1ay--9-iZNvbN0I_vgvXhkeNnTw"
const Addr string = "0.0.0.0:8080"
const TestGuildId tempest.Snowflake = 957738153442172958

// ====================================================================================================

func main() {
	cmds := 0
	client := tempest.CreateClient(tempest.ClientOptions{
		ApplicationId: AppId,
		PublicKey:     PublicKey,
		Token:         Token,
		PreCommandExecutionHandler: func(itx tempest.CommandInteraction) *tempest.ResponseData {
			cmds += 1
			log.Println("Somebody's running \"" + itx.Data.Name + "\" slash command. That's " + fmt.Sprint(cmds) + " command since app start.")
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

	log.Printf("Starting server at %s", Addr)
	log.Printf("Latency: %dms", client.Ping().Milliseconds())

	client.RegisterCommand(commands.Add)
	client.RegisterCommand(commands.Avatar)
	client.RegisterCommand(commands.Hello)
	client.RegisterCommand(commands.Menu)
	client.RegisterCommand(commands.Statistics)
	client.SyncCommands([]tempest.Snowflake{TestGuildId}, nil, nil)

	if err := client.ListenAndServe(Addr); err != nil {
		panic(err) // Will happen in situation where normal std/http would panic so most likely never.
	}
}
