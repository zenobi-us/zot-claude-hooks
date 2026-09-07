#!/bin/sh
set -eu
payload="$(cat)"
printf '{"action":"hook","payload":%s}\n' "$payload" >> "$ZOT_HOOK_TEST_LOG"
exit 2
