package tempest

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type Rest struct {
	Token    string // Discord App token. Remember to add "Bot" prefix.
	lockedTo int64  // Timestamp (in ms) to when it's locked, 0 means there's no lock.
	fails    uint8  // If request failed, try again up to 3 times (delay 250/500/750ms) - after 3rd failed attempt => panic
}

type rateLimitError struct {
	Global     bool    `json:"global"`
	Message    string  `json:"message"`
	RetryAfter float32 `json:"retry_after"`
}

// Handles communication with Discord API. It automatically handles various return types and controls global rate limit of your discord app.
func (rest *Rest) Request(method string, route string, jsonPayload interface{}) ([]byte, error) {
	now := time.Now().UnixMilli()
	if now < rest.lockedTo {
		time.Sleep(time.Millisecond * time.Duration(rest.lockedTo-now))
	}

	var req *http.Request
	if jsonPayload == nil {
		request, err := http.NewRequest(method, DISCORD_API_URL+route, nil)
		if err != nil {
			return nil, errors.New("failed to initialize new request: " + err.Error())
		}
		req = request
	} else {
		body, err := json.Marshal(jsonPayload)
		if err != nil {
			return nil, errors.New("failed to parse provided payload (make sure it's in JSON format)")
		}

		request, err := http.NewRequest(method, DISCORD_API_URL+route, bytes.NewBuffer(body))
		if err != nil {
			return nil, errors.New("failed to initialize new request: " + err.Error())
		}
		req = request
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "DiscordApp https://github.com/Amatsagu/tempest")
	req.Header.Add("Authorization", rest.Token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		rest.fails++
		if rest.fails == 3 {
			panic("failed to make http request 3 times to " + method + " :: " + route + " (check internet connection and/or app credentials)")
		} else {
			time.Sleep(time.Millisecond * time.Duration(250*rest.fails))
			return rest.Request(method, route, jsonPayload) // Try again after potential internet connection failure.
		}
	}
	defer res.Body.Close()
	rest.fails = 0

	if res.StatusCode == 204 {
		return nil, nil
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New("failed to parse response body (json): " + err.Error())
	}

	if res.StatusCode == 429 {
		rateErr := rateLimitError{}
		json.Unmarshal(body, &rateErr)
		rest.lockedTo = int64(rateErr.RetryAfter+3) * 1000
		time.Sleep(time.Second * time.Duration(rateErr.RetryAfter+3))
		return rest.Request(method, route, jsonPayload) // Try again after rate limit.
	} else if res.StatusCode >= 400 {
		return nil, errors.New(res.Status + " :: " + string(body))
	}

	return body, nil
}

// Creates standalone REST instance. Use CreateClient function if you want to create regular Discord App.
func CreateRest(token string) Rest {
	if !strings.HasPrefix(token, "Bot ") {
		panic("app token needs to start with \"Bot \" prefix (example: \"Bot XYZABCQEWQ\")")
	}

	return Rest{
		Token:    token,
		lockedTo: 0,
		fails:    0,
	}
}
