[![Go Reference](https://pkg.go.dev/badge/github.com/disgoorg/disgo.svg)](https://pkg.go.dev/github.com/Amatsagu/Tempest)
[![Go Report](https://goreportcard.com/badge/github.com/disgoorg/disgo)](https://goreportcard.com/report/github.com/Amatsagu/Tempest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/disgoorg/disgo)](https://golang.org/doc/devel/release.html)
[![License](https://img.shields.io/github/license/Amatsagu/tempest)](https://github.com/Amatsagu/Tempest/blob/development/LICENSE)
[![Maintenance Status](https://img.shields.io/maintenance/yes/2024)](https://github.com/Amatsagu/Tempest)
[![CodeQL](https://github.com/Amatsagu/Tempest/actions/workflows/github-code-scanning/codeql/badge.svg?branch=development)](https://github.com/Amatsagu/Tempest/actions/workflows/github-code-scanning/codeql)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-%23FE5196?logo=conventionalcommits&logoColor=white)](https://conventionalcommits.org)

<img align="left" src="/.github/tempest-logo.png" width=192 alt="Tempest library logo">

# Tempest
Tempest is a [Discord](https://discord.com) API wrapper for Applications, written in [Golang](https://golang.org/). It aims to be fast, use minimal caching and be easier to use than other Discord API wrappers using http.

It was created as a better alternative to [discord-interactions-go](https://github.com/bsdlp/discord-interactions-go) which is "low level" and outdated.

## Summary
1. [HTTP vs Gateway](#http-vs-gateway)
2. [Getting Started](#getting-started)
3. [Troubleshooting](#troubleshooting)
4. [Contributing](#contributing)

### HTTP vs Gateway
**TL;DR**: you probably should be using libraries like [DiscordGo](https://github.com/bwmarrin/discordgo) unless you know why you're here.

There are two ways for bots to receive events from Discord. Most API wrappers such as **DiscordGo** use a WebSocket connection called a "gateway" to receive events, but **Tempest** receives interaction events over HTTP. Using http hooks lets you scale code more easily & reduce resource usage at cost of greatly reduced number of events you can use. You can easily create bots for roles, minigames, custom messages or admin utils but it'll be very difficult / impossible to create music or moderation bots.

### Getting started
1. Install with: `go get -u github.com/Amatsagu/Tempest`
2. Check [example](https://github.com/Amatsagu/Tempest/blob/master/example) with few simple commands.



## Troubleshooting
For help feel free to open an issue on github.

## Contributing
Contributions are welcomed but for bigger changes I would like first reaching out via Discord (invite `amatsagu`, id: `390394829789593601`) or create an issue to discuss your problems, intentions and ideas.
Few rules before making a pull request:
* Use [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/) 
* Avoid using interfaces, generics or any/interface{} keywords
    - As we focus on max performance, those elements should be skipped unless required to go forward
* Add link to document for new structs
    - Since `v1.1.0`, all structs have links to corresponding discord docs



[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FAmatsagu%2FTempest.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2FAmatsagu%2FTempest?ref=badge_large)