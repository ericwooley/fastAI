#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_REVISION="${1:-}"
HEAD_REVISION="${2:-HEAD}"
MESSAGE_FILE="$(mktemp "${TMPDIR:-/tmp}/fastAI-commit-message.XXXXXX")"

cleanup() {
	rm -f "$MESSAGE_FILE"
}

trap cleanup EXIT

if ! git cat-file -e "${HEAD_REVISION}^{commit}" 2>/dev/null; then
	printf 'Head revision is not a commit: %s\n' "$HEAD_REVISION" >&2
	exit 2
fi

if [[ -z "$BASE_REVISION" || "$BASE_REVISION" =~ ^0+$ ]]; then
	RANGE="$HEAD_REVISION"
else
	if ! git cat-file -e "${BASE_REVISION}^{commit}" 2>/dev/null; then
		printf 'Base revision is not a commit: %s\n' "$BASE_REVISION" >&2
		exit 2
	fi
	RANGE="${BASE_REVISION}..${HEAD_REVISION}"
fi

FAILED=0
while IFS= read -r commit; do
	[[ -n "$commit" ]] || continue
	git show -s --format=%B "$commit" >"$MESSAGE_FILE"
	if ! bash "$ROOT_DIR/scripts/validate-commit-message.sh" "$MESSAGE_FILE"; then
		printf 'Invalid commit: %s (%s)\n' "$commit" "$(git show -s --format=%s "$commit")" >&2
		FAILED=1
	fi
done < <(git log --no-merges --format=%H "$RANGE")

if [[ "$FAILED" -ne 0 ]]; then
	exit 1
fi

printf 'All non-merge commits in %s use Conventional Commits.\n' "$RANGE"
