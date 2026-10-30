package tempest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/klauspost/compress/zstd"
)

// Each Connection with Gateway is:
// manager -> shards -> sockets

// A thread-safe wrapper around a Gorilla WebSocket connection.
// It's designed to handle the basic lifecycle and data framing for a
// connection to the Discord Gateway.
type socket struct {
	conn         atomic.Pointer[websocket.Conn]
	decoder      atomic.Pointer[json.Decoder]
	zstdDecoder  *zstd.Decoder // Reused across reconnects to retain its buffers.
	readDeadline atomic.Int64
	writeMu      sync.Mutex
	compress     bool
}

// Handles zero-allocation streaming from WebSocket frames.
type websocketStreamReader struct {
	conn   *websocket.Conn
	reader io.Reader
}

func (f *websocketStreamReader) Read(p []byte) (int, error) {
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

	targetURL, err := validateGatewayURL(urlStr)
	if err != nil {
		return err
	}

	if !gatewayURLPattern.MatchString(targetURL) {
		return errors.New("gateway URL is not allowed")
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, targetURL, nil)
	if err != nil {
		return err
	}
	conn.SetReadLimit(MAX_GATEWAY_MESSAGE_SIZE)

	// Publish the connection before setting up compression so the shutdown
	// watcher can close it even if the server never sends compressed data.
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

	stream := &websocketStreamReader{conn: conn}
	zr, err := s.zstdReader(stream)
	if err != nil {
		s.replaceConnAndClose(conn)
		return err
	}

	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		s.releaseCurrentDecoder()
		s.replaceConnAndClose(conn)
		return err
	}

	decoder := json.NewDecoder(zr)
	s.decoder.Store(decoder)
	if s.conn.Load() != conn {
		s.decoder.CompareAndSwap(decoder, nil)
		s.releaseCurrentDecoder()
		return errors.New("connection closed during compression setup")
	}

	return nil
}

// zstdReader returns the socket's reusable zstd decoder, reset to stream. The
// decoder's history ring is the largest part of its working set (about 0.7 MiB
// for a 256 KiB zstd window and 0.3 MiB for a 64 KiB window), so keeping it
// across reconnects avoids reallocating that memory on every disconnect.
func (s *socket) zstdReader(stream io.Reader) (*zstd.Decoder, error) {
	if s.zstdDecoder != nil {
		if err := s.zstdDecoder.Reset(stream); err == nil {
			return s.zstdDecoder, nil
		}
		// A closed decoder cannot be reset. Discard it and create a new one.
		s.zstdDecoder.Close()
		s.zstdDecoder = nil
	}

	// Concurrency 1 avoids async worker goroutines. Gateway payloads are
	// small, so the scheduling overhead of parallel block decoding would
	// outweigh any benefit here. The decoder sizes its window from the zstd
	// frame header, so no fixed window size is needed.
	zr, err := zstd.NewReader(
		stream,
		zstd.WithDecoderConcurrency(1),
		zstd.WithDecoderLowmem(true),
	)
	if err != nil {
		return nil, err
	}
	s.zstdDecoder = zr
	return zr, nil
}

// releaseCurrentDecoder drops the current stream from the decoders after the
// read loop returns. The zstd decoder keeps its buffers for the next reconnect;
// only its reference to the old websocket connection is cleared. This is
// intentionally not called from detach: the read loop may still be blocked
// inside the decoder while the socket is being closed.
func (s *socket) releaseCurrentDecoder() {
	if s.zstdDecoder != nil {
		_ = s.zstdDecoder.Reset(nil)
	}
}

// closeDecoders releases decoder memory when the shard stops permanently.
func (s *socket) closeDecoders() {
	if s.zstdDecoder != nil {
		s.zstdDecoder.Close()
		s.zstdDecoder = nil
	}
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

	// The decoder is released by readLoop after Decode returns. Resetting or
	// closing it here would race with a read goroutine still blocked inside it.
	return conn.Close()
}

// Closes ws connection by sending a CloseNormalClosure (1000) control frame.
// Used when shutting down the shard/bot permanently.
func (s *socket) closeGracefully() (bool, error) {
	conn := s.detach()
	if conn == nil {
		return false, nil // Nothing to do
	}

	_ = conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	)

	return true, conn.Close()
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
