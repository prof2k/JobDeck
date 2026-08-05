#!/bin/sh

set -eu

script_directory=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
project_root=$(CDPATH= cd -- "$script_directory/../.." && pwd)

if ! git -C "$project_root" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  exit 0
fi

tracked_private=$(git -C "$project_root" ls-files -- 'private/**' | awk '$0 != "private/README.md"')
staged_private=$(git -C "$project_root" diff --cached --name-only --diff-filter=ACMR -- 'private/**' | awk '$0 != "private/README.md"')

if [ -n "$tracked_private" ] || [ -n "$staged_private" ]; then
  echo "Privacy guard: only private/README.md may be tracked or staged." >&2
  exit 1
fi
