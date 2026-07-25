#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VALIDATOR="$ROOT_DIR/scripts/validate-commit-message.sh"
TEST_DIR="$(mktemp -d "${TMPDIR:-/tmp}/fastAI-commit-message-test.XXXXXX")"

cleanup() {
	rm -rf "$TEST_DIR"
}

trap cleanup EXIT

fail() {
	printf 'FAIL: %s\n' "$1" >&2
	exit 1
}

assert_valid() {
	local label="$1"
	local message="$2"
	local message_file="$TEST_DIR/message"

	printf '%s\n' "$message" >"$message_file"
	if ! bash "$VALIDATOR" "$message_file" >"$TEST_DIR/output" 2>&1; then
		cat "$TEST_DIR/output" >&2
		fail "$label should be valid"
	fi
}

assert_invalid() {
	local label="$1"
	local message="$2"
	local message_file="$TEST_DIR/message"

	printf '%s\n' "$message" >"$message_file"
	if bash "$VALIDATOR" "$message_file" >"$TEST_DIR/output" 2>&1; then
		fail "$label should be invalid"
	fi

	grep -Fq 'Conventional Commit' "$TEST_DIR/output" || fail "$label should explain the expected format"
}

assert_valid "feature" "feat: add release archives"
assert_valid "scoped fix" "fix(auth): preserve the selected provider"
assert_valid "breaking feature" "feat(cli)!: remove a legacy flag"
assert_valid "non-releasing documentation" "docs(readme): explain local setup"
assert_valid "breaking footer" $'refactor(storage): simplify sessions\n\nBREAKING CHANGE: stored sessions use a new schema'
assert_valid "generated merge commit" "Merge branch 'feature/releases'"

assert_invalid "plain subject" "Add release archives"
assert_invalid "unsupported type" "feature: add release archives"
assert_invalid "capitalized type" "Feat: add release archives"
assert_invalid "missing description" "fix:"
assert_invalid "missing colon" "fix(auth) preserve the provider"
assert_invalid "fixup commit" "fixup! fix(auth): preserve the provider"

printf 'Commit message validation tests passed.\n'
