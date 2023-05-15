package middleware

import (
	"fmt"
	"time"

	tempest "github.com/Amatsagu/Tempest"
)

const cdrValue = time.Second * 3
const cooldownBufferMaxDesiredSize = 10000

var cooldowns = make(map[tempest.Snowflake]time.Time, 0)

// Simple cooldown example implementation. Rate limit: 1/3s
func Cooldown(itx tempest.CommandInteraction) *tempest.ResponseMessageData {
	cdr, available := cooldowns[itx.Member.User.ID]
	if !available {
		cooldowns[itx.Member.User.ID] = time.Now().Add(cdrValue)
		if len(cooldowns) > cooldownBufferMaxDesiredSize {
			go tryCleanCooldownBuffer()
		}
		return nil
	}

	timeLeft := time.Until(cdr)
	if timeLeft <= 0 {
		delete(cooldowns, itx.Member.User.ID)
		return nil
	}

	return &tempest.ResponseMessageData{
		Content: fmt.Sprintf("You're being on cooldown. Try again in **%.2fs**.", timeLeft.Seconds()),
	}
}

// Clear old map entries owned by members who didn't used app commands from long time
func tryCleanCooldownBuffer() {
	for ID, cdr := range cooldowns {
		if time.Until(cdr) <= 0 {
			delete(cooldowns, ID)
		}
	}
}
