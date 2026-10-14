package discord

import "time"

// https://docs.discord.com/developers/resources/poll#layout-type
type PollLayoutType uint8

const (
	PollLayoutDefault PollLayoutType = iota + 1
)

// https://docs.discord.com/developers/resources/poll#poll-media-object
type PollMedia struct {
	Emoji *Emoji `json:"emoji,omitzero"`
	Text  string `json:"text,omitzero"`
}

// https://docs.discord.com/developers/resources/poll#poll-answer-object
type PollAnswer struct {
	PollMedia PollMedia `json:"poll_media"`
	AnswerID  int       `json:"answer_id,omitzero"`
}

// https://docs.discord.com/developers/resources/poll#poll-results-object
type PollAnswerCount struct {
	ID      int  `json:"id"`
	Count   int  `json:"count"`
	MeVoted bool `json:"me_voted,omitzero"`
}

// https://docs.discord.com/developers/resources/poll#poll-results-object
type PollResults struct {
	AnswerCounts []PollAnswerCount `json:"answer_counts,omitzero"`
	IsFinalized  bool              `json:"is_finalized"`
}

// https://docs.discord.com/developers/resources/poll#poll-object
type Poll struct {
	Expiry           *time.Time     `json:"expiry,omitzero"`
	Results          *PollResults   `json:"results,omitzero"`
	Question         PollMedia      `json:"question"`
	Answers          []PollAnswer   `json:"answers"`
	Duration         uint32         `json:"duration,omitzero"`
	LayoutType       PollLayoutType `json:"layout_type,omitzero"`
	AllowMultiselect bool           `json:"allow_multiselect,omitzero"`
}
