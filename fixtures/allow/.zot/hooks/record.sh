#!/bin/sh
set -eu
mkdir -p .zot
payload=$(cat)
printf '%s\n' "$payload" >> .zot/hook-events.jsonl
