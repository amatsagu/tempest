package main

import (
	"context"
	"qord"
)

func main() {
	client := qord.NewClient(
		qord.ClientOptions{
			Token: "{{BOT_TOKEN}}",
			Trace: true,
		},
	)

	err := client.Gateway.Start(context.Background(), 0, 0)
	if err != nil {
		panic(err)
	}
}
