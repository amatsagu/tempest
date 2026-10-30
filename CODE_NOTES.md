## Quick list
1. [Fields with nested structs vs pointers](#fields-with-nested-structs-vs-pointers)
2. [Why some Discord API structs feels incomplete?](#why-some-discord-api-structs-feels-incomplete)
3. [JSON Serialization & Parsing Rules (json/v2)](#json-serialization--parsing-rules-jsonv2)
4. [Performance-Tuning Guidelines](#performance-tuning-guidelines)

### Fields with nested structs vs pointers
**TL;DR**: there's no single best solution to whether you should define nested structs by their pointer or not. Most people will agree to use pointer when said struct is large, but otherwise it's mostly down to preference. Sometimes people think that using pointers should be faster as no data is being copied but it's not always true - especially in languages with GC like Golang. Depending how Go's compiler decides, your data may be swapped between stack and heap memories based on how it thinks will be more efficient.

In our case - I've decided to by default pass data by value (copy them) as it remains very cheap and in some edge cases can help. It feels like force using pointers everything is too risky. So, only use pointer if it's **optional field** or when copied struct is large (size >= 600). We're talking here about single instances, please don't use pointer to arrays/slices or maps as they are already giving you just a pointer. When in doubt, please check how other code is handled, for example [Modal Interaction](https://github.com/amatsagu/tempest/blob/de02d0ad11bde79058019ac818ffdfda6afad0e2/interaction.go#L62) struct.

**Wrapper Structs**: Specialized interaction wrappers like `CommandInteraction`, `ComponentInteraction`, `ModalInteraction` embeds `Interaction` **by value** instead pointer to reduce heap allocations on every interaction/event.

### Why some Discord API structs feels incomplete?
I've decided to simply cut, not include dead fields - data that I'm sure will never be provided/used by Application. Sadly, Discord API at times can be confusing - for example it's hard to sometimes see differences between Application, Bot or Activity. Not including those fields makes your final app slightly faster as json parser will have less bytes to handle.

### JSON Serialization & Parsing Rules (json/v2)
Tempest uses modern `encoding/json/v2` and `encoding/json/jsontext`. Follow these points in order to maintain optimal performance, smaller payload sizes, and consistency with the Discord API:

- **Prefer `omitzero` for optional fields**:
  `omitzero` is standard in `json/v2`. It omits fields whose value equals their Go zero value (`""`, `0`, `false`, `nil`, or an `IsZero() bool` returning true).
- **Boolean fields**:
  Use `omitzero` on boolean fields that default to `false` in Discord (e.g. `Disabled`, `NSFW`, `Required`, `AutoComplete`, `Spoiler`, `Divider`).
  - When `false` (default): omitted completely from outgoing JSON, saving bytes and parsing time.
  - When `true`: serialized as `"field": true`.
- **Snowflake fields**:
  Use `omitzero` on optional snowflakes (e.g. `GuildID`, `ApplicationID`). `Snowflake(0)` is omitted automatically without requiring pointer wrapping.
- **Slices and Maps (Discord Resource Clearing Semantics)**:
  `omitzero` naturally handles Discord's resource clearing semantics:
  - `nil` slice (`var embeds []Embed = nil`): zero value -> **omitted** from JSON (leaves existing embeds unchanged).
  - Empty non-nil slice (`embeds := []Embed{}`): non-zero value -> **serialized as `"embeds": []"`** (explicitly signals Discord to clear/remove element(s)).

### Performance-Tuning Guidelines
1. **Avoid Recursive `UnmarshalJSON` (Use Zero-Cost Accessors)**:
   Do not implement custom `UnmarshalJSON` with alias structs on high-frequency structs (like `User`) merely to set defaults or fallback values (e.g. `GlobalName -> Username`). This forces `json/v2` to buffer and decode twice. Instead, provide zero-cost accessor methods like `user.DisplayName()`.
2. **Stream REST Responses with `json.UnmarshalRead`**:
   Stream HTTP responses directly into target structs via `json.UnmarshalRead(res.Body, into)`. Do not read the entire body into memory with `io.ReadAll`. When `into == nil` (e.g. 204 No Content, DELETE, callbacks), discard the body with `io.Copy(io.Discard, res.Body)` without allocating buffers.
3. **Zero-Allocation Token Probing (`jsontext.Decoder`)**:
   For polymorphic payloads (message components, interaction callbacks), use pooled token scanners (`peekInteractionType`, `peekComponentType`) to read discriminator fields like `"type"` in a single forward pass without decoding intermediate map structures.
4. **Implement `encoding.TextAppender` for Primitives**:
   Implement `AppendText(b []byte) ([]byte, error)` and `UnmarshalText(b []byte) error` on custom primitive types like `Snowflake`. `json/v2` calls `AppendText` directly into its buffer, eliminating intermediate allocations.

<br>

### Interaction
- Represents all types of interactions between user/member and bot.
- Includes **slash commands**, **button clicks**, **select menus**, **modal submissions**, and other interactive components.
- Contains data such as the **interaction ID**, **user who triggered it**, **associated guild/channel**, and **interaction-specific metadata** (e.g. selected options in a menu).

### Mentionable
- Covers all elements that can be @mentioned in Discord.
- Includes **users/members**, **servers**, **channels**, and **roles**.
- There's additionally, there are **timestamp mentions** and **slash command mentions**, but these are client-side chat decorations rather than real API models.

### Message
- Describes everything that can be included in or attached to a message.
- Includes **text fields**, **author details**, **links**, **embeds**, **attachments**, **emojis**, **stickers**, and **files**.
- Also handles message components such as **buttons**, **select menus**, and **modals**.

### Response
- Represents different types of responses that a bot can send to Discord
- Includes **immediate responses** (via interaction callbacks), **delayed responses** (via follow-ups), and **message updates** (modifying existing messages).
- Supports rich content such as **embeds**, **attachments**, and **interactive components**.

### Warning
We'll try keeping all structs up to date but nobody can promise that. Be especially careful with `./permission.go` as discord commonly makes minor changes to permission names/positions.
