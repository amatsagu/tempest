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

type pollCommand struct{}

func newPollCommand() *pollCommand { return &pollCommand{} }

func (c *pollCommand) Definition() tempest.Command {
	return tempest.Command{
		Command: discord.Command{
			Name:        "poll",
			Description: "Create a native Discord poll via tempest/builder with an owner close button.",
			Options: []discord.CommandOption{
				{
					Type:        discord.StringOptionType,
					Name:        "question",
					Description: "Poll question.",
					Required:    true,
				},
				{
					Type:        discord.StringOptionType,
					Name:        "options",
					Description: "Comma-separated answer choices (defaults to Yes, No).",
				},
				{
					Type:        discord.IntegerOptionType,
					Name:        "hours",
					Description: "Duration in hours (1-168, default 24).",
					MinValue:    util.Ptr(1.0),
					MaxValue:    util.Ptr(168.0),
				},
			},
		},
		SlashCommandHandler: c.handle,
	}
}

func (c *pollCommand) handle(ctx context.Context, itx *tempest.CommandInteraction) error {
	question, _ := util.StringOption(itx, "question")
	rawOptions, _ := util.StringOption(itx, "options")
	hoursVal, ok := util.IntegerOption(itx, "hours")
	if !ok || hoursVal < 1 {
		hoursVal = 24
	}

	var answers []discord.PollAnswer
	for part := range strings.SplitSeq(rawOptions, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			answers = append(answers, builder.PollAnswer(trimmed))
		}
		if len(answers) == 10 {
			break
		}
	}
	if len(answers) < 2 {
		answers = []discord.PollAnswer{builder.PollAnswer("Yes"), builder.PollAnswer("No")}
	}

	poll := builder.Poll(question, uint16(hoursVal), answers...)
	owner := itx.BaseUser()
	ownerID := "0"
	if owner != nil {
		ownerID = owner.ID.String()
	}

	controls := builder.ActionRow(
		builder.Button("poll/"+ownerID+"/close", "Close Poll Control", discord.DangerButtonStyle),
	)

	// Verify FindInteractiveComponent works on constructed component slices cleanly.
	if btn, found := tempest.FindInteractiveComponent([]discord.MessageComponent{controls}, func(b discord.ButtonComponent) bool {
		return strings.HasPrefix(b.CustomID, "poll/")
	}); !found || btn.CustomID == "" {
		return fmt.Errorf("missing poll close control")
	}

	return itx.SendReply(ctx, discord.ResponseMessageData{
		Poll:       &poll,
		Components: []discord.MessageComponent{controls},
	}, false, nil)
}

func (c *pollCommand) handleClose(ctx context.Context, itx *tempest.ComponentInteraction) error {
	ownerID := itx.PathValue("owner_id")
	caller := itx.BaseUser()
	if ownerID != "0" && (caller == nil || caller.ID.String() != ownerID) {
		return itx.AcknowledgeWithLinearMessage(ctx, "Only the poll creator can close this control.", true)
	}

	return itx.AcknowledgeWithUpdate(ctx, discord.ResponseMessageData{
		Content:    "Poll control closed by owner.",
		Components: []discord.MessageComponent{},
	})
}
