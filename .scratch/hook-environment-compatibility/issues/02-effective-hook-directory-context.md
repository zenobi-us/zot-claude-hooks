# 02: Align hook directory and event context

**What to build:** Make hook input `cwd`, the hook process working directory, and project-directory variables follow one documented directory contract across all current hook events.

**Blocked by:** 01: Add the hook environment builder and project variables.

**Status:** ready-for-agent

- [ ] Define the difference between the session project directory and the active event directory.
- [ ] Use the effective event directory for the hook process working directory when zot provides it.
- [ ] Use the project directory as the fallback when no event directory exists.
- [ ] Keep `ZOT_PROJECT_DIR` and `CLAUDE_PROJECT_DIR` tied to the defined project-directory value.
- [ ] Keep hook input `cwd` consistent with the active hook directory.
- [ ] Verify the behavior for `PreToolUse`, `SessionStart`, `Stop`, and `Notification`.
- [ ] Add tests that compare the process directory, input `cwd`, and exported project variables.
