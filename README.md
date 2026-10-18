<div align="center">
    <img align="center" src="/.github/tempest-banner.png" height="165" alt="Tempest library banner">
</div>

<p align="center">
  <i align="center">Create lightning fast Discord Applications</i>
</p>

<h4 align="center">
    <a href="#features">Features</a> - <a href="#http-vs-gateway">HTTP vs Gateway</a> - <a href="#getting-started">Getting started</a> - <a href="#troubleshooting">Troubleshooting</a> - <a href="#contributing">Contributing</a>
</h4>

<h4 align="center">
    <a href="https://pkg.go.dev/github.com/amatsagu/tempest">
        <img src="https://pkg.go.dev/badge/github.com/amatsagu/tempest.svg" alt="Go Reference">
    </a>
    <a href="https://golang.org/doc/devel/release.html">
        <img src="https://img.shields.io/github/go-mod/go-version/amatsagu/tempest" alt="Go Version">
    </a>
    <a href="https://github.com/amatsagu/tempest/blob/master/LICENSE">
        <img src="https://img.shields.io/github/license/amatsagu/tempest" alt="License">
    </a>
    <a href="https://github.com/amatsagu/tempest">
        <img src="https://img.shields.io/maintenance/yes/2026" alt="Maintenance Status">
    </a>
    <a href="https://github.com/amatsagu/tempest/actions/workflows/github-code-scanning/codeql">
        <img src="https://github.com/amatsagu/tempest/actions/workflows/github-code-scanning/codeql/badge.svg?branch=master" alt="CodeQL">
    </a>
    <a href="https://conventionalcommits.org">
        <img src="https://img.shields.io/badge/Conventional%20Commits-1.0.0-%23FE5196?logo=conventionalcommits&logoColor=white" alt="Conventional Commits">
    </a>
</h4>

**Project: Tempest** is a modern, minimal [Discord](https://discord.com) library for building Discord Apps, written in [Go](https://golang.org/). It aims to be extremely fast, stay very close to the Discord API, and include little to no caching, making it a good fit for a small VPS or a serverless architecture.

In real-world projects using this library, your bottlenecks will almost always be your database or network bandwidth, not the app or bot itself.

The runtime is split into small packages, so you can take only what you need:

| Package | What it does |
| --- | --- |
| `tempest` | The runtime: config, registries, routing, hooks, HTTP endpoint, gateway wiring |
| `discord` | The Discord data model and its JSON codecs |
| `builder` | Ergonomic constructors for the data model |
| `rest` | Rate-limit aware REST client, including raw requests |
| `gateway` | Optional gateway connection, sharding, presence |
| `route` | The pattern router used for component and modal custom IDs |

### Features

- [x] Secure HTTPS-based communication with Discord's API using `crypto/ed25519`, verified in place with no body copy
- [x] Automatic dispatching of application commands, message components, autocomplete, and modals
- [x] One interaction object carries everything: reply, defer, update, follow-ups, file uploads, and input reading
- [x] Subcommands routed with `client.Command("parent sub", handler)`
- [x] Pattern routing for dynamic custom IDs, with runtime add and remove
- [x] Rate limiter that learns Discord's buckets, keeps channel and guild limits separate, and bounds its own memory
- [x] File uploads on replies and follow-ups
- [x] Messages, embeds, modals, guilds, bitfields, and [Components v2](https://docs.discord.com/developers/components/overview) with full type safety
- [x] Optional gateway connection with auto-sharding, reconnect and resume, presence, shard state, and heartbeat latency
- [x] Zero-cost debug logging: when `Debug` is off the log sites are guarded and allocate nothing
- [x] One opt-in helper, `tempest.TuneRuntime()`, for sizing `GOMAXPROCS` and `GOMEMLIMIT` to a container; it is never called for you
- [x] Monetization: SKUs, entitlements for one-time purchases and the shop, and app subscriptions, with the entitlement gateway events
- [x] REST client exposed for extension, including raw requests, response headers, and multipart uploads

<details>
<summary>Supports all currently available component types</summary>

  - [x] Action Row (1)
  - [x] Button (2)
  - [x] String Select (3)
  - [x] Text Input (4)
  - [x] User Select (5)
  - [x] Role Select (6)
  - [x] Mentionable Select (7)
  - [x] Channel Select (8)
  - [x] Section (9)
  - [x] Text Display (10)
  - [x] Thumbnail (11)
  - [x] Media Gallery (12)
  - [x] File (13)
  - [x] Separator (14)
  - [x] Container (17)
  - [x] Label (18)
  - [x] File Upload (19)
  - [x] Radio Group (21)
  - [x] Checkbox Group (22)
  - [x] Checkbox (23)

</details>

### HTTP vs Gateway

There are two ways for an app to receive interactions. Most Discord libraries use a WebSocket connection called the "gateway"; Tempest defaults to receiving interactions over HTTPS. HTTP scales more easily and uses fewer resources, at the cost of the gateway-only events. Roles, minigames, custom messages, or admin utilities work well over HTTP, while music or moderation bots need the gateway.

Set `Config.HTTPAddr` for HTTP mode, or `Config.Gateway` for gateway mode. Exactly one may be configured. The gateway exists for large applications that want lower latency and the full event stream.

### Getting started

```
go get -u github.com/amatsagu/tempest
```

```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/gateway"
	"github.com/amatsagu/tempest/route"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Optional: nothing in the library touches the Go runtime. This sizes GOMAXPROCS
	// and GOMEMLIMIT to the container's cgroup limits.
	tempest.TuneRuntime()

	// Setting HTTP_ADDR selects HTTP mode; otherwise the gateway is used.
	httpAddr := os.Getenv("HTTP_ADDR")

	client, err := tempest.New(tempest.Config{
		Token:     os.Getenv("DISCORD_BOT_TOKEN"),
		PublicKey: os.Getenv("DISCORD_PUBLIC_KEY"), // required for HTTP mode
		HTTPAddr:  httpAddr,
		Gateway:   httpAddr == "",
		Intents:   uint32(gateway.IntentGuilds),
	})
	if err != nil {
		slog.Error("cannot start", "err", err)
		os.Exit(1)
	}

	// A handler answers through the interaction it was given and returns only an error.
	err = client.Command("ping", func(_ context.Context, itx *discord.CommandInteraction) error {
		return itx.Reply(discord.ResponseMessageData{Content: "Pong!"}, false)
	})
	if err != nil {
		slog.Error("cannot register", "err", err)
		os.Exit(1)
	}

	// Components and modals register the same way and may be added or removed at runtime.
	err = client.Component("ping:again", func(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
		card := builder.Embed("Again", "Pong!")
		return itx.Update(discord.ResponseMessageData{Embeds: []discord.Embed{card}})
	})
	if err != nil {
		slog.Error("cannot register", "err", err)
		os.Exit(1)
	}

	if client.REST() != nil {
		commands := []discord.Command{{
			Name:        "ping",
			Description: "Check that the app is alive.",
		}}
		if err := client.SyncCommands(ctx, discord.GuildID{}, commands, nil); err != nil {
			slog.Error("cannot sync commands", "err", err)
			os.Exit(1)
		}
	}

	if client.Gateway() != nil {
		_ = client.Gateway().UpdateStatus(func(shard int) *gateway.Presence {
			return &gateway.Presence{
				Status:     gateway.StatusOnline,
				Activities: []gateway.Activity{{Name: "Type /ping", Type: gateway.ActivityCustom}},
			}
		})
	}

	if err := client.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("app stopped", "err", err)
		os.Exit(1)
	}
}
```

A handler answers through the interaction it receives:

```go
itx.Defer(true)                                     // acknowledge, then send later
itx.Reply(data, ephemeral)                          // first reply, or an edit of the original
itx.Reply(data, ephemeral, files...)                // reply with attachments
itx.Update(data)                                    // edit the component's own message
itx.FollowUp(data, ephemeral)                       // an extra message
itx.Respond(builder.ModalResponse(id, title, ...))  // open a modal
```

and reads input the same way:

```go
name, ok := itx.Data.StringOption("name")     // also searches subcommand options
option, ok := itx.Data.Option("value")
focused, value, ok := itx.Data.Focused()      // autocomplete
member, ok := itx.Data.Resolved.Member(userID)
input, ok := itx.Data.TextInput("field")      // modal
```

For anything the library does not wrap, the interaction reaches the runtime directly:

```go
body, err := itx.Runtime().Request(ctx, http.MethodGet, "/guilds/"+id.String(), nil, "")
```

Monetization is wrapped too. Gate premium features on entitlements rather than on a
subscription's status:

```go
skus, err := client.SKUs(ctx)

entitlements, err := client.Entitlements(ctx, &discord.EntitlementFilter{
    UserID:       user.ID,
    ExcludeEnded: true,
})

if len(entitlements) > 0 && entitlements[0].Active(time.Now()) {
    // grant the premium feature
}
```

Subscribe to `gateway.EventEntitlementCreate` to fulfil a purchase, and call
`client.ConsumeEntitlement` once a consumable has been delivered. The example bot
has a complete `/premium` command in `example/feature/premium`.

Set `Config.Debug` with a `Config.Logf` sink to log what the runtime rejects, including a diagnostic reply for an interaction with no matching route.

(Optional) Check the [examples folder](./example) for a complete bot that doubles as documentation.

## Troubleshooting

For help or feature requests, feel free to open an issue on GitHub. \
You can also contact me on [discord](https://discord.com/users/390394829789593601).

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines on how to contribute to this project.

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FAmatsagu%2FTempest.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2FAmatsagu%2FTempest?ref=badge_large)
