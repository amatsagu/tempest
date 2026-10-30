package tempest

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
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
	errFieldNotFound     = errors.New("type field not found")
)

func peekInteractionType(data []byte) (InteractionType, error) {
	r := jsonReaderPool.Get().(*bytes.Reader)
	r.Reset(data)
	dec := jsonDecoderPool.Get().(*jsontext.Decoder)
	dec.Reset(r)

	defer func() {
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
			u, err := valTok.Uint()
			if err != nil {
				return 0, err
			}
			return InteractionType(u), nil
		}
		if err := dec.SkipValue(); err != nil {
			return 0, err
		}
	}

	return 0, errFieldNotFound
}

func peekComponentType(data []byte) (ComponentType, error) {
	r := jsonReaderPool.Get().(*bytes.Reader)
	r.Reset(data)
	dec := jsonDecoderPool.Get().(*jsontext.Decoder)
	dec.Reset(r)

	defer func() {
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
			u, err := valTok.Uint()
			if err != nil {
				return 0, err
			}
			return ComponentType(u), nil
		}
		if err := dec.SkipValue(); err != nil {
			return 0, err
		}
	}

	return 0, errFieldNotFound
}
