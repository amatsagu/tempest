package tempest

import (
	"context"
	"testing"

	"github.com/amatsagu/tempest/discord"
)

// TestInteractionRuntimeReachesClient proves a handler can reach the host runtime
// through the interaction it was given, instead of searching for the client.
func TestInteractionRuntimeReachesClient(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	reached := false
	if err := c.Command("ping", func(_ context.Context, itx *discord.CommandInteraction) error {
		reached = itx.Runtime() == discord.Runtime(c)
		return itx.Reply(discord.ResponseMessageData{Content: "pong"}, false)
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	ctx := context.Background()
	r := acquireResponder()
	r.kind = responderHTTP
	r.ch = make(chan dispatchResult, 1)
	r.ctx = ctx
	r.client = c

	body := []byte(`{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"ping","type":1}}`)
	if _, err := c.dispatch(ctx, body, r); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	releaseResponder(r)

	if !reached {
		t.Fatal("handler could not reach the client through itx.Runtime()")
	}
}
