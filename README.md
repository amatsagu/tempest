[![Go Reference](https://pkg.go.dev/badge/github.com/disgoorg/disgo.svg)](https://pkg.go.dev/github.com/Amatsagu/Tempest)
[![Go Report](https://goreportcard.com/badge/github.com/disgoorg/disgo)](https://goreportcard.com/report/github.com/Amatsagu/Tempest)
[![License](https://img.shields.io/github/license/Amatsagu/tempest)](https://github.com/Amatsagu/Tempest/blob/master/LICENSE)
[![Maintenance Status](https://img.shields.io/maintenance/yes/2023)](https://github.com/Amatsagu/Tempest)
[![CodeQL](https://github.com/Amatsagu/Tempest/actions/workflows/github-code-scanning/codeql/badge.svg?branch=master)](https://github.com/Amatsagu/Tempest/actions/workflows/github-code-scanning/codeql)

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

There are two ways for bots to recieve events from Discord. Most API wrappers such as **DiscordGo** use a WebSocket connection called a "gateway" to receive events, but **Tempest** receives interaction events over HTTP. Using http connection lets you easily split your bot into microservices and use far less resources as opposed to gateway but will receive less events. As such, there are some major points to keep in mind before deciding against using gateway.

### Supported discord features
**Tempest** since `v1.1.0` supports all discord features (allowed over HTTP) except file transfer. You can accept files but there's currently no support for sending own files. Other elements like command auto complete, components or modals have full support.

### Special features
* [Easy to use & efficient handler for (/) commands & auto complete interactions](https://pkg.go.dev/github.com/Amatsagu/Tempest#Client.RegisterCommand)
    - Deep control with [command middleware(s)](https://pkg.go.dev/github.com/Amatsagu/Tempest#ClientOptions)
* [Exposed REST](https://pkg.go.dev/github.com/Amatsagu/Tempest#Client.Rest)
* [Easy component & modal handler](https://pkg.go.dev/github.com/Amatsagu/Tempest#Client.AwaitComponent)
    - Works with buttons, select menus, text inputs and modals,
    - Supports timeouts & gives a lot of freedom,
    - Works for both [static](https://pkg.go.dev/github.com/Amatsagu/Tempest#Client.RegisterComponent) and [dynamic](https://pkg.go.dev/github.com/Amatsagu/Tempest#Client.AwaitModal) ways
* [Simple way to sync (/) commands with API](https://pkg.go.dev/github.com/Amatsagu/Tempest#Client.SyncCommands)
* Auto panic recovery inherited from `std/http`
* Request failure auto recovery (3 attempts)
    - On failed attempts *(probably due to internet connection)*, it'll try again up to 3 times before returning error
* Cache is optional
    - Applications/Bots work without any state caching if they only prefer to (avoid dynamic handlers to do it).

### Getting started
1. Install with: `go get -u github.com/Amatsagu/Tempest`
2. Check [example](https://github.com/Amatsagu/Tempest/blob/master/example) with few simple commands.



## Troubleshooting
For help feel free to open an issue on github.

## Contributing
Contributions are welcomed but for bigger changes I would like first reaching out via Discord (invite `Amatsagu#0001`, id: `390394829789593601`) or create an issue to discuss your problems, intentions and ideas.
Few rules before making a pull request:
* Use [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/) 
* Avoid using interfaces, generics or any/interface{} keywords
    - As we focus on max performance, those elements should be skipped unless required to go forward
* Add link to document for new structs
    - Since `v1.1.0`, all structs have links to corresponding discord docs
