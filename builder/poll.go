package builder

import "github.com/amatsagu/tempest/discord"

// Poll builds a poll ready to send. Duration is in hours.
func Poll(question string, duration uint32, answers ...discord.PollAnswer) discord.Poll {
	return discord.Poll{
		Question:   PollMedia(question),
		Answers:    answers,
		Duration:   duration,
		LayoutType: discord.PollLayoutDefault,
	}
}

func PollMedia(text string) discord.PollMedia {
	return discord.PollMedia{Text: text}
}

func PollAnswer(text string) discord.PollAnswer {
	return discord.PollAnswer{PollMedia: PollMedia(text)}
}

func PollAnswerCount(id, count int) discord.PollAnswerCount {
	return discord.PollAnswerCount{
		ID:    id,
		Count: count,
	}
}

func PollResults(answerCounts ...discord.PollAnswerCount) discord.PollResults {
	return discord.PollResults{AnswerCounts: answerCounts}
}
