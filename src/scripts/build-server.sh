#!/bin/sh

set -eu

script_directory=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
project_root=$(CDPATH= cd -- "$script_directory/../.." && pwd)
server_root="$project_root/src/server"
web_dist="$project_root/src/web/dist"
embed_root="$server_root/webdist"
runtime_root="$project_root/private/runtime"
temporary_binary="$runtime_root/jobdeck.tmp"
root_binary="$project_root/jobdeck"

if [ ! -f "$web_dist/index.html" ]; then
  echo "Job Deck web output is missing. Build the UI only because dist is unavailable." >&2
  exit 1
fi

mkdir -p "$runtime_root"
rm -rf "$embed_root"
mkdir -p "$embed_root"
cp -R "$web_dist/." "$embed_root/"
touch "$embed_root/placeholder.txt"

cd "$server_root"
go test ./...
go build -trimpath -o "$temporary_binary" .
chmod 700 "$temporary_binary"
mv "$temporary_binary" "$root_binary"

cd "$project_root"
"$root_binary" record-build
"$root_binary" verify
