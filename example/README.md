# Tempest example bot

You're looking at a small Discord Application/Bot written in Go, using only Tempest - following feature slice style. General idea of this style is to keep code working together - close together (making it easier to maintain and extend).

This directory is its own Go module made so it always builds against the local
lib source.

> [!Note]
> Example code has been largely generated as example, based from other, real project - Discord
> Application running on 11k+ servers.

## Run it

```sh
# HTTP interaction app (needs the public key from the developer portal)
DISCORD_BOT_TOKEN=... DISCORD_PUBLIC_KEY=... HTTP_ADDR=127.0.0.1:8080 go run .
```

- Copy `example.dotenv` to `.env` if you prefer a file.
- Set `DEBUG=1` to turn on the runtime's debug logging.

`DISCORD_TEST_SERVER_ID` syncs commands to one server while iterating. Leave it
empty to sync globally.

## Layout

```text
main.go                       config, ClientHooks wiring, feature setup, command sync, shutdown
util/                         logger, option, embed, component, and modal helpers
feature/commands/             slash commands
feature/contextmenu/          USER and MESSAGE context menu commands
feature/components/           buttons, selects, modals, runtime component registration/unregistration
feature/patterns/             {param} / {rest...} component & modal pattern matching, builder, native polls
feature/showcase/             Components V2
feature/stats/                Components V2 runtime panel fed by Pre/Post/Error hooks
feature/image/                upload a PNG and get it back rotated
feature/tags/                 slash subcommands, two-level groups, and per-server state
feature/calc/                 pattern-matched interactive calculator ("calc:{session}:{key}")
feature/premium/              SKUs, entitlements, and subscriptions
bot_test.go                   end-to-end signed HTTP interaction integration test
```

## Compilation Sizes
Tempest bot binaries commonly ends up in range of 12 - 25MB mostly mostly due to Go internal machinery (HTTP2, JSON v2) and massive debug symbols. That's why we recommend disabling CGO (when not needed) and stripping debug symbols for production build - you can do it using command:
```sh
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" .
```

Some people additionally do binary packaging (e.g., upx --best / --lzma) to make binary even smaller at cost of slightly slower startup times.