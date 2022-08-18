package main

import (
	"example-bot/commands"
	"fmt"
	"log"

	. "github.com/Amatsagu/Tempest"
)

// ==[ CREDENTIALS (EXAMPLE) ]====================================================================

const AppId Snowflake = 1003423309444165733
const PublicKey string = "168b4f26de412c4fcaaf7166b58cc234f0746e54b791551de3f55e716624761a"
const Token string = "Bot MTAwMzQyMzMwOTQ0NDE2NTczMw.GhjosP.rkwPdZnNICM7xuCQCX6C-cf5D1SoySXzYVWaRk"
const Addr string = "0.0.0.0:8080"

// ===============================================================================================

func main() {
	cmds := 0
	client := CreateClient(ClientOptions{
		ApplicationId:          AppId,
		PublicKey:              PublicKey,
		Token:                  Token,
		GlobalRequestLimit:     50,
		MaxRequestsBeforeSweep: 50,
		PreCommandExecutionHandler: func(commandInteraction CommandInteraction) *ResponseData {
			cmds += 1
			log.Println("Somebody's running \"" + commandInteraction.Data.Name + "\" slash command. That's " + fmt.Sprint(cmds) + " command since app start.")
			return nil
		},
	})

	log.Printf("Starting server at %s", Addr)
	log.Printf("Latency: %dms", client.Ping().Milliseconds())

	client.RegisterCommand(commands.Add)
	client.RegisterCommand(commands.Avatar)
	client.RegisterCommand(commands.Hello)
	client.RegisterCommand(commands.Menu)
	client.RegisterCommand(commands.Statistics)
	client.SyncCommands([]Snowflake{957738153442172958}, nil)

	if err := client.ListenAndServe(Addr); err != nil {
		panic(err) // Will happen in situation where normal std/http would panic so most likely never.
	}
}

func mb(value uint64) float64 {
	return float64(value) / 1024.0 / 1024.0
}
