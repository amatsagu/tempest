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
    <a href="https://goreportcard.com/report/github.com/amatsagu/tempest">
        <img src="https://goreportcard.com/badge/github.com/amatsagu/tempest" alt="Go Report">
    </a>
    <a href="https://golang.org/doc/devel/release.html">
        <img src="https://img.shields.io/github/go-mod/go-version/amatsagu/tempest" alt="Go Version">
    </a>
    <a href="https://github.com/amatsagu/tempest/blob/development/LICENSE">
        <img src="https://img.shields.io/github/license/Amatsagu/tempest" alt="License">
    </a>
    <a href="https://github.com/amatsagu/tempest">
        <img src="https://img.shields.io/maintenance/yes/2025" alt="Maintenance Status">
    </a>
    <a href="https://github.com/amatsagu/tempest/actions/workflows/github-code-scanning/codeql">
        <img src="https://github.com/amatsagu/tempest/actions/workflows/github-code-scanning/codeql/badge.svg?branch=master" alt="CodeQL">
    </a>
    <a href="https://conventionalcommits.org">
        <img src="https://img.shields.io/badge/Conventional%20Commits-1.0.0-%23FE5196?logo=conventionalcommits&logoColor=white" alt="Conventional Commits">
    </a>
</h4>

**Project: Tempest** is a modern, minimal [Discord](https://discord.com) library for building Discord Apps, written in [Go](https://golang.org/). It aims to be extremely fast, stay very close to the Discord API, and include little to no caching - making it an excellent choice for small VPS or serverless architecture. In real-world projects using this lib, your bottlenecks will 9/10 cases be in the database or network bandwidth, not in app/bot itself.

It was created as a better alternative to [discord-interactions-go](https://github.com/bsdlp/discord-interactions-go), which is too low-level and outdated.

### Features

- [x] Secure HTTPS-based communication with the Discord API using `crypto/ed25519`
- [x] Automatic dispatching of:
    - [x] Application commands
    - [x] Message components (buttons, select menu, text input)
    - [x] Autocomplete interactions
    - [x] Modal interactions  
- [x] Built-in basic rate limit management that respects Discord’s HTTP limits
- [x] Basic file upload support (message attachments)
- [x] Lightweight, fast command manager for auto handling slash commands, their auto complete and subcommands
- [x] Performance focused approach:
    - Structs only contain fields usable without a Gateway session
    - Essentially no caching for very low resource usage & easier hosting
- [x] Built-in helpers for component & modal interaction flow:
  - [Supports buttons, select menus, text inputs, and modals](https://pkg.go.dev/github.com/amatsagu/tempest#Client.AwaitComponent)
  - Includes timeout support and flexible interaction flows
  - Works with both [static](https://pkg.go.dev/github.com/amatsagu/tempest#Client.RegisterComponent) and [dynamic](https://pkg.go.dev/github.com/amatsagu/tempest#Client.AwaitModal) handlers
- [x] Helper structs and methods to manage:
  - [x] Simple messages
  - [x] Embeds
  - [x] Components (buttons, selects, modals, etc.)
  - [x] Bitfields (flags, permissions, etc.)
  - [ ] Message Components v2 *(planned)*
- [x] Exposed Rest client and all API structs which allows to easily extend library capabilities if needed
- [ ] Support for new HTTP event webhooks *(planned)*:
  - [ ] Application Authorized
  - [ ] Application Deauthorized
  - [ ] Entitlement Create
- [ ] Support for Discord Monetization API *(planned)*



### HTTP vs Gateway
**TL;DR**: you probably should be using libraries like [DiscordGo](https://github.com/bwmarrin/discordgo) unless you know why you're here.

There are two ways for bots to receive events from Discord. Most API wrappers such as **DiscordGo** use a WebSocket connection called a "gateway" to receive events, but **Tempest** receives interaction events over HTTPS. Using http hooks lets you scale code more easily & reduce resource usage at cost of greatly reduced number of events you can use. You can easily create bots for roles, minigames, custom messages or admin utils but it'll be very difficult / impossible to create music or moderation bots.



### Getting started
1. Install with: `go get -u github.com/amatsagu/tempest`
2. Check [example](https://github.com/amatsagu/tempest/blob/master/example) with few simple commands.



## Troubleshooting
For help feel free to open an issue on github.
You can also inivite to contact me on [discord](https://discord.com/users/390394829789593601).

## Contributing
All contributions are welcomed.
Few rules before making a pull request:
* Use [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/),
* Add link to document for new structs,
* Check [extra code notes](https://github.com/amatsagu/tempest/blob/master/CODE_NOTES.md) to get familiar with few rules I use when writing writing this lib



[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FAmatsagu%2FTempest.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2FAmatsagu%2FTempest?ref=badge_large)
