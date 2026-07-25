#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMPLATE="$ROOT_DIR/packaging/homebrew/fastai.rb.tmpl"
VERSION="${1:-}"
CHECKSUMS_FILE="${2:-}"
OUTPUT_FILE="${3:-}"
SEMVER_PATTERN='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'

if [[ ! "$VERSION" =~ $SEMVER_PATTERN ]]; then
	printf 'Version must be plain semantic version x.y.z: %s\n' "${VERSION:-<empty>}" >&2
	exit 2
fi
if [[ -z "$CHECKSUMS_FILE" || ! -f "$CHECKSUMS_FILE" ]]; then
	printf 'Checksums file does not exist: %s\n' "${CHECKSUMS_FILE:-<empty>}" >&2
	exit 2
fi
if [[ -z "$OUTPUT_FILE" ]]; then
	printf 'Usage: %s <version> <checksums-file> <output-file>\n' "$(basename "$0")" >&2
	exit 2
fi

checksum_for() {
	local asset="$1"
	local checksum

	checksum="$(
		awk -v asset="$asset" '
			$2 == asset && $1 ~ /^[[:xdigit:]]{64}$/ {
				print tolower($1)
				found++
			}
			END {
				if (found != 1) {
					exit 1
				}
			}
		' "$CHECKSUMS_FILE"
	)" || {
		printf 'Missing checksum for release asset: %s\n' "$asset" >&2
		exit 1
	}
	printf '%s\n' "$checksum"
}

DARWIN_ARM64_SHA="$(checksum_for "fastAI_${VERSION}_darwin_arm64.tar.gz")"
DARWIN_AMD64_SHA="$(checksum_for "fastAI_${VERSION}_darwin_amd64.tar.gz")"
LINUX_ARM64_SHA="$(checksum_for "fastAI_${VERSION}_linux_arm64.tar.gz")"
LINUX_AMD64_SHA="$(checksum_for "fastAI_${VERSION}_linux_amd64.tar.gz")"

mkdir -p "$(dirname "$OUTPUT_FILE")"
TEMP_OUTPUT="$(mktemp "${TMPDIR:-/tmp}/fastai-formula.XXXXXX")"

cleanup() {
	rm -f "$TEMP_OUTPUT"
}

trap cleanup EXIT

awk \
	-v version="$VERSION" \
	-v darwin_arm64_sha="$DARWIN_ARM64_SHA" \
	-v darwin_amd64_sha="$DARWIN_AMD64_SHA" \
	-v linux_arm64_sha="$LINUX_ARM64_SHA" \
	-v linux_amd64_sha="$LINUX_AMD64_SHA" \
	'{
		gsub(/@VERSION@/, version)
		gsub(/@DARWIN_ARM64_SHA256@/, darwin_arm64_sha)
		gsub(/@DARWIN_AMD64_SHA256@/, darwin_amd64_sha)
		gsub(/@LINUX_ARM64_SHA256@/, linux_arm64_sha)
		gsub(/@LINUX_AMD64_SHA256@/, linux_amd64_sha)
		print
	}' "$TEMPLATE" >"$TEMP_OUTPUT"

mv "$TEMP_OUTPUT" "$OUTPUT_FILE"
chmod 0644 "$OUTPUT_FILE"
trap - EXIT
printf 'Rendered Homebrew formula at %s\n' "$OUTPUT_FILE"
