#!/usr/bin/env bash
set -euo pipefail

echo "==> Staging and committing changes..."
git add commit.sh rest.go
git commit -m "feat(rest): add randomized jitter to retry backoff to prevent thundering herds"
