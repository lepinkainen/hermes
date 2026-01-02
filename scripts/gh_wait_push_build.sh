#!/usr/bin/env bash
set -euo pipefail

GH_CMD=${GH_CMD:-gh}
GIT_CMD=${GIT_CMD:-git}

usage() {
  cat <<'USAGE'
Usage: gh_wait_push_build.sh [--branch <name>] [--repo <owner/repo>] [--interval <seconds>]

Wait for the latest GitHub Actions run triggered by a git push and report status.

Options:
  --branch    Branch name to filter runs (defaults to current git branch)
  --repo      Repository in owner/repo format (defaults to gh's current repo)
  --interval  Poll interval for gh run watch (default: 10)
  -h, --help  Show this help message
USAGE
}

branch=""
repo=""
interval="10"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --branch)
      branch="$2"
      shift 2
      ;;
    --repo)
      repo="$2"
      shift 2
      ;;
    --interval)
      interval="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if ! command -v "$GH_CMD" >/dev/null 2>&1; then
  echo "gh CLI not found: $GH_CMD" >&2
  exit 127
fi

if [[ -z "$branch" ]]; then
  branch="$($GIT_CMD rev-parse --abbrev-ref HEAD)"
fi

repo_args=()
if [[ -n "$repo" ]]; then
  repo_args=(--repo "$repo")
fi

run_id="$($GH_CMD run list "${repo_args[@]}" --event push --branch "$branch" --limit 1 --json databaseId -q '.[0].databaseId')"

if [[ -z "$run_id" || "$run_id" == "null" ]]; then
  echo "No push-triggered runs found for branch: $branch" >&2
  exit 1
fi

# Poll for completion instead of using interactive watch
while true; do
  status="$($GH_CMD run view "${repo_args[@]}" "$run_id" --json status,conclusion -q '.status + " " + .conclusion')"
  read -r run_status run_conclusion <<< "$status"

  if [[ "$run_status" == "completed" ]]; then
    break
  fi

  sleep "$interval"
done

conclusion="$run_conclusion"
if [[ "$conclusion" == "success" ]]; then
  echo "build ok"
  exit 0
fi

if ! $GH_CMD run view "${repo_args[@]}" "$run_id" --log-failed; then
  $GH_CMD run view "${repo_args[@]}" "$run_id" --log
fi
exit 1
