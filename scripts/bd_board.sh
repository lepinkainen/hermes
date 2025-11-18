#!/usr/bin/env bash
set -euo pipefail

# Usage: scripts/bd_board.sh [issues.jsonl]
# Renders a compact, human-readable view of open/in-progress bd issues with parent-child grouping.
# Requires: jq, gum.

issues_file="${1:-.beads/issues.jsonl}"

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required." >&2
  exit 1
fi

if ! command -v gum >/dev/null 2>&1; then
  echo "gum is required." >&2
  exit 1
fi

if [[ ! -f "$issues_file" ]]; then
  echo "Issues file not found: $issues_file" >&2
  exit 1
fi

# Build a JSON payload with items and child relationships (parent-child dependencies).
payload=$(jq -s '
  # Keep only open / in_progress issues
  map(select(.status != "closed")) as $items
  |
  # Map of parent -> [child ids] from parent-child dependencies
  (reduce $items[] as $i ({}; reduce ($i.dependencies // [])[]? as $d (.;
    if $d.type == "parent-child"
      then .[$d.depends_on_id] = (.[$d.depends_on_id] // []) + [$i.id]
      else .
    end))) as $children
  |
  # Set of ids that are children (to detect standalones)
  (reduce $children[]? as $arr ([]; . + $arr)) as $child_ids
  |
  $items
  | map({
      id,
      title,
      status,
      priority,
      issue_type,
      children: ($children[.id] // [])
    }) as $enriched
  |
  {
    top: $enriched
      | map(select(.issue_type == "epic" or (.id as $cid | ($child_ids | index($cid)) | not)))
      | sort_by(.priority, .issue_type, .id),
    lookup: ($enriched | INDEX(.id)),
    child_map: $children
  }
' "$issues_file")

print_issue() {
  local id="$1" title="$2" priority="$3" status="$4" type="$5" prefix="${6:-}"
  local status_color fg="white"
  case "$status" in
    in_progress) status_color="#f0c674" ;; # yellow-ish
    open) status_color="#81a2be" ;;        # blue-ish
    *) status_color="#c5c8c6" ;;           # grey
  esac
  local label="${id} (p${priority} ${status} ${type}) — ${title}"
  gum style --foreground "$status_color" "${prefix}${label}"
}

render_children() {
  local parent_id="$1" indent="$2"
  local child_ids
  child_ids=$(echo "$payload" | jq -r --arg id "$parent_id" '.child_map[$id][]?')
  if [[ -z "$child_ids" ]]; then
    return
  fi
  while IFS= read -r cid; do
    local child
    child=$(echo "$payload" | jq -r --arg id "$cid" '.lookup[$id]')
    if [[ "$child" == "null" ]]; then
      continue
    fi
    local title priority status type
    title=$(echo "$child" | jq -r '.title')
    priority=$(echo "$child" | jq -r '.priority')
    status=$(echo "$child" | jq -r '.status')
    type=$(echo "$child" | jq -r '.issue_type')
    print_issue "$cid" "$title" "$priority" "$status" "$type" "$indent- "
    # Render grandchildren with extra indent
    render_children "$cid" "  $indent"
  done <<< "$child_ids"
}

echo
gum style --bold --foreground "#b5bd68" "Open/In-Progress bd Issues"
echo

# Render top-level issues and their children.
echo "$payload" | jq -r '.top[].id' | while IFS= read -r tid; do
  top=$(echo "$payload" | jq -r --arg id "$tid" '.lookup[$id]')
  title=$(echo "$top" | jq -r '.title')
  priority=$(echo "$top" | jq -r '.priority')
  status=$(echo "$top" | jq -r '.status')
  type=$(echo "$top" | jq -r '.issue_type')

  print_issue "$tid" "$title" "$priority" "$status" "$type"
  render_children "$tid" "  "
  echo
done
