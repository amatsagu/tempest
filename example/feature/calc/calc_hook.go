package calc

import (
	"fmt"

	"github.com/amatsagu/tempest"

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
func calcButton(session, label, key string, style tempest.ButtonStyle) tempest.ButtonComponent {
	return util.Button(fmt.Sprintf("calc:%s:%s", session, key), label, style)
}

func calcComponents(session string) []tempest.MessageComponent {
	return []tempest.MessageComponent{
		util.ActionRow(
			calcButton(session, "7", "7", tempest.SECONDARY_BUTTON_STYLE),
			calcButton(session, "8", "8", tempest.SECONDARY_BUTTON_STYLE),
			calcButton(session, "9", "9", tempest.SECONDARY_BUTTON_STYLE),
			calcButton(session, "÷", "÷", tempest.PRIMARY_BUTTON_STYLE),
		),
		util.ActionRow(
			calcButton(session, "4", "4", tempest.SECONDARY_BUTTON_STYLE),
			calcButton(session, "5", "5", tempest.SECONDARY_BUTTON_STYLE),
			calcButton(session, "6", "6", tempest.SECONDARY_BUTTON_STYLE),
			calcButton(session, "×", "×", tempest.PRIMARY_BUTTON_STYLE),
		),
		util.ActionRow(
			calcButton(session, "1", "1", tempest.SECONDARY_BUTTON_STYLE),
			calcButton(session, "2", "2", tempest.SECONDARY_BUTTON_STYLE),
			calcButton(session, "3", "3", tempest.SECONDARY_BUTTON_STYLE),
			calcButton(session, "-", "-", tempest.PRIMARY_BUTTON_STYLE),
		),
		util.ActionRow(
			calcButton(session, ".", ".", tempest.SECONDARY_BUTTON_STYLE),
			calcButton(session, "0", "0", tempest.SECONDARY_BUTTON_STYLE),
			calcButton(session, "C", keyClear, tempest.DANGER_BUTTON_STYLE),
			calcButton(session, "+", "+", tempest.PRIMARY_BUTTON_STYLE),
		),
		util.ActionRow(
			calcButton(session, "⌫", keyBack, tempest.SECONDARY_BUTTON_STYLE),
			calcButton(session, "=", keyEquals, tempest.SUCCESS_BUTTON_STYLE),
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
func calcData(session, display string) tempest.ResponseMessageData {
	return tempest.ResponseMessageData{
		Content:    display,
		Components: calcComponents(session),
	}
}
