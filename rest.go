package tempest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Rest struct {
	HTTPClient http.Client
	MaxRetries uint8
	token      string
	mu         sync.RWMutex
	lockedTo   time.Time
}

// Represents file you can attach to message on Discord.
type File struct {
	Name   string // File's display name
	Reader io.Reader
}

type rateLimitError struct {
	Message    string  `json:"message"`
	RetryAfter float32 `json:"retry_after"`
	Global     bool    `json:"global"`
}

func NewRest(token string) *Rest {
	t := token
	if !strings.HasPrefix(t, "Bot ") {
		t = "Bot " + t
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   256,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 0,
	}

	return &Rest{
		HTTPClient: http.Client{Transport: transport, Timeout: 3 * time.Second},
		token:      t,
		MaxRetries: 3,
		lockedTo:   time.Time{},
	}
}

func (rest *Rest) Request(method, route string, jsonPayload any) ([]byte, error) {
	var body io.Reader

	if jsonPayload != nil {
		var buf bytes.Buffer

		encoder := json.NewEncoder(&buf)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(jsonPayload); err != nil {
			return nil, fmt.Errorf("failed to encode JSON payload: %w", err)
		}

		body = &buf
	}

	rest.mu.RLock()
	lockedUntil := rest.lockedTo
	rest.mu.RUnlock()

	if !lockedUntil.IsZero() {
		sleepFor := time.Until(lockedUntil)
		if sleepFor > 0 {
			time.Sleep(sleepFor)
		}
	}

	var i uint8
	for i = 0; i < rest.MaxRetries; i++ {
		res, err, done := rest.handleRequest(method, route, body, CONTENT_TYPE_JSON)

		if done {
			return res, err
		}

		time.Sleep(time.Millisecond * time.Duration(250*(i+1)))
	}

	return nil, fmt.Errorf("request failed after %d retries to %s %s", rest.MaxRetries, method, route)
}

func (rest *Rest) RequestWithFiles(method string, route string, jsonPayload any, files []File) ([]byte, error) {
	if len(files) == 0 {
		return rest.Request(method, route, jsonPayload)
	}

	rest.mu.RLock()
	lockedUntil := rest.lockedTo
	rest.mu.RUnlock()

	if !lockedUntil.IsZero() {
		sleepFor := time.Until(lockedUntil)
		if sleepFor > 0 {
			time.Sleep(sleepFor)
		}
	}

	// Prepare pipe for streaming multipart content without full buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer writer.Close()

		jsonPart, err := writer.CreatePart(textproto.MIMEHeader{
			"Content-Disposition": []string{CONTENT_MULTIPART_JSON_DESCRIPTION},
			"Content-Type":        []string{CONTENT_TYPE_JSON},
		})
		if err != nil {
			pw.CloseWithError(fmt.Errorf("failed to create payload_json part: %w", err))
			return
		}

		encoder := json.NewEncoder(jsonPart)
		encoder.SetEscapeHTML(false)

		if err := encoder.Encode(jsonPayload); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to encode payload_json: %w", err))
			return
		}

		for i, file := range files {
			filePart, err := writer.CreatePart(textproto.MIMEHeader{
				"Content-Disposition": []string{fmt.Sprintf(`form-data; name="files[%d]"; filename="%s"`, i, file.Name)},
				"Content-Type":        []string{CONTENT_TYPE_OCTET_STREAM},
			})

			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to create file part [%d]: %w", i, err))
				return
			}

			if _, err := io.Copy(filePart, file.Reader); err != nil {
				pw.CloseWithError(fmt.Errorf("failed to stream file [%s]: %w", file.Name, err))
				return
			}
		}
	}()

	var i uint8
	for i = 0; i < rest.MaxRetries; i++ {
		body, err, done := rest.handleRequest(method, route, pr, writer.FormDataContentType())

		if done {
			return body, err
		}

		time.Sleep(time.Millisecond * time.Duration(250*(i+1)))
	}

	return nil, fmt.Errorf("request failed after %d retries to %s %s", rest.MaxRetries, method, route)
}

func (rest *Rest) handleRequest(method string, route string, payload io.Reader, contentType string) ([]byte, error, bool) {
	req, err := http.NewRequest(method, DISCORD_API_URL+route, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize new request: %w", err), false
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", USER_AGENT)
	req.Header.Set("Authorization", rest.token)

	res, err := rest.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to process request: %w", err), false
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNoContent {
		return nil, nil, true
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err), true
	}

	if res.StatusCode == http.StatusTooManyRequests {
		var rateErr rateLimitError
		_ = json.Unmarshal(body, &rateErr) // even if this fails - it can still fall back

		var retryAfter time.Duration
		if hdr := res.Header.Get("Retry-After"); hdr != "" {
			if v, err := strconv.ParseFloat(hdr, 64); err == nil {
				retryAfter = time.Duration(v * float64(time.Second))
			}
		}
		if retryAfter == 0 {
			retryAfter = time.Duration(float64(rateErr.RetryAfter) * float64(time.Second))
		}
		if retryAfter <= 0 {
			retryAfter = 250 * time.Millisecond
		}

		if rateErr.Global {
			rest.mu.Lock()
			rest.lockedTo = time.Now().Add(retryAfter)
			rest.mu.Unlock()
		}

		time.Sleep(retryAfter)
		return nil, errors.New("rate limited"), false
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("%s :: %s", res.Status, string(body)), true
	}

	return body, nil, true
}
