#!/usr/bin/env python3
"""
LLM-oriented helper to fetch a movie or TV item from TMDB by ID.

- Reads TMDB_API_KEY from the environment (or --api-key).
- Accepts an explicit --type to avoid movie/TV ID collisions.
- Emits a single JSON object to stdout so agents can parse it easily.
"""

import argparse
import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request
from typing import Any, Dict, Optional


BASE_URL = "https://api.themoviedb.org/3"
DEFAULT_APPEND = "external_ids"


def build_url(media_type: str, tmdb_id: str, api_key: str, language: Optional[str], append: Optional[str]) -> str:
    """Construct the TMDB URL with query parameters."""
    params: Dict[str, Any] = {"api_key": api_key}
    if language:
        params["language"] = language
    if append:
        params["append_to_response"] = append

    query = urllib.parse.urlencode(params)
    return f"{BASE_URL}/{media_type}/{tmdb_id}?{query}"


def fetch_tmdb(url: str, timeout: float) -> Dict[str, Any]:
    """Fetch TMDB JSON and parse it."""
    request = urllib.request.Request(url, headers={"Accept": "application/json"})
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            body = response.read().decode("utf-8")
            return json.loads(body)
    except urllib.error.HTTPError as exc:
        error_body = exc.read().decode("utf-8", errors="ignore")
        raise RuntimeError(f"TMDB HTTP error {exc.code}: {error_body}") from exc
    except urllib.error.URLError as exc:
        raise RuntimeError(f"TMDB request failed: {exc.reason}") from exc
    except json.JSONDecodeError as exc:
        raise RuntimeError("TMDB response was not valid JSON") from exc


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Fetch a TMDB item by ID for movie or TV content. Outputs a single JSON object "
            "suitable for LLM consumption."
        )
    )
    parser.add_argument("--type", required=True, choices=["movie", "tv"], help="Explicit TMDB media type (movie or tv).")
    parser.add_argument("--id", dest="tmdb_id", required=True, help="TMDB numeric ID to fetch.")
    parser.add_argument(
        "--api-key",
        default=os.getenv("TMDB_API_KEY"),
        help="TMDB API key (defaults to TMDB_API_KEY env var).",
    )
    parser.add_argument(
        "--append",
        default=DEFAULT_APPEND,
        help=f"Comma-separated append_to_response extras (default: {DEFAULT_APPEND}).",
    )
    parser.add_argument(
        "--language",
        default=None,
        help="Optional language code (e.g., en-US).",
    )
    parser.add_argument(
        "--timeout",
        type=float,
        default=20.0,
        help="Request timeout in seconds (default: 20).",
    )
    parser.add_argument(
        "--pretty",
        action="store_true",
        help="Pretty-print JSON for humans (otherwise emits compact JSON for parsers).",
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()

    if not args.api_key:
        print("TMDB API key is required (set TMDB_API_KEY or pass --api-key).", file=sys.stderr)
        sys.exit(1)

    if not args.tmdb_id.isdigit():
        print("--id must be a numeric TMDB ID.", file=sys.stderr)
        sys.exit(1)

    url = build_url(args.type, args.tmdb_id, args.api_key, args.language, args.append)

    try:
        payload = fetch_tmdb(url, timeout=args.timeout)
    except RuntimeError as exc:
        print(str(exc), file=sys.stderr)
        sys.exit(1)

    output = {
        "media_type": args.type,
        "tmdb_id": int(args.tmdb_id),
        "append": args.append or "",
        "language": args.language or "",
        "source": f"{BASE_URL}/{args.type}/{args.tmdb_id}",
        "data": payload,
    }

    if args.pretty:
        print(json.dumps(output, indent=2, sort_keys=True))
    else:
        print(json.dumps(output, separators=(",", ":")))


if __name__ == "__main__":
    main()
