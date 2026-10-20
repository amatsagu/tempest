package tempest

import (
	"context"
	"testing"

	"github.com/amatsagu/tempest/discord"
)

// subCommandBody is an INTERACTION_CREATE for "tag" whose only option is the
// "add" subcommand carrying a "name" string.
const subCommandBody = `{"id":"99","application_id":"2","type":2,"token":"tok","data":{"id":"3","name":"tag","type":1,"options":[{"name":"add","type":1,"options":[{"name":"name","type":3,"value":"hi"}]}]}}`

// mirrors TestDispatchResultIdentity's HTTP responder.
func newDispatchResponder(ctx context.Context) *responder {
	r := acquireResponder()
	r.kind = responderHTTP
	r.ch = make(chan dispatchResult, 1)
	r.ctx = ctx
	return r
}

// Checks whether a handler registered under
// "parent sub" sees the subcommand's options flattened to the top level.
func TestDispatchFlattensRegisteredSubCommand(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var gotName, gotArg string
	if err := c.Command("tag add", func(_ context.Context, itx *discord.CommandInteraction) error {
		gotName = itx.Data.Name
		gotArg, _ = itx.Data.StringOption("name")
		return nil
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	ctx := context.Background()
	r := newDispatchResponder(ctx)

	if _, err := c.dispatch(ctx, []byte(subCommandBody), r); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	releaseResponder(r)

	if gotName != "tag add" {
		t.Fatalf("handler name = %q, want tag add", gotName)
	}

	if gotArg != "hi" {
		t.Fatalf("StringOption(name) = %q, want hi", gotArg)
	}
}

// Checks whether that without a
// registered subcommand handler a parent handler still sees the raw nesting, and
// the recursive search still finds the argument.
func TestDispatchKeepsParentOptionsWithoutSubCommandHandler(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var (
		gotName string
		gotType discord.OptionType
		gotArg  string
	)

	if err := c.Command("tag", func(_ context.Context, itx *discord.CommandInteraction) error {
		gotName = itx.Data.Name
		if len(itx.Data.Options) > 0 {
			gotType = itx.Data.Options[0].Type
		}

		gotArg, _ = itx.Data.StringOption("name")
		return nil
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	ctx := context.Background()
	r := newDispatchResponder(ctx)

	if _, err := c.dispatch(ctx, []byte(subCommandBody), r); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	releaseResponder(r)

	if gotName != "tag" {
		t.Fatalf("handler name = %q, want tag", gotName)
	}

	if gotType != discord.OptionSubCommand {
		t.Fatalf("Options[0].Type = %d, want OptionSubCommand", gotType)
	}

	if gotArg != "hi" {
		t.Fatalf("recursive StringOption(name) = %q, want hi", gotArg)
	}
}

// subCommandGroupBody is an INTERACTION_CREATE for "settings" whose option is the
// "profile" group containing the "set" subcommand carrying a "theme" string.
const subCommandGroupBody = `{"id":"100","application_id":"2","type":2,"token":"tok","data":{"id":"4","name":"settings","type":1,"options":[{"name":"profile","type":2,"options":[{"name":"set","type":1,"options":[{"name":"theme","type":3,"value":"dark"}]}]}]}}`

func TestDispatchFlattensSubCommandGroupLeaf(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var gotName, gotTheme string
	if err := c.Command("settings profile set", func(_ context.Context, itx *discord.CommandInteraction) error {
		gotName = itx.Data.Name
		gotTheme, _ = itx.Data.StringOption("theme")
		return nil
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	ctx := context.Background()
	r := newDispatchResponder(ctx)

	if _, err := c.dispatch(ctx, []byte(subCommandGroupBody), r); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	releaseResponder(r)

	if gotName != "settings profile set" {
		t.Fatalf("handler name = %q, want settings profile set", gotName)
	}

	if gotTheme != "dark" {
		t.Fatalf("StringOption(theme) = %q, want dark", gotTheme)
	}
}

func TestDispatchFlattensSubCommandGroupLevel(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var gotName string
	var optionsCount int
	if err := c.Command("settings profile", func(_ context.Context, itx *discord.CommandInteraction) error {
		gotName = itx.Data.Name
		optionsCount = len(itx.Data.Options)
		return nil
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	ctx := context.Background()
	r := newDispatchResponder(ctx)

	if _, err := c.dispatch(ctx, []byte(subCommandGroupBody), r); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	releaseResponder(r)

	if gotName != "settings profile" {
		t.Fatalf("handler name = %q, want settings profile", gotName)
	}

	if optionsCount != 1 {
		t.Fatalf("len(Options) = %d, want 1 (the sub command)", optionsCount)
	}
}
