package tempest

import (
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Each Connection with Gateway is:
// manager -> shards -> sockets

// A thread-safe wrapper around a Gorilla WebSocket connection.
// It's designed to handle the basic lifecycle and data framing for a
// connection to the Discord Gateway.
type socket struct {
	conn         atomic.Pointer[websocket.Conn]
	decoder      atomic.Pointer[json.Decoder]
	readDeadline atomic.Int64
	writeMu      sync.Mutex
	compress     bool
}

// Handles zero-allocation streaming from WebSocket frames.
type zlibFeeder struct {
	conn   *websocket.Conn
	reader io.Reader
}

func (f *zlibFeeder) Read(p []byte) (int, error) {
	for {
		if f.reader == nil {
			mt, r, err := f.conn.NextReader()
			if err != nil {
				return 0, err
			}

			if mt != websocket.BinaryMessage {
				continue
			}

			f.reader = r
		}

		n, err := f.reader.Read(p)
		if err == io.EOF {
			f.reader = nil
			if n > 0 {
				return n, nil
			}

			continue
		}

		return n, err
	}
}

func (s *socket) connect(ctx context.Context, urlStr string) error {
	if s.conn.Load() != nil {
		return errors.New("already connected")
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, urlStr, nil)
	if err != nil {
		return err
	}
	conn.SetReadLimit(MAX_GATEWAY_MESSAGE_SIZE)

	// Publish the connection before setting up compression so the shutdown
	// watcher can close it even if the server never sends a zlib header.
	if !s.conn.CompareAndSwap(nil, conn) {
		_ = conn.Close()
		return errors.New("already connected")
	}
	s.readDeadline.Store(0)

	// The shutdown watcher may have observed cancellation before the dial
	// completed and exited while conn was still unpublished.
	if err := ctx.Err(); err != nil {
		s.replaceConnAndClose(conn)
		return err
	}

	if !s.compress {
		return nil
	}

	if err := conn.SetReadDeadline(time.Now().Add(GATEWAY_HANDSHAKE_TIMEOUT)); err != nil {
		s.replaceConnAndClose(conn)
		return err
	}

	zr, err := zlib.NewReader(&zlibFeeder{conn: conn})
	if err != nil {
		s.replaceConnAndClose(conn)
		return err
	}

	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		_ = zr.Close()
		s.replaceConnAndClose(conn)
		return err
	}

	decoder := json.NewDecoder(zr)
	s.decoder.Store(decoder)
	if s.conn.Load() != conn {
		s.decoder.CompareAndSwap(decoder, nil)
		_ = zr.Close()
		return errors.New("connection closed during compression setup")
	}

	return nil
}

// detach removes the active connection from the socket and returns it.
// The caller closes it outside of any lock so a blocked write cannot prevent
// shutdown or reconnection.
func (s *socket) detach() *websocket.Conn {
	conn := s.conn.Swap(nil)
	s.decoder.Store(nil)
	s.readDeadline.Store(0)
	return conn
}

// replaceConnAndClose clears conn if it is still the active connection and
// closes it. It is used for setup failures before the socket is fully ready.
func (s *socket) replaceConnAndClose(conn *websocket.Conn) {
	if s.conn.CompareAndSwap(conn, nil) {
		s.decoder.Store(nil)
		s.readDeadline.Store(0)
	}

	_ = conn.Close()
}

// Closes the network connection without sending a CloseNormalClosure (1000) frame.
// This can preserve the session on Discord's side and allows the client to RESUME.
func (s *socket) closeForReconnect() error {
	conn := s.detach()
	if conn == nil {
		return nil // Nothing to do
	}

	// Intentionally skip closing the zlib reader as it only sets an internal
	// error flag (flate: f.err = io.EOF) and holds no OS resources. Calling it
	// here would race with a concurrent Decode() still draining the zlib stream
	// on the read goroutine. The abandoned reader is safely garbage collected.
	return conn.Close()
}

// Closes ws connection by sending a CloseNormalClosure (1000) control frame.
// Used when shutting down the shard/bot permanently.
func (s *socket) closeGracefully() error {
	conn := s.detach()
	if conn == nil {
		return nil // Nothing to do
	}

	_ = conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	)

	return conn.Close()
}

func (s *socket) readJSON(v any) error {
	conn := s.conn.Load()
	if conn == nil {
		return errors.New("not connected")
	}

	if err := s.refreshReadDeadline(conn); err != nil {
		return err
	}

	if !s.compress {
		return conn.ReadJSON(v)
	}

	decoder := s.decoder.Load()
	if decoder == nil {
		return errors.New("not connected")
	}

	return decoder.Decode(v)
}

// refreshReadDeadline extends the read deadline only when less than half of
// the timeout remains. Gateway traffic can produce thousands of frames per
// second, and resetting the runtime poll deadline for every one of them adds
// measurable CPU overhead without improving the timeout guarantee.
func (s *socket) refreshReadDeadline(conn *websocket.Conn) error {
	now := time.Now()
	remaining := s.readDeadline.Load() - now.UnixNano()
	if remaining > int64(GATEWAY_READ_TIMEOUT/2) {
		return nil
	}

	deadline := now.Add(GATEWAY_READ_TIMEOUT)
	if err := conn.SetReadDeadline(deadline); err != nil {
		return err
	}
	s.readDeadline.Store(deadline.UnixNano())
	return nil
}

func (s *socket) writeJSON(v any) error {
	conn := s.conn.Load()
	if conn == nil {
		return errors.New("not connected")
	}

	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if err := conn.SetWriteDeadline(time.Now().Add(GATEWAY_WRITE_TIMEOUT)); err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, payload)
}
