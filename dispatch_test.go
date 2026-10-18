package tempest

import (
	"context"
	"testing"

	"github.com/amatsagu/tempest/discord"
)

// TestInteractionTypeProbe checks the token scan that replaced the header
// decode: it finds "type" regardless of position, never mistakes a nested "type"
// for the top-level one, and defers malformed bodies to the semantic decode.
func TestInteractionTypeProbe(t *testing.T) {
	cases := []struct {
		name string
		body string
		want discord.InteractionType
	}{
		{"first member", `{"type":2,"id":"1"}`, discord.InteractionApplicationCommand},
		{"nested type ignored", `{"id":"1","data":{"type":99,"nested":{"x":[1,2,3]}},"type":3}`, discord.InteractionMessageComponent},
		{"absent", `{"id":"1","token":"t"}`, 0},
	}

	for _, tc := range cases {
		got, err := interactionType([]byte(tc.body))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}

		if got != tc.want {
			t.Fatalf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}

	if _, err := interactionType([]byte(`{"type":`)); err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
}

// TestDispatchResultIdentity covers the contract the gateway relies on: a
// dispatched interaction carries the ID and token needed to post its response
// through the callback, and a PING carries none because it never needs one.
func TestDispatchResultIdentity(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := c.Command("ping", func(_ context.Context, itx *discord.CommandInteraction) error {
		return itx.Reply(discord.ResponseMessageData{Content: "pong"}, false)
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	ctx := context.Background()

	// The handler answers through the interaction, so give dispatch a responder
	// whose channel is big enough to hold the early reply.
	r := acquireResponder()
	r.kind = responderHTTP
	r.ch = make(chan dispatchResult, 1)
	r.ctx = ctx

	result, err := c.dispatch(ctx, []byte(`{"id":"99","application_id":"2","type":2,"token":"tok","data":{"id":"3","name":"ping","type":1}}`), r)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	releaseResponder(r)
	if result.ID.String() != "99" || result.Token != "tok" {
		t.Fatalf("identity = %s/%q, want 99/tok", result.ID, result.Token)
	}

	if result.Response.Type != discord.ResponseChannelMessageWithSource {
		t.Fatalf("response type = %d", result.Response.Type)
	}

	// A dropped interaction still needs its identity, so the callback can carry
	// the unhandled notice back to Discord.
	dropped, err := c.dispatch(ctx, []byte(`{"id":"7","application_id":"2","type":2,"token":"d","data":{"id":"4","name":"missing","type":1}}`), nil)
	if err != nil {
		t.Fatalf("dispatch dropped: %v", err)
	}

	if dropped.ID.String() != "7" || dropped.Token != "d" {
		t.Fatalf("dropped identity = %s/%q, want 7/d", dropped.ID, dropped.Token)
	}

	// A PING needs no callback, so it reports no token.
	ping, err := c.dispatch(ctx, []byte(`{"id":"1","application_id":"2","type":1,"token":"t"}`), nil)
	if err != nil {
		t.Fatalf("dispatch ping: %v", err)
	}

	if ping.Token != "" {
		t.Fatalf("ping token = %q, want empty", ping.Token)
	}

	if ping.Response.Type != discord.ResponsePong {
		t.Fatalf("ping response type = %d, want pong", ping.Response.Type)
	}
}
