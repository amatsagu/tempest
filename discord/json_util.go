package discord

import (
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"sync"
)

var (
	emptyReader    = bytes.NewReader(nil)
	jsonReaderPool = sync.Pool{
		New: func() any {
			return bytes.NewReader(nil)
		},
	}
	jsonDecoderPool = sync.Pool{
		New: func() any {
			return jsontext.NewDecoder(emptyReader)
		},
	}
)

var (
	errInvalidJSONObject = errors.New("expected json object")
	errInvalidJSONArray  = errors.New("expected json array")
	errInvalidTypeValue  = errors.New("component type must be an integer")
	errFieldNotFound     = errors.New("type field not found")
)

func unmarshalWithDecoder(data []byte, fn func(*jsontext.Decoder) error) error {
	r := jsonReaderPool.Get().(*bytes.Reader)
	r.Reset(data)
	dec := jsonDecoderPool.Get().(*jsontext.Decoder)
	dec.Reset(r)

	err := fn(dec)
	if err == nil {
		if _, eofErr := dec.ReadToken(); eofErr != nil && eofErr != io.EOF {
			err = eofErr
		}
	}

	r.Reset(nil)
	jsonReaderPool.Put(r)
	jsonDecoderPool.Put(dec)
	return err
}

func consumeJSONObjectStart(dec *jsontext.Decoder) (bool, error) {
	if dec.PeekKind() == 'n' {
		return true, dec.SkipValue()
	}

	tok, err := dec.ReadToken()
	if err != nil {
		return false, err
	}
	if tok.Kind() != '{' {
		return false, errInvalidJSONObject
	}
	return false, nil
}

func skipJSONSpace(data []byte, i int) int {
	for i < len(data) {
		switch data[i] {
		case ' ', '\t', '\r', '\n':
			i++
		default:
			return i
		}
	}
	return i
}

func peekComponentType(data []byte) (ComponentType, error) {
	n := len(data)
	i := skipJSONSpace(data, 0)
	if i >= n || data[i] != '{' {
		return 0, errInvalidJSONObject
	}
	i++

	first := true
	for {
		i = skipJSONSpace(data, i)
		if i >= n {
			return 0, errInvalidJSONObject
		}
		if data[i] == '}' {
			return 0, errFieldNotFound
		}
		if !first {
			if data[i] != ',' {
				return 0, errInvalidJSONObject
			}
			i = skipJSONSpace(data, i+1)
			if i >= n {
				return 0, errInvalidJSONObject
			}
		}
		first = false

		if data[i] != '"' {
			return 0, errInvalidJSONObject
		}
		i++
		keyStart := i
		for i < n && data[i] != '"' {
			if data[i] == '\\' {
				return peekComponentTypeFallback(data)
			}
			i++
		}
		if i >= n {
			return 0, errInvalidJSONObject
		}
		key := data[keyStart:i]
		i++

		i = skipJSONSpace(data, i)
		if i >= n || data[i] != ':' {
			return 0, errInvalidJSONObject
		}
		i = skipJSONSpace(data, i+1)
		if i >= n {
			return 0, errInvalidJSONObject
		}

		if len(key) == 4 && string(key) == "type" {
			return parseComponentTypeToken(data, i)
		}

		switch data[i] {
		case '"':
			i++
			for i < n && data[i] != '"' {
				if data[i] == '\\' {
					i += 2
					continue
				}
				i++
			}
			if i >= n {
				return 0, errInvalidJSONObject
			}
			i++
		case '{', '[':
			depth := 1
			inStr := false
			i++
			for i < n && depth > 0 {
				c := data[i]
				if inStr {
					if c == '\\' {
						i += 2
						continue
					}
					if c == '"' {
						inStr = false
					}
					i++
					continue
				}
				switch c {
				case '"':
					inStr = true
				case '{', '[':
					depth++
				case '}', ']':
					depth--
				}
				i++
			}
			if depth != 0 {
				return 0, errInvalidJSONObject
			}
		default:
			for i < n {
				c := data[i]
				if c == ',' || c == '}' || c == ' ' || c == '\t' || c == '\r' || c == '\n' {
					break
				}
				i++
			}
		}
	}
}

func parseComponentTypeToken(data []byte, i int) (ComponentType, error) {
	n := len(data)
	if i >= n || data[i] < '0' || data[i] > '9' {
		return 0, errInvalidTypeValue
	}

	var val uint64
	if data[i] == '0' {
		i++
		if i < n && data[i] >= '0' && data[i] <= '9' {
			return 0, errInvalidTypeValue
		}
	} else {
		for i < n && data[i] >= '0' && data[i] <= '9' {
			val = val*10 + uint64(data[i]-'0')
			if val > 255 {
				return 0, fmt.Errorf("unknown component type: %d", val)
			}
			i++
		}
	}

	i = skipJSONSpace(data, i)
	if i >= n || (data[i] != ',' && data[i] != '}') {
		return 0, errInvalidTypeValue
	}

	return ComponentType(val), nil
}

func peekComponentTypeFallback(data []byte) (ComponentType, error) {
	r := jsonReaderPool.Get().(*bytes.Reader)
	r.Reset(data)
	dec := jsonDecoderPool.Get().(*jsontext.Decoder)
	dec.Reset(r)
	defer func() {
		r.Reset(nil)
		jsonReaderPool.Put(r)
		jsonDecoderPool.Put(dec)
	}()

	tok, err := dec.ReadToken()
	if err != nil || tok.Kind() != '{' {
		return 0, errInvalidJSONObject
	}

	for dec.PeekKind() != '}' {
		keyTok, err := dec.ReadToken()
		if err != nil {
			return 0, err
		}
		if keyTok.Kind() == '"' && keyTok.String() == "type" {
			valTok, err := dec.ReadToken()
			if err != nil {
				return 0, err
			}
			if valTok.Kind() != '0' {
				return 0, errInvalidTypeValue
			}
			u, err := valTok.Uint()
			if err != nil {
				return 0, err
			}
			if u > 255 {
				return 0, fmt.Errorf("unknown component type: %d", u)
			}
			return ComponentType(u), nil
		}
		if err := dec.SkipValue(); err != nil {
			return 0, err
		}
	}

	return 0, errFieldNotFound
}

func readJSONKeyBytes(dec *jsontext.Decoder) ([]byte, error) {
	raw, err := dec.ReadValue()
	if err != nil {
		return nil, err
	}
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return nil, errors.New("expected json string key")
	}
	inner := raw[1 : len(raw)-1]
	if bytes.IndexByte(inner, '\\') == -1 {
		return inner, nil
	}
	var decoded string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	return []byte(decoded), nil
}

func readJSONString(dec *jsontext.Decoder) (string, error) {
	if dec.PeekKind() == 'n' {
		return "", dec.SkipValue()
	}
	raw, err := dec.ReadValue()
	if err != nil {
		return "", err
	}
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", errors.New("expected json string")
	}
	inner := raw[1 : len(raw)-1]
	if bytes.IndexByte(inner, '\\') == -1 {
		return string(inner), nil
	}
	var decoded string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", err
	}
	return decoded, nil
}

func readJSONBool(dec *jsontext.Decoder) (bool, error) {
	if dec.PeekKind() == 'n' {
		return false, dec.SkipValue()
	}
	tok, err := dec.ReadToken()
	if err != nil {
		return false, err
	}
	switch tok.Kind() {
	case 't':
		return true, nil
	case 'f':
		return false, nil
	default:
		return false, errors.New("expected json boolean")
	}
}

func readJSONUint64(dec *jsontext.Decoder, maxVal uint64, overflowMsg string) (uint64, error) {
	if dec.PeekKind() == 'n' {
		return 0, dec.SkipValue()
	}
	raw, err := dec.ReadValue()
	if err != nil {
		return 0, err
	}
	v, ok := parseUint64Bytes(raw)
	if !ok {
		return 0, errors.New("expected unsigned json integer")
	}
	if v > maxVal {
		return 0, errors.New(overflowMsg)
	}
	return v, nil
}

func readJSONUint8(dec *jsontext.Decoder) (uint8, error) {
	v, err := readJSONUint64(dec, uint64(^uint8(0)), "uint8 overflow")
	return uint8(v), err
}

func readJSONUint32(dec *jsontext.Decoder) (uint32, error) {
	v, err := readJSONUint64(dec, uint64(^uint32(0)), "uint32 overflow")
	return uint32(v), err
}

func readJSONBitSet(dec *jsontext.Decoder) (BitSet, error) {
	v, err := readJSONUint64(dec, ^uint64(0), "uint64 overflow")
	return BitSet(v), err
}

func readJSONComponentType(dec *jsontext.Decoder) (ComponentType, error) {
	v, err := readJSONUint64(dec, 255, "component type overflow")
	return ComponentType(v), err
}
