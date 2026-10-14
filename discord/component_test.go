package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestComponentInterfaces(t *testing.T) {
	var _ Component = ActionRow{}
	var _ Component = Section{}
	var _ Component = Container{}
	var _ Component = Separator{}
	var _ Component = TextDisplay{}
	var _ Component = Thumbnail{}
	var _ Component = MediaGallery{}
	var _ Component = FileComponent{}
	var _ Component = Label{}
}

func TestUnmarshalComponentUnion(t *testing.T) {
	raw := []byte(`{"type":17,"accent_color":1,"components":[{"type":10,"content":"hi"},{"type":1,"components":[{"type":2,"style":1,"label":"x","custom_id":"c"}]}]}`)

	var container Container
	if err := json.Unmarshal(raw, &container); err != nil {
		t.Fatalf("unmarshal container: %v", err)
	}

	if len(container.Components) != 2 {
		t.Fatalf("container components = %d, want 2", len(container.Components))
	}

	if _, ok := container.Components[0].(TextDisplay); !ok {
		t.Errorf("component 0 = %T, want TextDisplay", container.Components[0])
	}

	if row, ok := container.Components[1].(ActionRow); !ok {
		t.Errorf("component 1 = %T, want ActionRow", container.Components[1])
	} else if len(row.Buttons) != 1 || row.Buttons[0].CustomID != "c" {
		t.Errorf("action row = %+v, want 1 button with custom id c", row)
	}
}

func TestUnmarshalMessageComponents(t *testing.T) {
	raw := []byte(`{"id":"1","channel_id":"2","type":0,"timestamp":"2024-01-01T00:00:00+00:00","author":{"id":"3","username":"u"},"components":[{"type":10,"content":"hello"}]}`)

	var msg Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}

	if len(msg.Components) != 1 {
		t.Fatalf("message components = %d, want 1", len(msg.Components))
	}

	if td, ok := msg.Components[0].(TextDisplay); !ok || td.Content != "hello" {
		t.Errorf("component = %+v, want TextDisplay hello", msg.Components[0])
	}
}
