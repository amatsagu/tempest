package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestSelectComponents(t *testing.T) {
	sel := NewStringSelect("my_select", "Choose an option",
		SelectOption{Label: "Opt 1", Value: "val_1"},
		SelectOption{Label: "Opt 2", Value: "val_2", Default: true},
	)

	if sel.Type != ComponentStringSelect {
		t.Errorf("got %d, want %d", sel.Type, ComponentStringSelect)
	}

	data, err := json.Marshal(sel)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed StringSelect
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.CustomID != sel.CustomID || len(parsed.Options) != 2 {
		t.Errorf("parsed = %+v, want %+v", parsed, sel)
	}

	userSel := NewUserSelect("user_pick", "Select user")
	if userSel.Type != ComponentUserSelect {
		t.Errorf("userSel.Type = %d, want %d", userSel.Type, ComponentUserSelect)
	}
}

func BenchmarkSelect_JSON(b *testing.B) {
	sel := NewStringSelect("my_select", "Choose an option",
		SelectOption{Label: "Opt 1", Value: "val_1"},
		SelectOption{Label: "Opt 2", Value: "val_2", Default: true},
	)
	data, _ := json.Marshal(sel)

	b.ReportAllocs()
	for b.Loop() {
		var parsed StringSelect
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(sel)
	}
}
