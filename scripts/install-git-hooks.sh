#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR"

if ! git rev-parse --git-dir >/dev/null 2>&1; then
	printf 'Not a Git repository: %s\n' "$ROOT_DIR" >&2
	exit 1
fi

git config --local core.hooksPath .githooks
printf 'Installed repository Git hooks from .githooks.\n'
