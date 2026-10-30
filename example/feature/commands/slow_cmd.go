package commands

import (
	"time"

	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

type slowCommand struct{}

func newSlowCommand() *slowCommand { return &slowCommand{} }

func (c *slowCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "slow",
		Description:         "Acknowledges at once, then finishes a slow task and edits the reply.",
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

// handle defers immediately, then does the slow work and replies; because a first
// reply already went out, SendReply edits the original response in place.
func (c *slowCommand) handle(itx *tempest.CommandInteraction) error {
	if err := itx.Defer(true); err != nil {
		return err
	}

	time.Sleep(1500 * time.Millisecond)

	return itx.SendReply(tempest.ResponseMessageData{Content: "The slow task finished."}, false, nil)
}
