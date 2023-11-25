package tempest

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

type Rest struct {
	mu         sync.RWMutex
	token      string
	httpClient *http.Client
	lockedTo   time.Time
}

type rateLimitError struct {
	Global     bool    `json:"global"`
	Message    string  `json:"message"`
	RetryAfter float32 `json:"retry_after"`
}

func (rest *Rest) Request(method string, route string, jsonPayload interface{}) ([]byte, error) {
	if !rest.lockedTo.IsZero() {
		timeLeft := time.Until(rest.lockedTo)
		if timeLeft > 0 {
			time.Sleep(timeLeft)
		}
	}

	for i := 1; i < 3; i++ {
		rest.mu.RLock()
		raw, err, finished := rest.handleRequest(method, route, jsonPayload)
		if finished {
			return raw, err
		}
		rest.mu.RUnlock()
		time.Sleep(time.Microsecond * time.Duration(250*i))
	}

	return nil, errors.New("failed to make http request 3 times to " + method + " :: " + route + " (check internet connection and/or app credentials)")
}

func (rest *Rest) handleRequest(method string, route string, jsonPayload interface{}) ([]byte, error, bool) {
	var req *http.Request
	if jsonPayload == nil {
		request, err := http.NewRequest(method, DISCORD_API_URL+route, nil)
		if err != nil {
			return nil, errors.New("failed to initialize new request: " + err.Error()), false
		}
		req = request
	} else {
		body, err := json.Marshal(jsonPayload)
		if err != nil {
			return nil, errors.New("failed to parse provided payload (make sure it's in JSON format)"), true
		}

		request, err := http.NewRequest(
			method,
			DISCORD_API_URL+route,
			bytes.NewBuffer(body),
		)

		if err != nil {
			return nil, errors.New("failed to initialize new request: " + err.Error()), false
		}
		req = request
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", USER_AGENT)
	req.Header.Add("Authorization", rest.token)

	res, err := rest.httpClient.Do(req)
	if err != nil {
		return nil, errors.New("failed to process request: " + err.Error()), false
	}

	if res.StatusCode == 204 {
		return nil, nil, true
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New("failed to parse response body (json): " + err.Error()), true
	}

	if res.StatusCode == 429 {
		rateErr := rateLimitError{}
		json.Unmarshal(body, &rateErr)

		rest.mu.Lock()
		timeLeft := time.Now().Add(time.Second * time.Duration(rateErr.RetryAfter+5))
		rest.lockedTo = timeLeft
		rest.mu.Unlock()

		time.Sleep(time.Until(timeLeft))

		rest.mu.Lock()
		rest.lockedTo = time.Time{}
		rest.mu.Unlock()
		return nil, errors.New("rate limit"), false
	} else if res.StatusCode >= 400 {
		return nil, errors.New(res.Status + " :: " + string(body)), true
	}

	return body, nil, true
}

func NewRest(token string) *Rest {
	return NewCustomRest(token, http.DefaultClient)
}

func NewCustomRest(token string, client *http.Client) *Rest {
	if _, err := extractUserIDFromToken(token); err != nil {
		panic(err)
	}

	return &Rest{
		token:      token,
		httpClient: client,
	}
}
