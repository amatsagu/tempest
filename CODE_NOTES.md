## Quick list
1. [Fields with nested structs vs pointers](#fields-with-nested-structs-vs-pointers)
2. [Why some Discord API structs feels incomplete?](#why-some-discord-api-structs-feels-incomplete)

### Fields with nested structs vs pointers
**TL;DR**: there's no single best solution to whether you should define nested structs by their pointer or not. Most people will agree to use pointer when said struct is large, but otherwise it's mostly down to preference. Sometimes people think that using pointers should be faster as no data is being copied but it's not always true - especially in languages with GC like Golang. Depending how Go's compiler decides, your data may be swapped between stack and heap memories based on how it thinks will be more efficient.

In our case - I've decided to by default pass data by value (copy them) as it remains very cheap and in some edge cases can help. It feels like force using pointers everything is too risky. So, only use pointer if it's **optional field** or when copied struct is large (size >= 600). We're talking here about single instances, please don't use pointer to arrays/slices or maps as they are already giving you just a pointer. When in doubt, please check how other code is handled, for example [Modal Interaction](https://github.com/amatsagu/tempest/blob/de02d0ad11bde79058019ac818ffdfda6afad0e2/interaction.go#L62) struct.

### Why some Discord API structs feels incomplete?
I've decided to simply cut, not include dead fields - data that I'm sure will never be provided/used by Application. Sadly, Discord API at times can be confusing - for example it's hard to sometimes see differences between Application, Bot or Activity. Not including those fields makes your final app slightly faster as json parser will have less bytes to handle.


### JSON Parsing of Optional Fields
To ensure consistent behavior with other languages and align with Discord API expectations, Tempest follows a specific convention for optional fields in Go:
- `omitempty` for optional fields like strings, numbers, etc.
- `omitzero` for **slices or maps** where an **explicit empty value** (e.g., `[]`) must be included in the payload to signal the removal or absence of a resource.

> 💡 This is required in cases like `Message#embeds`, where sending `"embeds": []` is necessary to explicitly clear embeds from a message. Using only `omitempty` would omit the field entirely, which would not trigger removal in the Discord API.

> ⚠️ Please at least for now avoid using `omitempty` on bool type struct fields. Go's default, zero value logic may overwrite expected outcome (in some cases). I'll try finding better solution later on but that works for now.

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