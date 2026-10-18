package gateway

import "testing"

// TestOpcodesMatchTheDocs pins the gateway opcodes, including the two newer send
// opcodes, since a wrong opcode is a silent protocol bug.
func TestOpcodesMatchTheDocs(t *testing.T) {
	opcodes := map[Opcode]uint8{
		OpDispatch:                0,
		OpHeartbeat:               1,
		OpIdentify:                2,
		OpPresenceUpdate:          3,
		OpVoiceStateUpdate:        4,
		OpResume:                  6,
		OpReconnect:               7,
		OpRequestGuildMembers:     8,
		OpInvalidSession:          9,
		OpHello:                   10,
		OpHeartbeatACK:            11,
		OpRequestSoundboardSounds: 31,
		OpRequestChannelInfo:      43,
	}

	for got, want := range opcodes {
		if uint8(got) != want {
			t.Errorf("opcode = %d, want %d", got, want)
		}
	}
}
