package tempest

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/gateway"
	"github.com/amatsagu/tempest/rest"
)

// Errors returned by the client at startup.
var (
	// ErrNoEndpoint is returned by Run when neither HTTPAddr nor Gateway is set.
	ErrNoEndpoint = errors.New("tempest: set Config.HTTPAddr or Config.Gateway")

	// ErrMultipleModes is returned by New when both HTTPAddr and Gateway are set.
	ErrMultipleModes = errors.New("tempest: configure either HTTPAddr or Gateway, not both")

	// ErrRunning is returned by Command and CommandAutocomplete once the client
	// is already running; components and modals may still be registered.
	ErrRunning = errors.New("tempest: commands must be registered before Run")
)

// Client is the central runtime: build one with New, register handlers, then call
// Run. Exactly one transport mode (HTTP or gateway) may be configured.
type Client struct {
	runCtx                 context.Context
	handlers               *handlerSet
	rest                   *rest.Client
	gateway                *gateway.Gateway
	httpSrv                *http.Server
	events                 map[string]EventHandler
	onRawInteraction       func(InteractionInfo) bool
	onCompletedInteraction func(InteractionInfo)
	onInteractionError     func(err error, info InteractionInfo, itx any)
	publicKey              ed25519.PublicKey
	cfg                    Config
	res                    resolved
	appID                  discord.AppID
	eventsMu               sync.RWMutex
	closeOnce              sync.Once
	httpSrvMu              sync.Mutex
	debug                  bool
	running                atomic.Bool
}

// The client is the runtime handlers reach through itx.Runtime.
var _ discord.Runtime = (*Client)(nil)

// New validates the configuration and applies host-adapted runtime tuning.
func New(cfg Config) (*Client, error) {
	if cfg.HTTPAddr != "" && cfg.Gateway {
		return nil, ErrMultipleModes
	}

	c := &Client{
		cfg:      cfg,
		handlers: newHandlerSet(),
		events:   make(map[string]EventHandler),
	}

	c.res = cfg.resolve()
	c.debug = cfg.Debug && cfg.Logf != nil
	c.onRawInteraction = cfg.OnRawInteraction
	c.onCompletedInteraction = cfg.OnCompletedInteraction
	c.onInteractionError = cfg.OnInteractionError

	// With no error hook but debug logging on, install a default that logs and
	// tells the user something went wrong.
	if c.onInteractionError == nil && c.debug {
		c.onInteractionError = c.defaultOnInteractionError
	}

	switch {
	case cfg.PublicKey != "":
		pub, err := parsePublicKey(cfg.PublicKey)
		if err != nil {
			return nil, err
		}
		c.publicKey = pub
	case cfg.HTTPAddr != "":
		return nil, errors.New("tempest: Config.PublicKey is required to serve HTTP interactions")
	}

	if cfg.Token != "" {
		c.rest = rest.New(cfg.Token, rest.Options{
			MaxIdleConnsPerHost: c.res.IdleConnsPerHost,
			MaxRetries:          3,
		})

		c.appID = cfg.ApplicationID
		if c.appID.Snowflake == 0 {
			if id, ok := appIDFromToken(cfg.Token); ok {
				c.appID = id
			}
		}
	}

	// The gateway is built here so it is available to configure before Run, for
	// example to set the bot status with Client.Gateway().UpdateStatus.
	if cfg.Gateway {
		c.gateway = gateway.New(gateway.Config{
			Token:       cfg.Token,
			Intents:     cfg.Intents,
			Compression: cfg.Compression,
			ShardCount:  cfg.ShardCount,
			Presence:    cfg.Presence,
			AutoShard:   true,
			OnDispatch:  c.onGatewayDispatch,
			OnError:     c.onGatewayError,
			OnReshard:   c.onReshard,
			Debug:       c.debug,
			Logf:        cfg.Logf,
		})
	}

	return c, nil
}

// Config returns the configuration the client was built with.
func (c *Client) Config() Config { return c.cfg }

// Resolved exposes the host-adapted tuning in effect (useful for diagnostics).
func (c *Client) Resolved() resolved { return c.res }

// REST exposes the shared REST client, or nil when no token is configured.
func (c *Client) REST() *rest.Client { return c.rest }

// Gateway exposes the running gateway, or nil when gateway mode is disabled.
func (c *Client) Gateway() *gateway.Gateway { return c.gateway }

// ApplicationID returns the application ID (from Config or derived from Token).
func (c *Client) ApplicationID() discord.AppID { return c.appID }

// Command registers a handler for a slash/user/message command by name, or returns
// ErrRunning once the client is running.
func (c *Client) Command(name string, h CommandHandler) error {
	if c.running.Load() {
		return ErrRunning
	}

	c.handlers.command(name, h)
	return nil
}

// CommandAutocomplete registers an autocomplete handler for a command by name.
// It returns ErrRunning if the client is already running (see Command).
func (c *Client) CommandAutocomplete(name string, h AutocompleteHandler) error {
	if c.running.Load() {
		return ErrRunning
	}

	c.handlers.autocompleteCommand(name, h)
	return nil
}

// Component registers a handler for a component custom ID, safe to call at runtime;
// an ID containing {name} is a pattern matching on '/' and ':'.
func (c *Client) Component(id string, h ComponentHandler) error {
	return c.handlers.components.register(id, h)
}

// RemoveComponent unregisters a component handler by custom ID. Removing an
// unknown ID is a no-op.
func (c *Client) RemoveComponent(id string) {
	c.handlers.components.remove(id)
}

// Modal registers a handler for a modal custom ID, like Component.
func (c *Client) Modal(id string, h ModalHandler) error {
	return c.handlers.modals.register(id, h)
}

// RemoveModal unregisters a modal handler by custom ID. Removing an unknown ID
// is a no-op.
func (c *Client) RemoveModal(id string) {
	c.handlers.modals.remove(id)
}

// Event registers a handler for a gateway event (only when Gateway is enabled).
// Registering the handler is the subscription, so there is no separate name list
// to keep in sync. Register before Run so the gateway sees it.
func (c *Client) Event(name string, h EventHandler) *Client {
	c.eventsMu.Lock()
	c.events[name] = h
	c.eventsMu.Unlock()
	return c
}

// Handler returns the interaction endpoint as an http.Handler. Developers may
// mount it in their own server and add HTTP-level middleware around it.
func (c *Client) Handler() http.Handler { return c }

// CommandFilter selects which commands SyncCommands sends. A nil filter sends
// every command. Names match top-level command names.
type CommandFilter struct {
	// Names to keep, or to drop when Exclude is true.
	Names []string
	// Exclude flips Names from a whitelist into a blacklist.
	Exclude bool
}

func (f *CommandFilter) allows(name string) bool {
	if f == nil || len(f.Names) == 0 {
		return true
	}

	listed := false
	for _, n := range f.Names {
		if n == name {
			listed = true
			break
		}
	}
	return listed != f.Exclude
}

// SyncCommands replaces the global command set, or a guild's when guildID is
// non-zero. filter may be nil to send every command. A definition with a zero
// Type is sent as discord.CommandChatInput, and one without Contexts inherits
// Config.DefaultContexts.
func (c *Client) SyncCommands(ctx context.Context, guildID discord.GuildID, commands []discord.Command, filter *CommandFilter) error {
	if c.rest == nil {
		return errors.New("tempest: Config.Token is required to sync commands")
	}

	if c.appID.Snowflake == 0 {
		return errors.New("tempest: application ID is unknown; set Config.ApplicationID")
	}

	path := "/applications/" + c.appID.String() + "/commands"
	if guildID.Snowflake != 0 {
		path = "/applications/" + c.appID.String() + "/guilds/" + guildID.String() + "/commands"
	}

	return c.rest.RequestInto(ctx, http.MethodPut, path, c.prepareCommands(commands, filter), "", nil)
}

// prepareCommands copies the commands that pass the filter, applying the
// CommandChatInput and DefaultContexts defaults. Copies are made so the caller's
// definitions are left as registered.
func (c *Client) prepareCommands(commands []discord.Command, filter *CommandFilter) []discord.Command {
	synced := make([]discord.Command, 0, len(commands))

	for _, cmd := range commands {
		if !filter.allows(cmd.Name) {
			if c.debug {
				c.cfg.Logf("tempest: skipping command %q (filtered out)", cmd.Name)
			}
			continue
		}

		if cmd.Type == 0 {
			cmd.Type = discord.CommandChatInput
			if c.debug {
				c.cfg.Logf("tempest: command %q had no type, using CommandChatInput", cmd.Name)
			}
		}

		if len(cmd.Contexts) == 0 && len(c.cfg.DefaultContexts) > 0 {
			cmd.Contexts = c.cfg.DefaultContexts
		}

		synced = append(synced, cmd)
	}
	return synced
}

// EditOriginal replaces the message an interaction created; handlers normally use
// itx.Reply, which does this automatically after a first reply.
func (c *Client) EditOriginal(ctx context.Context, itx *discord.Interaction, data discord.ResponseMessageData, files ...discord.File) error {
	return c.editOriginal(ctx, itx.ApplicationID, itx.Token, data, files...)
}

func (c *Client) editOriginal(ctx context.Context, appID discord.AppID, token string, data discord.ResponseMessageData, files ...discord.File) error {
	if c.rest == nil {
		return errors.New("tempest: Config.Token is required to edit an interaction response")
	}

	path := editOriginalPath(appID, token)
	if len(files) > 0 {
		_, err := c.rest.RequestWithFiles(ctx, http.MethodPatch, path, data, files, "")
		return err
	}
	return c.rest.RequestInto(ctx, http.MethodPatch, path, data, "", nil)
}

func editOriginalPath(appID discord.AppID, token string) string {
	return "/webhooks/" + appID.String() + "/" + token + "/messages/@original"
}

func (c *Client) followUp(ctx context.Context, appID discord.AppID, token string, data discord.ResponseMessageData, ephemeral bool, files []discord.File) (*discord.Message, error) {
	if c.rest == nil {
		return nil, errors.New("tempest: Config.Token is required to send a follow-up")
	}

	if ephemeral {
		data.Flags |= discord.MessageFlagEphemeral
	}

	path := "/webhooks/" + appID.String() + "/" + token

	var msg discord.Message
	if len(files) > 0 {
		body, err := c.rest.RequestWithFiles(ctx, http.MethodPost, path, data, files, "")
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(body, &msg); err != nil {
			return nil, err
		}
		return &msg, nil
	}

	if err := c.rest.RequestInto(ctx, http.MethodPost, path, data, "", &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (c *Client) editFollowUp(ctx context.Context, appID discord.AppID, token string, messageID discord.MessageID, data discord.ResponseMessageData, files []discord.File) error {
	if c.rest == nil {
		return errors.New("tempest: Config.Token is required to edit a follow-up")
	}

	path := "/webhooks/" + appID.String() + "/" + token + "/messages/" + messageID.String()
	if len(files) > 0 {
		_, err := c.rest.RequestWithFiles(ctx, http.MethodPatch, path, data, files, "")
		return err
	}
	return c.rest.RequestInto(ctx, http.MethodPatch, path, data, "", nil)
}

func (c *Client) deleteFollowUp(ctx context.Context, appID discord.AppID, token string, messageID discord.MessageID) error {
	if c.rest == nil {
		return errors.New("tempest: Config.Token is required to delete a follow-up")
	}

	path := "/webhooks/" + appID.String() + "/" + token + "/messages/" + messageID.String()
	_, err := c.rest.Request(ctx, http.MethodDelete, path, nil, "")
	return err
}

// SendMessage posts a message to a channel, with files when given.
func (c *Client) SendMessage(ctx context.Context, channelID discord.ChannelID, data discord.ResponseMessageData, files ...discord.File) (*discord.Message, error) {
	if c.rest == nil {
		return nil, errors.New("tempest: Config.Token is required to send a message")
	}

	path := "/channels/" + channelID.String() + "/messages"

	var msg discord.Message
	if len(files) > 0 {
		body, err := c.rest.RequestWithFiles(ctx, http.MethodPost, path, data, files, "")
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(body, &msg); err != nil {
			return nil, err
		}
		return &msg, nil
	}

	if err := c.rest.RequestInto(ctx, http.MethodPost, path, data, "", &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// EditMessage replaces a message a channel already has.
func (c *Client) EditMessage(ctx context.Context, channelID discord.ChannelID, messageID discord.MessageID, data discord.ResponseMessageData) error {
	if c.rest == nil {
		return errors.New("tempest: Config.Token is required to edit a message")
	}

	path := "/channels/" + channelID.String() + "/messages/" + messageID.String()
	return c.rest.RequestInto(ctx, http.MethodPatch, path, data, "", nil)
}

// DeleteMessage removes a message from a channel.
func (c *Client) DeleteMessage(ctx context.Context, channelID discord.ChannelID, messageID discord.MessageID) error {
	if c.rest == nil {
		return errors.New("tempest: Config.Token is required to delete a message")
	}

	path := "/channels/" + channelID.String() + "/messages/" + messageID.String()
	_, err := c.rest.Request(ctx, http.MethodDelete, path, nil, "")
	return err
}

// FetchUser returns a user by ID.
func (c *Client) FetchUser(ctx context.Context, id discord.UserID) (discord.User, error) {
	var user discord.User
	if c.rest == nil {
		return user, errors.New("tempest: Config.Token is required to fetch a user")
	}

	err := c.rest.RequestInto(ctx, http.MethodGet, "/users/"+id.String(), nil, "", &user)
	return user, err
}

// FetchMember returns a guild member by user ID.
func (c *Client) FetchMember(ctx context.Context, guildID discord.GuildID, userID discord.UserID) (discord.Member, error) {
	var member discord.Member
	if c.rest == nil {
		return member, errors.New("tempest: Config.Token is required to fetch a member")
	}

	path := "/guilds/" + guildID.String() + "/members/" + userID.String()
	err := c.rest.RequestInto(ctx, http.MethodGet, path, nil, "", &member)
	return member, err
}

// Request performs one raw REST call and returns the response body, for endpoints
// the library does not wrap. path is relative to the API base.
func (c *Client) Request(ctx context.Context, method, path string, payload any, auditReason string) ([]byte, error) {
	if c.rest == nil {
		return nil, errors.New("tempest: Config.Token is required to make a request")
	}
	return c.rest.Request(ctx, method, path, payload, auditReason)
}

// RequestWithFiles is Request with a multipart body.
func (c *Client) RequestWithFiles(ctx context.Context, method, path string, payload any, files []discord.File, auditReason string) ([]byte, error) {
	if c.rest == nil {
		return nil, errors.New("tempest: Config.Token is required to make a request")
	}
	return c.rest.RequestWithFiles(ctx, method, path, payload, files, auditReason)
}

// RequestHeaders is Request with the response headers, for endpoints driven by them.
func (c *Client) RequestHeaders(ctx context.Context, method, path string, payload any, auditReason string) ([]byte, http.Header, error) {
	if c.rest == nil {
		return nil, nil, errors.New("tempest: Config.Token is required to make a request")
	}
	return c.rest.RequestHeaders(ctx, method, path, payload, auditReason)
}

// Run starts the configured endpoint and blocks until ctx is cancelled or it
// stops with an error.
func (c *Client) Run(ctx context.Context) error {
	if !c.cfg.Gateway && c.cfg.HTTPAddr == "" {
		return ErrNoEndpoint
	}

	if c.cfg.Gateway && c.cfg.Token == "" {
		return errors.New("tempest: Config.Token is required for the gateway")
	}

	c.running.Store(true)

	runCtx, cancel := context.WithCancel(ctx)
	c.runCtx = runCtx
	defer cancel()

	errCh := make(chan error, 1)

	if c.cfg.Gateway {
		c.configureGatewayShards(runCtx)
		go func() { errCh <- c.gateway.Run(runCtx) }()
	} else {
		c.startHTTP()
		go func() { errCh <- c.httpSrv.ListenAndServe() }()
	}

	select {
	case <-runCtx.Done():
		return c.shutdown()
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) || errors.Is(err, context.Canceled) {
			return c.shutdown()
		}
		_ = c.shutdown()
		return err
	}
}

// shutdownGrace bounds how long Run waits for in-flight work to drain.
const shutdownGrace = 10 * time.Second

func (c *Client) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	return c.Close(ctx)
}

// Close stops the endpoint gracefully, waiting for in-flight work until ctx is
// done. It is safe to call multiple times; only the first call does the work.
func (c *Client) Close(ctx context.Context) error {
	var err error
	c.closeOnce.Do(func() {
		if c.gateway != nil {
			// Stop the shards first: once they are down no more work is spawned.
			if gwErr := c.gateway.Close(ctx); gwErr != nil {
				err = gwErr
			}
		}

		c.httpSrvMu.Lock()
		srv := c.httpSrv
		c.httpSrvMu.Unlock()

		if srv != nil {
			if shutdownErr := srv.Shutdown(ctx); shutdownErr != nil {
				err = shutdownErr
			}
		}
	})
	return err
}

func (c *Client) startHTTP() {
	c.httpSrvMu.Lock()
	defer c.httpSrvMu.Unlock()

	c.httpSrv = &http.Server{
		Addr:              c.cfg.HTTPAddr,
		Handler:           c,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       75 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}
}

// configureGatewayShards asks Discord for the recommended shard count and
// endpoint, overriding the configured values. It is best effort: on failure the
// gateway starts with whatever Config set.
func (c *Client) configureGatewayShards(ctx context.Context) {
	if c.cfg.ShardCount > 0 || c.rest == nil {
		return
	}

	bot, err := c.rest.GatewayBot(ctx)
	if err != nil {
		return
	}

	c.gateway.SetBotInfo(bot.Shards, bot.URL, bot.MaxConcurrency)
	if c.debug {
		c.cfg.Logf("gateway: auto-configured %d shard(s), endpoint %s", bot.Shards, bot.URL)
	}
}

func (c *Client) onGatewayError(shard int, err error) {
	if c.debug {
		c.cfg.Logf("gateway: shard %d error: %v", shard, err)
	}
}

func (c *Client) onReshard(shards int) {
	if c.debug {
		c.cfg.Logf("gateway: resharding to %d shard(s)", shards)
	}
}

// onGatewayDispatch runs on the shard goroutine and must return quickly, so it
// hands real work to its own goroutine. Only handled events are copied.
func (c *Client) onGatewayDispatch(shard int, event string, data jsontext.Value) {
	if c.cfg.OnRawEvent != nil {
		c.cfg.OnRawEvent(shard, event, data)
	}

	if event != gateway.EventInteraction && !c.wantsEvent(event) {
		return
	}

	// data is copied because the shard reuses its read and decompression
	// buffers the moment this returns.
	go c.runGatewayJob(shard, event, bytes.Clone(data))
}

func (c *Client) wantsEvent(event string) bool {
	c.eventsMu.RLock()
	_, ok := c.events[event]
	c.eventsMu.RUnlock()
	return ok
}

func (c *Client) runGatewayJob(shard int, event string, data jsontext.Value) {
	if event == gateway.EventInteraction {
		c.handleGatewayInteraction(c.runCtx, shard, data)
		return
	}

	c.eventsMu.RLock()
	handler, ok := c.events[event]
	c.eventsMu.RUnlock()
	if ok {
		handler(c.runCtx, shard, event, data)
	}
}

func (c *Client) handleGatewayInteraction(ctx context.Context, shard int, data []byte) {
	responder := acquireResponder()
	defer releaseResponder(responder)
	responder.kind = responderGateway
	responder.client = c
	responder.ctx = ctx
	responder.shard = uint16(shard)

	result, err := c.dispatch(ctx, data, responder)
	if err != nil {
		if c.debug {
			c.cfg.Logf("gateway: failed to parse interaction: %v", err)
		}
		return
	}

	// close claims the reply for the transport: an early Defer, Respond, or the
	// handler's return value, whichever happened.
	if !responder.close() {
		return
	}

	// An empty token (PING or unknown type) or a declined interaction is not posted.
	if c.rest == nil || result.Token == "" || result.NoResponse {
		return
	}

	_ = c.answer(ctx, result.ID, result.Token, result.Response)
}

func (c *Client) answer(ctx context.Context, id discord.Snowflake, token string, r discord.Response) error {
	if c.rest == nil {
		return errors.New("tempest: Config.Token is required to answer interactions")
	}
	_, err := c.rest.Request(ctx, http.MethodPost, "/interactions/"+id.String()+"/"+token+"/callback", r, "")
	return err
}
