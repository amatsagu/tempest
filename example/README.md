# Tempest example bot

A small but complete Discord bot built on Tempest's HTTP interactions API. It is a tour of
the things a bot usually does, written in the same shape as a real project - following
event driven or feature slice style. General idea of this style is to group code that
interacts frequently with each other to life together, making it easier to write and maintain.

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
