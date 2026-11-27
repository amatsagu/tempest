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
- **Numbers and Flags**:
  Always use `omitzero` rather than `omitempty` on primitive numbers (`uint*`, `int*`, `float*`) and bitset flags. In `json/v2`, `omitempty` does not omit `0`, which causes Discord API rejection (e.g. `max_values: 0`, `duration: 0`).
- **Boolean fields**:
  - Use plain `bool` with `omitzero` on boolean fields whose Discord default is `false` (e.g. `Disabled`, `NSFW`, `CommandOption.Required`, `AutoComplete`, `Spoiler`, `RepliedUser`, `Inline`, `Multiselect`).
    - When `false` (default): omitted completely from outgoing JSON, saving bytes and parsing time.
    - When `true`: serialized as `"field": true`.
  - Use `*bool` with `omitzero` on optional boolean fields whose Discord default is `true` (e.g. `Required` on `StringSelectComponent`, `SelectComponent`, `TextInputComponent`, `FileUploadComponent`, `RadioGroupComponent`, `CheckboxGroupComponent`; `Divider` on `SeparatorComponent`; `FailIfNotExists` on `MessageReference`).
    - When `nil` (default): omitted completely from JSON so Discord applies its `true` default cleanly (and shared select components used in messages never emit invalid `"required"` keys).
    - When `new(false)` or `new(true)` (Go 1.27): explicitly serialized as `"field": false` or `"field": true`.
- **Nullable zero vs unset (`*float64`, `*uint8`)**:
  Use pointer primitives with `omitzero` on numeric fields where omitting the key means a non-zero Discord default or "unbounded", while `0` is a distinct valid value:
  - `CommandOption.MinValue` / `MaxValue` (`*float64`): `nil` omits, `new(0.0)` emits `0`, `new(1.0)` emits `1`, etc.
  - `MinValues` (`*uint8` on `StringSelectComponent`, `SelectComponent`, `FileUploadComponent`, `CheckboxGroupComponent`): Discord defaults omitted `min_values` to `1`, while `0` is needed for optional selection when `Required` is `new(false)`.
- **Pointer/string tags (`omitzero` over `omitempty`)**:
  Prefer `,omitzero` over `,omitempty` on optional `string` and pointer (`*T`) fields so `json/v2` checks Go zero/nil directly without encoding/probing the underlying value. Keep `,omitempty` only on `any` (`CommandInteractionOption.Value`).
- **Snowflake fields**:
  Use `omitzero` on optional snowflakes (e.g. `GuildID`, `ApplicationID`). `Snowflake(0)` (and single-field wrappers like `GuildID{}`) is omitted natively by `json/v2` without pointer wrapping or implementing `IsZero() bool`. (Avoid defining `IsZero() bool` on `Snowflake`, as `json/v2` calls it via `reflect` interface boxing on every marshaled `omitzero` field, adding a heap allocation per field.)
- **Slices and Maps (Discord Resource Clearing Semantics)**:
  `omitzero` naturally handles Discord's resource clearing semantics:
  - `nil` slice (`var embeds []Embed = nil`): zero value -> **omitted** from JSON (leaves existing embeds unchanged).
  - Empty non-nil slice (`embeds := []Embed{}`): non-zero value -> **serialized as `"embeds": []"`** (explicitly signals Discord to clear/remove element(s)).

### Performance-Tuning Guidelines
1. **Avoid Recursive `UnmarshalJSON` (Use Zero-Cost Accessors)**:
   Do not implement custom `UnmarshalJSON` with alias structs on high-frequency structs (like `User`) merely to set defaults or fallback values (e.g. `GlobalName -> Username`). This forces `json/v2` to buffer and decode twice. Instead, provide zero-cost accessor methods like `user.DisplayName()`.
2. **Stream REST Responses with `json.UnmarshalRead`**:
   Stream HTTP responses directly into target structs via `json.UnmarshalRead(res.Body, into)`. Do not read the entire body into memory with `io.ReadAll`. When `into == nil` (e.g. 204 No Content, DELETE, callbacks), discard the body with `io.Copy(io.Discard, res.Body)` without allocating buffers.
3. **Zero-Allocation Token Probing (`peekComponentType`, `peekInteractionType`)**:
   For polymorphic payloads (message components, interaction callbacks), use direct top-level byte scanning (`peekComponentType`, safely rejecting non-numeric `"type"` tokens without panicking, and falling back to pooled `jsontext.Decoder` only on escaped keys `\"`).
4. **Zero-Copy Streaming Union Decoding (`UnmarshalJSONFrom`)**:
   For container structs holding union interfaces (`Message`, `ActionRowComponent`, `ContainerComponent`, `SectionComponent`, `LabelComponent`, `ModalInteractionData`), implement `UnmarshalJSONFrom(*jsontext.Decoder)` and decode slice items directly from `dec.ReadValue()` (`componentSlice[T]`) instead of materializing intermediate `[]jsontext.Value` copies or heap-escaping `type alias` wrappers (`Message`).
   - Compare keys via `dec.ReadValue()` byte slices (`switch string(inner)`) rather than `tok.String()` so key comparisons remain 0-alloc.
   - Keep **leaf component structs** (`ButtonComponent`, `StringSelectComponent`, etc.) on `json/v2`'s native struct decoder (`json.Unmarshal(raw, &cmp)`) instead of hand-decoding leaf string fields or wrapping `raw` in `bytes.NewReader`, so `json/v2` retains its direct `[]byte` fast path and internal 256-entry `stringCache`.
   - Decode homogenous component slices (`SectionComponent.Components []TextDisplayComponent`) directly without boxing into `AnyComponent`.
5. **Implement `TextAppender` & `MarshalJSONTo` / `UnmarshalJSONFrom` for Primitives**:
   - Implement `AppendText` and `UnmarshalJSONFrom(*jsontext.Decoder)` on `Snowflake` and `UnmarshalJSONFrom` on `Nonce` so `json/v2` reads/writes directly against the decoder/encoder buffer.
   - Implement `MarshalJSONTo(*jsontext.Encoder)` on `uint8`-backed enums used in slices (e.g. `ChannelType`) to write JSON integers directly without temporary `[]byte` / string allocations or `[]uint8` base64 handling.
6. **Single-Allocation CDN URL Builders & Struct Alignment**:
   - Build Discord CDN URLs with stack `[20]byte` digit formatting (`strconv.AppendUint`) and a pre-grown `strings.Builder` (`Grow`) rather than chaining `ID.String()` and `+` concatenations.
   - Keep pointer-containing fields grouped first so `fieldalignment ./...` stays at `0` issues with minimal GC pointer scan prefixes.

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
We'll try keeping all structs up to date but nobody can promise that. Be especially careful with `./discord/permission.go` as discord commonly makes minor changes to permission names/positions.
