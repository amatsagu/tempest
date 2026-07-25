//go:build !goexperiment.jsonv2

package json

import (
	"encoding/json"
	"io"
)

type (
	RawMessage         = json.RawMessage
	Marshaler          = json.Marshaler
	Unmarshaler        = json.Unmarshaler
	SyntaxError        = json.SyntaxError
	UnmarshalTypeError = json.UnmarshalTypeError
)

func Marshal(v any) ([]byte, error) { return json.Marshal(v) }
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}
func Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }

// Expose standard Encoder/Decoder wrappers
type Encoder struct{ *json.Encoder }

func NewEncoder(w io.Writer) *Encoder { return &Encoder{json.NewEncoder(w)} }

type Decoder struct{ *json.Decoder }

func NewDecoder(r io.Reader) *Decoder { return &Decoder{json.NewDecoder(r)} }
