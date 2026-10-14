package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestPoll(t *testing.T) {
	p := Poll{
		Question: PollMedia{Text: "Favorite fruit?"},
		Answers: []PollAnswer{
			{PollMedia: PollMedia{Text: "Apple"}},
			{PollMedia: PollMedia{Text: "Banana"}},
		},
		Duration:         24,
		AllowMultiselect: false,
		LayoutType:       PollLayoutDefault,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Poll
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Question.Text != "Favorite fruit?" || len(parsed.Answers) != 2 {
		t.Errorf("parsed = %+v, want %+v", parsed, p)
	}
}

func BenchmarkPoll_JSON(b *testing.B) {
	p := Poll{
		Question: PollMedia{Text: "Favorite fruit?"},
		Answers: []PollAnswer{
			{PollMedia: PollMedia{Text: "Apple"}},
			{PollMedia: PollMedia{Text: "Banana"}},
		},
		Duration:         24,
		AllowMultiselect: false,
		LayoutType:       PollLayoutDefault,
	}
	data, _ := json.Marshal(p)

	b.ReportAllocs()
	for b.Loop() {
		var parsed Poll
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(p)
	}
}
