package builder

import "github.com/amatsagu/tempest/discord"

// Builds a poll ready to send. durationHours specifies how long the poll remains open (up to 32 days).
func Poll(question string, durationHours uint16, answers ...discord.PollAnswer) discord.Poll {
	return discord.Poll{
		Question:   PollMedia(question),
		Answers:    answers,
		Duration:   durationHours,
		LayoutType: discord.DefaultPollLayoutType,
	}
}

// Builds a poll question or answer media entry.
func PollMedia(text string) discord.PollMedia {
	return discord.PollMedia{Text: text}
}

// Builds a poll answer option with text.
func PollAnswer(text string) discord.PollAnswer {
	return discord.PollAnswer{PollMedia: PollMedia(text)}
}

// Builds vote count metadata for one poll answer ID.
func PollAnswerCounts(id, count uint32) discord.PollAnswerCounts {
	return discord.PollAnswerCounts{
		ID:    id,
		Count: count,
	}
}

// Builds poll vote result tallies.
func PollResult(answerCounts ...discord.PollAnswerCounts) discord.PollResult {
	return discord.PollResult{AnswerCounts: answerCounts}
}
