#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PACKAGER="$ROOT_DIR/scripts/package-release.sh"
TEST_DIR="$(mktemp -d "${TMPDIR:-/tmp}/fastAI-package-release-test.XXXXXX")"
TARGET_OS="$(go env GOOS)"
TARGET_ARCH="$(go env GOARCH)"
VERSION="9.8.7"

cleanup() {
	rm -rf "$TEST_DIR"
}

trap cleanup EXIT

fail() {
	printf 'FAIL: %s\n' "$1" >&2
	exit 1
}

FASTAI_RELEASE_TARGETS="$TARGET_OS/$TARGET_ARCH" bash "$PACKAGER" "$VERSION" "$TEST_DIR/dist"

case "$TARGET_OS" in
	windows)
		archive="fastAI_${VERSION}_${TARGET_OS}_${TARGET_ARCH}.zip"
		mkdir -p "$TEST_DIR/extracted"
		unzip -q "$TEST_DIR/dist/$archive" -d "$TEST_DIR/extracted"
		binary="$TEST_DIR/extracted/fastAI.exe"
		;;
	*)
		archive="fastAI_${VERSION}_${TARGET_OS}_${TARGET_ARCH}.tar.gz"
		mkdir -p "$TEST_DIR/extracted"
		tar -xzf "$TEST_DIR/dist/$archive" -C "$TEST_DIR/extracted"
		binary="$TEST_DIR/extracted/fastAI"
		;;
esac

[[ -f "$TEST_DIR/dist/$archive" ]] || fail "release archive should exist"
[[ -f "$TEST_DIR/dist/checksums.txt" ]] || fail "checksums file should exist"
grep -Fq "$archive" "$TEST_DIR/dist/checksums.txt" || fail "checksums should include the archive"

if [[ "$TARGET_OS" != "windows" ]]; then
	version_output="$("$binary" --version)"
	grep -Fq "$VERSION" <<<"$version_output" || fail "packaged binary should report the release version"
fi

if command -v shasum >/dev/null 2>&1; then
	(cd "$TEST_DIR/dist" && shasum -a 256 -c checksums.txt) >/dev/null || fail "checksums should verify"
else
	(cd "$TEST_DIR/dist" && sha256sum -c checksums.txt) >/dev/null || fail "checksums should verify"
fi

printf 'Release packaging tests passed.\n'
