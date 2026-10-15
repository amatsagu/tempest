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

// handle defers immediately, then does the slow work and replies; because a first
// reply already went out, the Reply edits the original response in place.
func (c *slowCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	if err := itx.Defer(true); err != nil {
		return err
	}

	time.Sleep(3 * time.Second)

	return itx.Reply(discord.ResponseMessageData{Content: "The slow task finished."}, false)
}
