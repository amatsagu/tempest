package rest

import (
	json "encoding/json/v2"
	"errors"
	"fmt"
)

var (
	errRetryable       = errors.New("rest: retryable error")
	errGlobalRateLimit = errors.New("rest: global rate limit exceeded")

	// Returned when concurrent retryable failures exceed RetryThreshold.
	ErrTooManyRetries = errors.New("rest: retry threshold exceeded")
)

// Non-2xx Discord REST response.
type HTTPError struct {
	Status     string
	Message    string `json:"message"`
	Body       []byte
	StatusCode int
	Code       int `json:"code"`
}

func newHTTPError(statusCode int, status string, body []byte) *HTTPError {
	httpErr := &HTTPError{
		StatusCode: statusCode,
		Status:     status,
		Body:       body,
	}

	if len(body) > 0 {
		_ = json.Unmarshal(body, httpErr)
	}

	return httpErr
}

// Formats the HTTP error with the extracted Discord API code when present and original body.
func (e *HTTPError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("Discord API error %d (%s): %s", e.Code, e.Status, string(e.Body))
	}

	return fmt.Sprintf("request failed with status %s: %s", e.Status, string(e.Body))
}

type rateLimitError struct {
	Message    string  `json:"message"`
	RetryAfter float64 `json:"retry_after"`
	Global     bool    `json:"global"`
}
