#!/usr/bin/env bash
set -euo pipefail

require_arg() { [[ -n "${1:-}" ]] || { echo "Missing ${2:-argument}" >&2; exit 1; }; }
resolve_commit() {
  git rev-parse --verify "$1^{commit}" 2>/dev/null || {
    echo "Unknown commit or branch: $1" >&2; exit 1;
  }
}
root_commit() { git merge-base "$(resolve_commit "$1")" "$(resolve_commit "$2")"; }

case "${1:-}" in
  is-root-commit)
    require_arg "${2:-}" commit; require_arg "${3:-}" release_branch
    [[ "${3}" == release/* ]] || { echo "Branch name '${3}' does not match pattern" >&2; exit 1; }
    [[ "$(resolve_commit "${2}")" == "$(root_commit "${3}" "${4:-origin/master}")" ]]
    ;;
  is-hotfix-merge)
    require_arg "${2:-}" commit
    commit="$(resolve_commit "${2}")"
    while read -r branch; do
      root="$(root_commit "${branch}" "${3:-origin/master}")"
      if git merge-base --is-ancestor "${root}" "${commit}" &&
         [[ "$(resolve_commit "${branch}")" != "${root}" ]]; then
        exit 0
      fi
    done < <(git for-each-ref --format='%(refname:short)' refs/remotes/origin/release refs/heads/release)
    exit 1
    ;;
  *) echo "Usage: $0 {is-root-commit COMMIT RELEASE_BRANCH [BASE]|is-hotfix-merge COMMIT [BASE]}" >&2; exit 1 ;;
esac
