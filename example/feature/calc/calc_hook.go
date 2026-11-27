package calc

import (
	"fmt"

	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

// Non-numeric keys are words so they cannot collide with the ID separators.
const (
	keyBack   = "back"
	keyClear  = "C"
	keyEquals = "="
)

// calcKeys is the keypad order used both for the message and for AwaitComponent.
var calcKeys = []string{
	"7", "8", "9", "÷",
	"4", "5", "6", "×",
	"1", "2", "3", "-",
	".", "0", keyClear, "+",
	keyBack, keyEquals,
}

// calcButton builds one key, packing its session and key into its custom ID.
func calcButton(session, label, key string, style discord.ButtonStyle) discord.ButtonComponent {
	return util.Button(fmt.Sprintf("calc:%s:%s", session, key), label, style)
}

func calcComponents(session string) []discord.MessageComponent {
	return []discord.MessageComponent{
		util.ActionRow(
			calcButton(session, "7", "7", discord.SecondaryButtonStyle),
			calcButton(session, "8", "8", discord.SecondaryButtonStyle),
			calcButton(session, "9", "9", discord.SecondaryButtonStyle),
			calcButton(session, "÷", "÷", discord.PrimaryButtonStyle),
		),
		util.ActionRow(
			calcButton(session, "4", "4", discord.SecondaryButtonStyle),
			calcButton(session, "5", "5", discord.SecondaryButtonStyle),
			calcButton(session, "6", "6", discord.SecondaryButtonStyle),
			calcButton(session, "×", "×", discord.PrimaryButtonStyle),
		),
		util.ActionRow(
			calcButton(session, "1", "1", discord.SecondaryButtonStyle),
			calcButton(session, "2", "2", discord.SecondaryButtonStyle),
			calcButton(session, "3", "3", discord.SecondaryButtonStyle),
			calcButton(session, "-", "-", discord.PrimaryButtonStyle),
		),
		util.ActionRow(
			calcButton(session, ".", ".", discord.SecondaryButtonStyle),
			calcButton(session, "0", "0", discord.SecondaryButtonStyle),
			calcButton(session, "C", keyClear, discord.DangerButtonStyle),
			calcButton(session, "+", "+", discord.PrimaryButtonStyle),
		),
		util.ActionRow(
			calcButton(session, "⌫", keyBack, discord.SecondaryButtonStyle),
			calcButton(session, "=", keyEquals, discord.SuccessButtonStyle),
		),
	}
}

func calcButtonIDs(session string) []string {
	ids := make([]string, 0, len(calcKeys))
	for _, key := range calcKeys {
		ids = append(ids, fmt.Sprintf("calc:%s:%s", session, key))
	}
	return ids
}

// calcData renders a session's display and keypad.
func calcData(session, display string) discord.ResponseMessageData {
	return discord.ResponseMessageData{
		Content:    display,
		Components: calcComponents(session),
	}
}
