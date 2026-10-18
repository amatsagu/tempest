package gateway

import (
	"bytes"
	"encoding/json/jsontext"
	"testing"
)

// BenchmarkDecode measures one gateway message parse, the single pass that replaced
// a ReadValue followed by an Unmarshal.
func BenchmarkDecode(b *testing.B) {
	body := []byte(`{"t":"READY","s":1,"op":0,"d":{"session_id":"abc","resume_gateway_url":"wss://gateway.discord.gg","guilds":[{"id":"1"},{"id":"2"}],"shard":[0,1]}}`)

	b.ReportAllocs()

	for b.Loop() {
		dec := &messageDecoder{dec: jsontext.NewDecoder(bytes.NewReader(body))}

		var p payload
		if err := dec.decode(&p); err != nil {
			b.Fatalf("decode: %v", err)
		}
	}
}
