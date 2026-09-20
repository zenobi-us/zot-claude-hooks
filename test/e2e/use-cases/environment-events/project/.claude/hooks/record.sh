#!/bin/sh
set -eu
payload=$(cat)
printf '{"action":"hook","zot_project_dir":"%s","claude_project_dir":"%s","payload":%s}\n' \
  "${ZOT_PROJECT_DIR}" "${CLAUDE_PROJECT_DIR}" "${payload}" >> "${ZOT_HOOK_TEST_LOG}"
