#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
OUTPUT_DIR="${2:-$ROOT_DIR/dist}"
SEMVER_PATTERN='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'
TARGETS="${FASTAI_RELEASE_TARGETS:-darwin/arm64 darwin/amd64 linux/arm64 linux/amd64 windows/amd64}"
STAGING_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/fastAI-release.XXXXXX")"

cleanup() {
	rm -rf "$STAGING_ROOT"
}

trap cleanup EXIT

if [[ ! "$VERSION" =~ $SEMVER_PATTERN ]]; then
	printf 'Version must be plain semantic version x.y.z: %s\n' "${VERSION:-<empty>}" >&2
	exit 2
fi

mkdir -p "$OUTPUT_DIR"
OUTPUT_DIR="$(cd "$OUTPUT_DIR" && pwd)"
archives=()

for target in $TARGETS; do
	case "$target" in
		darwin/arm64|darwin/amd64|linux/arm64|linux/amd64|windows/amd64) ;;
		*)
			printf 'Unsupported release target: %s\n' "$target" >&2
			exit 2
			;;
	esac

	target_os="${target%/*}"
	target_arch="${target#*/}"
	stage_dir="$STAGING_ROOT/${target_os}-${target_arch}"
	binary_name="fastAI"
	archive_name="fastAI_${VERSION}_${target_os}_${target_arch}.tar.gz"

	if [[ "$target_os" == "windows" ]]; then
		binary_name="fastAI.exe"
		archive_name="fastAI_${VERSION}_${target_os}_${target_arch}.zip"
	fi

	mkdir -p "$stage_dir"
	(
		cd "$ROOT_DIR"
		CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" go build \
			-trimpath \
			-buildvcs=false \
			-ldflags "-s -w -X main.version=$VERSION" \
			-o "$stage_dir/$binary_name" \
			./cmd/fastAI
	)

	if [[ "$target_os" == "windows" ]]; then
		command -v zip >/dev/null 2>&1 || {
			printf 'Missing required command: zip\n' >&2
			exit 1
		}
		(
			cd "$stage_dir"
			zip -q "$OUTPUT_DIR/$archive_name" "$binary_name"
		)
	else
		tar -C "$stage_dir" -czf "$OUTPUT_DIR/$archive_name" "$binary_name"
	fi
	archives+=("$archive_name")
done

: >"$OUTPUT_DIR/checksums.txt"
for archive in "${archives[@]}"; do
	if command -v shasum >/dev/null 2>&1; then
		(
			cd "$OUTPUT_DIR"
			shasum -a 256 "$archive"
		) >>"$OUTPUT_DIR/checksums.txt"
	else
		(
			cd "$OUTPUT_DIR"
			sha256sum "$archive"
		) >>"$OUTPUT_DIR/checksums.txt"
	fi
done

printf 'Created release artifacts in %s\n' "$OUTPUT_DIR"
