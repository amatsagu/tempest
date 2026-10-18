package rest

import (
	"context"
	"encoding/json/v2"
)

// GatewayBot is the response of GET /gateway/bot.
type GatewayBot struct {
	URL               string `json:"url"`
	Shards            int    `json:"shards"`
	MaxConcurrency    int    `json:"-"`
	SessionStartLimit struct {
		Total          int `json:"total"`
		Remaining      int `json:"remaining"`
		ResetAfter     int `json:"reset_after"`
		MaxConcurrency int `json:"max_concurrency"`
	} `json:"session_start_limit"`
}

// GatewayBot fetches the recommended gateway endpoint, shard count, and concurrency.
func (c *Client) GatewayBot(ctx context.Context) (*GatewayBot, error) {
	raw, err := c.Request(ctx, "GET", "/gateway/bot", nil, "")
	if err != nil {
		return nil, err
	}

	var bot GatewayBot
	if err := json.Unmarshal(raw, &bot); err != nil {
		return nil, err
	}

	bot.MaxConcurrency = bot.SessionStartLimit.MaxConcurrency
	if bot.MaxConcurrency <= 0 {
		bot.MaxConcurrency = 1
	}

	if bot.Shards <= 0 {
		bot.Shards = 1
	}

	return &bot, nil
}
