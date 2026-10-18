package discord

import "testing"

// TestBitFieldsPinDocumentedValues pins the flags that were missing or wrong, so a
// later edit cannot silently shift them.
func TestBitFieldsPinDocumentedValues(t *testing.T) {
	permissions := map[Permissions]uint8{
		PermissionSendVoiceMessages:     46,
		PermissionSetVoiceChannelStatus: 48,
		PermissionSendPolls:             49,
		PermissionUseExternalApps:       50,
		PermissionPinMessages:           51,
		PermissionBypassSlowmode:        52,
	}
	checkBits(t, "permission", permissions)

	channelFlags := map[ChannelFlags]uint8{
		ChannelFlagPinned:                   1,
		ChannelFlagRequireTag:               4,
		ChannelFlagHideMediaDownloadOptions: 15,
		ChannelFlagObfuscated:               17,
		ChannelFlagIsSpoilerChannel:         21,
	}
	checkBits(t, "channel flag", channelFlags)

	attachmentFlags := map[AttachmentFlags]uint8{
		AttachmentFlagIsClip:      0,
		AttachmentFlagIsThumbnail: 1,
		AttachmentFlagIsRemix:     2,
		AttachmentFlagIsSpoiler:   3,
		AttachmentFlagIsAnimated:  5,
	}
	checkBits(t, "attachment flag", attachmentFlags)

	memberFlags := map[MemberFlags]uint8{
		MemberFlagDidRejoin:                    0,
		MemberFlagAutomodQuarantinedUsername:   7,
		MemberFlagDmSettingsUpsellAcknowledged: 9,
		MemberFlagAutomodQuarantinedGuildTag:   10,
	}
	checkBits(t, "member flag", memberFlags)

	messageFlags := map[MessageFlags]uint8{
		MessageFlagCrossposted:           0,
		MessageFlagSuppressNotifications: 12,
		MessageFlagHasSnapshot:           14,
		MessageFlagIsComponentsV2:        15,
	}
	checkBits(t, "message flag", messageFlags)

	// The type constants are not bits, so they are checked by value.
	messageTypes := map[MessageType]uint8{
		MessageTypeDefault:                             0,
		MessageTypeChannelPinnedMessage:                6,
		MessageTypeGuildApplicationPremiumSubscription: 32,
		MessageTypeGuildIncidentAlertModeEnabled:       36,
		MessageTypeGuildIncidentAlertModeDisabled:      37,
		MessageTypeGuildIncidentReportRaid:             38,
		MessageTypeGuildIncidentReportFalseAlarm:       39,
		MessageTypePurchaseNotification:                44,
		MessageTypePollResult:                          46,
	}

	for got, want := range messageTypes {
		if uint8(got) != want {
			t.Errorf("message type = %d, want %d", got, want)
		}
	}

	componentTypes := map[ComponentType]uint8{
		ComponentActionRow:     1,
		ComponentButton:        2,
		ComponentTextInput:     4,
		ComponentSeparator:     14,
		ComponentContainer:     17,
		ComponentLabel:         18,
		ComponentFileUpload:    19,
		ComponentRadioGroup:    21,
		ComponentCheckboxGroup: 22,
		ComponentCheckbox:      23,
	}

	for got, want := range componentTypes {
		if uint8(got) != want {
			t.Errorf("component type = %d, want %d", got, want)
		}
	}
}

func checkBits[T ~uint64](t *testing.T, name string, values map[T]uint8) {
	t.Helper()

	for got, bit := range values {
		if uint64(got) != uint64(1)<<bit {
			t.Errorf("%s = %d, want 1 << %d", name, got, bit)
		}
	}
}
