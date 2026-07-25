#!/usr/bin/env bash

set -euo pipefail

DEFAULT_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROOT_DIR="${RELEASE_WORKFLOW_TEST_ROOT:-$DEFAULT_ROOT_DIR}"
PUBLISH_ACTION="$ROOT_DIR/.github/actions/update-homebrew/action.yml"
TESTDATA_ROOT="$DEFAULT_ROOT_DIR/scripts/testdata/release-workflow"

fail() {
	printf 'FAIL: %s\n' "$1" >&2
	exit 1
}

extract_job() {
	local workflow="$1"
	local job_name="$2"

	awk -v job_name="$job_name" '
		$0 == "jobs:" {
			in_jobs = 1
			next
		}
		!in_jobs {
			next
		}
		$0 == "  " job_name ":" {
			in_job = 1
		}
		in_job && /^  [[:alnum:]_-]+:/ && $0 != "  " job_name ":" {
			exit
		}
		in_job && /^[^[:space:]]/ {
			exit
		}
		in_job {
			print
		}
	' "$workflow"
}

publish_job_names() {
	local workflow="$1"

	awk '
		function finish_job() {
			if (job_name != "" && publishes_homebrew) {
				print job_name
			}
		}
		$0 == "jobs:" {
			in_jobs = 1
			next
		}
		!in_jobs {
			next
		}
		in_jobs && /^[^[:space:]]/ {
			finish_job()
			exit
		}
		/^  [[:alnum:]_-]+:$/ {
			finish_job()
			job_name = substr($0, 3, length($0) - 3)
			publishes_homebrew = 0
			next
		}
		job_name != "" && /uses: \.\/\.github\/actions\/update-homebrew/ {
			publishes_homebrew = 1
		}
		END {
			finish_job()
		}
	' "$workflow"
}

extract_action_step() {
	local action="$1"
	local step_name="$2"

	awk -v step_name="$step_name" '
		$0 == "    - name: " step_name {
			in_step = 1
		}
		in_step && /^    - name:/ && $0 != "    - name: " step_name {
			exit
		}
		in_step {
			print
		}
	' "$action"
}

assert_publish_job() {
	local label="$1"
	local job="$2"

	grep -Fq '    environment: release' <<<"$job" ||
		fail "$label publishing job must target the release environment"
	grep -Fq 'uses: ./.github/actions/update-homebrew' <<<"$job" ||
		fail "$label publishing job must use the shared Homebrew action"
	grep -Eq '^[[:space:]]+version: \$\{\{ .+ \}\}$' <<<"$job" ||
		fail "$label publishing job must pass a release version"
	grep -Fq 'tap-token: ${{ secrets.TAP_GITHUB_TOKEN }}' <<<"$job" ||
		fail "$label publishing job must pass the release environment token"

	case "$label" in
		release.yml:*)
			grep -Fq 'publish-release' <<<"$job" ||
				fail "$label publishing job must wait for the GitHub release"
			;;
	esac
}

publish_job_count=0
automatic_publish_job_count=0
repair_publish_job_count=0
for workflow in "$ROOT_DIR"/.github/workflows/*.yml; do
	while IFS= read -r job_name; do
		[[ -n "$job_name" ]] || continue
		assert_publish_job "$(basename "$workflow"):$job_name" "$(extract_job "$workflow" "$job_name")"
		publish_job_count=$((publish_job_count + 1))
		case "$(basename "$workflow")" in
			release.yml)
				automatic_publish_job_count=$((automatic_publish_job_count + 1))
				;;
			update-homebrew.yml)
				repair_publish_job_count=$((repair_publish_job_count + 1))
				;;
		esac
	done < <(publish_job_names "$workflow")
done

if grep -Fq 'uses: ./.github/workflows/update-homebrew.yml' "$ROOT_DIR"/.github/workflows/*.yml; then
	fail "Homebrew publishing must not cross a reusable workflow secret boundary"
fi

[[ "$automatic_publish_job_count" -gt 0 ]] ||
	fail "release.yml must publish Homebrew automatically from the release environment"
[[ "$repair_publish_job_count" -gt 0 ]] ||
	fail "update-homebrew.yml must provide a release-environment repair path"
[[ "$publish_job_count" -gt 0 ]] ||
	fail "at least one workflow must publish through the shared Homebrew action"

[[ -f "$PUBLISH_ACTION" ]] ||
	fail "shared Homebrew publishing action must exist"
grep -Fq 'tap-token:' "$PUBLISH_ACTION" ||
	fail "shared Homebrew action must declare the tap token input"
grep -Fq 'TAP_GITHUB_TOKEN: ${{ inputs.tap-token }}' <<<"$(extract_action_step "$PUBLISH_ACTION" "Require Homebrew tap token")" ||
	fail "tap token guard must consume the tap-token input"
grep -Fq 'GH_TOKEN: ${{ github.token }}' <<<"$(extract_action_step "$PUBLISH_ACTION" "Download release archives")" ||
	fail "release downloads must use the workflow token"
grep -Fq 'token: ${{ inputs.tap-token }}' <<<"$(extract_action_step "$PUBLISH_ACTION" "Checkout Homebrew tap")" ||
	fail "tap checkout must consume the tap-token input"
grep -Fq 'GH_TOKEN: ${{ inputs.tap-token }}' <<<"$(extract_action_step "$PUBLISH_ACTION" "Push Homebrew formula")" ||
	fail "tap push must consume the tap-token input"

commit_step="$(extract_action_step "$PUBLISH_ACTION" "Commit Homebrew formula")"
push_step="$(extract_action_step "$PUBLISH_ACTION" "Push Homebrew formula")"
grep -Fq '      id: commit' <<<"$commit_step" ||
	fail "formula commit step must expose a stable output id"
grep -Fq 'changed=' <<<"$commit_step" ||
	fail "formula commit step must report whether the formula changed"
grep -Fq '"$GITHUB_OUTPUT"' <<<"$commit_step" ||
	fail "formula commit step must publish its changed output"
grep -Fq "if: steps.commit.outputs.changed == 'true'" <<<"$push_step" ||
	fail "tap push must use the formula commit output"

assert_fixture_failure() {
	local label="$1"
	local fixture="$2"
	local expected="$3"
	local output
	local status

	set +e
	output="$(
		RELEASE_WORKFLOW_TEST_ROOT="$fixture" \
			bash "$DEFAULT_ROOT_DIR/scripts/release-workflow.test.sh" 2>&1
	)"
	status=$?
	set -e

	[[ "$status" -ne 0 ]] ||
		fail "$label fixture must fail validation"
	grep -Fq "$expected" <<<"$output" ||
		fail "$label fixture failed for an unexpected reason: $output"
}

if [[ "$ROOT_DIR" == "$DEFAULT_ROOT_DIR" ]]; then
	assert_fixture_failure \
		"reusable workflow" \
		"$TESTDATA_ROOT/reusable-workflow" \
		"Homebrew publishing must not cross a reusable workflow secret boundary"
	assert_fixture_failure \
		"missing environment" \
		"$TESTDATA_ROOT/missing-environment" \
		"publishing job must target the release environment"
fi

printf 'release workflow tests passed\n'
