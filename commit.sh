#!/usr/bin/env bash
set -euo pipefail

echo "==> Staging and committing changes..."
git add commit.sh component.go component_test.go CODE_NOTES.md
git commit -m "fix: serialize required and divider when false to avoid discord default-true regression"
