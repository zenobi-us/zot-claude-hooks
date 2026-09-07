#!/bin/sh
set -eu
payload="$(cat)"
printf '{"action":"hook","payload":%s}\n' "$payload" >> "$ZOT_HOOK_TEST_LOG"
printf '%s\n' '{"decision":"block","reason":"fixture blocked this tool"}'
