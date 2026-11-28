// Package tempest provides a lightweight, HTTP-first Discord interactions client
// and REST toolkit. External Gateway addons can plug into Client.HandleInteraction
// while reusing the same command, component, modal, and REST pipelines.
package tempest

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/rest"
)

const (
	ContentTypeJSON        = rest.ContentTypeJSON
	maxRequestBodySize     = 1024 * 1024 // 1 MiB interaction HTTP body limit.
	defaultAckTimeout      = 2500 * time.Millisecond
	maxPooledBodyBufferCap = 128 * 1024
)

var (
	ErrClientClosed     = errors.New("tempest: client is closed")
	ErrUnknownCommand   = errors.New("tempest: unknown command")
	ErrUnknownComponent = errors.New("tempest: unknown component")
	ErrUnknownModal     = errors.New("tempest: unknown modal")
	ErrAckTimeout       = errors.New("tempest: initial HTTP interaction acknowledgment timed out")
	ErrAlreadyResponded = errors.New("tempest: interaction has already been responded to")
	ErrAlreadyDeferred  = errors.New("tempest: interaction has already been responded to or deferred")
)

// Re-exported REST types so callers importing only github.com/amatsagu/tempest
// can configure REST, uploads, and rate limits cleanly.
type (
	Rest              = rest.RestClient
	RestClient        = rest.RestClient
	RestClientConfig  = rest.RestClientConfig
	CustomRequest     = rest.CustomRequest
	RateLimiter       = rest.RateLimiter
	RateLimiterConfig = rest.RateLimiterConfig
	File              = rest.File
)

// Creates a standalone Discord REST client.
func NewRest(cfg RestClientConfig) *Rest {
	return rest.NewRestClient(cfg)
}

// Optional lifecycle hooks executed around command, component, and modal handlers.
type ClientHooks struct {
	PreCommand     func(ctx context.Context, cmd Command, itx *CommandInteraction) bool
	PostCommand    func(ctx context.Context, cmd Command, itx *CommandInteraction)
	CommandError   func(ctx context.Context, cmd Command, itx *CommandInteraction, err error)
	PreComponent   func(ctx context.Context, itx *ComponentInteraction) bool
	PostComponent  func(ctx context.Context, itx *ComponentInteraction)
	ComponentError func(ctx context.Context, itx *ComponentInteraction, err error)
	PreModal       func(ctx context.Context, itx *ModalInteraction) bool
	PostModal      func(ctx context.Context, itx *ModalInteraction)
	ModalError     func(ctx context.Context, itx *ModalInteraction, err error)
}

// Configuration for constructing a [Client]. Token and PublicKey are required.
type ClientConfig struct {
	Context                    context.Context                  // Base lifecycle context for async/deferred handlers (defaults to context.Background()).
	TraceLogger                *log.Logger                      // Optional logger used when tracing is enabled.
	Rest                       *rest.RestClient                 // Optional pre-built REST client; nil builds one from RestConfig.
	Token                      string                           // Bot token (with or without "Bot " prefix).
	PublicKey                  string                           // Hex-encoded 32-byte Ed25519 application public key.
	DefaultInteractionContexts []discord.InteractionContextType // Default command contexts; defaults to [discord.GuildContextType].
	Hooks                      ClientHooks                      // Optional Pre/Post/Error hooks.
	RestConfig                 rest.RestClientConfig            // Used only when Rest is nil.
	AckTimeout                 time.Duration                    // Max wait for initial HTTP interaction reply (defaults to 2.5s).
	MaxConcurrentHandlers      int                              // Concurrency cap (0 uses max(GOMAXPROCS*64, 512); < 0 disables limit).
	Trace                      bool                             // Enables verbose diagnostic tracing.
}

// Core HTTP-first Discord interactions client. Implements [http.Handler] and exposes
// [Client.HandleInteraction] for external Gateway addons.
type Client struct {
	hooks           ClientHooks
	baseCtx         context.Context
	cancel          context.CancelFunc
	Rest            *rest.RestClient
	traceLogger     *log.Logger
	sem             chan struct{}
	commands        map[string]Command
	components      *patternRouter[ComponentHandler]
	modals          *patternRouter[ModalHandler]
	sessionPool     *sync.Pool
	timerPool       *sync.Pool
	readerPool      *sync.Pool
	bufferPool      *sync.Pool
	PublicKey       ed25519.PublicKey
	defaultContexts []discord.InteractionContextType
	wg              sync.WaitGroup
	cmdMu           sync.RWMutex
	ackTimeout      time.Duration
	ApplicationID   discord.AppID
	closed          atomic.Bool
	trace           bool
}

// Validates credentials, initializes internal registries and pools, and returns a configured [Client].
func NewClient(cfg ClientConfig) *Client {
	botUserID, err := rest.ExtractUserIDFromToken(cfg.Token)
	if err != nil {
		panic("failed to extract bot user ID from bot token: " + err.Error())
	}

	publicKey, err := hex.DecodeString(cfg.PublicKey)
	if err != nil {
		panic("failed to decode discord public key: " + err.Error())
	}

	if len(publicKey) != ed25519.PublicKeySize {
		panic("discord public key must be exactly 32 bytes")
	}

	contexts := []discord.InteractionContextType{discord.GuildContextType}
	if len(cfg.DefaultInteractionContexts) > 0 {
		contexts = append([]discord.InteractionContextType(nil), cfg.DefaultInteractionContexts...)
	}

	traceLogger := cfg.TraceLogger
	if traceLogger == nil {
		if cfg.Trace {
			traceLogger = log.New(os.Stdout, "[TEMPEST] ", log.LstdFlags)
		} else {
			traceLogger = log.New(io.Discard, "[TEMPEST] ", log.LstdFlags)
		}
	}

	trace := cfg.Trace || traceLogger.Writer() != io.Discard

	parentCtx := cfg.Context
	if parentCtx == nil {
		parentCtx = context.Background()
	}

	baseCtx, cancel := context.WithCancel(parentCtx)

	restClient := cfg.Rest
	if restClient == nil {
		restCfg := cfg.RestConfig
		if restCfg.Token == "" {
			restCfg.Token = cfg.Token
		}

		restCfg.Trace = restCfg.Trace || trace
		if restCfg.TraceLogger == nil {
			restCfg.TraceLogger = traceLogger
		}

		if restCfg.RateLimiterConfig.TraceLogger == nil {
			restCfg.RateLimiterConfig.TraceLogger = restCfg.TraceLogger
		}

		restClient = rest.NewRestClient(restCfg)
	}

	ackTimeout := cfg.AckTimeout
	if ackTimeout <= 0 {
		ackTimeout = defaultAckTimeout
	}

	var sem chan struct{}
	switch {
	case cfg.MaxConcurrentHandlers == 0:
		sem = make(chan struct{}, max(runtime.GOMAXPROCS(0)*64, 512))
	case cfg.MaxConcurrentHandlers > 0:
		sem = make(chan struct{}, cfg.MaxConcurrentHandlers)
	}

	return &Client{
		baseCtx:         baseCtx,
		cancel:          cancel,
		Rest:            restClient,
		PublicKey:       ed25519.PublicKey(publicKey),
		ApplicationID:   discord.AppID{Snowflake: discord.Snowflake(botUserID)},
		defaultContexts: contexts,
		hooks:           cfg.Hooks,
		traceLogger:     traceLogger,
		trace:           trace,
		ackTimeout:      ackTimeout,
		sem:             sem,
		commands:        make(map[string]Command),
		components:      newPatternRouter[ComponentHandler](),
		modals:          newPatternRouter[ModalHandler](),
		sessionPool: &sync.Pool{
			New: func() any {
				return &httpSession{
					responseCh: make(chan responsePayload, 1),
				}
			},
		},
		timerPool: &sync.Pool{
			New: func() any {
				t := time.NewTimer(time.Hour)
				t.Stop()
				return t
			},
		},
		readerPool: &sync.Pool{
			New: func() any {
				return new(io.LimitedReader)
			},
		},
		bufferPool: &sync.Pool{
			New: func() any {
				b := new(bytes.Buffer)
				b.Grow(32 * 1024)
				return b
			},
		},
	}
}

func (c *Client) tracef(format string, v ...any) {
	if !c.trace {
		return
	}

	c.traceLogger.Printf("[CLIENT] "+format, v...)
}

// Marks the client closed, cancels active interaction contexts, and waits until all
// in-flight handlers finish or ctx expires.
func (c *Client) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	c.closed.Store(true)
	if c.cancel != nil {
		c.cancel()
	}

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) SendMessage(ctx context.Context, channelID discord.ChannelID, message discord.Message, files []File) (discord.Message, error) {
	var res discord.Message
	err := c.Rest.RequestWithFiles(ctx, http.MethodPost, "/channels/"+channelID.String()+"/messages", message, files, &res)
	if err != nil {
		return discord.Message{}, err
	}

	c.tracef("Sent message ID = %d to channel ID = %d.", res.ID, channelID)
	return res, nil
}

func (c *Client) SendLinearMessage(ctx context.Context, channelID discord.ChannelID, content string) (discord.Message, error) {
	return c.SendMessage(ctx, channelID, discord.Message{Content: content}, nil)
}

// Opens (or reuses) a DM channel for userID and sends a message into it.
func (c *Client) SendPrivateMessage(ctx context.Context, userID discord.UserID, content discord.Message, files []File) (discord.Message, error) {
	var dmResp struct {
		ID discord.ChannelID `json:"id"`
	}

	err := c.Rest.Request(ctx, http.MethodPost, "/users/@me/channels", map[string]any{"recipient_id": userID}, &dmResp)
	if err != nil {
		return discord.Message{}, err
	}

	msg, err := c.SendMessage(ctx, dmResp.ID, content, files)
	msg.ChannelID = dmResp.ID

	return msg, err
}

func (c *Client) EditMessage(ctx context.Context, channelID discord.ChannelID, messageID discord.MessageID, content discord.Message) error {
	err := c.Rest.Request(ctx, http.MethodPatch, "/channels/"+channelID.String()+"/messages/"+messageID.String(), content, nil)
	if err == nil {
		c.tracef("Edited message ID = %d in channel ID = %d.", messageID, channelID)
	}

	return err
}

func (c *Client) DeleteMessage(ctx context.Context, channelID discord.ChannelID, messageID discord.MessageID) error {
	err := c.Rest.Request(ctx, http.MethodDelete, "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil, nil)
	if err == nil {
		c.tracef("Deleted message ID = %d in channel ID = %d.", messageID, channelID)
	}

	return err
}

func (c *Client) CrosspostMessage(ctx context.Context, channelID discord.ChannelID, messageID discord.MessageID) error {
	err := c.Rest.Request(ctx, http.MethodPost, "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/crosspost", nil, nil)
	if err == nil {
		c.tracef("Crossposted message ID = %d in channel ID = %d.", messageID, channelID)
	}

	return err
}

func (c *Client) FetchUser(ctx context.Context, id discord.UserID) (discord.User, error) {
	var res discord.User
	err := c.Rest.Request(ctx, http.MethodGet, "/users/"+id.String(), nil, &res)
	if err != nil {
		return discord.User{}, err
	}

	c.tracef("Fetched user %q (ID = %d).", res.GlobalName, res.ID)
	return res, nil
}

func (c *Client) FetchMember(ctx context.Context, guildID discord.GuildID, memberID discord.UserID) (discord.Member, error) {
	var res discord.Member
	err := c.Rest.Request(ctx, http.MethodGet, "/guilds/"+guildID.String()+"/members/"+memberID.String(), nil, &res)
	if err != nil {
		return discord.Member{}, err
	}

	if res.User != nil {
		c.tracef("Fetched member %q (ID = %d).", res.User.GlobalName, res.User.ID)
	}

	return res, nil
}

func (c *Client) FetchEntitlementsPage(ctx context.Context, queryFilter string) ([]discord.Entitlement, error) {
	if queryFilter != "" && queryFilter[0] != '?' {
		queryFilter = "?" + queryFilter
	}

	var res []discord.Entitlement
	err := c.Rest.Request(ctx, http.MethodGet, "/applications/"+c.ApplicationID.String()+"/entitlements"+queryFilter, nil, &res)
	if err != nil {
		return nil, err
	}

	c.tracef("Fetched %d entitlement(s).", len(res))
	return res, nil
}

func (c *Client) FetchEntitlement(ctx context.Context, entitlementID discord.EntitlementID) (discord.Entitlement, error) {
	var res discord.Entitlement
	err := c.Rest.Request(ctx, http.MethodGet, "/applications/"+c.ApplicationID.String()+"/entitlements/"+entitlementID.String(), nil, &res)
	if err != nil {
		return discord.Entitlement{}, err
	}

	c.tracef("Fetched entitlement ID = %d.", entitlementID)
	return res, nil
}

func (c *Client) ConsumeEntitlement(ctx context.Context, entitlementID discord.EntitlementID) error {
	err := c.Rest.Request(ctx, http.MethodPost, "/applications/"+c.ApplicationID.String()+"/entitlements/"+entitlementID.String()+"/consume", nil, nil)
	if err == nil {
		c.tracef("Consumed entitlement ID = %d.", entitlementID)
	}

	return err
}

func (c *Client) CreateTestEntitlement(ctx context.Context, payload discord.TestEntitlementPayload) error {
	err := c.Rest.Request(ctx, http.MethodPost, "/applications/"+c.ApplicationID.String()+"/entitlements", payload, nil)
	if err == nil {
		c.tracef("Created test entitlement.")
	}

	return err
}

func (c *Client) DeleteTestEntitlement(ctx context.Context, entitlementID discord.EntitlementID) error {
	err := c.Rest.Request(ctx, http.MethodDelete, "/applications/"+c.ApplicationID.String()+"/entitlements/"+entitlementID.String(), nil, nil)
	if err == nil {
		c.tracef("Deleted test entitlement ID = %d.", entitlementID)
	}

	return err
}
