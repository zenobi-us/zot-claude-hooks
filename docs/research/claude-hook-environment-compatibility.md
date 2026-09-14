# Claude Code hook environment compatibility

**Research date:** 2026-09-15
**Scope:** Environment variables and command interpolation used by Claude Code hooks.
**Target:** `zot-claude-hooks`

## Executive summary

Claude Code hooks use two environment layers:

1. The hook process inherits the parent environment.
2. Claude Code adds variables and expands documented placeholders.

There is no finite list of all variables that a hook can read. A hook can read user-defined variables such as `PATH`, `CI`, or `MY_HOOK_MODE` when the parent process provides them. Claude Code can also remove variables from child processes through its environment scrub rules.

The first compatibility step should preserve the full inherited environment and set the one value that zot can calculate with high confidence:

```text
CLAUDE_PROJECT_DIR=<effective project directory>
```

The complete plan below also defines how to handle variables that zot can map later, variables that must remain inherited only, plugin-only variables, and environment persistence.

## Research findings

### 1. Inherited environment

A command hook inherits the parent Claude Code environment. Claude Code removes `OTEL_*` exporter variables from spawned processes. It can also scrub credential-like variables when `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`.

The extension currently leaves `exec.Cmd.Env` unset. Go then inherits the environment of the zot process. This already preserves user variables. The extension must keep this behavior when it adds compatibility variables.

The extension MUST NOT use a narrow allowlist. A narrow list would break hooks that depend on user, CI, toolchain, locale, proxy, or authentication variables.

**Confidence:** High. The behavior is stated in the official hook reference.

### 2. Project and plugin path variables

Claude Code documents these path variables and placeholders:

| Variable or placeholder | Meaning | Safe zot mapping |
| --- | --- | --- |
| `CLAUDE_PROJECT_DIR` / `${CLAUDE_PROJECT_DIR}` | Project root where the session started | Yes. Map to the effective zot project directory. |
| `CLAUDE_PLUGIN_ROOT` / `${CLAUDE_PLUGIN_ROOT}` | Installed plugin directory | Not yet. Zot extensions are not Claude plugins. |
| `CLAUDE_PLUGIN_DATA` / `${CLAUDE_PLUGIN_DATA}` | Persistent plugin data directory | Not yet. Define a zot-specific mapping first. |

`CLAUDE_PROJECT_DIR` stays at the original project root when Claude enters a worktree. The hook input `cwd` follows the active worktree or directory after `cd`.

For zot, the first implementation must define this distinction:

- `CLAUDE_PROJECT_DIR` means the project directory known by the zot host.
- Hook JSON `cwd` and the command working directory mean the active directory for the event.

Until zot exposes a separate session-root value, the current host `CWD` is the best available value for both.

**Confidence:** High for Claude behavior. Medium for the zot mapping until the host API exposes separate session-root and event-cwd values.

### 3. Session and runtime variables

Claude Code documents these variables for hook processes:

```text
CLAUDE_CODE_SESSION_ID
CLAUDE_EFFORT
CLAUDE_CODE_REMOTE
CLAUDE_CODE_REMOTE_SESSION_ID
CLAUDE_CODE_BRIDGE_SESSION_ID
CLAUDE_CODE_MESSAGING_SOCKET
CLAUDE_CODE_MESSAGING_TOKEN
CLAUDE_PID
CLAUDECODE
CLAUDE_CODE_CHILD_SESSION
TRACEPARENT
CLAUDE_CODE_SHELL_PREFIX
```

Their meanings include session identity, effort level, cloud or Remote Control state, messaging access, parent PID, nested-process state, trace propagation, and shell wrappers.

Zot does not currently expose a proven equivalent for every value. The extension MUST preserve an inherited value. It MUST NOT create a guessed value. An unset value means “unknown”. This is safer than setting a false or synthetic value that can change hook behavior.

Some of these values can become supported after the extension maps them to zot event or host data:

| Variable | Possible zot source | Initial action |
| --- | --- | --- |
| `CLAUDE_CODE_SESSION_ID` | `HostInfo` or `Event.SessionID` | Preserve only. Add after a semantic mapping is documented. |
| `CLAUDE_EFFORT` | Event effort data, if zot exposes it | Set only when the event has a matching effort level. |
| `CLAUDE_CODE_REMOTE` | No current equivalent | Preserve only. |
| `CLAUDE_CODE_REMOTE_SESSION_ID` | No current equivalent | Preserve only. |
| `CLAUDE_CODE_BRIDGE_SESSION_ID` | No current equivalent | Preserve only. |
| `CLAUDE_CODE_MESSAGING_SOCKET` | No current equivalent | Preserve only. |
| `CLAUDE_CODE_MESSAGING_TOKEN` | No current equivalent | Preserve only. |
| `CLAUDE_PID` | The zot process PID is not the Claude process PID | Preserve only. |
| `CLAUDECODE` | No semantic need in zot | Preserve only. |
| `CLAUDE_CODE_CHILD_SESSION` | Agent or child-session data, if exposed | Preserve only until mapped. |
| `TRACEPARENT` | Parent environment | Preserve only. |
| `CLAUDE_CODE_SHELL_PREFIX` | No current equivalent | Preserve only. |

**Confidence:** High for the Claude definitions. Medium for the zot source column because it depends on the installed zot SDK version.

### 4. Zot-native environment variables

Zot should expose its own namespaced variables for the same concepts. This avoids claiming that zot is Claude Code while giving hooks a stable native interface.

| Claude variable | Zot equivalent | Meaning and rule |
| --- | --- | --- |
| `CLAUDE_PROJECT_DIR` | `ZOT_PROJECT_DIR` | Effective zot project directory. Set this in every hook process. |
| `CLAUDE_PLUGIN_ROOT` | `ZOT_EXTENSION_ROOT` | Root directory for the zot extension that owns the hook. Set only for extension-owned hook files. |
| `CLAUDE_PLUGIN_DATA` | `ZOT_EXTENSION_DATA` | Persistent data directory for the owning zot extension. Set only when the directory contract exists. |
| `CLAUDE_CODE_SESSION_ID` | `ZOT_SESSION_ID` | Zot session identifier. Set only when the host provides one. |
| `CLAUDE_EFFORT` | `ZOT_EFFORT` | Zot effort level, if the host exposes the same concept. Do not invent a value. |
| `CLAUDE_ENV_FILE` | `ZOT_ENV_FILE` | Zot environment persistence file. Do not set until the persistence feature is implemented. |
| `CLAUDE_CODE_REMOTE` | `ZOT_REMOTE` | Zot remote-session marker. Set only when zot exposes a remote state. |
| `CLAUDE_CODE_REMOTE_SESSION_ID` | `ZOT_REMOTE_SESSION_ID` | Zot remote session identifier. |
| `CLAUDE_CODE_BRIDGE_SESSION_ID` | `ZOT_BRIDGE_SESSION_ID` | Zot bridge or control-session identifier. |
| `CLAUDE_CODE_MESSAGING_SOCKET` | `ZOT_MESSAGING_SOCKET` | Zot session messaging socket. Do not set until zot exposes one. |
| `CLAUDE_CODE_MESSAGING_TOKEN` | `ZOT_MESSAGING_TOKEN` | Zot messaging token. Never log this value. |
| `CLAUDE_PID` | `ZOT_PID` | PID of the parent zot process, not the hook process. |
| `CLAUDECODE` | `ZOTCODE` | Marker that zot spawned the subprocess. Set to `1` when applicable. |
| `CLAUDE_CODE_CHILD_SESSION` | `ZOT_CHILD_SESSION` | Marker for a zot child or agent session. |
| `CLAUDE_CODE_SHELL_PREFIX` | `ZOT_SHELL_PREFIX` | Zot shell wrapper, if zot provides one. |
| `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB` | `ZOT_SUBPROCESS_ENV_SCRUB` | Zot environment scrub setting. |
| `CLAUDE_CODE_SESSIONEND_HOOKS_TIMEOUT_MS` | `ZOT_SESSIONEND_HOOKS_TIMEOUT_MS` | Zot session-end hook budget. |
| `CLAUDE_CODE_DEBUG_LOG_LEVEL` | `ZOT_DEBUG_LOG_LEVEL` | Zot extension diagnostic level. |
| `CLAUDE_CODE_PROPAGATE_TRACEPARENT` | `ZOT_PROPAGATE_TRACEPARENT` | Zot trace propagation setting. |
| `CLAUDE_CODE_SAFE_MODE` | `ZOT_SAFE_MODE` | Zot safe-mode setting, if implemented. |
| `CLAUDE_CODE_SIMPLE` | `ZOT_SIMPLE` | Zot minimal-mode setting, if implemented. |
| `CLAUDE_PLUGIN_OPTION_<KEY>` | `ZOT_EXTENSION_OPTION_<KEY>` | Explicit option for the zot extension that owns the hook. |

The `ZOT_*` names are the native contract. They MUST NOT be created by blind string replacement. Each variable needs a defined source, scope, and unset behavior.

The extension SHOULD set both names only when the meanings are equivalent. For example, a zot session can set `ZOT_SESSION_ID`, but it MUST NOT set `CLAUDE_CODE_SESSION_ID` unless zot has confirmed that its session identifier has the same lifecycle and format.

Zot-native variables MUST take precedence over stale inherited `ZOT_*` values for variables owned by the extension. Unowned variables MUST remain inherited. The environment builder MUST remove duplicate owned keys before it adds the current values.

Do not create a `ZOT_*` alias for `TRACEPARENT`. It is a standard W3C trace variable, not a Claude-specific variable. Preserve it and apply zot trace policy separately.

**Confidence:** High for the naming policy. The exact source and lifecycle for each variable require verification against the zot SDK before implementation.

### 5. Environment persistence

`CLAUDE_ENV_FILE` is available to `SessionStart`, `Setup`, `CwdChanged`, and `FileChanged` hooks. Claude Code reads shell `export` statements written to this file and applies them to later Bash commands.

Zot does not currently define the same session environment lifecycle. The extension MUST NOT set `CLAUDE_ENV_FILE`, source a caller-provided file, or apply shell exports without a separate design. Sourcing a file would execute arbitrary shell code and would change the environment of later hooks in a way that is not yet defined.

A later implementation can support a safe subset:

- create a private file per zot session;
- accept only `export NAME=value` lines;
- reject command substitution, redirects, and other shell syntax;
- apply changes only to later hook subprocesses;
- never change the zot process environment;
- define behavior on malformed lines and session end.

**Confidence:** High for Claude behavior. High that zot does not currently implement this contract.

### 6. Plugin configuration variables

Claude plugin hooks can use:

```text
${user_config.<key>}
CLAUDE_PLUGIN_OPTION_<KEY>
```

Exec-form hooks can substitute `${user_config.<key>}` into `command` and `args`. Shell-form plugin hooks must read `CLAUDE_PLUGIN_OPTION_<KEY>` instead. Current Claude versions reject `${user_config.*}` in shell-form plugin hooks.

Zot extension hook files do not currently have a Claude plugin option model. The extension MUST NOT interpret `${user_config.*}` or create `CLAUDE_PLUGIN_OPTION_*` values from arbitrary JSON. Doing so would require a source-specific configuration model and a safe value policy.

**Confidence:** High.

### 7. HTTP hook variables

Claude HTTP hooks support `$VAR_NAME` and `${VAR_NAME}` interpolation in header values. The `allowedEnvVars` field controls which variables can be resolved. The settings-level `httpHookAllowedEnvVars` setting can also define an allowlist.

This behavior does not apply to command hooks. The extension currently runs command hooks only. It MUST NOT add `allowedEnvVars` behavior to command parsing. If HTTP handlers are added later, header interpolation MUST use an explicit allowlist and MUST NOT resolve every inherited variable.

**Confidence:** High.

### 8. Shell interpolation

Claude expands path placeholders before command execution. Normal shell variables such as `$PATH` and `${CLAUDE_PROJECT_DIR}` are expanded by the selected shell.

The current zot implementation runs:

- `sh -c` on Unix;
- `cmd.exe /d /s /c` on Windows.

The extension MUST NOT pre-expand shell syntax in Go. Pre-expansion would change quoting, escaping, command substitution, globbing, and platform behavior.

The current parser stores `args` and `shell`, but the runtime does not yet implement those fields. Environment compatibility MUST remain separate from that work.

**Confidence:** High.

## Complete compatibility plan

### Phase 1: Preserve the inherited environment

Add a single environment builder at the command execution boundary.

```go
type HookEnvironment struct {
    Variables []string
}

func buildHookEnvironment(parent []string, cwd string) []string
```

The builder MUST:

1. Start with `os.Environ()`.
2. Remove existing entries for variables owned by the extension.
3. Add one `ZOT_PROJECT_DIR=<cwd>` entry and one matching `CLAUDE_PROJECT_DIR=<cwd>` entry.
4. Preserve every other entry, including empty values.
5. Avoid duplicate keys.

The first owned-key set is:

```text
ZOT_PROJECT_DIR
CLAUDE_PROJECT_DIR
```

Do not add `CLAUDE_CODE_REMOTE=false`, `ZOT_REMOTE=false`, `CLAUDE_ENV_FILE`, `ZOT_ENV_FILE`, or guessed session values.

Set the result on `exec.Cmd.Env`. This makes the behavior explicit and testable while preserving the current inherited environment.

### Phase 2: Define the effective directory contract

Create one value for each hook invocation:

```go
type HookContext struct {
    ProjectDir string
    EventCWD   string
}
```

Until zot provides a separate session-root value:

- use the host project CWD for `ZOT_PROJECT_DIR` and `CLAUDE_PROJECT_DIR`;
- use the event CWD for `command.Dir` and payload `cwd` when available;
- use the project CWD as a fallback.

Do not silently use `os.Getwd()`. The extension can run from a different directory than the active zot project.

Add tests that compare:

```sh
pwd
printf '%s\n' "$CLAUDE_PROJECT_DIR"
```

The test must document whether these values are expected to match for the current zot model.

### Phase 3: Add event-derived variables only with explicit mappings

Create a mapping table in code. Each entry must include:

- variable name;
- source field;
- event scope;
- behavior when missing;
- version or capability requirement.

Example:

```go
type DerivedEnvironmentValue struct {
    Name      string
    Value     string
    Available bool
}
```

Rules:

- Set a variable only when the source value has the same meaning as the Claude value.
- Leave the variable unset when the value is unknown.
- Do not use an empty string as a false value unless Claude uses an empty string for the same meaning.
- Keep inherited values when the derived value is unavailable.
- Remove an inherited value only when zot owns and replaces that variable by contract.

Candidate variables for a later phase:

- `ZOT_SESSION_ID` from the zot session ID.
- `ZOT_EFFORT` from event effort data.
- `ZOT_CHILD_SESSION` from an explicit child-agent event.

Add the matching `CLAUDE_*` alias only after the semantic mapping is verified.

Do not map `CLAUDE_PID` to the zot PID. The process identity differs.

### Phase 5: Support plugin-backed extension sources

The current extension discovers hook files under:

```text
$ZOT_HOME/extensions/<extension-name>/hooks/*.json
```

If these files need Claude plugin-style variables, define a zot-specific source model first:

```go
type HookSource struct {
    Path       string
    Owner      string
    Root       string
    DataDir    string
    Options    map[string]string
}
```

Only then decide whether to expose:

```text
CLAUDE_PLUGIN_ROOT
CLAUDE_PLUGIN_DATA
CLAUDE_PLUGIN_OPTION_<KEY>
```

Recommended mapping:

- `ZOT_EXTENSION_ROOT` → extension owner directory;
- `ZOT_EXTENSION_DATA` → a persistent owner data directory under `$ZOT_HOME`;
- `ZOT_EXTENSION_OPTION_<KEY>` → explicit, validated source options only;
- add Claude plugin aliases only when the source is a true Claude-compatible plugin source.

Do not use this mapping for ordinary `.claude/settings.json` hooks. A project hook is not automatically a plugin hook.

### Phase 6: Add safe placeholder expansion

Support only placeholders that have a defined source context:

```text
${CLAUDE_PROJECT_DIR}
```

For command hooks with `args`:

- expand the placeholder in `command` and each argument as a plain string;
- pass each argument without shell tokenization.

For shell-form hooks:

- keep shell variable expansion in the shell;
- quote paths in generated documentation;
- do not parse or rewrite arbitrary shell syntax.

Do not implement `${user_config.*}` until plugin options exist.

On Windows, define the shell behavior before adding placeholder rewriting. The Claude PowerShell rules differ from `cmd.exe` rules. A compatible implementation may need `shell` support before it can support PowerShell placeholder syntax correctly.

### Phase 7: Model environment persistence

Only implement `CLAUDE_ENV_FILE` after the lifecycle is defined.

Required decisions:

1. Which zot events can write environment state?
2. Does state apply to later hooks only, or also to tool execution?
3. Is the file private to the session and project?
4. Which syntax is accepted?
5. How are malformed lines handled?
6. How is state cleared after `CwdChanged` or session end?
7. Does state survive a zot restart?

Recommended first version:

- support only `export NAME=value`;
- parse with a shell-aware but non-executing parser;
- reject command substitution and shell operators;
- store values in a per-session map;
- pass the map to later hooks;
- do not source the file;
- do not change the zot process environment.

### Phase 8: Document the compatibility boundary

Add a README section with four tables:

1. **Supported by zot:** inherited environment, `ZOT_PROJECT_DIR`, and the matching `CLAUDE_PROJECT_DIR` alias.
2. **Zot-native variables:** every `ZOT_*` value that has a verified zot source.
3. **Claude aliases:** matching `CLAUDE_*` values only when the semantics are identical.
4. **Preserved when inherited:** variables that zot cannot create or verify.

Do not synthesize plugin, remote, messaging, PID, or environment-persistence variables without a zot mapping.

State clearly that command hooks read process variables through the selected shell. State that event data arrives through JSON on stdin, not through environment variables.

### Phase 9: Test the full matrix

#### Environment builder tests

- Preserve custom variables.
- Preserve `ZOT_*` variables.
- Preserve inherited Claude variables that zot does not own.
- Replace one stale `CLAUDE_PROJECT_DIR` entry.
- Do not create duplicate keys.
- Preserve empty values for non-owned variables.
- Add no guessed variables.

#### Command execution tests

- Expand `$CLAUDE_PROJECT_DIR` in `sh`.
- Expand `${CLAUDE_PROJECT_DIR}` in `sh`.
- Keep shell syntax unchanged.
- Confirm command `Dir` and payload `cwd` use the effective directory.
- Test a Windows command path when Windows CI is available.

#### Source-context tests

- Project hooks do not receive plugin-root variables unless explicitly inherited.
- Extension-owned hook files receive plugin variables only after the source mapping is implemented.
- Unknown plugin options remain unset.

#### Lifecycle tests

- `PreToolUse` sees the environment.
- `SessionStart` sees the environment.
- `Stop` sees the environment.
- `Notification` sees the environment.
- Future asynchronous hooks use the same builder.

#### Security tests

- Do not execute `CLAUDE_ENV_FILE` contents during environment construction.
- Do not expand arbitrary `${user_config.*}` values.
- Do not interpolate HTTP header variables without an allowlist.
- Do not leak messaging tokens into logs.

## Recommended implementation order

1. Inventory the zot SDK fields and confirm the source and lifecycle for each proposed `ZOT_*` value.
2. Add `buildHookEnvironment` and unit tests.
3. Set `exec.Cmd.Env` in `runHook`.
4. Export `ZOT_PROJECT_DIR` and the matching `CLAUDE_PROJECT_DIR` alias from the effective project directory.
5. Add tests for inherited variables, stale-value replacement, and duplicate-key removal.
6. Add end-to-end environment fixtures for both `ZOT_*` and `CLAUDE_*` names.
7. Add event-derived `ZOT_*` variables only after zot source fields are verified.
8. Add source context for extension-owned hook files.
9. Add `ZOT_EXTENSION_ROOT`, `ZOT_EXTENSION_DATA`, and option variables only for extension-owned hooks.
10. Add safe command `args` and `shell` support.
11. Add plugin option compatibility only if a real extension option source exists.
12. Add `ZOT_ENV_FILE` and `CLAUDE_ENV_FILE` as a separate persistence design.
13. Update the README and compatibility matrix.

## Risks and unknowns

- Zot may expose values that look like Claude values but have different lifecycle semantics.
- `CLAUDE_PROJECT_DIR` may need a separate session-root field when zot supports worktrees.
- Shell behavior differs between Unix, Windows `cmd.exe`, and PowerShell.
- Claude adds variables and changes version behavior over time.
- Inherited credentials can be sensitive. The extension should preserve them for compatibility but must not print them.
- Plugin variables must remain source-scoped. A hook file under a generic extension directory is not automatically a Claude plugin hook.

## Sources

All sources were checked on 2026-09-15.

1. [Claude Code Hooks reference](https://code.claude.com/docs/en/hooks) — primary source for hook process environments, path placeholders, shell behavior, common input, `CLAUDE_ENV_FILE`, and HTTP allowlists.
2. [Claude Code Environment Variables](https://code.claude.com/docs/en/env-vars) — primary source for exported variables, environment scrubbing, remote values, session values, and version caveats.
3. [Claude Code Plugin Reference](https://code.claude.com/docs/en/plugins-reference) — primary source for plugin paths, persistent data, `user_config`, and plugin option variables.
4. [Claude Code Settings](https://code.claude.com/docs/en/settings) — primary source for settings environment values and HTTP environment allowlists.
5. [Claude Code Settings Reference](https://code.claude.com/docs/en/settings-reference) — primary source for settings fields such as `httpHookAllowedEnvVars`.
6. [Claude Code Skills](https://code.claude.com/docs/en/skills) — source for skill-only placeholder behavior.
7. [Claude Code Subagents](https://code.claude.com/docs/en/sub-agents) — source for subagent hook context and agent fields.

## Summary

The safe compatibility boundary is:

- inherit the complete parent environment;
- set native `ZOT_*` variables for values that zot owns;
- set a matching `CLAUDE_*` alias only when the meanings are identical;
- replace stale values for extension-owned variables without creating duplicates;
- preserve unknown Claude variables without inventing values;
- keep event data in stdin JSON;
- keep plugin variables source-scoped;
- treat environment persistence as a separate feature;
- test both environment namespaces and shell expansion at the command boundary.

This plan supports common Claude hooks now and leaves a clear path for session, plugin, remote, and persistence features later.
