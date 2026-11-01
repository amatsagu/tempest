#!/usr/bin/env bash
set -euo pipefail

echo "==> Staging and committing changes..."
git add commit.sh gateway-client.go http-client.go interaction-helper.go rate-limit.go rest.go
git commit -m "perf: optimize interaction dispatch, REST request buffering, and rate limit routing"
