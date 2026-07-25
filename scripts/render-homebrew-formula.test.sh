#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RENDERER="$ROOT_DIR/scripts/render-homebrew-formula.sh"
TEST_DIR="$(mktemp -d "${TMPDIR:-/tmp}/fastAI-homebrew-formula-test.XXXXXX")"

cleanup() {
	rm -rf "$TEST_DIR"
}

trap cleanup EXIT

fail() {
	printf 'FAIL: %s\n' "$1" >&2
	exit 1
}

cat >"$TEST_DIR/checksums.txt" <<'EOF'
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  fastAI_1.2.3_darwin_arm64.tar.gz
bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  fastAI_1.2.3_darwin_amd64.tar.gz
cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc  fastAI_1.2.3_linux_arm64.tar.gz
dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd  fastAI_1.2.3_linux_amd64.tar.gz
eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee  fastAI_1.2.3_windows_amd64.zip
EOF

bash "$RENDERER" 1.2.3 "$TEST_DIR/checksums.txt" "$TEST_DIR/fastai.rb"

ruby -c "$TEST_DIR/fastai.rb" >/dev/null || fail "rendered formula should be valid Ruby"
if stat -f '%Lp' "$TEST_DIR/fastai.rb" >/dev/null 2>&1; then
	formula_mode="$(stat -f '%Lp' "$TEST_DIR/fastai.rb")"
else
	formula_mode="$(stat -c '%a' "$TEST_DIR/fastai.rb")"
fi
[[ "$formula_mode" == "644" ]] || fail "formula mode should be 644, got $formula_mode"
grep -Fq 'version "1.2.3"' "$TEST_DIR/fastai.rb" || fail "formula should contain the version"
grep -Fq '/releases/download/v1.2.3/' "$TEST_DIR/fastai.rb" || fail "formula should use the Go-compatible release tag"
grep -Fq 'fastAI_1.2.3_darwin_arm64.tar.gz' "$TEST_DIR/fastai.rb" || fail "formula should contain the Apple Silicon asset"
grep -Fq 'sha256 "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"' "$TEST_DIR/fastai.rb" || fail "formula should contain the Apple Silicon checksum"
grep -Fq 'fastAI_1.2.3_linux_amd64.tar.gz' "$TEST_DIR/fastai.rb" || fail "formula should contain the Linux Intel asset"
if grep -Eq '@[A-Z0-9_]+@' "$TEST_DIR/fastai.rb"; then
	fail "formula should not contain unresolved placeholders"
fi

sed '/linux_amd64/d' "$TEST_DIR/checksums.txt" >"$TEST_DIR/incomplete-checksums.txt"
if bash "$RENDERER" 1.2.3 "$TEST_DIR/incomplete-checksums.txt" "$TEST_DIR/incomplete.rb" >"$TEST_DIR/output" 2>&1; then
	fail "missing release checksum should fail"
fi
grep -Fq 'Missing checksum' "$TEST_DIR/output" || fail "missing checksum failure should be actionable"

printf 'Homebrew formula rendering tests passed.\n'
