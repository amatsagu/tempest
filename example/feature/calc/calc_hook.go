package calc

import (
	"context"
	"fmt"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/route"
)

// Non-numeric keys are words so they cannot collide with the pattern separators.
const (
	keyBack   = "back"
	keyClear  = "C"
	keyEquals = "="
)

// calcButton builds one key, packing its session and key into the registered pattern.
func calcButton(session, label, key string, style discord.ButtonStyle) discord.Button {
	return builder.Button(fmt.Sprintf("calc:%s:%s", session, key), label, style)
}

func calcComponents(session string) discord.Components {
	return discord.Components{
		builder.ActionRow(
			calcButton(session, "7", "7", discord.ButtonSecondary),
			calcButton(session, "8", "8", discord.ButtonSecondary),
			calcButton(session, "9", "9", discord.ButtonSecondary),
			calcButton(session, "÷", "÷", discord.ButtonPrimary),
		),
		builder.ActionRow(
			calcButton(session, "4", "4", discord.ButtonSecondary),
			calcButton(session, "5", "5", discord.ButtonSecondary),
			calcButton(session, "6", "6", discord.ButtonSecondary),
			calcButton(session, "×", "×", discord.ButtonPrimary),
		),
		builder.ActionRow(
			calcButton(session, "1", "1", discord.ButtonSecondary),
			calcButton(session, "2", "2", discord.ButtonSecondary),
			calcButton(session, "3", "3", discord.ButtonSecondary),
			calcButton(session, "-", "-", discord.ButtonPrimary),
		),
		builder.ActionRow(
			calcButton(session, ".", ".", discord.ButtonSecondary),
			calcButton(session, "0", "0", discord.ButtonSecondary),
			calcButton(session, "C", keyClear, discord.ButtonDanger),
			calcButton(session, "+", "+", discord.ButtonPrimary),
		),
		builder.ActionRow(
			calcButton(session, "⌫", keyBack, discord.ButtonSecondary),
			calcButton(session, "=", keyEquals, discord.ButtonSuccess),
		),
	}
}

// calcData renders a session's display and keypad for either Reply or Update.
func calcData(session, display string) discord.ResponseMessageData {
	return discord.ResponseMessageData{
		Content:    display,
		Components: calcComponents(session),
	}
}

// handleButton mutates the session in the custom ID and refreshes the message; a
// missing session answers with an ephemeral hint.
func (c *calcCommand) handleButton(_ context.Context, itx *discord.ComponentInteraction, params *route.Params) error {
	session, _ := params.Get("session")
	key, _ := params.Get("key")

	state, ok := c.press(session, key)
	if !ok {
		return itx.Reply(discord.ResponseMessageData{Content: "This calculator expired. Run /calc again."}, true)
	}
	return itx.Update(calcData(session, state.display))
}
