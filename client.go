package tempest

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type ClientOptions struct {
	Rest                       rest
	ApplicationId              Snowflake                                                 // Your app/bot's user id.
	PublicKey                  string                                                    // Hash like key used to verify incoming payloads from Discord.
	InteractionHandler         func(interaction Interaction)                             // Function to call on all unhandled interactions.
	PreCommandExecutionHandler func(commandInteraction CommandInteraction) *ResponseData // Function to call after doing initial processing but before executing slash command. Allows to attach own, global logic to all slash commands (similar to routing). Return pointer to ResponseData struct if you want to send messageand stop execution or <nil> to continue.
}

type client struct {
	Rest          rest
	User          User
	ApplicationId Snowflake
	PublicKey     ed25519.PublicKey

	commands                   map[string]map[string]Command                             // Search by command name, then subcommand name (if it's main command then provide "-" as subcommand name)
	interactionHandler         func(interaction Interaction)                             // From options, called on all unhandled interactions.
	preCommandExecutionHandler func(commandInteraction CommandInteraction) *ResponseData // From options, called before each slash command.
	running                    bool                                                      // Whether client's web server is already launched.
}

// Returns time it took to communicate with Discord API (in milliseconds).
func (client client) GetLatency() int64 {
	start := time.Now()
	client.Rest.Request("GET", "/gateway", nil)
	return time.Since(start).Milliseconds()
}

func (client client) SendMessage(channelId Snowflake, content Message) (Message, error) {
	raw, err := client.Rest.Request("POST", "/channels/"+channelId.String()+"/messages", content)
	if err != nil {
		return Message{}, err
	}

	res := Message{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return Message{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

// Use that for simple text messages that won't be modified.
func (client client) SendLinearMessage(channelId Snowflake, content string) (Message, error) {
	raw, err := client.Rest.Request("POST", "/channels/"+channelId.String()+"/messages", Message{Content: content})
	if err != nil {
		return Message{}, err
	}

	res := Message{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return Message{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

func (client client) EditMessage(channelId Snowflake, messageId Snowflake, content Message) error {
	_, err := client.Rest.Request("PATCH", "/channels/"+channelId.String()+"/messages"+messageId.String(), content)
	if err != nil {
		return err
	}
	return nil
}

func (client client) DeleteMessage(channelId Snowflake, messageId Snowflake) error {
	_, err := client.Rest.Request("DELETE", "/channels/"+channelId.String()+"/messages"+messageId.String(), nil)
	if err != nil {
		return err
	}
	return nil
}

func (client client) CrosspostMessage(channelId Snowflake, messageId Snowflake) error {
	_, err := client.Rest.Request("POST", "/channels/"+channelId.String()+"/messages"+messageId.String()+"/crosspost", nil)
	if err != nil {
		return err
	}
	return nil
}

func (client client) FetchUser(id Snowflake) (User, error) {
	raw, err := client.Rest.Request("GET", "/users/"+id.String(), nil)
	if err != nil {
		return User{}, err
	}

	res := User{}
	json.Unmarshal(raw, &res)
	if err != nil {
		return User{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

func (client client) FetchMember(guildId Snowflake, memberId Snowflake) (Member, error) {
	raw, err := client.Rest.Request("GET", "/guilds/"+guildId.String()+"/members/"+memberId.String(), nil)
	if err != nil {
		return Member{}, err
	}

	res := Member{}
	json.Unmarshal(raw, &res)
	if err != nil {
		return Member{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

func (client client) RegisterCommand(command Command) {
	if _, ok := client.commands[command.Name]; !ok {
		if command.Options == nil {
			command.Options = []Option{}
		}

		tree := make(map[string]Command)
		tree["-"] = command
		client.commands[command.Name] = tree
		return
	}

	panic("found already registered \"" + command.Name + "\" slash command")
}

func (client client) RegisterSubCommand(subCommand Command, rootCommandName string) {
	if _, ok := client.commands[rootCommandName]; ok {
		client.commands[rootCommandName][subCommand.Name] = subCommand
		return
	}

	panic("missing \"" + rootCommandName + "\" slash command in registry (register root command first before adding subcommands)")
}

// Sync currently cached slash commands to discord API. By default it'll try to make (bulk) global update (limit 100 updates per day), provide array with guild id snowflakes to update data only for specific guilds.
// You can also add second param -> slice with all command names you want to update (whitelist).
func (client client) SyncCommands(guildIds []Snowflake, commandsToInclude []string) {
	payload := parseCommandsToDiscordObjects(&client, commandsToInclude)

	if len(guildIds) == 0 {
		client.Rest.Request("PUT", "/applications/"+client.ApplicationId.String()+"/commands", payload)
		return
	}

	for _, guildId := range guildIds {
		client.Rest.Request("PUT", "/applications/"+client.ApplicationId.String()+"/guilds/"+guildId.String()+"/commands", payload)
	}
}

func (client client) ListenAndServe(address string) error {
	if client.running {
		panic("client's web server is already launched")
	}

	user, err := client.FetchUser(client.ApplicationId)
	if err != nil {
		panic("failed to fetch bot user's details (check if application id is correct & your internet connection)\n")
	}
	client.User = user

	http.HandleFunc("/", client.handleDiscordWebhookRequests)
	return http.ListenAndServe(address, nil)
}

func CreateClient(options ClientOptions) client {
	discordPublicKey, err := hex.DecodeString(options.PublicKey)
	if err != nil {
		panic("failed to decode \"%s\" discord's public key (check if it's correct key)")
	}

	client := client{
		Rest:               options.Rest,
		ApplicationId:      options.ApplicationId,
		PublicKey:          ed25519.PublicKey(discordPublicKey),
		commands:           make(map[string]map[string]Command, 50), // Allocate space for 50 global slash commands
		interactionHandler: options.InteractionHandler,
		running:            false,
	}

	return client
}

func (client client) handleDiscordWebhookRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed.", http.StatusMethodNotAllowed)
		return
	}

	verified := verifyRequest(r, ed25519.PublicKey(client.PublicKey))
	if !verified {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var interaction Interaction
	err := json.NewDecoder(r.Body).Decode(&interaction)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		panic(err)

	}
	defer r.Body.Close()

	interaction.Client = &client // Bind access to client instance which is needed for methods.
	switch interaction.Type {
	case PING_TYPE:
		// io.WriteString(w, `{"type":1}`)
		w.Write([]byte(`{"type":1}`))
		return
	case APPLICATION_COMMAND_TYPE:
		ctx := CommandInteraction(interaction)
		command := func() Command {
			if len(interaction.Data.Options) != 0 && interaction.Data.Options[0].Type == OPTION_SUB_COMMAND {
				rootName := interaction.Data.Name
				ctx.Data.Name, ctx.Data.Options = interaction.Data.Options[0].Name, interaction.Data.Options[0].Options
				return client.commands[rootName][ctx.Data.Name]
			} else {
				return client.commands[interaction.Data.Name]["-"]
			}
		}()

		if command.Name == "" {
			terminateCommandInteraction(w)
			return
		}

		if ctx.GuildID == 0 && !command.AvailableInDM {
			w.WriteHeader(http.StatusNoContent)
			return // Stop execution since this command doesn't want to be used inside DM.
		}

		if client.preCommandExecutionHandler != nil {
			content := client.preCommandExecutionHandler(ctx)
			if content != nil {
				body, err := json.Marshal(Response{
					Type: ACKNOWLEDGE_WITH_SOURCE_RESPONSE,
					Data: *content,
				})

				if err != nil {
					panic("failed to parse payload received from client's \"pre command execution\" handler (make sure it's in JSON format)")
				}

				w.Header().Add("Content-Type", "application/json")
				w.Write(body)
				return
			}
		}

		w.WriteHeader(http.StatusNoContent)
		command.SlashCommandHandler(ctx)
		return
	default:
		if client.interactionHandler != nil {
			client.interactionHandler(interaction)
		}
	}
}
