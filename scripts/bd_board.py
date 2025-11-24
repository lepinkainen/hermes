#!/usr/bin/env python3
"""
Renders a compact, human-readable view of open/in-progress bd issues with parent-child grouping.
Requires: bd
"""

import json
import subprocess
import sys
from typing import Any


# ANSI color codes
class Colors:
    RESET = "\033[0m"
    BOLD = "\033[1m"

    # Status colors
    IN_PROGRESS = "\033[38;2;240;198;116m"  # yellow
    OPEN = "\033[38;2;129;162;190m"         # blue
    DEFAULT = "\033[38;2;197;200;198m"      # grey

    # Priority colors
    P0 = "\033[38;2;204;102;102m"  # red - critical
    P1 = "\033[38;2;222;147;95m"   # orange - high
    P2 = "\033[38;2;240;198;116m"  # yellow - medium
    P3 = "\033[38;2;181;189;104m"  # green - low
    P4 = "\033[38;2;150;152;150m"  # grey - backlog

    # Other
    HEADER = "\033[38;2;181;189;104m"  # green
    MUTED = "\033[38;2;150;152;150m"   # grey
    CYAN = "\033[38;2;138;190;183m"    # cyan for type


def run_bd_command(args: list[str]) -> str:
    """Run a bd command and return stdout."""
    try:
        result = subprocess.run(
            ["bd"] + args,
            capture_output=True,
            text=True,
            check=True
        )
        return result.stdout
    except subprocess.CalledProcessError as e:
        print(f"Error running bd {' '.join(args)}: {e.stderr}", file=sys.stderr)
        sys.exit(1)
    except FileNotFoundError:
        print("bd is required.", file=sys.stderr)
        sys.exit(1)


def get_issues() -> list[dict[str, Any]]:
    """Get all open/in_progress issues with full details."""
    # Get list of issues
    list_output = run_bd_command(["list", "--json"])
    if not list_output.strip():
        return []

    all_issues = json.loads(list_output)

    # Filter to open/in_progress
    open_ids = [
        issue["id"]
        for issue in all_issues
        if issue.get("status") in ("open", "in_progress")
    ]

    if not open_ids:
        return []

    # Get full details with dependencies
    show_output = run_bd_command(["show", "--json"] + open_ids)
    if not show_output.strip():
        return []

    return json.loads(show_output)


def build_tree(issues: list[dict[str, Any]]) -> tuple[list[str], dict[str, list[str]], dict[str, dict]]:
    """
    Build parent-child tree structure.
    Returns: (top_level_ids, child_map, lookup)
    """
    lookup = {issue["id"]: issue for issue in issues}

    # Build child_map: parent_id -> [child_ids]
    child_map: dict[str, list[str]] = {}
    child_ids_set: set[str] = set()

    # Process dependents field (parent perspective - epics have this)
    for issue in issues:
        parent_id = issue["id"]
        dependents = issue.get("dependents") or []
        if dependents:
            for dependent in dependents:
                child_id = dependent["id"]
                if parent_id not in child_map:
                    child_map[parent_id] = []
                child_map[parent_id].append(child_id)
                child_ids_set.add(child_id)

    # Also process dependencies field (child perspective - for backwards compatibility)
    for issue in issues:
        for dep in issue.get("dependencies") or []:
            if dep.get("dependency_type") == "parent-child":
                parent_id = dep["id"]
                child_id = issue["id"]
                if parent_id not in child_map:
                    child_map[parent_id] = []
                if child_id not in child_map[parent_id]:  # Avoid duplicates
                    child_map[parent_id].append(child_id)
                child_ids_set.add(child_id)

    # Top-level: issues that aren't children
    top_level = [
        issue["id"]
        for issue in issues
        if issue["id"] not in child_ids_set
    ]

    # Sort: epics first (by priority), then other issues (by priority)
    def sort_key(issue_id: str) -> tuple:
        issue = lookup[issue_id]
        is_epic = 0 if issue.get("issue_type") == "epic" else 1
        return (is_epic, issue.get("priority", 2), issue_id)

    top_level.sort(key=sort_key)

    return top_level, child_map, lookup


def truncate(text: str, max_len: int = 50) -> str:
    """Truncate text with ellipsis if too long."""
    if len(text) > max_len:
        return text[:max_len - 1] + "…"
    return text


def get_status_style(status: str) -> tuple[str, str]:
    """Get color and icon for status."""
    if status == "in_progress":
        return Colors.IN_PROGRESS, "●"
    elif status == "open":
        return Colors.OPEN, "○"
    else:
        return Colors.DEFAULT, "·"


def get_type_short(issue_type: str) -> str:
    """Get abbreviated type name."""
    abbrevs = {
        "feature": "feat",
        "task": "task",
        "bug": "bug",
        "epic": "epic",
        "chore": "chore",
    }
    return abbrevs.get(issue_type, issue_type)


def print_issue(issue: dict[str, Any], prefix: str = "") -> None:
    """Print a single issue with formatting."""
    status = issue.get("status", "open")
    priority = issue.get("priority", 2)
    issue_type = issue.get("issue_type", "task")
    title = truncate(issue.get("title", ""))
    issue_id = issue["id"]

    color, icon = get_status_style(status)
    type_short = get_type_short(issue_type)

    # Format: prefix + icon + id + [priority type] + title
    line = f"{prefix}{icon} {issue_id:<10} [p{priority} {type_short}] {title}"
    print(f"{color}{line}{Colors.RESET}")


def render_children(
    parent_id: str,
    child_map: dict[str, list[str]],
    lookup: dict[str, dict],
    indent: str = ""
) -> None:
    """Recursively render children with tree characters."""
    children = child_map.get(parent_id, [])
    if not children:
        return

    total = len(children)
    for i, child_id in enumerate(children):
        if child_id not in lookup:
            continue

        is_last = (i == total - 1)
        branch = "└── " if is_last else "├── "

        print_issue(lookup[child_id], f"{indent}{branch}")

        # Render grandchildren with appropriate continuation
        next_indent = f"{indent}    " if is_last else f"{indent}│   "
        render_children(child_id, child_map, lookup, next_indent)


def main() -> None:
    issues = get_issues()
    if not issues:
        print("No open issues found.")
        return

    top_level, child_map, lookup = build_tree(issues)

    # Calculate counts
    total = len(issues)
    in_progress = sum(1 for i in issues if i.get("status") == "in_progress")
    open_count = sum(1 for i in issues if i.get("status") == "open")

    # Print header
    print()
    print(f"{Colors.BOLD}{Colors.HEADER}bd Board{Colors.RESET}")
    print(f"{Colors.MUTED}{total} issues ({in_progress} in progress, {open_count} open){Colors.RESET}")
    print()

    # Render issues
    for issue_id in top_level:
        if issue_id not in lookup:
            continue
        print_issue(lookup[issue_id])
        render_children(issue_id, child_map, lookup, "")
        print()


if __name__ == "__main__":
    main()
