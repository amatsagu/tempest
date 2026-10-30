# Tempest example bot

A small but complete Discord bot built on Tempest's master API. It is a tour of
the things a bot usually does, written in the same shape as a real project: one
package per feature, handlers split into command and component/modal files, and
small shared helpers instead of package-level handlers.

This directory is its own Go module and `replace`s `github.com/amatsagu/tempest`
with the runtime in the parent directory, so it always builds against the local
source.

## Run it

```sh
# gateway mode (default)
DISCORD_BOT_TOKEN=... go run .

# HTTP interaction app (needs the public key from the developer portal)
DISCORD_BOT_TOKEN=... DISCORD_PUBLIC_KEY=... HTTP_ADDR=127.0.0.1:8080 go run .
```

Copy `example.dotenv` to `.env` if you prefer a file; it is loaded automatically.
Set `DEBUG=1` to turn on the runtime's debug logging. On start the bot registers
all features, syncs command definitions, and then either connects to the gateway
or serves the HTTP interaction endpoint.

`DISCORD_TEST_SERVER_ID` syncs commands to one server while iterating. Leave it
empty to sync globally. `GATEWAY_ZLIB=1` turns on gateway compression.

## Layout

```text
main.go                       config, feature wiring, command sync, shutdown
util/                         logger, option, handler, embed, component, and modal helpers
events/                       one gateway CustomEventHandler with a switch
feature/commands/             slash commands
feature/contextmenu/          USER and MESSAGE context menu commands
feature/components/           buttons, selects, modals, dynamic handlers
feature/showcase/             Components V2
feature/stats/                a Components V2 runtime stats panel
feature/image/                upload a PNG and get it back rotated
feature/tags/                 slash subcommands and per-server in-memory state
feature/calc/                 an interactive calculator with in-memory sessions
feature/premium/              SKUs, entitlements, and subscriptions
```

Every feature exposes a `Register` function that registers its handlers and
returns its command definitions. Registration errors are returned rather than
swallowed, so `main.go` stops on the first one instead of running with a missing
handler. `main.go` collects the definitions and calls
`BaseClient.SyncCommandsWithDiscord` once.

## What each feature shows

- `commands`
  - `/ping`: the smallest possible reply.
  - `/echo`: string and boolean options, an ephemeral reply, and `dm`/`channel`
    options demonstrating `SendPrivateMessage` and `SendLinearMessage`.
  - `/embed`: a rich embed assembled from plain Tempest types.
  - `/color`: a string option with autocomplete.
  - `/slow`: defers immediately, then edits the original response when the task
    finishes.
- `contextmenu`
  - `User info`: a USER command that reads the resolved target user.
  - `Quote`: a MESSAGE command that reads the resolved target message.
  - `Publish`: a MESSAGE command that crossposts an announcement.
- `components`
  - `/panel`: a message with static buttons and a string select.
  - The `Open form` button answers with a modal; modal submissions reply with
    an ephemeral embed.
  - `/ticket`: registers a one-shot component listener after replying, then
    removes it when it fires or times out.
- `showcase`
  - `/showcase`: a Components V2 message with a container, text displays, a
    separator, a section with a thumbnail, a media gallery, and a button.
- `stats`
  - `/stats`: one Components V2 message with uptime, command count, memory from
    `runtime.ReadMemStats`, goroutine count, and REST/gateway latency.
- `image`
  - `/rotate`: opens a modal whose file upload accepts `.png`, downloads the
    attachment, rotates it 90 degrees clockwise, and posts it back.
- `tags`
  - `/tag add|show|remove|list`: subcommands sharing one mutex-guarded store
    keyed by server. `show` and `remove` autocomplete the tag name.
  - `/tag admin clear`: demonstrates a subcommand group route.
- `calc`
  - `/calc`: an interactive calculator. Each invocation registers its own
    button listener with `AwaitComponent`, and each press edits the message in
    place.
- `events`
  - `events.NewHandler` returns the single `CustomEventHandler` used by
    `GatewayClient`: a plain switch over `packet.Event`.
  - It decodes `GUILD_CREATE`, `GUILD_DELETE`, and the three entitlement events.
    Add another case when a feature needs another gateway event.
- `premium`
  - `/premium status`: active user and guild entitlements.
  - `/premium skus`: the app's SKUs.
  - `/premium subscriptions`: subscriptions for one autocompleted SKU.
  - Entitlement lifecycle events are shown in `events/`, which is where
    fulfillment belongs.

## Master API notes

This example targets the current `tempest` package, not the rewrite branch. A
few differences are visible in the code:

- The entrypoint creates a `GatewayClient` or `HTTPClient` directly; there is no
  unified `tempest.Client`.
- Feature handlers use master's `RegisterCommand`, `RegisterSubCommand`,
  `RegisterComponent`, and `RegisterModal`. Handler signatures do not return an
  error, so `util.CommandHandler`, `util.ComponentHandler`, and
  `util.ModalHandler` adapt the feature methods and log failures.
- There is no per-interaction `context.Context` in handlers yet. `/slow` uses a
  timer, and image/modal work is bounded by the REST client.
- Components are registered by exact custom ID. Dynamic buttons use
  `AwaitComponent`, which acknowledges the component and leaves the feature to
  edit the original message with `CommandInteraction.EditReply`. This is why
  `/ticket` and `/calc` use command closures rather than a component pattern
  router.
- Modal interactions have no file-reply helper on master, so `/rotate` defers,
  then PATCHes `/webhooks/{app}/{token}/messages/@original` with
  `Rest.RequestWithFiles`.
- Master has `PreCommandHook`/`PostCommandHook` but no completed-interaction or
  error hooks. `/stats` therefore counts commands through `PreCommandHook`; it
  does not claim to count components or modals.
- Master has entitlement methods but no SKU or subscription client methods.
  `/premium` uses `Rest.Request` against the documented `/skus` and
  `/skus/{sku}/subscriptions` endpoints with local response types.
- Only one `CustomEventHandler` exists. `events.NewHandler` returns that single
  function with a `switch packet.Event`, matching how `GatewayClientOptions`
  exposes custom events. Add cases instead of registering separate handlers.
- Tempest logs through the standard library's `*log.Logger`. The example uses
  the same logger (`util.Logger`), and passes it into `BaseClientOptions.Logger`
  when `DEBUG` is set so runtime and example logs share one destination.
- Gateway mode is context-controlled: canceling the context passed to
  `Gateway.Start` closes active shards, stops startup, and returns. There is no
  separate `Stop()` call, matching how HTTP mode shuts down its `http.Server`
  with a context.
