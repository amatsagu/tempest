package gateway

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"io"

	"github.com/gorilla/websocket"
	"github.com/klauspost/compress/zstd"
)

// Compression selects the optional gateway transport compression.
// https://docs.discord.com/developers/events/gateway#transport-compression
type Compression uint8

const (
	// CompressionNone receives plain JSON text frames (the default).
	CompressionNone Compression = iota
	// CompressionZstd requests zstd-stream transport compression.
	CompressionZstd
)

func (c Compression) queryValue() string {
	if c == CompressionZstd {
		return "zstd-stream"
	}
	return ""
}

// frameSource turns a connection's message payloads into one continuous io.Reader
// of concatenated JSON. With onlyBinary (zstd) text frames are skipped.
type frameSource struct {
	conn       *websocket.Conn
	current    io.Reader
	initial    io.Reader
	onlyBinary bool
}

func (f *frameSource) Read(p []byte) (int, error) {
	for {
		if f.current == nil {
			if f.initial != nil {
				f.current, f.initial = f.initial, nil
			} else {
				mt, r, err := f.conn.NextReader()
				if err != nil {
					return 0, err
				}

				if f.onlyBinary && mt != websocket.BinaryMessage {
					continue
				}
				f.current = r
			}
		}

		n, err := f.current.Read(p)
		if errors.Is(err, io.EOF) {
			f.current = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

// messageDecoder reads whole gateway messages. With zstd it decodes Discord's
// continuous stream, where messages and websocket frames do not align, so reading
// is driven by JSON values rather than frame boundaries.
type messageDecoder struct {
	dec  *jsontext.Decoder
	zstd *zstd.Decoder
}

func newMessageDecoder(conn *websocket.Conn, c Compression) (*messageDecoder, error) {
	if c != CompressionZstd {
		return &messageDecoder{dec: jsontext.NewDecoder(&frameSource{conn: conn})}, nil
	}

	// Probe the first frame: a server that ignored compress=zstd-stream answers text.
	mt, first, err := conn.NextReader()
	if err != nil {
		return nil, err
	}

	if mt != websocket.BinaryMessage {
		src := &frameSource{conn: conn, initial: first}
		return &messageDecoder{dec: jsontext.NewDecoder(src)}, nil
	}

	zr, err := zstd.NewReader(&frameSource{conn: conn, initial: first, onlyBinary: true},
		zstd.WithDecoderConcurrency(1),
		zstd.WithDecoderLowmem(true),
		zstd.WithDecoderMaxMemory(1<<26),
	)
	if err != nil {
		return nil, err
	}
	return &messageDecoder{dec: jsontext.NewDecoder(zr), zstd: zr}, nil
}

// decode streams one whole JSON value straight into v, without materialising the
// raw value first.
func (d *messageDecoder) decode(v any) error {
	return json.UnmarshalDecode(d.dec, v)
}

func (d *messageDecoder) close() {
	if d.zstd != nil {
		d.zstd.Close()
	}
}
