[![Go Reference](https://pkg.go.dev/badge/github.com/disgoorg/disgo.svg)](https://pkg.go.dev/github.com/Amatsagu/Tempest)
[![Go Report](https://goreportcard.com/badge/github.com/disgoorg/disgo)](https://goreportcard.com/report/github.com/Amatsagu/Tempest)
[![License](https://img.shields.io/github/license/Amatsagu/tempest)](https://github.com/Amatsagu/Tempest/blob/master/LICENSE)
[![Maintenance Status](https://img.shields.io/maintenance/yes/2023)](https://github.com/Amatsagu/Tempest)

# Tempest
Tempest is a [Discord](https://discord.com) API wrapper for Applications (interactions), written in [Golang](https://golang.org/). It aims to be fast, cache free and higher level than other Discord API wrappers made for Discord Applications.

It was created as a better alternative to [discord-interactions-go](https://github.com/bsdlp/discord-interactions-go) which is "low level" and outdated.

## Summary
1. [Stability](#stability)
2. [Supported discord features](#supported-discord-features)
3. [Missing or partially supported discord features](#missing-or-partially-supported-discord-features)
4. [Special features](#special-features)
5. [Getting Started](#getting-started)
6. [Troubleshooting](#troubleshooting)
7. [Contributing](#contributing)

### Stability
The public API of Tempest is fully stable at point of releasing new version. Smaller breaking changes can happen if Discord requires them. Each function/method that may potentially fail (usually from dev bad code) returns optional error to handle the same way as Go std lib.

In race for efficiency, interaction & component structures are bare metal without wall of sanity checks so you need to to understand how those works to avoid silly issues. Know with what you're working with.

### Supported discord features
* Full Rest interaction & component API coverage
* [HTTP interactions](https://discord.com/developers/docs/interactions/slash-commands#receiving-an-interaction)
* [Application commands](https://discord.com/developers/docs/interactions/application-commands)
* [User/Message commands](https://discord.com/developers/docs/interactions/application-commands#user-commands)
* [Message components](https://discord.com/developers/docs/interactions/message-components)
* [Built-in rate limits](https://discord.com/developers/docs/topics/rate-limits)

### Missing or partially supported discord features
* [Modals (P)](https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-response-object-modal) - Technically you can use them but it'll require handling them manually using client's [interaction handler](https://pkg.go.dev/github.com/Amatsagu/Tempest#ClientOptions).
* [Attachments (P)](https://discord.com/developers/docs/resources/channel#attachment-object) - Tempest application can accept attachments but there's no support for sending own files.
* [Localization (M)](https://discord.com/developers/docs/interactions/application-commands#localization) - Multi-language support is still highly unstable and barely ever used by bots so I'm going to ignore it for now.

### Special features
* [Easy to use & efficient handler for (/) commands & auto complete interactions](https://pkg.go.dev/github.com/Amatsagu/Tempest#Client.RegisterCommand)
    - Deep control with [pre command execution client's handler](https://pkg.go.dev/github.com/Amatsagu/Tempest#ClientOptions).
* [Exposed REST](https://pkg.go.dev/github.com/Amatsagu/Tempest#Client.Rest)
* [Easy component handler](https://pkg.go.dev/github.com/Amatsagu/Tempest#Client.AwaitComponent)
    - Works with buttons, select menus & text inputs.
    - Supports timeouts & gives a lot of freedom.
* [Simple way to sync (/) commands with API](https://pkg.go.dev/github.com/Amatsagu/Tempest#Client.SyncCommands)
* Auto panic recovery inherited from `std/http`
* Request failure auto recovery (3 attempts)
    - On failed attempts *(probably due to internet connection)*, it'll try again up to 3 times before panicking.
* [Cooldown system for commands (optional)](https://pkg.go.dev/github.com/Amatsagu/Tempest#ClientCooldownOptions)

### Getting started
1. Install with: `go get -u github.com/Amatsagu/Tempest`
2. Check [example](https://github.com/Amatsagu/Tempest/blob/master/example/main.go) with few simple commands.



## Troubleshooting
For help feel free to open an issue on github.

## Contributing
Contributions are welcomed but for bigger changes I would like first reaching out via Discord (invite `Amatsagu#0001`) or create an issue to discuss your problems, intentions and ideas.
