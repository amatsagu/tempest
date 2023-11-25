[![Go Reference](https://pkg.go.dev/badge/github.com/disgoorg/disgo.svg)](https://pkg.go.dev/github.com/Amatsagu/Tempest)
[![Go Report](https://goreportcard.com/badge/github.com/disgoorg/disgo)](https://goreportcard.com/report/github.com/Amatsagu/Tempest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/disgoorg/disgo)](https://golang.org/doc/devel/release.html)
[![License](https://img.shields.io/github/license/Amatsagu/tempest)](https://github.com/Amatsagu/Tempest/blob/development/LICENSE)
[![Library Version](https://img.shields.io/github/v/tag/disgoorg/disgo?label=release)](https://github.com/Amatsagu/Tempest/releases/latest)
[![Maintenance Status](https://img.shields.io/maintenance/yes/2024)](https://github.com/Amatsagu/Tempest)
[![CodeQL](https://github.com/Amatsagu/Tempest/actions/workflows/github-code-scanning/codeql/badge.svg?branch=development)](https://github.com/Amatsagu/Tempest/actions/workflows/github-code-scanning/codeql)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-%23FE5196?logo=conventionalcommits&logoColor=white)](https://conventionalcommits.org)

<img align="left" src="/.github/tempest-logo.png" width=192 alt="Tempest library logo">

# Tempest
Tempest is a [Discord](https://discord.com) API wrapper for Applications, written in [Golang](https://golang.org/). It aims to be fast, use minimal caching and be easier to use than other Discord API wrappers using http.

It was created as a better alternative to [discord-interactions-go](https://github.com/bsdlp/discord-interactions-go) which is "low level" and outdated.

## Summary
1. [HTTP vs Gateway](#http-vs-gateway)
2. [Supported discord features](#supported-discord-features)
3. [Special features](#special-features)
4. [Getting Started](#getting-started)
5. [Troubleshooting](#troubleshooting)
6. [Contributing](#contributing)

### HTTP vs Gateway
**TL;DR**: you probably should be using libraries like [DiscordGo](https://github.com/bwmarrin/discordgo) unless you know why you're here.

There are two ways for bots to recieve events from Discord. Most API wrappers such as **DiscordGo** use a WebSocket connection called a "gateway" to receive events, but **Tempest** receives interaction events over HTTP. Using http hooks lets you split & scale code more easily. 