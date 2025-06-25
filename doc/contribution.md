## Contributing
All contributions are welcome!
Before submitting a pull request, please follow these rules:
* Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/),
* Follow the code guidelines below.

## Code Guidelines
1. When working on Discord structs (creating new ones or updating existing ones), make sure they include a correct link to the relevant section of the Discord API documentation in a comment. While the Discord API docs are improving, they can still be messy—mistakes happen, and it's easy to miss a field or provide the wrong type.
2. Design with usability in mind. This library aims to stay close to the original Discord API, but it's acceptable to rename specific fields if it improves clarity or to omit fields that serve no purpose for bots/apps focused solely on interactions and components. Discord often sends excessive data that this library will never use.
3. Try to maintain backward compatibility. This doesn’t mean you should never change existing code, but do so only when necessary or when it clearly improves the library.
4. Organize code into modules:
    - `discord`: for all enums, structs, and constants defined by the Discord API,
    - `gateway`: for everything related to the Discord Gateway connection,
    - `core`: for unique logic provided by the library (e.g., REST client or component listener),
    - `helper`: for transformers or helper structs that improve usability (e.g., embed pagination builder),
    - `util`: for anything else that doesn’t fit the above categories (if needed).
5. Avoid plural naming unless it is explicitly used in the API documentation.
6. Follow (my) established conventions for when to use pointers versus structs. See the section below for more details.

### Fields: Nested Structs vs. Pointers
**TL;DR**: There’s no universal rule for when to use a pointer vs. a value for nested structs. In general, use a pointer when the struct is large. While some think pointers are always faster because they avoid copying data, this isn’t necessarily true—especially in garbage-collected languages like Go. The compiler may choose to place data on the stack or heap depending on what it deems more efficient.

In this library, the default is to pass data by value (i.e., copy it). This is usually cheap and, in some edge cases, even beneficial. Using pointers everywhere can lead to subtle issues, so they are reserved for:

* **Optional fields**, or
* **Large structs** (size ≥ 512 bytes).

> [!NOTE]
> This rule applies to single instances only. Don’t use pointers for slices, arrays, or maps—they are already reference types.

When in doubt, refer to existing implementations. For example, see the [`ModalInteraction`](https://github.com/amatsagu/tempest/blob/de02d0ad11bde79058019ac818ffdfda6afad0e2/interaction.go#L62) struct.
