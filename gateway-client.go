package tempest

import (
	"encoding/json"
	"net/http"
	"os"
)

type GatewayClient struct {
	BaseClient
	Gateway            *ShardManager
	customEventHandler func(shardID uint16, packet EventPacket)
}

type GatewayClientOptions struct {
	BaseClientOptions
	PublicKey          string
	Trace              bool // Whether to enable detailed logging for shard manager and basic client actions.
	CustomEventHandler func(shardID uint16, packet EventPacket)
}

func NewGatewayClient(opt GatewayClientOptions) GatewayClient {
	client := GatewayClient{
		BaseClient: NewBaseClient(BaseClientOptions{
			Token:                      opt.Token,
			DefaultInteractionContexts: opt.DefaultInteractionContexts,
			PreCommandHook:             opt.PreCommandHook,
			PostCommandHook:            opt.PostCommandHook,
			ComponentHandler:           opt.ComponentHandler,
			ModalHandler:               opt.ModalHandler,
		}),
		customEventHandler: opt.CustomEventHandler,
	}

	client.interactionResponder = func(itx *Interaction, resp Response) error {
		// Gateway always responds via rest api.
		_, err := client.Rest.Request(http.MethodPost, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", resp)
		return err
	}

	client.Gateway = NewShardManager(opt.Token, opt.Trace, client.eventHandler)

	if opt.Trace {
		client.traceLogger.SetOutput(os.Stdout)
		client.tracef("Gateway Client tracing enabled.")
	}

	return client
}

func (m *GatewayClient) tracef(format string, v ...any) {
	m.traceLogger.Printf("[(GATEWAY) CLIENT] "+format, v...)
}

func (client *GatewayClient) eventHandler(shardID uint16, packet EventPacket) {
	if packet.Event != INTERACTION_CREATE_EVENT {
		if client.customEventHandler != nil {
			client.customEventHandler(shardID, packet)
		}
		return
	}

	var interaction Interaction
	if err := json.Unmarshal(packet.Data, &interaction); err != nil {
		client.tracef("Received interaction event but failed to parse it: %v", err)
		return
	}

	interaction.ShardID = shardID
	interaction.Client = &client.BaseClient

	switch interaction.Type {
	case APPLICATION_COMMAND_INTERACTION_TYPE:
		var data CommandInteractionData
		if err := json.Unmarshal(interaction.Data, &data); err != nil {
			client.tracef("Received command interaction event but failed to parse its data: %v", err)
			return
		}

		client.commandInteractionHandler(CommandInteraction{
			Interaction: &interaction,
			Data:        data,
		})
	case MESSAGE_COMPONENT_INTERACTION_TYPE:
		var componentData ComponentInteractionData
		if err := json.Unmarshal(interaction.Data, &componentData); err != nil {
			client.tracef("Received component interaction event but failed to parse its data: %v", err)
			return
		}

		client.componentInteractionHandler(ComponentInteraction{
			Interaction: &interaction,
			Data:        componentData,
		})
	case APPLICATION_COMMAND_AUTO_COMPLETE_INTERACTION_TYPE:
		var commandData CommandInteractionData
		if err := json.Unmarshal(interaction.Data, &commandData); err != nil {
			client.tracef("Received auto complete interaction event but failed to parse its data: %v", err)
			return
		}

		cmdInteraction := CommandInteraction{
			Interaction: &interaction,
			Data:        commandData,
		}

		_, command, available := client.handleInteraction(cmdInteraction)
		if !available {
			client.tracef("Dropped auto complete interaction. You see this trace message because client received slash command's auto complete interaction but there's no defined handler for it.")
			return
		}

		// Auto complete is time sensitive, run it in goroutine
		go func() {
			choices := command.AutoCompleteHandler(cmdInteraction)
			err := cmdInteraction.Client.interactionResponder(cmdInteraction.Interaction, Response{
				Type: AUTOCOMPLETE_RESPONSE_TYPE,
				Data: &ResponseAutoCompleteData{
					Choices: choices,
				},
			})

			if err != nil {
				client.tracef("failed to acknowledge auto complete interaction: %v", err)
			}
		}()

	case MODAL_SUBMIT_INTERACTION_TYPE:
		var modalData ModalInteractionData
		if err := json.Unmarshal(interaction.Data, &modalData); err != nil {
			client.tracef("Received modal interaction event but failed to parse its data: %v", err)
			return
		}

		client.modalInteractionHandler(ModalInteraction{
			Interaction: &interaction,
			Data:        modalData,
		})
	}
}

func (client *GatewayClient) commandInteractionHandler(interaction CommandInteraction) {
	itx, command, available := client.handleInteraction(interaction)
	if !available {
		client.tracef("Received command interaction but there's no matching command! (requested \"%s\")", interaction.Data.Name)
		return
	}

	itx.Client = &client.BaseClient
	client.tracef("Received command interaction - moved to target command's handler.")

	// Run command handler in goroutine
	go func() {
		if client.preCommandHandler != nil && !client.preCommandHandler(command, &itx) {
			return
		}

		command.SlashCommandHandler(&itx)

		if client.postCommandHandler != nil {
			client.postCommandHandler(command, &itx)
		}
	}()
}

func (client *GatewayClient) componentInteractionHandler(interaction ComponentInteraction) {
	if fn, ok := client.staticComponents.Get(interaction.Data.CustomID); ok {
		client.tracef("Received component interaction with matching custom ID for static handler - moved to registered handler.")
		go fn(interaction)
		return
	}

	isQueued := client.queuedComponents.Has(interaction.Data.CustomID)
	hasGlobal := client.componentHandler != nil

	if isQueued || hasGlobal {
		if isQueued {
			client.tracef("Received component interaction with matching custom ID for dynamic handler - moved to listener.")
		} else {
			client.tracef("Received component interaction - moved to defined component handler.")
		}

		go func() {
			interaction.Client.interactionResponder(interaction.Interaction, Response{Type: DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE})

			if isQueued {
				client.queuedComponents.mu.RLock()
				if signalChan, ok := client.queuedComponents.cache[interaction.Data.CustomID]; ok {
					select {
					case signalChan <- &interaction:
					default:
					}
					client.queuedComponents.mu.RUnlock()
					return
				}
				client.queuedComponents.mu.RUnlock()
			}

			if client.componentHandler != nil {
				client.componentHandler(&interaction)
			}
		}()
		return
	}

	client.tracef("Dropped component interaction. You see this trace message because client received component interaction but there's no defined handler for it.")
}

func (client *GatewayClient) modalInteractionHandler(interaction ModalInteraction) {
	if fn, ok := client.staticModals.Get(interaction.Data.CustomID); ok {
		client.tracef("Received modal interaction with matching custom ID for static handler - moved to registered handler.")
		go fn(interaction)
		return
	}

	isQueued := client.queuedModals.Has(interaction.Data.CustomID)
	hasGlobal := client.modalHandler != nil

	if isQueued || hasGlobal {
		if isQueued {
			client.tracef("Received modal interaction with matching custom ID for dynamic handler - moved to listener.")
		} else {
			client.tracef("Received modal interaction - moved to defined modal handler.")
		}

		go func() {
			interaction.Client.interactionResponder(interaction.Interaction, Response{Type: DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE})

			if isQueued {
				client.queuedModals.mu.RLock()
				if signalChan, ok := client.queuedModals.cache[interaction.Data.CustomID]; ok {
					select {
					case signalChan <- &interaction:
					default:
					}
					client.queuedModals.mu.RUnlock()
					return
				}
				client.queuedModals.mu.RUnlock()
			}

			if client.modalHandler != nil {
				client.modalHandler(&interaction)
			}
		}()
		return
	}

	client.tracef("Dropped modal interaction. You see this trace message because client received modal interaction but there's no defined handler for it.")
}
