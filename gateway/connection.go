package gateway

import (
	"context"
	"errors"
	"sync"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type wsConn struct {
	mu       sync.Mutex
	cancelFn context.CancelFunc
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
		conn:     conn,
		cancelFn: cancel,
	}, nil
}

// Writes data to the socket as JSON payload.
func (c *wsConn) sendPayload(ctx context.Context, v any) error {
	c.mu.Lock()

	if c.closed {
		c.mu.Unlock()
		return errors.New("ws connection is closed")
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
	c.closed = true
	err := c.conn.Close(exitCode, reason)
	c.mu.Unlock()
	return err
}

func (c *wsConn) close() error {
	return c.closeWithReason(websocket.StatusNormalClosure, "closed by client")
}

// Allows for reuse of wsConn. It will silently drop current connection if it's still connected and dial new target url.
// In case this method returns error - assume conn is in broken state and there's no way to safely use it anymore.
func (c *wsConn) reconnect(ctx context.Context, url string) error {
	if !c.closed {
		if err := c.close(); err != nil {
			return err
		}
	}

	c.mu.Lock()

	ctx, cancel := context.WithCancel(ctx)
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		cancel()
		c.mu.Unlock()
		return err
	}

	c.conn = conn
	c.cancelFn = cancel
	c.mu.Unlock()
	return nil
}
