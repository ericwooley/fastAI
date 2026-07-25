#!/usr/bin/env bash

set -euo pipefail

MESSAGE_FILE="${1:-}"

if [[ -z "$MESSAGE_FILE" || ! -f "$MESSAGE_FILE" ]]; then
	printf 'Usage: %s <commit-message-file>\n' "$(basename "$0")" >&2
	exit 2
fi

HEADER="$(sed -n '1p' "$MESSAGE_FILE")"
CONVENTIONAL_PATTERN='^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([^()[:space:]]+\))?!?: .+'

# Git creates these subjects for non-fast-forward local merges. Merge commits
# do not drive release versions; their individual commits still do.
if [[ "$HEADER" =~ ^Merge[[:space:]] ]]; then
	exit 0
fi

if grep -Eq "$CONVENTIONAL_PATTERN" <<<"$HEADER"; then
	exit 0
fi

cat >&2 <<'EOF'
Commit message must use the Conventional Commits format:

  <type>[optional scope][!]: <description>

Examples:
  fix: preserve session history
  feat(cli): add release archives
  feat(api)!: replace the provider contract

Allowed types: build, chore, ci, docs, feat, fix, perf, refactor, revert, style, test.
Use feat for a minor release, fix or perf for a patch release, and ! or a
BREAKING CHANGE footer for a major release.
EOF

printf '\nReceived: %s\n' "${HEADER:-<empty>}" >&2
exit 1
