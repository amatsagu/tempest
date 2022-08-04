package main

import (
	"log"
	. "tempest"
)

const AppId Snowflake = 1003423309444165733
const PublicKey string = "168b4f26de412c4fcaaf7166b58cc234f0746e54b791551de3f55e716624761a"
const Token string = "Bot MTAwMzQyMzMwOTQ0NDE2NTczMw.G1Mx5f.ZnObIzZ9W1mtBEKFary6yB3HsA76XySHo7i4m8"
const Addr string = "0.0.0.0:7788"

func main() {
	client := CreateClient(ClientOptions{
		Rest:          CreateRest(Token, 100),
		ApplicationId: AppId,
		PublicKey:     PublicKey,
	})

	log.Printf("Starting server at %s", Addr)
	log.Printf("Latency: %dms", client.GetLatency())

	command := Command{
		Name:        "test",
		Description: "Replies with hello message! (v3)",
		Options: []Option{
			{
				Name:        "user",
				Description: "User to greet.",
				Type:        OPTION_USER,
				Required:    true,
			},
			{
				Name:        "message",
				Description: "A custom text to send for mentioned user.",
				Type:        OPTION_STRING,
			},
		},
		AvailableInDM: false,
		SlashCommandHandler: func(ctx CommandInteraction) {
			PrettyStructPrint(ctx)
			ctx.SendLinearReply("Hello world! v3", false)
		},
	}

	client.RegisterCommand(command)
	// client.SyncCommands([]Snowflake{957738153442172958}, nil)

	if err := client.ListenAndServe(Addr); err != nil {
		panic(err)
	}
}
