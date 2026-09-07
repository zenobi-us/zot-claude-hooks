#!/bin/sh
set -eu
mkdir -p .zot
payload=$(cat)
printf '%s\n' "$payload" >> .zot/hook-events.jsonl
printf '%s\n' '{"decision":"block","reason":"blocked by the zot-hooks E2E fixture"}'
exit 2
