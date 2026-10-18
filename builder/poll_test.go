package builder

import (
	"encoding/json/v2"
	"testing"

	"github.com/amatsagu/tempest/discord"
)

func buildPollViaHelpers() discord.Poll {
	return Poll("Favorite fruit?", 24,
		PollAnswer("Apple"),
		PollAnswer("Banana"),
	)
}

func buildPollDirect() discord.Poll {
	return discord.Poll{
		Question: discord.PollMedia{Text: "Favorite fruit?"},
		Answers: []discord.PollAnswer{
			{PollMedia: discord.PollMedia{Text: "Apple"}},
			{PollMedia: discord.PollMedia{Text: "Banana"}},
		},
		Duration:   24,
		LayoutType: discord.PollLayoutDefault,
	}
}

func TestPollBuildersMatchDirect(t *testing.T) {
	assertSameJSON(t, buildPollViaHelpers(), buildPollDirect())

	assertSameJSON(t, PollMedia("t"), discord.PollMedia{Text: "t"})
	assertSameJSON(t, PollAnswer("a"), discord.PollAnswer{PollMedia: discord.PollMedia{Text: "a"}})
	assertSameJSON(t, PollAnswerCount(1, 2), discord.PollAnswerCount{ID: 1, Count: 2})
	assertSameJSON(t,
		PollResults(PollAnswerCount(1, 2)),
		discord.PollResults{AnswerCounts: []discord.PollAnswerCount{{ID: 1, Count: 2}}},
	)
}

func TestPollRoundTrip(t *testing.T) {
	raw, err := json.Marshal(ptr(buildPollViaHelpers()))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed discord.Poll
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed.Question.Text != "Favorite fruit?" || len(parsed.Answers) != 2 || parsed.Duration != 24 {
		t.Errorf("round-trip mismatch: %+v", parsed)
	}
}

func BenchmarkPoll_JSON(b *testing.B) {
	poll := buildPollViaHelpers()
	data, _ := json.Marshal(&poll)

	var parsed discord.Poll
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&poll)
	}
}

func BenchmarkBuildPoll_Helpers(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkPoll = buildPollViaHelpers()
	}
}

func BenchmarkBuildPoll_Direct(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkPoll = buildPollDirect()
	}
}

var sinkPoll discord.Poll
