package middleware

import (
	"fmt"
	"sync"
	"time"

	tempest "github.com/Amatsagu/Tempest"
)

const cdrValue = time.Second * 3
const cooldownBufferMaxDesiredSize = 10000

var cdrMu sync.RWMutex
var cooldowns = make(map[tempest.Snowflake]time.Time, 0)

// Simple cooldown example implementation. Rate limit: 1/3s
func Cooldown(itx tempest.CommandInteraction) *tempest.ResponseMessageData {
	cdrMu.RLock()
	cdr, available := cooldowns[itx.Member.User.ID]
	cdrMu.RUnlock()
	if !available {
		cdrMu.Lock()
		cooldowns[itx.Member.User.ID] = time.Now().Add(cdrValue)
		cdrMu.Unlock()
		if len(cooldowns) > cooldownBufferMaxDesiredSize {
			go tryCleanCooldownBuffer()
		}
		return nil
	}

	timeLeft := time.Until(cdr)
	if timeLeft <= 0 {
		cdrMu.Lock()
		delete(cooldowns, itx.Member.User.ID)
		cdrMu.Unlock()
		return nil
	}

	return &tempest.ResponseMessageData{
		Content: fmt.Sprintf("You're being on cooldown. Try again in **%.2fs**.", timeLeft.Seconds()),
	}
}

// Clear old map entries owned by members who didn't used app commands from long time
func tryCleanCooldownBuffer() {
	cdrMu.Lock()
	for ID, cdr := range cooldowns {
		if time.Until(cdr) <= 0 {
			delete(cooldowns, ID)
		}
	}
	cdrMu.Unlock()
}
