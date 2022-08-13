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
	// Please avoid creating raw Rest struct unless you know what you're doing. Use CreateRest function instead.
	Rest                       Rest
	ApplicationId              Snowflake                                                 // Your app/bot's user id.
	PublicKey                  string                                                    // Hash like key used to verify incoming payloads from Discord.
	PreCommandExecutionHandler func(commandInteraction CommandInteraction) *ResponseData // Function to call after doing initial processing but before executing slash command. Allows to attach own, global logic to all slash commands (similar to routing). Return pointer to ResponseData struct if you want to send messageand stop execution or <nil> to continue.
	InteractionHandler         func(interaction Interaction)                             // Function to call on all unhandled interactions.
}

type QueueComponent struct {
	CustomIds []string
	TargetId  Snowflake // User/Member id who can trigger button. If set and button is clicked by someone else - ignore.
	Handler   func(interaction *Interaction)
}

// Please avoid creating raw Client struct unless you know what you're doing. Use CreateClient function instead.
type Client struct {
	Rest          Rest
	User          User
	ApplicationId Snowflake
	PublicKey     ed25519.PublicKey

	commands                   map[string]map[string]Command                             // Search by command name, then subcommand name (if it's main command then provide "-" as subcommand name)
	queuedComponents           map[string]*QueueComponent                                // Map with all currently running button queues.
	preCommandExecutionHandler func(commandInteraction CommandInteraction) *ResponseData // From options, called before each slash command.
	interactionHandler         func(interaction Interaction)                             // From options, called on all unhandled interactions.
	running                    bool                                                      // Whether client's web server is already launched.
}

// Returns time it took to communicate with Discord API (in milliseconds).
func (client Client) GetLatency() int64 {
	start := time.Now()
	client.Rest.Request("GET", "/gateway", nil)
	return time.Since(start).Milliseconds()
}

// Makes client "listen" incoming component type interactions.
// When there's component with custom id matching queued component(s) it'll trigger your handler function.
// Provided function will be also called with <nil> param on timeout.
//
// Warning! Automatically handled components will be already acknowledged by client.
//
// Set timeout equal to 0 to make it last infinitely.
func (client Client) AwaitComponent(queue QueueComponent, timeout time.Duration) {
	if timeout != 0 && time.Second*3 > timeout {
		timeout = time.Second * 3 // Min 3 seconds
	}

	for _, key := range queue.CustomIds {
		client.queuedComponents[key] = &queue
	}

	if timeout == 0 {
		return
	}

	time.AfterFunc(timeout, func() {
		_, exists := client.queuedComponents[queue.CustomIds[0]]
		if !exists {
			return
		}

		for _, key := range queue.CustomIds {
			delete(client.queuedComponents, key)
		}
		queue.Handler(nil)
	})
}

func (client Client) SendMessage(channelId Snowflake, content Message) (Message, error) {
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
func (client Client) SendLinearMessage(channelId Snowflake, content string) (Message, error) {
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

func (client Client) EditMessage(channelId Snowflake, messageId Snowflake, content Message) error {
	_, err := client.Rest.Request("PATCH", "/channels/"+channelId.String()+"/messages"+messageId.String(), content)
	if err != nil {
		return err
	}
	return nil
}

func (client Client) DeleteMessage(channelId Snowflake, messageId Snowflake) error {
	_, err := client.Rest.Request("DELETE", "/channels/"+channelId.String()+"/messages"+messageId.String(), nil)
	if err != nil {
		return err
	}
	return nil
}

func (client Client) CrosspostMessage(channelId Snowflake, messageId Snowflake) error {
	_, err := client.Rest.Request("POST", "/channels/"+channelId.String()+"/messages"+messageId.String()+"/crosspost", nil)
	if err != nil {
		return err
	}
	return nil
}

func (client Client) FetchUser(id Snowflake) (User, error) {
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

func (client Client) FetchMember(guildId Snowflake, memberId Snowflake) (Member, error) {
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

func (client Client) RegisterCommand(command Command) {
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

func (client Client) RegisterSubCommand(subCommand Command, rootCommandName string) {
	if _, ok := client.commands[rootCommandName]; ok {
		client.commands[rootCommandName][subCommand.Name] = subCommand
		return
	}

	panic("missing \"" + rootCommandName + "\" slash command in registry (register root command first before adding subcommands)")
}

// Sync currently cached slash commands to discord API. By default it'll try to make (bulk) global update (limit 100 updates per day), provide array with guild id snowflakes to update data only for specific guilds.
// You can also add second param -> slice with all command names you want to update (whitelist).
func (client Client) SyncCommands(guildIds []Snowflake, commandsToInclude []string) {
	payload := parseCommandsToDiscordObjects(&client, commandsToInclude)

	if len(guildIds) == 0 {
		client.Rest.Request("PUT", "/applications/"+client.ApplicationId.String()+"/commands", payload)
		return
	}

	for _, guildId := range guildIds {
		client.Rest.Request("PUT", "/applications/"+client.ApplicationId.String()+"/guilds/"+guildId.String()+"/commands", payload)
	}
}

func (client Client) ListenAndServe(address string) error {
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

func CreateClient(options ClientOptions) Client {
	discordPublicKey, err := hex.DecodeString(options.PublicKey)
	if err != nil {
		panic("failed to decode \"%s\" discord's public key (check if it's correct key)")
	}

	client := Client{
		Rest:                       options.Rest,
		ApplicationId:              options.ApplicationId,
		PublicKey:                  ed25519.PublicKey(discordPublicKey),
		commands:                   make(map[string]map[string]Command, 50),
		queuedComponents:           make(map[string]*QueueComponent, 25),
		preCommandExecutionHandler: options.PreCommandExecutionHandler,
		interactionHandler:         options.InteractionHandler,
		running:                    false,
	}

	return client
}

func (client Client) handleDiscordWebhookRequests(w http.ResponseWriter, r *http.Request) {
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
		w.Write([]byte(`{"type":1}`))
		return
	case APPLICATION_COMMAND_TYPE:
		command, interaction, exists := client.getCommand(interaction)
		if !exists {
			terminateCommandInteraction(w)
			return
		}

		if !command.AvailableInDM && interaction.GuildId == 0 {
			w.WriteHeader(http.StatusNoContent)
			return // Stop execution since this command doesn't want to be used inside DM.
		}

		ctx := CommandInteraction(interaction)
		if client.preCommandExecutionHandler != nil {
			content := client.preCommandExecutionHandler(ctx)
			if content != nil {
				body, err := json.Marshal(Response{
					Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE,
					Data: content,
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
	case MESSAGE_COMPONENT_TYPE:
		queue, exists := client.queuedComponents[interaction.Data.CustomId]
		var targetId Snowflake

		if interaction.GuildId == 0 {
			targetId = interaction.User.Id
		} else {
			targetId = interaction.Member.User.Id
		}

		if exists && (queue.TargetId == 0 || queue.TargetId == targetId) {
			queue.Handler(&interaction)

			for _, key := range queue.CustomIds {
				delete(client.queuedComponents, key)
			}
		} else if client.interactionHandler != nil {
			client.interactionHandler(interaction)
		}

		acknowledgeComponentInteraction(w)
		return
	case APPLICATION_COMMAND_AUTO_COMPLETE_TYPE:
		command, interaction, exists := client.getCommand(interaction)
		if !exists || command.AutoCompleteHandler == nil || len(command.Options) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		choices := command.AutoCompleteHandler(AutoCompleteInteraction(interaction))
		body, err := json.Marshal(ResponseChoice{
			Type: AUTOCOMPLETE_RESPONSE,
			Data: ResponseChoiceData{
				Choices: choices,
			},
		})

		if err != nil {
			panic("failed to parse payload received from client's \"auto complete\" handler (make sure it's in JSON format)")
		}

		w.Header().Add("Content-Type", "application/json")
		w.Write(body)
		return
	default:
		if client.interactionHandler != nil {
			client.interactionHandler(interaction)
		}
	}
}

// Returns command, subcommand, a command context (updated interaction) and bool to check whether it succeeded and is safe to use.
func (client Client) getCommand(interaction Interaction) (Command, Interaction, bool) {
	if len(interaction.Data.Options) != 0 && interaction.Data.Options[0].Type == OPTION_SUB_COMMAND {
		rootName := interaction.Data.Name
		interaction.Data.Name, interaction.Data.Options = interaction.Data.Options[0].Name, interaction.Data.Options[0].Options
		command, exists := client.commands[rootName][interaction.Data.Name]
		if !exists {
			return Command{}, interaction, false
		}
		return command, interaction, true
	}

	command, exists := client.commands[interaction.Data.Name]["-"]
	if !exists {
		return Command{}, interaction, false
	}

	return command, interaction, true
}
