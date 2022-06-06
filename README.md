# Tempest

<img alt="Github license badge" src="https://img.shields.io/github/license/Amatsagu/tempest" />
<img alt="Maintenance badge" src="https://img.shields.io/maintenance/yes/2024" />

> A simple, robust framework for creating discord applications in typescript (for deno runtime).

- Easily scalable!
- No cache by default,
- Fully typed,

Library is already usable _(even used in production!)_ but still misses a lot of elements:

## Supported parts

- [x] Webhook _(reversed rest api)_ web server for receiving incoming payloads
- [x] (Slash) Command handler _(both normal & sub commands)_
- [x] Button menus handler
- [x] Button handler
- [x] Creating/Editing/Deleting/Crossposting regular messages
- [x] REST handler wish built-in rate limit protection
- [x] Followes camelCase _(all Discord's snake_case payloads follows JS/TS standards)_
- [x] Data compression to lower memory footprint _(ids are turn into bigints & some codes into hashes)_
- [x] Helpful error messages when creating interactions
- [ ] Select menus _(no way to handle created menus)_
- [ ] User/Text messages commands
- [ ] Modals
- [ ] Multi-language support

## Performance

Tempest is interaction focused library for Discord apps. We don't relay on gateway so there's far less spam and we can
handle more at cost of a bit higher ping. How much?

Deno uses Rust's Hyper crate for dealing with networking
_([benchmark](https://deno.land/benchmarks#http-server-throughput))_. Average deno http server can handle around
`40K req/sec on Windows` and about `70K req/sec on Linux`. Assuming you use linux server - your app would need
_(approximately)_ `~300K discord guilds` to hit throughput issues. That's efficiency of `~120 gateway shards`! On top of
that - single webhook will likely take far less resources than process with 60 ws sockets. Additionally - scalling
discord apps is super easy. Just spawn new mirror process and link it with for example nginx's balanceloader. Scalling
gateway based bot can be a nightmare.

All of that cost you just a bit higher average ping and of course Discord apps are still a bit limited in functionality.
Pick your poison :)
