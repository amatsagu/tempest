# Contributing to Tempest
Thank you for considering contributing to Tempest!
We accept contributions of all shapes and sizes, from bug fixes and typo corrections to new features and enhancements.

## Prerequisites
- Go 1.26.2 or later (downloadable from [the official Go website](https://go.dev/dl/))
- A suitable version of `golangci-lint` installed on your device (instructions on [their website](https://golangci-lint.run/docs/welcome/install/local/))
- The repository [forked](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/fork-a-repo) and [cloned](https://docs.github.com/en/repositories/creating-and-managing-repositories/cloning-a-repository) locally on your device

## Lefthook Setup
This project uses [Lefthook](https://lefthook.dev/) to run pre-commit hooks for linting and formatting. \
It is included as a tool dependency in `go.mod`, but requires a separate command to initialize the various hooks:

```bash
go tool github.com/evilmartians/lefthook/v2 install
```

**You only need to run this command once after cloning the repository.**

> [!TIP]
> For those using VSCode for development, add the following settings to your workspace `settings.json` to ensure the editor uses the same formatting and linting tools as the pre-commit hooks:
> ```json
> {
>     "go.lintTool": "golangci-lint-v2",
>     "go.formatTool": "custom",
>     "go.alternateTools": {
>         "customFormatter": "golangci-lint-v2"
>     },
>     "go.formatFlags": [
>         "fmt",
>         "--stdin"
>     ],
> }
> ```

## Pull Request Guidelines
The following are guidelines to follow to ensure that your contributions can be merged as quickly as possible. \
PRs will _not_ be merged if they do not follow these conventions.

- Use [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/) for the PR title. This helps us maintain a clean commit history and makes it easier to generate changelogs.
- Add links to [Discord's documentation site](https://docs.discord.com/developers/intro) when creating new structs or modifying existing ones.
- Review the [extra code notes](./CODE_NOTES.md) for guidelines on how to write code for this library.
