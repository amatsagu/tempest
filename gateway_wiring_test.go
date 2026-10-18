package tempest

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"testing"
	"time"

	"github.com/amatsagu/tempest/discord"
)

// TestGatewayInteractionCarriesShard proves the shard an interaction arrived on
// reaches the handler, instead of being dropped on the way into dispatch.
func TestGatewayInteractionCarriesShard(t *testing.T) {
	c := newGatewayClient(t, Config{})

	got := make(chan uint16, 1)
	err := c.Command("ping", func(_ context.Context, itx *discord.CommandInteraction) error {
		got <- itx.ShardID
		return errors.New("captured")
	})
	if err != nil {
		t.Fatalf("Command: %v", err)
	}

	c.handleGatewayInteraction(c.runCtx, 3, []byte(`{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"ping","type":1}}`))

	select {
	case shard := <-got:
		if shard != 3 {
			t.Fatalf("ShardID = %d, want 3", shard)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not run")
	}
}

// TestEventRegistrationIsTheSubscription proves a registered handler fires with no
// second name list to keep in sync.
func TestEventRegistrationIsTheSubscription(t *testing.T) {
	c := newGatewayClient(t, Config{})

	got := make(chan int, 1)
	c.Event("GUILD_CREATE", func(_ context.Context, shard int, _ string, _ jsontext.Value) {
		got <- shard
	})

	c.onGatewayDispatch(1, "GUILD_CREATE", jsontext.Value(`{"id":"1"}`))

	select {
	case shard := <-got:
		if shard != 1 {
			t.Fatalf("shard = %d, want 1", shard)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("event handler did not run")
	}
}
