package tempest

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"
)

type GatewayClient struct {
	*BaseClient
	Gateway            *ShardManager
	customEventHandler func(shardID uint16, packet EventPacket)
}

type GatewayClientOptions struct {
	BaseClientOptions
	Trace              bool // Whether to enable detailed logging for shard manager and basic client actions.
	CustomEventHandler func(shardID uint16, packet EventPacket)
}

func NewGatewayClient(opt GatewayClientOptions) *GatewayClient {
	client := GatewayClient{
		BaseClient: NewBaseClient(BaseClientOptions{
			Token:                      opt.Token,
			DefaultInteractionContexts: opt.DefaultInteractionContexts,
			PreCommandHook:             opt.PreCommandHook,
			PostCommandHook:            opt.PostCommandHook,
			ComponentHandler:           opt.ComponentHandler,
			ModalHandler:               opt.ModalHandler,
			Logger:                     opt.Logger,
		}),
		customEventHandler: opt.CustomEventHandler,
	}

	if opt.Trace {
		w := client.traceLogger.Writer()
		if w == nil || w == io.Discard {
			client.traceLogger.SetOutput(os.Stdout)
		}
		client.tracef("Gateway Client tracing enabled.")
	}

	client.Gateway = NewShardManager(opt.Token, opt.Trace, client.eventHandler, client.traceLogger)

	return &client
}

func (client *GatewayClient) tracef(format string, v ...any) {
	client.traceLogger.Printf("[(GATEWAY) CLIENT] "+format, v...)
}

// This handler already runs in dedicated goroutine (from shard).
func (client *GatewayClient) eventHandler(shardID uint16, packet EventPacket) {
	if packet.Event != INTERACTION_CREATE_EVENT {
		if client.customEventHandler != nil {
			client.customEventHandler(shardID, packet)
		}
		return
	}

	var extractor InteractionTypeExtractor
	if err := json.Unmarshal(packet.Data, &extractor); err != nil {
		client.tracef("Received interaction event but failed to extract type: %v", err)
		return
	}

	switch extractor.Type {
	case APPLICATION_COMMAND_INTERACTION_TYPE:
		var interaction CommandInteraction
		if err := json.Unmarshal(packet.Data, &interaction); err != nil {
			client.tracef("Received command interaction event but failed to parse its data: %v", err)
			return
		}
		interaction.BaseClient = client.BaseClient
		interaction.GatewayClient = client
		interaction.ShardID = shardID
		interaction.responder = func(res Response) error {
			_, err := client.Rest.Request(http.MethodPost, "/interactions/"+interaction.ID.String()+"/"+interaction.Token+"/callback", res)
			return err
		}

		client.commandInteractionHandler(interaction)
	case MESSAGE_COMPONENT_INTERACTION_TYPE:
		var interaction ComponentInteraction
		if err := json.Unmarshal(packet.Data, &interaction); err != nil {
			client.tracef("Received component interaction event but failed to parse its data: %v", err)
			return
		}

		interaction.BaseClient = client.BaseClient
		interaction.GatewayClient = client
		interaction.ShardID = shardID
		interaction.responder = func(res Response) error {
			_, err := client.Rest.Request(http.MethodPost, "/interactions/"+interaction.ID.String()+"/"+interaction.Token+"/callback", res)
			return err
		}

		client.componentInteractionHandler(interaction)
	case APPLICATION_COMMAND_AUTO_COMPLETE_INTERACTION_TYPE:
		var interaction CommandInteraction
		if err := json.Unmarshal(packet.Data, &interaction); err != nil {
			client.tracef("Received auto complete interaction event but failed to parse its data: %v", err)
			return
		}

		interaction.BaseClient = client.BaseClient
		interaction.GatewayClient = client
		interaction.ShardID = shardID
		interaction.responder = func(res Response) error {
			_, err := client.Rest.Request(http.MethodPost, "/interactions/"+interaction.ID.String()+"/"+interaction.Token+"/callback", res)
			return err
		}

		client.autoCompleteInteractionHandler(interaction)
	case MODAL_SUBMIT_INTERACTION_TYPE:
		var interaction ModalInteraction
		if err := json.Unmarshal(packet.Data, &interaction); err != nil {
			client.tracef("Received modal interaction event but failed to parse its data: %v", err)
			return
		}

		interaction.BaseClient = client.BaseClient
		interaction.GatewayClient = client
		interaction.ShardID = shardID
		interaction.responder = func(res Response) error {
			_, err := client.Rest.Request(http.MethodPost, "/interactions/"+interaction.ID.String()+"/"+interaction.Token+"/callback", res)
			return err
		}

		client.modalInteractionHandler(interaction)
	}
}

func (client *GatewayClient) commandInteractionHandler(interaction CommandInteraction) {
	itx, command, available := client.handleInteraction(interaction)
	if !available {
		client.tracef("Received command interaction but there's no matching command! (requested \"%s\")", itx.Data.Name)
		return
	}

	client.tracef("Received command interaction - moved to target command's handler.")

	if client.preCommandHandler != nil && !client.preCommandHandler(command, &itx) {
		return
	}

	command.SlashCommandHandler(&itx)

	if client.postCommandHandler != nil {
		client.postCommandHandler(command, &itx)
	}
}

func (client *GatewayClient) autoCompleteInteractionHandler(interaction CommandInteraction) {
	itx, command, available := client.handleInteraction(interaction)
	if !available || command.AutoCompleteHandler == nil {
		client.tracef("Dropped auto complete interaction. You see this trace message because client received slash command's auto complete interaction but there's no defined handler for it.")
		return
	}

	client.tracef("Received slash command's auto complete interaction - moved to target (sub) command auto complete handler.")
	choices := command.AutoCompleteHandler(itx)
	err := itx.responder(Response{
		Type: AUTOCOMPLETE_RESPONSE_TYPE,
		Data: &ResponseAutoCompleteData{
			Choices: choices,
		},
	})

	if err != nil {
		client.tracef("Failed to acknowledge auto complete interaction: %v.", err)
	}
}

func (client *GatewayClient) componentInteractionHandler(interaction ComponentInteraction) {
	if fn, ok := client.staticComponents.Get(interaction.Data.CustomID); ok {
		client.tracef("Received component interaction with matching custom ID for static handler - moved to registered handler.")
		fn(interaction)
		return
	}

	handler, isQueued := client.queuedComponents.Get(interaction.Data.CustomID)
	if isQueued && time.Now().After(handler.Expire) {
		isQueued = false
		client.queuedComponents.Delete(interaction.Data.CustomID)
		if handler.OnTimeout != nil {
			go handler.OnTimeout()
		}
	}

	hasGlobal := client.componentHandler != nil

	if isQueued {
		client.tracef("Received component interaction with matching custom ID for dynamic handler - moved to listener.")
		if err := interaction.responder(Response{Type: DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE}); err != nil {
			client.tracef("failed to send deferred update message response: %v", err)
		}
		handler.Handler(&interaction)
		return
	}

	if hasGlobal {
		client.tracef("Received component interaction - moved to defined component handler.")
		client.componentHandler(&interaction)
		return
	}

	client.tracef("Dropped component interaction. You see this trace message because client received component interaction but there's no defined handler for it.")
}

func (client *GatewayClient) modalInteractionHandler(interaction ModalInteraction) {
	if fn, ok := client.staticModals.Get(interaction.Data.CustomID); ok {
		client.tracef("Received modal interaction with matching custom ID for static handler - moved to registered handler.")
		fn(interaction)
		return
	}

	handler, isQueued := client.queuedModals.Get(interaction.Data.CustomID)
	if isQueued && time.Now().After(handler.Expire) {
		isQueued = false
		client.queuedModals.Delete(interaction.Data.CustomID)
		if handler.OnTimeout != nil {
			go handler.OnTimeout()
		}
	}

	hasGlobal := client.modalHandler != nil

	if isQueued {
		client.tracef("Received modal interaction with matching custom ID for dynamic handler - moved to listener.")
		if err := interaction.responder(Response{Type: DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE}); err != nil {
			client.tracef("failed to send deferred update message response: %v", err)
		}
		handler.Handler(&interaction)
		return
	}

	if hasGlobal {
		client.tracef("Received modal interaction - moved to defined modal handler.")
		client.modalHandler(&interaction)
		return
	}

	client.tracef("Dropped modal interaction. You see this trace message because client received modal interaction but there's no defined handler for it.")
}

