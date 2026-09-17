# 01: Add the hook environment builder and project variables

**What to build:** Make command hooks inherit the complete parent environment and expose stable project-directory variables for zot and Claude-compatible hooks.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] Preserve all inherited environment variables, including custom, CI, toolchain, and empty values.
- [ ] Set `ZOT_PROJECT_DIR` to the effective zot project directory.
- [ ] Set `CLAUDE_PROJECT_DIR` to the same value.
- [ ] Replace stale inherited values for both owned project variables.
- [ ] Emit one entry for each owned variable.
- [ ] Keep unsupported variables inherited when the parent provides them.
- [ ] Leave unsupported variables unset when the parent does not provide them.
- [ ] Run hooks with the built environment through the existing command execution seam.
- [ ] Add unit and end-to-end tests for inheritance, replacement, duplicates, and shell expansion.
