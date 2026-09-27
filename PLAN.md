# zot-cluade-hooks plan

Discussion: https://github.com/patriceckhart/zot/discussions/170

`zot-cluade-hooks` reads Claude-style hook definitions from JSON files and runs matching command hooks through the zot extension protocol.

## 1. Possible now

These features use the current zot extension protocol.

- Discover hook definitions in the fixed files `~/.claude/settings.json`, `$ZOT_HOME/zot-cluade-hooks.json`, `.claude/settings.json`, `.claude/settings.local.json`, `.zot/zot-cluade-hooks.json`, `.zot/zot-cluade-hooks.local.json`, and `$ZOT_HOOKS_PATH`; then check the shared hook directories `$HOME/.agents/hooks/`, `$HOME/.claude/hooks/`, `$ZOT_HOME/hooks/`, `<project>/.agents/hooks/`, `<project>/.claude/hooks/`, and `<project>/.zot/hooks/` for direct, sorted `.json` files; finally check `$ZOT_HOME/extensions/*/hooks/*.json`.
- Read a top-level `hooks` object from each JSON file.
- Support `type: "command"` hook entries.
- Support `matcher` as a regular expression against the zot tool name.
- Run `PreToolUse` hooks through the synchronous `tool_call` interceptor.
- Pass a Claude-style JSON payload to each command on standard input.
- Map exit status `0` to allow.
- Map exit status `2` to block.
- Parse `decision: "block"` and `reason` from JSON command output.
- Subscribe to current asynchronous events:
  - `SessionStart`
  - `Stop` through `turn_end`
  - `Notification` through available zot event notifications
  - `tool_call` for audit information
  - `assistant_message` for audit information
- Enforce a command timeout.
- Discover JSON hook files contributed by installed extensions.
- Write diagnostics to standard error.
- Keep standard output reserved for the zot protocol.
- Fail open when a hook command times out or returns an invalid response, except that a valid hook exit status `2` blocks the tool.

The Go implementation uses zot's extension SDK for protocol handling and implements hook discovery, `PreToolUse`, current event forwarding, and the `list` diagnostic command. Remaining policy details are tracked below.

## 2. TODO: requires missing zot events

Track implementation against [discussion #170](https://github.com/patriceckhart/zot/discussions/170).

- [x] Add `user_prompt_submit` support when zot exposes the event.
- [x] Add `tool_result` support for `PostToolUse`.
- [x] Include the effective tool arguments and the final tool status.
- [x] Forward the SDK's execution status and `executed` flag for completed, failed, blocked, cancelled, and timed-out tool calls.
- [x] Add `session_end` support.
- [x] Add `pre_compact` and `post_compact` support.
- [x] Add `subagent_start` and `subagent_stop` support.
- [x] Add `permission_decision` support (forwarded as `PermissionRequest`).
- [ ] Add synchronous prompt replacement or blocking after zot defines its semantics.
- [ ] Add event ordering tests for blocked and cancelled tool calls.
- [ ] Revisit fail-open behavior if zot adds a fail-closed policy mode.

## Open design decisions

- Define the trust prompt or opt-in rule for arbitrary commands from hook files.
- Define how hook command output maps to tool argument changes.
- Define whether invalid matchers disable one hook or the full file.

The current discovery order is global Claude settings, the user-level `$ZOT_HOME` hook file, project Claude settings, project-local overrides, project Zot settings, project-local Zot overrides, `ZOT_HOOKS_PATH`, the six shared hook directories listed above, and installed extension hook files. Directory files are direct-only, sorted by filename, and deduplicated by canonical path. `settings.json` files are now the supported configuration format.

Environment compatibility is implemented and documented in the README. The contract preserves inherited variables, owns verified `ZOT_*` values, adds only matching Claude aliases, keeps extension values source-scoped, and uses a safe session-scoped persistence file for later hooks. The complete unit and end-to-end environment matrix is covered by the Go tests.
