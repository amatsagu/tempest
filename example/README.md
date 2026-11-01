# Tempest example bot

A small but complete Discord bot built on Tempest's master API. It is a tour of
the things a bot usually does, written in the same shape as a real project - following
event driven or feature slice style. General idea of this style is to group code that
interacts frequently with each other to life together, making it easier to write and maintain.

This directory is its own Go module made so it always builds against the local
lib source.

> [!Note]
> Example code has been largely generated as example, based from other, real project - Discord
Application running on 11k+ servers. Tempest normally focuses on HTTP client but this example
bot code intentionally attempts to mix both to show how things can be implemented from either side.

## Run it

```sh
# HTTP interaction app (needs the public key from the developer portal)
DISCORD_BOT_TOKEN=... DISCORD_PUBLIC_KEY=... HTTP_ADDR=127.0.0.1:8080 go run .

# gateway mode (alternative client, default fallback)
DISCORD_BOT_TOKEN=... go run .
```

- Copy `example.dotenv` to `.env` if you prefer a file.
- Set `DEBUG=1` to turn on the runtime's debug logging.

`DISCORD_TEST_SERVER_ID` syncs commands to one server while iterating. Leave it
empty to sync globally. Adding `GATEWAY_ZLIB=1` will enable compression while using gateway.

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