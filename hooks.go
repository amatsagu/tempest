package tempest

import (
	"time"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/route"
)

// RouteKind names the registry a matched route belongs to.
type RouteKind uint8

const (
	// RouteNone means no route matched and the interaction is dropped.
	RouteNone RouteKind = iota
	// RouteCommand is a slash, user, or message command handler.
	RouteCommand
	// RouteAutocomplete is a command autocomplete handler.
	RouteAutocomplete
	// RouteComponent is a message component handler.
	RouteComponent
	// RouteModal is a modal submit handler.
	RouteModal
)

// Route identifies the registry handler that will run for an interaction.
type Route struct {
	// Pattern is the registered key, empty when nothing matched.
	Pattern string
	// Kind is the handler family.
	Kind RouteKind
	// Exact is true when the route was matched exactly rather than through a
	// pattern.
	Exact bool
}

// InteractionInfo summarizes an interaction for the lifecycle hooks. It is passed
// by value and is only valid for the duration of the call.
type InteractionInfo struct {
	// Params holds wildcard captures for a pattern component or modal route.
	Params *route.Params
	// CustomID is set for component and modal interactions.
	CustomID string
	// CommandName is set for command and autocomplete interactions.
	CommandName string
	// Token is the interaction token, usable for follow-up responses.
	Token string
	// Route is the matched handler. Route.Kind is RouteNone when Handled is
	// false.
	Route Route
	// ApplicationID is the application the interaction targets.
	ApplicationID discord.AppID
	// ID is the interaction ID.
	ID discord.Snowflake
	// GuildID is zero in DMs.
	GuildID discord.GuildID
	// ChannelID is the channel the interaction came from.
	ChannelID discord.ChannelID
	// UserID is the invoking user, taken from the member when present.
	UserID discord.UserID
	// Duration is the total handler time, zero in OnRawInteraction. The
	// slow-handler warning instead measures time until the first reply.
	Duration time.Duration
	// ShardID is the gateway shard, or zero for HTTP.
	ShardID uint16
	// Kind is the interaction type.
	Kind discord.InteractionType
	// Response is the response type the handler returned. It is only set in
	// OnCompletedInteraction.
	Response discord.ResponseType
	// Handled reports whether a route matched and its handler will run.
	Handled bool
}

func (c *Client) interactionInfo(kind discord.InteractionType, base *discord.Interaction) InteractionInfo {
	info := InteractionInfo{
		Kind:          kind,
		ID:            base.ID,
		ApplicationID: base.ApplicationID,
		GuildID:       base.GuildID,
		ChannelID:     base.ChannelID,
		Token:         base.Token,
		ShardID:       base.ShardID,
	}

	switch {
	case base.Member != nil && base.Member.User != nil:
		info.UserID = base.Member.User.ID
	case base.User != nil:
		info.UserID = base.User.ID
	}

	return info
}

// verdict is what the registry and OnRawInteraction decided to do.
type verdict uint8

const (
	// verdictRun runs the matched registry handler.
	verdictRun verdict = iota
	// verdictUnhandled means no registry route matched.
	verdictUnhandled
	// verdictVetoed means OnRawInteraction returned false: the matched handler
	// is skipped and the interaction is not acknowledged.
	verdictVetoed
)

// begin fires OnRawInteraction and reports what should happen next.
func (c *Client) begin(info InteractionInfo) verdict {
	if c.onRawInteraction != nil && !c.onRawInteraction(info) {
		return verdictVetoed
	}

	if !info.Handled {
		return verdictUnhandled
	}
	return verdictRun
}

// slowInteraction is where a handler starts to risk Discord's 3 second response
// deadline once network latency is counted.
const slowInteraction = 2500 * time.Millisecond

// startTimer records the handler start time, but only when a hook or debug needs it.
func (c *Client) startTimer() time.Time {
	if c.onCompletedInteraction != nil || c.onInteractionError != nil || c.debug {
		return time.Now()
	}
	return time.Time{}
}

// finish reports a completed handler and, in debug mode, warns when the first
// reply missed Discord's deadline. r is nil for routes that answer without one.
func (c *Client) finish(info InteractionInfo, start time.Time, r *responder, response discord.Response) {
	if start.IsZero() {
		return
	}

	elapsed := time.Since(start)
	firstReply := r.firstReplyAfter(start)

	if c.debug && firstReply > slowInteraction {
		c.cfg.Logf("tempest: handler for route %q sent its first reply after %v; Discord needs the first response within 3s. Answer early with itx.Defer", info.Route.Pattern, firstReply)
	}

	if c.onCompletedInteraction == nil {
		return
	}
	info.Duration = elapsed
	info.Response = response.Type
	c.onCompletedInteraction(info)
}

// defaultOnInteractionError is installed in debug mode when there is no hook: it
// logs and replies with a generic ephemeral message.
func (c *Client) defaultOnInteractionError(err error, info InteractionInfo, itx any) {
	c.cfg.Logf("tempest: handler for route %q (interaction %s) failed: %v", info.Route.Pattern, info.ID, err)
	_ = replyAny(itx, discord.ResponseMessageData{Content: "Something went wrong."}, true)
}

func replyAny(itx any, data discord.ResponseMessageData, ephemeral bool) error {
	switch v := itx.(type) {
	case *discord.CommandInteraction:
		return v.Reply(data, ephemeral)
	case *discord.ComponentInteraction:
		return v.Reply(data, ephemeral)
	case *discord.ModalInteraction:
		return v.Reply(data, ephemeral)
	default:
		return nil
	}
}
