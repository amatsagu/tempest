# Tempest example bot

A small but complete Discord bot built on the Tempest runtime. It is a tour of
the things a bot usually does, written in the same shape as a real project: one
package per feature, handlers split into command and component/modals files, and
builders used everywhere they apply.

This directory is its own Go module and `replace`s `github.com/amatsagu/tempest`
with the runtime in the parent directory, so it always builds against the local
source.

## Run it

```
# gateway mode (default)
DISCORD_BOT_TOKEN=... go run .

# HTTP interaction app (needs the public key from the developer portal)
DISCORD_BOT_TOKEN=... DISCORD_PUBLIC_KEY=... HTTP_ADDR=127.0.0.1:8080 go run .
```

Set `DEBUG=1` to turn on the runtime's debug logging (which costs nothing when
off). On start the bot syncs its command definitions and then connects.

In HTTP mode the same flag turns the endpoint into a diagnostic: opening it in a
browser returns a short status page, and failed requests explain what to fix.
With debug off, a GET answers with no content and errors stay terse.

## Layout

```
main.go                       config, feature wiring, command sync, graceful shutdown
util/                         tiny shared helpers
events/                       raw gateway events
feature/commands/             slash commands
feature/contextmenu/          USER and MESSAGE context menu commands
feature/components/           buttons, select menus, modals, dynamic handlers
feature/showcase/             Components V2
feature/stats/                a Components V2 runtime stats panel
feature/image/                upload a PNG and get it back rotated
feature/tags/                 slash subcommands and per-server in-memory state
feature/calc/                 an interactive calculator with in-memory sessions
```

Every feature exposes a `Register(client) ([]discord.Command, error)`: it
registers its handlers and returns the command definitions to sync. Registration
errors are returned rather than swallowed, so `main.go` stops on the first one
instead of running with a missing handler. `main.go` collects the definitions and
calls `SyncCommands` once. Gateway event subscription lives in `events/`, outside
`feature/`, because it is not a command.

Each command is a small struct built by a `newXCommand(deps...)` constructor that
captures what it needs (usually the client). `Definition()` returns its
`discord.Command`, and its `handle` (or `handleX`) methods answer interactions;
a feature's `registry.go` constructs the structs, registers their methods, and
returns the definitions, so there are no bare package-level handlers.

## What each feature shows

- `commands`
  - `/ping`: the smallest possible reply.
  - `/echo`: string and boolean options, an ephemeral reply.
  - `/embed`: a rich embed assembled entirely with the `builder` package.
  - `/color`: string option with autocomplete.
  - `/slow`: acknowledge at once with `itx.Defer`, then finish the slow work and
    send the result with `itx.Reply`. Because the first reply already went out as
    the defer, the reply edits the original response instead of posting a new
    message.
- `contextmenu`
  - `User info`: a USER command that reads the resolved target user.
  - `Quote`: a MESSAGE command that reads the resolved target message.
- `components`
  - `/panel`: a message with static buttons and a string select.
  - The `Open form` button answers with a modal; the modal submission is read
    back by custom ID.
  - `/ticket`: registers a one-shot component handler at runtime, keyed by the
    interaction, and removes it when it fires. This is the dynamic routing
    pattern, and the router does not accumulate entries because the handler is
    removed on use.
- `showcase`
  - `/showcase`: a Components V2 message with a container, text displays, a
    separator, a section with a thumbnail, a media gallery, and a button.
- `stats`
  - `/stats`: one Components V2 message with uptime (days/hours/minutes/seconds),
    interaction counts, memory from `runtime.ReadMemStats` and `NumGoroutine`,
    and latency. The interaction counts are fed by `OnCompletedInteraction`
    (split by route kind) and the error counter by `OnInteractionError`, so the
    panel reports what the runtime has actually handled. Latency times one cheap
    authenticated REST call (`GET /gateway/bot`) and reads the gateway shard's
    heartbeat round trip; in HTTP mode it says so instead.
- `image`
  - `/rotate`: opens a modal whose file upload accepts only `image/png`, then
    downloads the attachment, rotates it 90 degrees clockwise and posts it back.
    It uses the standard library only (`net/http`, `image`, `image/png`) plus the
    REST client's new multipart `RequestWithFiles` for the upload.
- `tags`
  - `/tag add|show|remove|list`: a slash command whose subcommands share one
    mutex-guarded store of tags, keyed by server, so state lives in memory
    rather than a database. `show` and `remove` autocomplete the tag name.
  - A `show` for a tag that does not exist returns an error instead of replying,
    which is the example's demonstration of `Config.OnInteractionError`.
- `calc`
  - `/calc`: an interactive calculator. One button handler is registered with a
    custom-ID pattern and every session's expression lives in an in-memory
    store, which each press mutates before editing the message in place.
- `events`
  - Subscribes to `GUILD_CREATE` and `GUILD_DELETE`, and observes every raw
    dispatch through `Config.OnRawEvent`.

## A few things to notice

- Registration is checked at startup. `Command` and `CommandAutocomplete` return
  an error once the client is running, because command definitions are synced
  with Discord up front. Components and modals, by contrast, can be added and
  removed at any time; that is what `/ticket` relies on. Each feature's
  `Register` returns that error (and wraps it with context); there is no
  `util.Must` panicking on it, and `util` now only holds `Ptr`.
- Lifecycle hooks replace middleware. `Config.OnRawInteraction` fires for every
  interaction except PING, as soon as the route is known. Returning true runs the
  matched handler; returning false vetoes it, skipping the handler and leaving
  the interaction unacknowledged, like Tempest's pre-command hook. An interaction
  with no matching route is dropped too, except with `DEBUG` set, where it gets an
  ephemeral hint explaining how to register a handler.
  `Config.OnCompletedInteraction` fires after a matched handler returns, with how
  long it took and the response type. All hooks cost nothing when nil.
- With `DEBUG=1` the runtime warns with an `itx.Defer` suggestion when a handler
  runs past 2.5 seconds.
- A handler returns an `error` and answers through the interaction itself:
  `itx.Reply(data, ephemeral)` sends the first response, or edits the original
  response when one already went out, which is why `/slow` can `Defer` and then
  just `Reply` its result. `itx.Update(data)` is the same swap for a component's
  own message, and `itx.Respond(resp)` is the raw escape hatch for anything else,
  such as the modal in `/panel`. The answer is delivered immediately over the
  shortest path, the HTTP response body in HTTP mode or the callback in gateway
  mode, so a slow handler is acknowledged inside Discord's 3 second window.
- A handler that fails returns an error. `Config.OnInteractionError` is called
  once per failure with the route info and the typed interaction pointer, so the
  hook can type switch and answer the user. `main.go` wires one that logs the
  error and replies `"Something went wrong."`. Leave the hook nil and the runtime
  installs a default that does the same when `DEBUG=1`; with no hook and debug
  off the error is dropped, so set it in production.
- A handler that returns nil without replying is still acknowledged: the runtime
  sends an implicit `Defer`. With `DEBUG=1` it also logs that the handler
  returned without responding.
- The runtime adapts its own tuning (connection pool, memory limit) to the host,
  so the example does not configure any of it.
