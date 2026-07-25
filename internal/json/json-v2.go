//go:build goexperiment.jsonv2

package json

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"io"
)

type (
	RawMessage         = jsontext.Value
	Marshaler          = jsonv2.Marshaler
	Unmarshaler        = jsonv2.Unmarshaler
	SyntaxError        = jsontext.SyntacticError
	UnmarshalTypeError = jsonv2.SemanticError
)

func Marshal(v any) ([]byte, error) { return jsonv2.Marshal(v) }
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return jsonv2.Marshal(v, jsontext.WithIndentPrefix(prefix), jsontext.WithIndent(indent))
}
func Unmarshal(data []byte, v any) error { return jsonv2.Unmarshal(data, v) }

// Shim the v1 Encoder/Decoder syntax to v2
type Encoder struct {
	w          io.Writer
	escapeHTML bool
}

func NewEncoder(w io.Writer) *Encoder    { return &Encoder{w: w, escapeHTML: true} }
func (e *Encoder) SetEscapeHTML(on bool) { e.escapeHTML = on }
func (e *Encoder) Encode(v any) error {
	return jsonv2.MarshalWrite(e.w, v, jsontext.EscapeForHTML(e.escapeHTML))
}

type Decoder struct{ dec *jsontext.Decoder }

func NewDecoder(r io.Reader) *Decoder { return &Decoder{dec: jsontext.NewDecoder(r)} }
func (d *Decoder) Decode(v any) error { return jsonv2.UnmarshalDecode(d.dec, v) }
