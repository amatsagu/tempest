package tempest

import (
	"errors"
	"sync"

	"github.com/gorilla/websocket"
)

// Each Qord connection with Gateway is:
// manager -> shards -> sockets

// A thread-safe wrapper around a Gorilla WebSocket connection.
// It's designed to handle the basic lifecycle and data framing for a
// connection to the Discord Gateway.
type socket struct {
	conn *websocket.Conn
	mu   sync.Mutex
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

	return err
}

func (s *socket) readJSON(v any) error {
	s.mu.Lock()
	if s.conn == nil {
		s.mu.Unlock()
		return errors.New("not connected")
	}
	s.mu.Unlock() // Unlock early to allow writes while we block on read.

	return s.conn.ReadJSON(v)
}

func (s *socket) writeJSON(v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		return errors.New("not connected")
	}

	return s.conn.WriteJSON(v)
}
