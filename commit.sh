#!/usr/bin/env bash
set -euo pipefail

echo "==> Staging and committing changes..."
git add commit.sh interaction-helper.go example/feature/calc/calc_cmd.go example/feature/components/ticket_cmd.go example/feature/image/rotate_hook.go
git commit -m "feat: support files in EditReply and add in-place edit helpers for components and modals"
