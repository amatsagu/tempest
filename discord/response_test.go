package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestResponse(t *testing.T) {
	pong := NewPongResponse()
	if pong.Type != ResponsePong {
		t.Errorf("got %d, want %d", pong.Type, ResponsePong)
	}

	data, err := json.Marshal(pong)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expected := `{"type":1}`
	if string(data) != expected {
		t.Errorf("got %s, want %s", string(data), expected)
	}

	def := NewDeferredResponse(true)
	if def.Type != ResponseDeferredChannelMessageWithSource {
		t.Errorf("def.Type = %d, want %d", def.Type, ResponseDeferredChannelMessageWithSource)
	}

	defData, err := json.Marshal(def)
	if err != nil {
		t.Fatalf("Marshal deferred failed: %v", err)
	}

	expectedDef := `{"data":{"flags":64},"type":5}`
	if string(defData) != expectedDef {
		t.Errorf("got %s, want %s", string(defData), expectedDef)
	}
}

func BenchmarkResponse_JSON(b *testing.B) {
	def := NewDeferredResponse(true)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = json.Marshal(&def)
	}
}
