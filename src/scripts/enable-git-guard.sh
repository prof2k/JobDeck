#!/bin/sh

set -eu

script_directory=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
project_root=$(CDPATH= cd -- "$script_directory/../.." && pwd)

if ! git -C "$project_root" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  exit 0
fi

git -C "$project_root" config core.hooksPath .githooks
"$project_root/src/scripts/privacy-check.sh"
