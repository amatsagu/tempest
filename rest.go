package tempest

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Please avoid creating raw Rest struct unless you know what you're doing. Use CreateRest function instead.
type Rest struct {
	Token                  string // Discord Bot/App token. Remember to add "Bot" prefix.
	MaxRequestsBeforeSweep uint16
	GlobalRequestLimit     uint16
	globalRequests         uint16
	requestsSinceSweep     uint16
	lockedTo               int64 // Timestamp (in ms) to when it's locked, 0 means there's no lock.
	locks                  map[string]int64
	fails                  uint16 // If request failed, try again up to 3 times (delay 250/500/750ms) - after 3rd failed attempt => panic
}

type rateLimitError struct {
	Global     bool    `json:"global"`
	Message    string  `json:"message"`
	RetryAfter float32 `json:"retry_after"`
}

func (rest Rest) Request(method string, route string, jsonPayload interface{}) ([]byte, error) {
	rest.globalRequests++
	rest.requestsSinceSweep++

	if rest.locks == nil {
		rest.locks = make(map[string]int64, rest.MaxRequestsBeforeSweep)
	}

	now := time.Now().Unix()
	var offset uint16 = 0
	var req *http.Request

	if rest.globalRequests == rest.GlobalRequestLimit && now < rest.lockedTo {
		rest.globalRequests = 0
		offset += 8
	}

	expiresTimestamp, exists := rest.locks[route]
	if exists && expiresTimestamp > now {
		offset += uint16(expiresTimestamp - now)
	}

	if rest.requestsSinceSweep%rest.MaxRequestsBeforeSweep == 0 {
		rest.requestsSinceSweep = 0

		go func() {
			for key, value := range rest.locks {
				if now > value {
					delete(rest.locks, key)
				}
			}
		}()
	}

	if offset != 0 {
		time.Sleep(time.Second * time.Duration(offset))
	}

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
			panic("failed to make http request 3 times to " + method + " :: " + route + " (check internet connection and app credentials)")
		} else {
			time.Sleep(time.Millisecond * time.Duration(250*rest.fails))
			return rest.Request(method, route, jsonPayload) // Try again after potential internet connection failure.
		}
	}
	defer res.Body.Close()

	rest.fails = 0
	remaining, err := strconv.ParseFloat(res.Header.Get("x-ratelimit-remaining"), 32)
	if err == nil && remaining == 0 {
		resetAt, _ := strconv.ParseFloat(res.Header.Get("x-ratelimit-reset"), 64) // If first succeeded then there's no need to check this one.
		rest.locks[route] = int64(resetAt + 6)
	}

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
		time.Sleep(time.Second * time.Duration(rateErr.RetryAfter+2.5))
		return rest.Request(method, route, jsonPayload) // Try again after rate limit.
	} else if res.StatusCode >= 400 {
		return nil, errors.New(res.Status + " :: " + string(body))
	}

	return body, nil
}

func CreateRest(token string, requestsBeforeSweep uint16) Rest {
	if !strings.HasPrefix(token, "Bot ") {
		panic("app token needs to start with \"Bot \" prefix (example: \"Bot XYZABCQEWQ\")")
	}

	return Rest{
		Token:                  token,
		MaxRequestsBeforeSweep: requestsBeforeSweep,
		GlobalRequestLimit:     50,
		globalRequests:         0,
		requestsSinceSweep:     0,
		lockedTo:               0,
		locks:                  make(map[string]int64, requestsBeforeSweep),
		fails:                  0,
	}
}
