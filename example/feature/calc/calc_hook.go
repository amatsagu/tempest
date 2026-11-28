package calc

import (
	"fmt"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

const (
	keyBack   = "back"
	keyClear  = "C"
	keyEquals = "="
)

func calcButton(session, label, key string, style discord.ButtonStyle) discord.ButtonComponent {
	return builder.Button(fmt.Sprintf("calc:%s:%s", session, key), label, style)
}

func calcComponents(session string) []discord.MessageComponent {
	return []discord.MessageComponent{
		builder.ActionRow(
			calcButton(session, "7", "7", discord.SecondaryButtonStyle),
			calcButton(session, "8", "8", discord.SecondaryButtonStyle),
			calcButton(session, "9", "9", discord.SecondaryButtonStyle),
			calcButton(session, "÷", "÷", discord.PrimaryButtonStyle),
		),
		builder.ActionRow(
			calcButton(session, "4", "4", discord.SecondaryButtonStyle),
			calcButton(session, "5", "5", discord.SecondaryButtonStyle),
			calcButton(session, "6", "6", discord.SecondaryButtonStyle),
			calcButton(session, "×", "×", discord.PrimaryButtonStyle),
		),
		builder.ActionRow(
			calcButton(session, "1", "1", discord.SecondaryButtonStyle),
			calcButton(session, "2", "2", discord.SecondaryButtonStyle),
			calcButton(session, "3", "3", discord.SecondaryButtonStyle),
			calcButton(session, "-", "-", discord.PrimaryButtonStyle),
		),
		builder.ActionRow(
			calcButton(session, ".", ".", discord.SecondaryButtonStyle),
			calcButton(session, "0", "0", discord.SecondaryButtonStyle),
			calcButton(session, "C", keyClear, discord.DangerButtonStyle),
			calcButton(session, "+", "+", discord.PrimaryButtonStyle),
		),
		builder.ActionRow(
			calcButton(session, "⌫", keyBack, discord.SecondaryButtonStyle),
			calcButton(session, "=", keyEquals, discord.SuccessButtonStyle),
		),
	}
}

func calcData(session, display string) discord.ResponseMessageData {
	return discord.ResponseMessageData{
		Content:    display,
		Components: calcComponents(session),
	}
}
