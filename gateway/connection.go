package gateway

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Represents single shard's connection with discord gateway.
type wsConn struct {
	mu       sync.Mutex
	cancelFn context.CancelFunc
	limiter  *TokenRateLimiter
	conn     *websocket.Conn
	closed   bool
}

// Makes new wsConn and starts the read loop.
func createwsConnection(ctx context.Context, url string) (*wsConn, error) {
	ctx, cancel := context.WithCancel(ctx)
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		cancel()
		return nil, err
	}

	// It's being created so no need to use mutex here.
	return &wsConn{
		cancelFn: cancel,
		// https://discord.com/developers/docs/events/gateway#rate-limiting
		limiter: NewTokenRateLimiter(120, time.Millisecond*500),
		conn:    conn,
	}, nil
}

// Writes data to the socket as JSON payload.
func (c *wsConn) sendPayload(ctx context.Context, v any) error {
	c.mu.Lock()

	if c.closed {
		c.mu.Unlock()
		return errors.New("ws connection is closed")
	}

	if !c.limiter.TryConsume() {
		c.mu.Unlock()
		return errors.New("reached rate limit (over 120 payloads in last 60s)")
	}

	err := wsjson.Write(ctx, c.conn, v)
	c.mu.Unlock()
	return err
}

func (c *wsConn) readJSON(ctx context.Context, v *EventPacket) error {
	c.mu.Lock()
	closed := c.closed
	conn := c.conn
	c.mu.Unlock()

	if closed {
		return errors.New("ws connection is closed")
	}

	return wsjson.Read(ctx, conn, v)
}

// Tries to safely close ws connection.
func (c *wsConn) closeWithReason(exitCode websocket.StatusCode, reason string) error {
	c.mu.Lock()

	if c.closed {
		c.mu.Unlock()
		return errors.New("ws connection is already closed")
	}

	c.cancelFn()
	c.limiter = nil
	c.closed = true
	err := c.conn.Close(exitCode, reason)
	c.mu.Unlock()
	return err
}

func (c *wsConn) close() error {
	return c.closeWithReason(websocket.StatusNormalClosure, "closed by client")
}
