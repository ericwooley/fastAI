#!/usr/bin/env bash

set -euo pipefail

usage() {
	cat <<'EOF'
Usage: ./scripts/install.sh [--dir <install-dir>]

Builds fastAI from the current checkout and installs it into a user-level bin directory.

Options:
  --dir <install-dir>   Override the install directory.
  -h, --help            Show this help message.

Environment:
  FASTAI_INSTALL_DIR    Override the install directory.
EOF
}

install_dir="${FASTAI_INSTALL_DIR:-}"

while [[ $# -gt 0 ]]; do
	case "$1" in
		--dir)
			if [[ $# -lt 2 ]]; then
				echo "error: --dir requires a value" >&2
				exit 1
			fi
			install_dir="$2"
			shift 2
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			echo "error: unknown argument: $1" >&2
			usage >&2
			exit 1
			;;
	esac
done

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/.." && pwd)"
binary_name="fastAI"
build_dir="$repo_root/tmp/bin"
build_path="$build_dir/$binary_name"

resolve_install_dir() {
	if [[ -n "$install_dir" ]]; then
		printf '%s\n' "$install_dir"
		return
	fi

	case "$(uname -s)" in
		Darwin)
			if [[ -d "/usr/local/bin" && -w "/usr/local/bin" ]]; then
				printf '%s\n' "/usr/local/bin"
				return
			fi
			printf '%s\n' "$HOME/.local/bin"
			;;
		Linux)
			printf '%s\n' "$HOME/.local/bin"
			;;
		MINGW*|MSYS*|CYGWIN*)
			if [[ -n "${USERPROFILE:-}" ]]; then
				printf '%s\n' "$USERPROFILE/bin"
				return
			fi
			printf '%s\n' "$HOME/bin"
			;;
		*)
			printf '%s\n' "$HOME/.local/bin"
			;;
	esac
}

target_dir="$(resolve_install_dir)"
target_path="$target_dir/$binary_name"

mkdir -p "$build_dir"
mkdir -p "$target_dir"

echo "Building $binary_name..."
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$build_path" ./cmd/fastAI

cp "$build_path" "$target_path"
chmod 0755 "$target_path"

echo "Installed $binary_name to $target_path"

case ":$PATH:" in
	*":$target_dir:"*)
		echo "$target_dir is already on PATH"
		;;
	*)
		echo "warning: $target_dir is not on PATH" >&2
		echo "Add it to your shell profile to run '$binary_name' directly." >&2
		;;
esac
