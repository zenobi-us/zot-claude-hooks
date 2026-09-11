#!/usr/bin/env bash
set -euo pipefail

payload="${1:-}"
if [[ -z "${payload}" ]]; then
  payload="$(cat)"
fi
jq -e . >/dev/null <<<"${payload}"
{
  echo '## Release Please Outputs'
  echo
  echo '| field | value |'
  echo '|---|---|'
  echo "| releases_created | \`$(jq -r '.releases_created // "false"' <<<"${payload}")\` |"
  echo "| prs_created | \`$(jq -r '.prs_created // "false"' <<<"${payload}")\` |"
  echo
  echo '```json'
  jq . <<<"${payload}"
  echo '```'
} >> "${GITHUB_STEP_SUMMARY}"
