package commands

import (
	"context"
	"time"

	"github.com/amatsagu/tempest/discord"
)

type slowCommand struct{}

func newSlowCommand() *slowCommand { return &slowCommand{} }

func (c *slowCommand) Definition() discord.Command {
	return discord.Command{
		Name:        "slow",
		Description: "Acknowledges at once, then finishes a slow task and edits the reply.",
	}
}

// handle acknowledges the interaction immediately and then finishes the work in
// place. Defer sends the acknowledgement over whichever transport is running, so
// Discord has an answer well inside its three second deadline while the handler
// keeps working, and this works the same in HTTP and gateway mode.
//
// The follow-up is a plain Reply: because the first reply already went out as
// the defer, Reply edits the original response instead of sending a new message.
func (c *slowCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	if err := itx.Defer(true); err != nil {
		return err
	}

	time.Sleep(3 * time.Second)

	return itx.Reply(discord.ResponseMessageData{Content: "The slow task finished."}, false)
}
