#!/usr/bin/env bash
set -euo pipefail

echo "==> Staging and committing changes..."
git add component.go response.go message.go guild.go mentionable.go interaction.go event.go CODE_NOTES.md
git commit -m "fix(json): standardize omitzero across all discord models for numeric, flag, and enum fields"
