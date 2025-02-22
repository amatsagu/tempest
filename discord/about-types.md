# Discord Module  
This module contains all required by Ashara lib struct data types used by Discord API. Here are the main groups:

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
