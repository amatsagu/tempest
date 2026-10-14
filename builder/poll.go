package builder

import "github.com/amatsagu/tempest/discord"

// NewPoll builds a poll ready to send. Duration is in hours; answers are
// message answers built with NewPollAnswer (or plain PollAnswer values).
func NewPoll(question string, duration uint32, answers ...discord.PollAnswer) discord.Poll {
	return discord.Poll{
		Question:   NewPollMedia(question),
		Answers:    answers,
		Duration:   duration,
		LayoutType: discord.PollLayoutDefault,
	}
}

func NewPollMedia(text string) discord.PollMedia {
	return discord.PollMedia{Text: text}
}

func NewPollAnswer(text string) discord.PollAnswer {
	return discord.PollAnswer{PollMedia: NewPollMedia(text)}
}

func NewPollAnswerCount(id, count int) discord.PollAnswerCount {
	return discord.PollAnswerCount{
		ID:    id,
		Count: count,
	}
}

func NewPollResults(answerCounts ...discord.PollAnswerCount) discord.PollResults {
	return discord.PollResults{AnswerCounts: answerCounts}
}
