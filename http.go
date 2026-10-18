package tempest

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amatsagu/tempest/discord"
)

// ServeHTTP implements the Discord interaction endpoint as a plain http.Handler,
// so it can be served by Run or mounted in a developer's own server.
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		c.serveProbe(w)
		return
	}

	if c.publicKey == nil {
		c.httpError(w, http.StatusInternalServerError,
			"interactions are not configured",
			"Config.PublicKey is empty. Set it to the application's Ed25519 public key so Discord's signatures can be verified.")
		return
	}

	body, buf, ok := c.readSignedBody(w, r)
	if !ok {
		return
	}

	typ, err := interactionType(body)
	if err != nil {
		c.releaseBody(buf)
		c.httpError(w, http.StatusBadRequest,
			"cannot parse interaction",
			"The request was signed correctly but its body is not a valid interaction payload.")
		return
	}

	// A PING has no handler to run, so it is answered on the request goroutine.
	if typ == discord.InteractionPing {
		defer c.releaseBody(buf)
		w.Header().Set("Content-Type", "application/json")
		_ = json.MarshalWrite(w, discord.Response{Type: discord.ResponsePong})
		return
	}

	// Every other interaction runs on its own goroutine and hands its first
	// response back through a channel.
	ctx := context.WithoutCancel(r.Context())
	ch := make(chan dispatchResult, 1)

	responder := acquireResponder()
	responder.kind = responderHTTP
	responder.ch = ch
	responder.client = c
	responder.ctx = ctx

	go func() {
		// The body buffer is released here, inside the goroutine, so it outlives ServeHTTP.
		defer c.releaseBody(buf)
		defer releaseResponder(responder)

		result, err := c.dispatchType(ctx, typ, body, responder)
		if err != nil {
			if c.debug {
				c.cfg.Logf("tempest: cannot parse interaction: %v", err)
			}
			responder.deliver(dispatchResult{NoResponse: true})
			return
		}
		responder.deliver(result)
	}()

	result := <-ch
	if result.NoResponse {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.MarshalWrite(w, result.Response)
}

// bodyPool reuses interaction-body buffers across requests.
var bodyPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

// readSignedBody buffers the body once, prefixed with the signature timestamp, and
// verifies Discord's Ed25519 signature over the whole message in place. Ed25519
// covers timestamp||body, so this read cannot be streamed, but the body is never
// copied. The caller must hand buf to releaseBody; on failure it writes the error.
func (c *Client) readSignedBody(w http.ResponseWriter, r *http.Request) (body []byte, buf *bytes.Buffer, ok bool) {
	timestamp := r.Header.Get("X-Signature-Timestamp")

	if c.cfg.TimestampWindow > 0 && !withinWindow(timestamp, c.cfg.TimestampWindow, time.Now()) {
		c.httpError(w, http.StatusUnauthorized,
			"stale request timestamp",
			"Config.TimestampWindow rejected this interaction. Discord retries deliveries with the original signature, so raise the window or set it to zero if legitimate requests are being rejected.")
		return nil, nil, false
	}

	buf = bodyPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.WriteString(timestamp)
	bodyStart := buf.Len()

	// MaxBytesReader caps the body even without a Content-Length.
	if _, err := buf.ReadFrom(http.MaxBytesReader(w, r.Body, c.res.MaxBodyBytes)); err != nil {
		c.releaseBody(buf)
		c.httpError(w, http.StatusBadRequest,
			"cannot read request body",
			"Raise Config.MaxBodyBytes if the interaction payload is larger than the current limit.")
		return nil, nil, false
	}

	if !verifySignature(c.publicKey, buf.Bytes(), r.Header.Get("X-Signature-Ed25519")) {
		c.releaseBody(buf)
		c.httpError(w, http.StatusUnauthorized,
			"invalid request signature",
			"Discord signs every interaction with the app's Ed25519 private key. Check that Config.PublicKey matches the application and that Discord's Interactions Endpoint URL points at this exact URL.")
		return nil, nil, false
	}

	return buf.Bytes()[bodyStart:], buf, true
}

// releaseBody returns a buffer to the pool, dropping oversized ones.
func (c *Client) releaseBody(buf *bytes.Buffer) {
	if int64(buf.Cap()) <= c.res.MaxBodyBytes {
		buf.Reset()
		bodyPool.Put(buf)
	}
}

// withinWindow reports whether a Discord signature timestamp, Unix seconds, is
// within window of now in either direction. It bounds replay of a captured signed
// request.
func withinWindow(timestamp string, window time.Duration, now time.Time) bool {
	secs, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}

	diff := now.Sub(time.Unix(secs, 0))
	if diff < 0 {
		diff = -diff
	}
	return diff <= window
}

// serveProbe answers a non-POST with no content, or a status page in debug mode.
func (c *Client) serveProbe(w http.ResponseWriter) {
	w.Header().Set("Allow", http.MethodPost)

	if !c.debug {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, c.endpointStatus())
}

func (c *Client) httpError(w http.ResponseWriter, status int, message, hint string) {
	if c.debug && hint != "" {
		http.Error(w, message+"\n\n"+hint, status)
		return
	}
	http.Error(w, message, status)
}

func (c *Client) endpointStatus() string {
	var b strings.Builder
	b.WriteString("Tempest interactions endpoint is running.\n\n")
	b.WriteString("Discord must POST signed interactions here. This URL is not a web page.\n\n")

	if c.publicKey == nil {
		b.WriteString("Public key: MISSING. Set Config.PublicKey to the application's Ed25519 public key.\n")
	} else {
		b.WriteString("Public key: configured.\n")
	}

	fmt.Fprintf(&b, "Max body: %d bytes.\n", c.res.MaxBodyBytes)
	return b.String()
}
