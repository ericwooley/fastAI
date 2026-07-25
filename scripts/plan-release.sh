#!/usr/bin/env bash

set -euo pipefail

REVISION="${1:-HEAD}"
TAG_PATTERN='^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'
CONVENTIONAL_PATTERN='^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([^()[:space:]]+\))?!?: .+'
BREAKING_HEADER_PATTERN='^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([^()[:space:]]+\))?!: .+'
FEATURE_PATTERN='^feat(\([^()[:space:]]+\))?: .+'
PATCH_PATTERN='^(fix|perf)(\([^()[:space:]]+\))?: .+'
CURRENT_VERSION="0.0.0"
CURRENT_TAG=""
RELEASE_TYPE="none"

if ! git cat-file -e "${REVISION}^{commit}" 2>/dev/null; then
	printf 'Revision is not a commit: %s\n' "$REVISION" >&2
	exit 2
fi

while IFS= read -r tag; do
	if [[ "$tag" =~ $TAG_PATTERN ]]; then
		CURRENT_VERSION="${tag#v}"
		CURRENT_TAG="$tag"
		break
	fi
done < <(git tag --merged "$REVISION" --sort=-version:refname)

if [[ -n "$CURRENT_TAG" ]]; then
	RANGE="${CURRENT_TAG}..${REVISION}"
else
	RANGE="$REVISION"
fi

while IFS= read -r commit; do
	[[ -n "$commit" ]] || continue

	MESSAGE="$(git show -s --format=%B "$commit")"
	HEADER="${MESSAGE%%$'\n'*}"

	# Invalid messages are rejected by the local hook and CI. Ignoring legacy
	# messages keeps the first release deterministic for an existing repository.
	if ! grep -Eq "$CONVENTIONAL_PATTERN" <<<"$HEADER"; then
		continue
	fi

	if grep -Eq "$BREAKING_HEADER_PATTERN" <<<"$HEADER" || grep -Eq '^BREAKING[ -]CHANGE: .+' <<<"$MESSAGE"; then
		RELEASE_TYPE="major"
		break
	fi

	if grep -Eq "$FEATURE_PATTERN" <<<"$HEADER"; then
		RELEASE_TYPE="minor"
	elif [[ "$RELEASE_TYPE" == "none" ]] && grep -Eq "$PATCH_PATTERN" <<<"$HEADER"; then
		RELEASE_TYPE="patch"
	fi
done < <(git log --no-merges --format=%H "$RANGE")

IFS=. read -r MAJOR MINOR PATCH <<<"$CURRENT_VERSION"
case "$RELEASE_TYPE" in
	major)
		NEXT_VERSION="$((MAJOR + 1)).0.0"
		;;
	minor)
		NEXT_VERSION="${MAJOR}.$((MINOR + 1)).0"
		;;
	patch)
		NEXT_VERSION="${MAJOR}.${MINOR}.$((PATCH + 1))"
		;;
	none)
		NEXT_VERSION=""
		;;
	*)
		printf 'Unsupported release type: %s\n' "$RELEASE_TYPE" >&2
		exit 2
		;;
esac

if [[ "$RELEASE_TYPE" == "none" ]]; then
	RELEASE_REQUIRED="false"
else
	RELEASE_REQUIRED="true"
fi

printf 'release_required=%s\n' "$RELEASE_REQUIRED"
printf 'current_version=%s\n' "$CURRENT_VERSION"
printf 'release_type=%s\n' "$RELEASE_TYPE"
printf 'next_version=%s\n' "$NEXT_VERSION"
