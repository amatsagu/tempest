package tempest

import (
	"compress/zlib"
	"encoding/json"
	"errors"
	"io"
	"sync"

	"github.com/gorilla/websocket"
)

// Each Connection with Gateway is:
// manager -> shards -> sockets

// A thread-safe wrapper around a Gorilla WebSocket connection.
// It's designed to handle the basic lifecycle and data framing for a
// connection to the Discord Gateway.
type socket struct {

	// zlib-stream
	zreader  io.ReadCloser
	conn     *websocket.Conn
	decoder  *json.Decoder
	mu       sync.Mutex
	compress bool
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

func (s *socket) connect(urlStr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		return errors.New("already connected")
	}

	conn, _, err := websocket.DefaultDialer.Dial(urlStr, nil)
	if err != nil {
		return err
	}

	s.conn = conn

	if s.compress {
		zr, err := zlib.NewReader(&zlibFeeder{conn: conn})
		if err != nil {
			_ = conn.Close()
			s.conn = nil
			return err
		}

		s.zreader = zr
		s.decoder = json.NewDecoder(zr)
	}

	return nil
}

func (s *socket) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		return nil // Nothing to do
	}

	_ = s.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)

	// Close the underlying TCP connection.
	err := s.conn.Close()
	s.conn = nil // Mark as disconnected.

	if s.zreader != nil {
		_ = s.zreader.Close()
		s.zreader = nil
	}

	s.decoder = nil
	return err
}

func (s *socket) readJSON(v any) error {
	s.mu.Lock()
	conn := s.conn
	if conn == nil {
		s.mu.Unlock()
		return errors.New("not connected")
	}

	if !s.compress {
		s.mu.Unlock()
		return conn.ReadJSON(v)
	}

	decoder := s.decoder
	if decoder == nil {
		s.mu.Unlock()
		return errors.New("not connected")
	}
	s.mu.Unlock()

	return decoder.Decode(v)
}

func (s *socket) writeJSON(v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		return errors.New("not connected")
	}

	return s.conn.WriteJSON(v)
}
