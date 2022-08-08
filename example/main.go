package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	. "github.com/Amatsagu/Tempest"
)

// ==[ CREDENTIALS (EXAMPLE) ]====================================================================

const AppId Snowflake = 1003423300000000000
const PublicKey string = "168b4f26de412c4fcaaf7166b58cc234f0746e54b791551de000000000000000"
const Token string = "Bot MTAwMz0yMzMwOTQ0NDE2NTczMw.GQncL8.PL_MWQfx21dxz4dVmBQ-H2CNGV000000000090"
const Addr string = "0.0.0.0:8080"

// ===============================================================================================

func main() {
	startedAt := time.Now()
	cmds := 0

	client := CreateClient(ClientOptions{
		Rest:          CreateRest(Token, 100),
		ApplicationId: AppId,
		PublicKey:     PublicKey,
		PreCommandExecutionHandler: func(commandInteraction CommandInteraction) *ResponseData {
			cmds += 1
			log.Println("Somebody's running \"" + commandInteraction.Data.Name + "\" slash command. That's " + fmt.Sprint(cmds) + " command since app start.")
			return nil
		},
	})

	log.Printf("Starting server at %s", Addr)
	log.Printf("Latency: %dms", client.GetLatency())

	helloCommand := Command{
		Name:        "hello",
		Description: "Replies with hello message!",
		Options: []Option{
			{
				Name:        "user",
				Description: "User to greet.",
				Type:        OPTION_USER,
				// Required:    true,
			},
		},
		SlashCommandHandler: func(ctx CommandInteraction) {
			raw, available := ctx.GetOptionValue("user")
			value, valid := raw.(string)

			if available && valid {
				user, err := ctx.Client.FetchUser(StringToSnowflake(value))
				if err != nil {
					ctx.SendLinearReply(err.Error(), false)
				}

				ctx.SendLinearReply("Hello "+user.Tag(), false)
			} else {
				ctx.SendLinearReply("Hello "+ctx.Member.User.Tag(), false)
			}
		},
	}

	avatarCommand := Command{
		Name:        "avatar",
		Description: "Returns link to member's avatar!",
		Options: []Option{
			{
				Name:        "user",
				Description: "User to steal avatar from.",
				Type:        OPTION_USER,
				Required:    true,
			},
		},
		SlashCommandHandler: func(ctx CommandInteraction) {
			raw, _ := ctx.GetOptionValue("user")
			value, valid := raw.(string)
			if !valid {
				ctx.SendLinearReply("Failed to cast raw user id value to string data type.", false)
			}

			user, err := ctx.Client.FetchUser(StringToSnowflake(value))
			if err != nil {
				ctx.SendLinearReply(err.Error(), false)
			}

			avatar := user.FetchAvatarUrl()
			embed := Embed{
				Title: user.Tag() + " avatar",
				Url:   avatar,
				Image: &EmbedImage{
					Url: avatar,
				},
			}

			ctx.SendReply(ResponseData{
				Embeds: []*Embed{&embed},
			}, false)
		},
	}

	runtimeCommand := Command{
		Name:        "runtime",
		Description: "Example description...",
	}

	statsCommand := Command{
		Name:        "stats",
		Description: "Displays basic runtime statistics.",
		SlashCommandHandler: func(ctx CommandInteraction) {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			reply := fmt.Sprintf(`
	Current memory usage: %.2fMB
	 => Heap usage: %.2fMB (Allocated: %.2fMB)
	 => Stack usage: %.2fMB (Allocated: %.2fMB)

	Total system allocated memory: %.2fMB
	GC cycles: %d
	Uptime: %.2f minute(s)`, mb(m.Alloc), mb(m.HeapInuse), mb(m.HeapSys), mb(m.StackInuse), mb(m.StackSys), mb(m.Sys), m.NumGC, time.Since(startedAt).Minutes())

			ctx.SendLinearReply(reply, false)
		},
	}

	sweepCommand := Command{
		Name:        "sweep",
		Description: "Force starts GC sweep.",
		SlashCommandHandler: func(ctx CommandInteraction) {
			runtime.GC()
			ctx.SendLinearReply("Finished!", false)
		},
	}

	askCommand := Command{
		Name:        "ask",
		Description: "Replies with answer!",
		Options: []Option{
			{
				Name:         "question",
				Description:  "Question to ask.",
				Type:         OPTION_STRING,
				AutoComplete: true,
				Required:     true,
			},
		},
		AutoCompleteHandler: func(ctx AutoCompleteInteraction) []Choice {
			_, rawValue := ctx.GetFocusedValue()
			value := rawValue.(string)
			if len(value) == 0 {
				return []Choice{}
			}

			data := []Choice{
				{
					Name:  "Result of 1 + 1",
					Value: "2",
				},
				{
					Name:  "Color from combination of red & blue",
					Value: "Purple",
				},
				{
					Name:  "You typed: " + value,
					Value: value,
				},
			}

			return data
		},
		SlashCommandHandler: func(ctx CommandInteraction) {
			raw, _ := ctx.GetOptionValue("question")
			value, _ := raw.(string)
			buttonId := ctx.Id.String() + "_example" // Some unique id to filter for later. It's recommended to use id or token of interaction because it's always unique.

			ctx.SendReply(ResponseData{
				Content: "Response: " + value,
				Components: []*Component{
					{
						Type: COMPONENT_ROW,
						Components: []*Component{
							{
								CustomId: buttonId,
								Type:     COMPONENT_BUTTON,
								Style:    BUTTON_PRIMARY,
								Label:    "Click me!",
							},
						},
					},
				},
			}, false)

			ctx.Client.AwaitComponent(QueueComponent{
				CustomIds: []string{buttonId},
				TargetId:  ctx.Member.User.Id,
				Handler: func(btx *Interaction) {
					if btx == nil {
						ctx.SendFollowUp(ResponseData{Content: "You haven't clicked button within last 5 minutes!"}, false)
						return
					}

					ctx.SendFollowUp(ResponseData{
						Content: "Successfully clicked button within 5 minutes! Button Component id: " + btx.Data.CustomId,
					}, false)
				},
			}, time.Minute*5)
		},
	}

	selectCommand := Command{
		Name:        "select",
		Description: "Another example slash command to test select menu handler.",
		SlashCommandHandler: func(ctx CommandInteraction) {
			selectMenuId := ctx.Id.String() + "_select" // Some unique id to filter for later. It's recommended to use id or token of interaction because it's always unique.

			ctx.SendReply(ResponseData{
				Content: "Example text",
				Components: []*Component{
					{
						Type: COMPONENT_ROW,
						Components: []*Component{
							{
								Type:     COMPONENT_SELECT_MENU,
								CustomId: selectMenuId,
								Options: []*SelectMenuOption{
									{
										Label:       "First",
										Description: "Example description...",
										Value:       "First",
									},
									{
										Label: "Second",
										Value: "Second",
									},
								},
							},
						},
					},
				},
			}, false)

			ctx.Client.AwaitComponent(QueueComponent{
				CustomIds: []string{selectMenuId},
				TargetId:  ctx.Member.User.Id,
				Handler: func(stx *Interaction) {
					if stx == nil {
						ctx.SendFollowUp(ResponseData{Content: "You haven't selected any option from 30 seconds!"}, false)
						return
					}

					ctx.EditReply(ResponseData{
						Content: "Received list selection event!",
					}, false)
				},
			}, time.Second*30)
		},
	}

	client.RegisterCommand(helloCommand)
	client.RegisterCommand(avatarCommand)
	client.RegisterCommand(runtimeCommand)
	client.RegisterCommand(askCommand)
	client.RegisterCommand(selectCommand)
	client.RegisterSubCommand(statsCommand, "runtime")
	client.RegisterSubCommand(sweepCommand, "runtime")
	client.SyncCommands([]Snowflake{957738153442172958}, nil)

	if err := client.ListenAndServe(Addr); err != nil {
		panic(err) // Will happen in situation where normal std/http would panic so most likely never.
	}
}

func mb(value uint64) float64 {
	return float64(value) / 1024.0 / 1024.0
}
