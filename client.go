package ashara

import (
	"crypto/ed25519"
	"encoding/hex"
	"sync"
)

// Client is the core Ashara entrypoint
type Client struct {
	ApplicationID   Snowflake
	PublicKey       ed25519.PublicKey
	Rest            RestHandler
	CommandRegistry SlashCommandRegistry

	jsonBufferPool *sync.Pool
}

type ClientOptions struct {
	Token          string
	PublicKey      string
	JSONBufferSize uint
}

func NewClient(opt ClientOptions) Client {
	discordPublicKey, err := hex.DecodeString(opt.PublicKey)
	if err != nil {
		panic("failed to decode discord's public key (check if it's correct key): " + err.Error())
	}

	botUserID, err := extractUserIDFromToken(opt.Token)
	if err != nil {
		panic("failed to extract bot user ID from bot token: " + err.Error())
	}

	var poolSize uint = 4096
	if opt.JSONBufferSize > poolSize {
		poolSize = opt.JSONBufferSize
	}

	return Client{
		ApplicationID:   botUserID,
		PublicKey:       discordPublicKey,
		Rest:            NewBaseRestHandler(opt.Token),
		CommandRegistry: NewBaseSlashCommandRegistry(botUserID),
		jsonBufferPool: &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, poolSize) // start with a decent buffer
				return &buf
			},
		},
	}
}
