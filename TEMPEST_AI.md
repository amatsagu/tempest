# Tempest AI Guide

Tempest: High-performance, minimal Discord library for Go. Supports both HTTPS (webhooks) and Gateway (WebSockets) architectures. Optimized for low-latency interactions and Message Components v2.

## Strategy
Refer to specific task chapters:
- [Core](./AI-docs/Core.md): HTTP/Gateway clients, config, logging.
- [Commands](./AI-docs/Commands.md): Registration, subcommands, autocomplete.
- [Interactions](./AI-docs/Interactions.md): Responders, defer, follow-up.
- [Components](./AI-docs/Components.md): Layout v2 (Sections, Containers) & v1.
- [Modals](./AI-docs/Modals.md): Label system, input handling.
- [REST](./AI-docs/REST.md): API methods, Fetch, Entitlements.

## Standards
- JSON Slices: Use `omitzero` for optional slices (ensures `[]` is sent).
- Booleans: No `omitempty` on `bool` (prevents zero-value overwrite).
- Pointers: Pass by value unless optional or size >= 600 bytes.
- URLs: Use `https://docs.discord.com/developers/`.

## Quick Start (HTTP)
```go
client := tempest.NewHTTPClient(tempest.HTTPClientOptions{
    PublicKey: "KEY",
    BaseClientOptions: tempest.BaseClientOptions{Token: "Bot TOKEN"},
})
```

## Quick Start (Gateway)
```go
client := tempest.NewGatewayClient(tempest.GatewayClientOptions{
    BaseClientOptions: tempest.BaseClientOptions{Token: "Bot TOKEN"},
})
client.Gateway.Start(context.Background(), 0, 0, nil)
```
