package patterns

import (
	"context"
	"fmt"
	"strings"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

const (
	actionHelpID             = "act:help"
	actionReasonField        = "act:modal:reason"
	actionAccent      uint32 = 0x5865F2
)

type actionCommand struct {
	presets []string
}

func newActionCommand() *actionCommand {
	return &actionCommand{
		presets: []string{"Spamming links", "Off-topic flood", "Repeated warnings", "Automated review"},
	}
}

func (c *actionCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:        "action",
		Description: "Open an interactive action panel showcasing custom_id pattern matching.",
		Options: []discord.CommandOption{
			{
				Type:        discord.UserOptionType,
				Name:        "target",
				Description: "Target user ID packed into component/modal pattern routes.",
				Required:    true,
			},
			{
				Type:         discord.StringOptionType,
				Name:         "note",
				Description:  "Optional audit note trail passed through a {trail...} catch-all route.",
				AutoComplete: true,
			},
		},
		AutoCompleteHandler: c.autocomplete,
		SlashCommandHandler: c.handle,
	}
}

func (c *actionCommand) autocomplete(_ context.Context, itx *tempest.CommandInteraction) ([]discord.CommandOptionChoice, error) {
	typed := strings.ToLower(util.FocusedString(itx))

	choices := make([]discord.CommandOptionChoice, 0, len(c.presets))
	for _, preset := range c.presets {
		if strings.Contains(strings.ToLower(preset), typed) {
			choices = append(choices, discord.CommandOptionChoice{Name: preset, Value: preset})
		}
	}
	return choices, nil
}

func (c *actionCommand) handle(ctx context.Context, itx *tempest.CommandInteraction) error {
	targetID, ok := util.SnowflakeOption(itx, "target")
	if !ok {
		return itx.SendLinearReply(ctx, "Pick a target user.", true)
	}

	note, _ := util.StringOption(itx, "note")
	if strings.TrimSpace(note) == "" {
		note = "audit/case-1"
	}

	targetStr := targetID.String()
	uid := discord.UserID{Snowflake: targetID}

	card := builder.Container(actionAccent,
		builder.TextDisplay("## Action Pattern Panel"),
		builder.TextDisplay("Target: "+uid.Mention()+"\nButtons encode parameters directly inside `custom_id` without per-instance memory maps."),
		builder.Separator(true),
		builder.ActionRow(
			builder.Button("act/"+targetStr+"/warn", "Warn (Modal)", discord.PrimaryButtonStyle),
			builder.Button("act/"+targetStr+"/mute/10m", "Mute 10m", discord.SecondaryButtonStyle),
			builder.Button("act/"+targetStr+"/inspect", "Inspect", discord.SecondaryButtonStyle),
			builder.Button("act:note:"+note, "Audit Trail", discord.SecondaryButtonStyle),
			builder.Button(actionHelpID, "Help", discord.SuccessButtonStyle),
		),
	)

	return itx.SendReply(ctx, discord.ResponseMessageData{
		Flags:      discord.IsComponentsV2MessageFlag,
		Components: []discord.MessageComponent{card},
	}, false, nil)
}

func (c *actionCommand) handleHelp(ctx context.Context, itx *tempest.ComponentInteraction) error {
	return itx.AcknowledgeWithLinearMessage(ctx,
		"Routes registered here:\n- `act:help` (exact O(1))\n- `act/{user_id}/mute/{duration}` (2-wildcard specific)\n- `act/{user_id}/{verb}` (single-wildcard fallback)\n- `act:note:{trail...}` (catch-all tail)\n- Modal `act:modal:{user_id}:{kind}`",
		true,
	)
}

func (c *actionCommand) handleMute(ctx context.Context, itx *tempest.ComponentInteraction) error {
	userID := itx.PathValue("user_id")
	duration, _ := itx.Param("duration")

	return itx.AcknowledgeWithUpdate(ctx, discord.ResponseMessageData{
		Flags: discord.IsComponentsV2MessageFlag,
		Components: []discord.MessageComponent{
			builder.TextDisplay(fmt.Sprintf("Matched `act/{user_id}/mute/{duration}` -> muted <@%s> for **%s**.", userID, duration)),
		},
	})
}

func (c *actionCommand) handleVerb(ctx context.Context, itx *tempest.ComponentInteraction) error {
	userID := itx.PathValue("user_id")
	verb := itx.PathValue("verb")

	if verb == "warn" {
		modal := builder.ModalData(
			"act:modal:"+userID+":warn",
			"Warn User "+userID,
			builder.Label("Reason", builder.ShortTextInput(actionReasonField, "Enter warning reason...")),
		)
		return itx.AcknowledgeWithModal(ctx, modal)
	}

	return itx.AcknowledgeWithLinearMessage(ctx, fmt.Sprintf("Matched `act/{user_id}/{verb}` -> action=%q target=<@%s>.", verb, userID), true)
}

func (c *actionCommand) handleNoteTrail(ctx context.Context, itx *tempest.ComponentInteraction) error {
	trail := itx.PathValue("trail")
	return itx.AcknowledgeWithLinearMessage(ctx, fmt.Sprintf("Matched catch-all `{trail...}` -> %q.", trail), true)
}

func (c *actionCommand) handleModalSubmit(ctx context.Context, itx *tempest.ModalInteraction) error {
	userID := itx.PathValue("user_id")
	kind := itx.PathValue("kind")
	reason := strings.TrimSpace(itx.GetInputValue(actionReasonField))
	if reason == "" {
		reason = "(no reason provided)"
	}

	return itx.AcknowledgeWithLinearMessage(ctx, fmt.Sprintf("Submitted modal `act:modal:{user_id}:{kind}` -> kind=%q user=<@%s> reason=%q.", kind, userID, reason), true)
}
