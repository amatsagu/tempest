<div align="center">
    <img align="center" src="/.github/tempest-banner.png" height="165" alt="Tempest library banner">
</div>

<p align="center">
  <i align="center">Create lightning fast Discord Applications</i>
</p>

<h4 align="center">
    <a href="#http-vs-gateway">HTTP vs Gateway</a> - <a href="#project-goals">Project goals</a> - <a href="#getting-started">Getting started</a> - <a href="#troubleshooting">Troubleshooting</a> - <a href="#contributing">Contributing</a>
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

### HTTP vs Gateway
**TL;DR**: you probably should be using libraries like [DiscordGo](https://github.com/bwmarrin/discordgo) unless you know why you're here.

There are two ways for bots to receive events from Discord. Most API wrappers such as **DiscordGo** use a WebSocket connection called a "gateway" to receive events, but **Tempest** receives interaction events over HTTP. Using http hooks lets you scale code more easily & reduce resource usage at cost of greatly reduced number of events you can use. You can easily create bots for roles, minigames, custom messages or admin utils but it'll be very difficult / impossible to create music or moderation bots.

### Project goals
*  [Efficient handler for (/) commands & auto complete interactions](https://pkg.go.dev/github.com/amatsagu/tempest#Client.RegisterCommand)
    - Deep control with [command middleware(s)](https://pkg.go.dev/github.com/amatsagu/tempest#ClientOptions)
* [Exposed REST](https://pkg.go.dev/github.com/amatsagu/tempest#Client.Rest)
[x] [Mixed component & modal handling](https://pkg.go.dev/github.com/amatsagu/tempest#Client.AwaitComponent)
    - Works with buttons, select menus, text inputs and modals,
    - Supports timeouts & gives a lot of freedom,
    - Works for both [static](https://pkg.go.dev/github.com/amatsagu/tempest#Client.RegisterComponent) and [dynamic](https://pkg.go.dev/github.com/amatsagu/tempest#Client.AwaitModal) ways
* [Simple way to sync (/) commands with API](https://pkg.go.dev/github.com/amatsagu/tempest#Client.SyncCommands)
* Request failure auto recovery (3 attempts by default)
    - On failed attempts *(probably due to internet connection)*, it'll try again set number of times before returning error
* Minimal Discord data caching by default
* Attempts to use debloated structs (only defines fields you may actually get in Application connection mode)
* Focus on performance over being beginner-friendly

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
