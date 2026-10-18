package tempest

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"sync"
	"time"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/route"
)

// unhandledResponse is the debug diagnostic for an interaction with no route.
func unhandledResponse(typ discord.InteractionType, key string) discord.Response {
	noun, hint := "command", "client.Command(\""+key+"\", handler)"
	switch typ {
	case discord.InteractionMessageComponent:
		noun, hint = "component", "client.Component(\""+key+"\", handler)"
	case discord.InteractionModalSubmit:
		noun, hint = "modal", "client.Modal(\""+key+"\", handler)"
	}

	return discord.Response{
		Type: discord.ResponseChannelMessageWithSource,
		Data: discord.ResponseMessageData{
			Content: "Tempest: received a " + noun + " interaction with no matching route (\"" + key +
				"\"). Register it with " + hint + ".",
			Flags: discord.MessageFlagEphemeral,
		},
	}
}

// explainOrDrop fills result for an unrouted interaction: a debug diagnostic, or
// a deliberate drop.
func (c *Client) explainOrDrop(result *dispatchResult, typ discord.InteractionType, key string) {
	if c.debug {
		result.Response = unhandledResponse(typ, key)
		return
	}
	result.NoResponse = true
}

// dispatchResult is the outcome of dispatching one interaction. An empty Token
// means nothing needs posting (PING or an unknown type); NoResponse means the
// runtime deliberately dropped the interaction without acknowledging it.
type dispatchResult struct {
	Response   discord.Response
	Token      string
	ID         discord.Snowflake
	NoResponse bool
}

// flattenSubCommand rewrites data to the registered subcommand route when one
// exists under "parent sub", so the handler sees the subcommand's own options at
// the top level and Data.Name identifies the route that ran. It leaves data alone
// when no subcommand handler is registered, letting one parent handler dispatch.
func (c *Client) flattenSubCommand(data *discord.CommandInteractionData, registered func(string) bool) {
	path, options, ok := data.SubCommand()
	if !ok {
		return
	}

	key := data.Name + " " + path
	if !registered(key) {
		return
	}
	data.Name, data.Options = key, options
}

// dispatch probes the interaction type and hands off to dispatchType.
func (c *Client) dispatch(ctx context.Context, body []byte, r *responder) (dispatchResult, error) {
	typ, err := interactionType(body)
	if err != nil {
		return dispatchResult{}, err
	}
	return c.dispatchType(ctx, typ, body, r)
}

// dispatchType is dispatch for a caller that already probed the type, so the token
// scan runs once per request instead of twice on the HTTP endpoint. It parses the
// body, resolves its route, runs the lifecycle hooks, and executes the matching
// handler.
func (c *Client) dispatchType(ctx context.Context, typ discord.InteractionType, body []byte, r *responder) (dispatchResult, error) {
	switch typ {
	case discord.InteractionPing:
		return dispatchResult{Response: discord.Response{Type: discord.ResponsePong}}, nil

	case discord.InteractionApplicationCommand:
		var itx discord.CommandInteraction
		if err := json.Unmarshal(body, &itx); err != nil {
			return dispatchResult{}, err
		}

		c.flattenSubCommand(&itx.Data, func(key string) bool {
			_, ok := c.handlers.commandHandler(key)
			return ok
		})

		handler, ok := c.handlers.commandHandler(itx.Data.Name)
		info := c.interactionInfo(typ, &itx.Interaction)
		info.CommandName = itx.Data.Name
		info.Handled = ok
		if ok {
			info.Route = Route{Kind: RouteCommand, Pattern: itx.Data.Name, Exact: true}
		}

		result := dispatchResult{ID: info.ID, Token: info.Token}
		switch c.begin(info) {
		case verdictRun:
			r.install(&itx.Interaction)
			defer r.uninstall(&itx.Interaction)

			start := c.startTimer()
			herr := handler(ctx, &itx)
			c.settle(info, start, r, &itx.Interaction, &itx, herr, &result)
		case verdictVetoed:
			result.NoResponse = true
		default:
			c.explainOrDrop(&result, typ, itx.Data.Name)
		}

		return result, nil

	case discord.InteractionApplicationCommandAutocomplete:
		var itx discord.CommandInteraction
		if err := json.Unmarshal(body, &itx); err != nil {
			return dispatchResult{}, err
		}

		c.flattenSubCommand(&itx.Data, func(key string) bool {
			_, ok := c.handlers.autocompleteHandler(key)
			return ok
		})

		r.install(&itx.Interaction)
		defer r.uninstall(&itx.Interaction)

		h, ok := c.handlers.autocompleteHandler(itx.Data.Name)
		info := c.interactionInfo(typ, &itx.Interaction)
		info.CommandName = itx.Data.Name
		info.Handled = ok
		if ok {
			info.Route = Route{Kind: RouteAutocomplete, Pattern: itx.Data.Name, Exact: true}
		}

		result := dispatchResult{ID: info.ID, Token: info.Token}
		if c.begin(info) != verdictRun {
			result.Response = autocompleteResponse(nil)
			return result, nil
		}
		start := c.startTimer()
		choices, herr := h(ctx, &itx)
		if herr != nil {
			c.reportInteractionError(herr, info, start, &itx)
		}
		result.Response = autocompleteResponse(choices)
		c.finish(info, start, nil, result.Response)
		return result, nil

	case discord.InteractionMessageComponent:
		var itx discord.ComponentInteraction
		if err := json.Unmarshal(body, &itx); err != nil {
			return dispatchResult{}, err
		}

		r.install(&itx.Interaction)
		defer r.uninstall(&itx.Interaction)

		var params route.Params
		h, matched, ok := c.handlers.components.match(itx.Data.CustomID, &params)
		info := c.interactionInfo(typ, &itx.Interaction)
		info.CustomID = itx.Data.CustomID
		info.Params = &params
		info.Handled = ok
		if ok {
			matched.Kind = RouteComponent
			info.Route = matched
		}

		result := dispatchResult{ID: info.ID, Token: info.Token}
		switch c.begin(info) {
		case verdictRun:
			start := c.startTimer()
			herr := h(ctx, &itx, &params)
			c.settle(info, start, r, &itx.Interaction, &itx, herr, &result)
		case verdictVetoed:
			result.NoResponse = true
		default:
			c.explainOrDrop(&result, typ, itx.Data.CustomID)
		}
		return result, nil

	case discord.InteractionModalSubmit:
		var itx discord.ModalInteraction
		if err := json.Unmarshal(body, &itx); err != nil {
			return dispatchResult{}, err
		}

		r.install(&itx.Interaction)
		defer r.uninstall(&itx.Interaction)

		var params route.Params
		h, matched, ok := c.handlers.modals.match(itx.Data.CustomID, &params)
		info := c.interactionInfo(typ, &itx.Interaction)
		info.CustomID = itx.Data.CustomID
		info.Params = &params
		info.Handled = ok
		if ok {
			matched.Kind = RouteModal
			info.Route = matched
		}

		result := dispatchResult{ID: info.ID, Token: info.Token}
		switch c.begin(info) {
		case verdictRun:
			start := c.startTimer()
			herr := h(ctx, &itx, &params)
			c.settle(info, start, r, &itx.Interaction, &itx, herr, &result)
		case verdictVetoed:
			result.NoResponse = true
		default:
			c.explainOrDrop(&result, typ, itx.Data.CustomID)
		}
		return result, nil
	}

	// An unrecognised interaction type has no route and no meaningful key.
	return dispatchResult{NoResponse: true}, nil
}

// settle folds a handler's outcome into the transport result and reports completion.
func (c *Client) settle(info InteractionInfo, start time.Time, r *responder, itx *discord.Interaction, typed any, err error, result *dispatchResult) {
	if err != nil {
		c.reportInteractionError(err, info, start, typed)
	}

	switch {
	case r.responded():
		result.Response = r.effective(discord.Response{})
	case err != nil:
		// The error hook had its chance; without a reply the interaction is left
		// unanswered, the documented production behaviour.
		result.NoResponse = true
	default:
		// A handler that returns nil without replying still gets an implicit defer.
		if derr := itx.Defer(false); derr != nil {
			result.NoResponse = true
		} else if c.debug {
			c.cfg.Logf("tempest: route %q returned without responding; sent an implicit defer", info.Route.Pattern)
		}
		result.Response = r.effective(discord.Response{})
	}

	c.finish(info, start, r, result.Response)
}

func (c *Client) reportInteractionError(err error, info InteractionInfo, start time.Time, itx any) {
	if c.onInteractionError == nil {
		return
	}

	if !start.IsZero() {
		info.Duration = time.Since(start)
	}
	c.onInteractionError(err, info, itx)
}

func autocompleteResponse(choices []discord.CommandOptionChoice) discord.Response {
	return discord.Response{
		Type: discord.ResponseApplicationCommandAutocomplete,
		Data: discord.ResponseAutocompleteData{Choices: choices},
	}
}

// typeProbePool reuses a token decoder and reader; after warmup the scan allocates nothing.
var typeProbePool = sync.Pool{
	New: func() any {
		reader := bytes.NewReader(nil)
		return &typeProbe{
			reader:  reader,
			decoder: jsontext.NewDecoder(reader),
		}
	},
}

type typeProbe struct {
	decoder *jsontext.Decoder
	reader  *bytes.Reader
}

// interactionType returns the top-level "type" via a token scan that skips every
// other member. It does not validate the whole body: anything it cannot interpret
// falls back to a semantic decode, which surfaces malformed JSON as an error.
func interactionType(body []byte) (discord.InteractionType, error) {
	probe := typeProbePool.Get().(*typeProbe)

	probe.reader.Reset(body)
	probe.decoder.Reset(probe.reader)

	typ, ok := scanInteractionType(probe.decoder)

	// Drop the body reference so the pooled reader does not keep it alive.
	probe.reader.Reset(nil)
	typeProbePool.Put(probe)

	if ok {
		return typ, nil
	}

	var header struct {
		Type discord.InteractionType `json:"type"`
	}

	if err := json.Unmarshal(body, &header); err != nil {
		return 0, err
	}
	return header.Type, nil
}

func scanInteractionType(dec *jsontext.Decoder) (discord.InteractionType, bool) {
	if dec.PeekKind() != jsontext.KindBeginObject {
		return 0, false
	}

	if _, err := dec.ReadToken(); err != nil {
		return 0, false
	}

	for dec.PeekKind() != jsontext.KindEndObject {
		name, err := dec.ReadToken()
		if err != nil {
			return 0, false
		}

		if name.Kind() == jsontext.KindString && name.String() == "type" {
			value, err := dec.ReadToken()
			if err != nil {
				return 0, false
			}
			n, err := value.Int()
			if err != nil {
				return 0, false
			}
			return discord.InteractionType(n), true
		}

		if err := dec.SkipValue(); err != nil {
			return 0, false
		}
	}

	return 0, false
}
