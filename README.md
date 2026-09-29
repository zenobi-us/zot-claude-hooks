# zot-cluade-hooks

`zot-cluade-hooks` is a [zot](https://github.com/patriceckhart/zot) extension that runs command hooks defined in Claude-style JSON settings files. It lets existing `PreToolUse` hooks participate in zot tool calls and forwards the lifecycle events that zot currently exposes.

> The repository and extension name intentionally use `cluade` for compatibility with the existing package and manifest names.

## Start here: run a hook in five minutes

### 1. Install the extension

```sh
zot ext install https://github.com/zenobi-us/zot-cluade-hooks
```


### 2. Add a hook configuration

Create `.claude/settings.json` in the project where you run zot:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "^bash$",
        "hooks": [
          {
            "type": "command",
            "command": "sh .zot/hooks/check-bash.sh"
          }
        ]
      }
    ]
  }
}
```
Zot also looks for hooks in other places, see [Discovery paths](#discovery-paths) below.


Create the command referenced above at `.zot/hooks/check-bash.sh`:

```sh
#!/bin/sh
set -eu

payload=$(cat)
printf 'checking %s\\n' "$payload" >&2

# Exit 0 to allow the tool call.
exit 0
```

Make it executable if you invoke it directly, or leave it non-executable when calling it through `sh`:

```sh
chmod +x .zot/hooks/check-bash.sh
```

The hook receives one JSON object on standard input. For a `PreToolUse` hook, it includes `hook_event_name`, `cwd`, `tool_name`, and `tool_input`.

### 3. Start zot with the extension

Run zot from the project directory:

```sh
zot --ext /path/to/zot-claude-hooks
```

Ask zot to use the `bash` tool. The hook runs before the tool call. Because the matcher is `^bash$`, calls to other tools are not matched.

### 4. Try blocking a tool call

Change the script to return exit status `2`:

```sh
#!/bin/sh
set -eu

printf '%s\\n' '{"decision":"block","reason":"bash is disabled for this project"}'
exit 2
```

The tool call is blocked and zot receives the supplied reason. A JSON response with `"decision":"block"` also blocks the call; exit status `2` is the command-level blocking signal.

## How-to guides

### Configure a lifecycle hook

Hooks use the Claude-style JSON settings shape: the `hooks` object maps an
**event name** to matcher groups. Each group has a regular-expression
`matcher` and one or more command handlers.

```json
{
  "hooks": {
    "TurnStart": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "sh .zot/hooks/turn-start.sh",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
```

Matchers apply to tool names for tool events such as `PreToolUse`,
`PostToolUse`, and `PermissionRequest`. For lifecycle events without a tool
name, use `.*`. Commands receive one JSON object on stdin.

### Manage hooks from zot

The extension registers a `/hooks` slash command with these forms:

```text
/hooks                    # open the interactive hook panel
/hooks locations          # show every valid discovery location
/hooks add                # open the panel directly in add mode
/hooks add PreToolUse sh .zot/hooks/check-bash.sh
```

The hook panel lists active hooks merged from all discovered files. Press
`a` to enter the add flow. The hook-event field provides a filtered dropdown:
type to narrow the choices, use Up/Down to select one, and press Enter. Then
type the command and press Enter again. Use Backspace to edit, Escape to
cancel, and `r` to reload the hook files. The panel updates after a successful
add.

`/hooks add <hook-event> <command>` remains available for scripted use. It
creates or updates the project-local `.zot/zot-cluade-hooks.json`, preserves
existing settings, adds the command to the event's default `.*` matcher group,
and reloads the hook list immediately after an add.

Run the extension's diagnostic command from the project directory when you
want tab-separated output for scripts:

```sh
./zot-cluade-hooks list
```

Each discovered hook is printed as an event, matcher, source file, and
command. This command uses the same discovery logic as the zot extension
process. Hooks loaded from another extension retain that extension as their
owner in `/hooks` output.

### Use a project-local hook without changing Claude settings

Create `.zot/zot-cluade-hooks.json`:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "sh .zot/hooks/on-session-start.sh",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
```

`timeout` is specified in seconds. The default is 10 seconds. The command is terminated when its timeout expires.

### Capture hook payloads for debugging

A hook can read its JSON payload from standard input and append it to a file:

```sh
#!/bin/sh
set -eu
mkdir -p .zot
cat >> .zot/hook-events.jsonl
```

Keep hook diagnostics on standard error. The extension uses standard output for its JSONL protocol when it is running under zot; arbitrary output from the extension process can break the protocol.

The Go implementation delegates protocol handling to zot's extension SDK. For protocol-level debugging, use zot's extension logs and tracing facilities. Hook payloads and command diagnostics remain available through the project-local files and stderr as shown above.

### Use a shared hook file

Set `ZOT_HOOKS_PATH` to a JSON file. Absolute paths are used as-is; relative paths resolve from the active project directory:

```sh
ZOT_HOOKS_PATH="$HOME/.config/zot/hooks.json" \
  zot --ext /path/to/zot-cluade-hooks
```

The extension also checks `$ZOT_HOME/zot-cluade-hooks.json` for user-level hooks. `$ZOT_HOME` follows zot's normal resolution: `ZOT_HOME`, then `$XDG_STATE_HOME/zot`, then `~/.local/state/zot` on Linux.

Installed extensions may contribute hook files in:

```text
$ZOT_HOME/extensions/<extension-name>/hooks/*.json
```

These files are loaded in deterministic extension-name and filename order. The current hook extension is excluded, and commands still run with the active project directory as their working directory. Use `/hooks locations` to inspect discovered extension hook files.

### Hook process environment

The hook environment has one compatibility rule: preserve the parent environment, then replace values owned by this extension. The extension never prints the complete environment or secret values.

| Variable | Behaviour |
| --- | --- |
| `ZOT_PROJECT_DIR` | Supported. The zot host project directory. |
| `CLAUDE_PROJECT_DIR` | Supported alias with the same value and lifecycle as `ZOT_PROJECT_DIR`. |
| `ZOT_SESSION_ID` | Supported only when zot provides a session ID. |
| `CLAUDE_CODE_SESSION_ID` | Set only as the matching alias for a verified `ZOT_SESSION_ID`. |
| `ZOT_CHILD_SESSION` | Supported only for `SubagentStart` and `SubagentStop` when zot provides `agent_id`. |
| `ZOT_EXTENSION_ROOT` | Set only for hooks loaded from an installed extension source. |
| `ZOT_ENV_FILE` | Set only for `SessionStart` hooks. It points to zot's private persistence file. |

All other variables are preserved when inherited, including `PATH`, `CI`, `TRACEPARENT`, custom `ZOT_*` values, and Claude variables that zot cannot reproduce. Unsupported variables are not synthesized. An unset variable stays unset; zot does not convert unknown state to an empty or false value. Owned variables replace stale inherited values and appear once.

Every hook inherits the complete environment of the zot process. The extension sets `ZOT_PROJECT_DIR` and `CLAUDE_PROJECT_DIR` to the host project directory, even when the hook process runs in an event directory.

Hooks loaded from an installed extension also receive `ZOT_EXTENSION_ROOT`, set to the owning extension directory. Project hooks do not receive this value unless it was already present in the inherited environment. Each hook gets the root of its own extension.

Lifecycle events expose the persisted or runtime-generated zot conversation ID as `ZOT_SESSION_ID`. `CLAUDE_CODE_SESSION_ID` is an alias with the same value and lifecycle. The extension replaces inherited values for both names when zot provides a verified session ID, and preserves inherited values when zot does not provide one. The extension reuses the current session ID when a later event omits it. A new `SessionStart` clears the previous cached session before it processes the new session. Cached session values are cleared after `session_end`. Subagent lifecycle events expose the SDK `agent_id` as `ZOT_CHILD_SESSION`; this value is available only for `SubagentStart` and `SubagentStop` hooks. The SDK does not expose an effort value, remote or bridge identity, messaging channel, child PID, or shell contract, so the extension leaves `ZOT_EFFORT`, remote, bridge, messaging, PID, and shell variables unset. It also does not add a Claude alias for `ZOT_CHILD_SESSION`.

When zot does not provide a session or child identity, the extension does not invent one. Existing inherited values remain unchanged. When zot provides a value, the extension replaces inherited values for that owned variable. Command hooks read these variables through the selected shell; `$ZOT_PROJECT_DIR` and `${CLAUDE_PROJECT_DIR}` are expanded by that shell. Event data is sent as JSON on standard input, and is not copied into environment variables. The extension never logs these environment values.

Zot extensions are not Claude plugins. Project hooks do not receive extension variables unless the caller already inherited them. Installed extension hooks receive only their own `ZOT_EXTENSION_ROOT`; zot does not invent `ZOT_EXTENSION_DATA` or `ZOT_EXTENSION_OPTION_<KEY>`. Remote, bridge, messaging, PID, effort, shell, and other Claude-specific values are preserved only when inherited. Zot does not create false values for them.

#### Safe hook environment persistence

`SessionStart` hooks may persist safe assignments for later hook processes. Zot
creates a private, session-scoped file and exposes its path as `ZOT_ENV_FILE`
only to those `SessionStart` processes. Zot does not set `CLAUDE_ENV_FILE`,
because it does not claim Claude's environment-file contract. This is a zot
extension contract, not a claim of Claude compatibility.

Write one assignment per line using `NAME=VALUE` or `export NAME=VALUE`. Names must use shell
identifier characters. Zot accepts values as data and rejects malformed lines
and shell execution characters, including command substitution, redirects,
pipelines, and separators. Zot parses the file after each `SessionStart` hook;
it never sources the file or executes its contents. Accepted values are added
to later hook processes, but never to zot's own process environment.

The state belongs to the active zot process, project, and session. A new
`SessionStart` clears the previous state. `SessionEnd` clears and removes the
file after its hooks run. A process restart starts with no persisted state.
Only `SessionStart` can write the state. `PreToolUse`, `Stop`, and
`Notification` can read accepted values from earlier `SessionStart` hooks.
Malformed assignments are ignored.

### Run the test suite

Run the Go tests:

```sh
go test ./...
```

The Go test suite can be run against the extension:

```sh
go test ./...
```

The end-to-end tests launch zot with temporary configuration and a local fake OpenAI-compatible provider. They verify allowing a matching hook, blocking with either exit status `2` or a JSON decision, and running the current `SessionStart`, `PreToolUse`, `Notification`, and `Stop` hooks. The environment fixture records selected hook payloads only; it does not record the complete environment or secrets. No external model provider or API credentials are required.

Manual fixtures are documented in [`fixtures/README.md`](fixtures/README.md).

## Configuration reference

### Configuration shape

Every discovered JSON file must contain a top-level `hooks` object. Each event maps to an array of hook groups. A group may provide a regular-expression `matcher` and contains an array of command hooks:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "^(bash|edit)$",
        "hooks": [
          {
            "type": "command",
            "command": "./.zot/hooks/validate.sh",
            "timeout": 10
          }
        ]
      }
    ]
  }
}
```

Supported command fields:

| Field | Required | Meaning |
| --- | --- | --- |
| `type` | yes | Must be `"command"`. Other hook types are ignored. |
| `command` | yes | Shell command executed with the project directory as its working directory. |
| `timeout` | no | Maximum runtime in seconds. Defaults to `10`; values are clamped to at least `0.1` seconds. |
| `matcher` | no | JavaScript regular expression matched against the zot tool name. Defaults to `.*`. |

Relative command paths and relative configuration paths resolve from the project directory. `ZOT_HOOKS_PATH` must identify a JSON file; absolute values are used as-is and relative values resolve from the project directory. Invalid regular expressions are logged and do not match. A malformed or unreadable configuration file is logged and skipped.

### Discovery paths

The active project directory is the project directory supplied to zot. Project
paths below are resolved beneath it; home and `$ZOT_HOME` paths are resolved
independently. Existing files are checked in this order:

1. `~/.claude/settings.json`
2. `$ZOT_HOME/zot-cluade-hooks.json`
3. `.claude/settings.json`
4. `.claude/settings.local.json`
5. `.zot/zot-cluade-hooks.json`
6. `.zot/zot-cluade-hooks.local.json`
7. `$ZOT_HOOKS_PATH` (when `ZOT_HOOKS_PATH` is set)
8. The shared hook directories listed below
9. `$ZOT_HOME/extensions/*/hooks/*.json` (excluding `zot-cluade-hooks` itself)

All valid definitions found at these paths are loaded. Discovery is additive:
later files do not replace earlier files. When multiple hooks match an event,
they execute in discovery order; files in a shared directory execute in
filename order. Use matchers and commands that make multiple matching hooks
safe. The same file is loaded only once when it is reachable through symlinks
or multiple discovery locations.

### Shared hook directories

After the fixed-file locations and `ZOT_HOOKS_PATH`, zot checks these directories in this order:

1. `$HOME/.agents/hooks/`
2. `$HOME/.claude/hooks/`
3. `$ZOT_HOME/hooks/`
4. `<project>/.agents/hooks/`
5. `<project>/.claude/hooks/`
6. `<project>/.zot/hooks/`

Zot reads direct regular `.json` files only. It does not recurse into
subdirectories. Symlinked directories and files are supported. Files are
sorted by filename and deduplicated by canonical path. Missing directories are
ignored. Use `/hooks locations` to see these candidate directory sources.

Installed extensions contribute direct regular `.json` files from
`$ZOT_HOME/extensions/<extension-name>/hooks/`. Nested directories are not
scanned, extension names and filenames are processed in sorted order, and the
`zot-cluade-hooks` extension itself is excluded.

`/hooks locations` displays candidate fixed paths and shared directories,
including locations that do not currently exist. It displays installed
extension entries only when their hook files are present. Use `/hooks` or
`./zot-cluade-hooks list` to inspect hooks that were actually parsed and
loaded.

### Hook events

The table is the complete command-hook compatibility surface. The first column
is the PascalCase name used in the configuration file and links to the detailed
reference below. `ClaudeHookName` identifies the corresponding Claude Code
concept; `ZotHookName` is the native lifecycle event emitted by zot.

| PascalCaseHookName | ClaudeHookName | ZotHookName | Description (influenced by facts in code) |
| --- | --- | --- | --- |
| [`SessionStart`](#sessionstart) | `SessionStart` | `session_start` | Runs when an active conversation opens. Initializes the hook session environment. |
| [`SessionEnd`](#sessionend) | `SessionEnd` | `session_end` | Runs when the conversation closes and receives the shutdown reason. |
| [`UserPromptSubmit`](#userpromptsubmit) | `UserPromptSubmit` | `user_prompt_submit` | Runs when processed user input is accepted, before append or queue. |
| [`TurnStart`](#turnstart) | — | `turn_start` | Zot-specific observational hook that runs when a model step begins. |
| [`Stop`](#stop) | `Stop` | `turn_end` | Runs when a model response attempt ends, before client tool execution. |
| [`PreToolUse`](#pretooluse) | `PreToolUse` | `tool_call` | Runs synchronously before a client tool call and can block it. |
| [`PostToolUse`](#posttooluse) | `PostToolUse` | `tool_result` | Runs after a client tool reaches a terminal result. |
| [`PermissionRequest`](#permissionrequest) | `PermissionRequest` | `tool_confirmation_requested`, `permission_decision` | Runs for an interactive approval request and the resulting decision. |
| [`Notification`](#notification) | `Notification` | `tool_call`, `assistant_message` | Runs for tool-call and visible-assistant-message notifications. |
| [`PreCompact`](#precompact) | `PreCompact` | `pre_compact` | Runs when context compaction begins. |
| [`PostCompact`](#postcompact) | — | `post_compact` | Runs when context compaction finishes, including failure. |
| [`SubagentStart`](#subagentstart) | `SubagentStart` | `subagent_start` | Runs when a local swarm runner starts or resumes. |
| [`SubagentStop`](#subagentstop) | `SubagentStop` | `subagent_stop` | Runs when a local swarm runner returns. |
| [`BeforeAgentStart`](#beforeagentstart) | — | `before_agent_start` interception | Runs after prompt assembly and before the first model call; may replace the complete system prompt. |

Every lifecycle payload also includes `session_id`, `cwd`, and `sequence`. Tool
payloads include tool identity and arguments; compaction payloads include
counts and estimates; subagent payloads include agent identity. Observational
hooks cannot change the event. `PreToolUse` can block a tool call, and
`BeforeAgentStart` can replace the system prompt using the zot-specific
interception protocol described below.

#### <a id="sessionstart"></a>SessionStart

Receives `hook_event_name`, `session_id`, `cwd`, and the event metadata. The
extension initializes its per-session environment before running these hooks.
A command's output is observational; exit status does not block session start.

#### <a id="sessionend"></a>SessionEnd

Receives the session shutdown `reason`. Output is observational and cannot
prevent session shutdown.

#### <a id="userpromptsubmit"></a>UserPromptSubmit

Receives the processed prompt as `message`, plus `queued` when the prompt was
queued and `image_count` when images were attached. This is not raw editor
keystroke input. Output is observational.

#### <a id="turnstart"></a>TurnStart

Zot-specific. Receives `step` and runs when a model step begins. Output is
observational; this command-hook integration does not block the turn. Native
zot extensions can use the separate `turn_start` interceptor.

#### <a id="stop"></a>Stop

Maps zot's `turn_end` event and receives `stop_reason`; an optional `error` is
included when the model response attempt failed. It runs before client tool
execution. Output is observational.

#### <a id="pretooluse"></a>PreToolUse

Runs synchronously before a client tool call. It receives `tool_name`,
`tool_input`, and `tool_id`. This is the one command-hook event whose result
can change execution:

- exit code `0` allows the call;
- exit code `2` blocks the call;
- JSON `{"decision":"block","reason":"..."}` blocks the call and supplies the reason;
- other exit codes, malformed JSON, and timeouts fail open.

A matcher is applied to `tool_name` for this event.

#### <a id="posttooluse"></a>PostToolUse

Runs after a client tool reaches a terminal outcome. It receives effective
`tool_input`, `tool_status`, `tool_executed`, and `tool_result`. Effective
arguments include accepted interceptor rewrites. Output is observational.

#### <a id="permissionrequest"></a>PermissionRequest

Runs for both `tool_confirmation_requested` and `permission_decision`.
Confirmation includes `tool_preview`; the later decision includes `decision`,
`source`, `stage`, and optional `reason`. It is emitted only when zot actually
requests or resolves an applicable approval. Output is observational.

#### <a id="notification"></a>Notification

Runs for `tool_call` and `assistant_message`. Tool notifications include the
proposed tool fields; assistant notifications include the visible text as
`message`. The command cannot rewrite or suppress the assistant message.

#### <a id="precompact"></a>PreCompact

Runs when compaction begins and receives `compaction_id`, `message_count`, and
`token_estimate`. Output is observational.

#### <a id="postcompact"></a>PostCompact

Runs when compaction finishes. It receives the same compaction metadata plus
`tool_status`-style `status` information and an optional `error`. Output is
observational, including when compaction fails.

#### <a id="subagentstart"></a>SubagentStart

Runs when a local swarm runner starts or resumes. Payloads include `agent_id`,
`agent_run_id`, and `agent_name`. Output is observational.

#### <a id="subagentstop"></a>SubagentStop

Runs when that swarm runner returns. Payloads include the same identity fields
plus status and optional error information. Output is observational.

#### <a id="beforeagentstart"></a>BeforeAgentStart

This is zot-specific; Claude Code has no equivalent hook name. It runs after
zot assembles the complete system prompt and before the first model call. The
payload includes `system_prompt`, `session_id`, `agent_run_id`, `cwd`,
`provider`, and `model`. The complete prompt can contain private context-file,
skill, and extension instructions. Do not enable this hook for untrusted
project configuration.

Return `{}` or `{"decision":"allow"}` to preserve the prompt. Return
`{"decision":"replace","system_prompt":"..."}` to replace it. An empty
string intentionally removes the prompt. Multiple matching hooks run serially;
each hook sees the previous hook's replacement. Invalid JSON, timeout, crashes,
and invalid fields fail open and preserve the current prompt. Hook execution is
capped at zot's five-second interception deadline.

### Hook examples

#### Replace the system prompt before the first model call

Definition in `.claude/settings.json`:

```json
{
  "hooks": {
    "BeforeAgentStart": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "sh .zot/hooks/add-project-rule.sh",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
```

`.zot/hooks/add-project-rule.sh`:

```sh
#!/bin/sh
set -eu

payload=$(cat)
prompt=$(printf '%s' "$payload" | jq -r '.system_prompt')
addition='Additional project rule: run the focused test before reporting completion.'

jq -n --arg prompt "$prompt" --arg addition "$addition" \
  '{decision: "replace", system_prompt: ($prompt + "\\n\\n" + $addition)}'
```

Return `{}` or `{"decision":"allow"}` to leave the prompt unchanged. An
empty `system_prompt` is a deliberate replacement that removes the prompt.

#### Block a dangerous tool with exit status

Definition in `.claude/settings.json`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "^bash$",
        "hooks": [
          {
            "type": "command",
            "command": "sh .zot/hooks/block-dangerous-bash.sh"
          }
        ]
      }
    ]
  }
}
```

`.zot/hooks/block-dangerous-bash.sh`:

```sh
#!/bin/sh
set -eu

payload=$(cat)
command=$(printf '%s' "$payload" | jq -r '.tool_input.command // ""')
case "$command" in
  *"rm -rf"*)
    printf '%s\\n' 'blocked by project policy' >&2
    exit 2
    ;;
esac

exit 0
```

#### Block a tool with a JSON response

The same `PreToolUse` command can return a structured reason:

```sh
#!/bin/sh
set -eu

payload=$(cat)
command=$(printf '%s' "$payload" | jq -r '.tool_input.command // ""')
if printf '%s' "$command" | grep -q 'curl.*production'; then
  printf '%s\\n' '{"decision":"block","reason":"production network access is disabled"}'
  exit 0
fi

printf '%s\\n' '{"decision":"allow"}'
exit 0
```

`decision: "block"` is acted on by `PreToolUse`; `decision: "allow"` is
currently informational and the call proceeds. For all other hook events,
JSON and exit status are observational and do not alter zot behavior.

#### Audit an observational event

Definition:

```json
{
  "hooks": {
    "TurnStart": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "sh .zot/hooks/audit-turn.sh"
          }
        ]
      }
    ]
  }
}
```

Script:

```sh
#!/bin/sh
set -eu

jq -c '{event: .hook_event_name, step, session_id, cwd}' >> .zot/turns.jsonl
# Exit status and JSON output are ignored for observational events.
printf '%s\\n' '{"recorded":true}'
exit 0
```

### Command results and failure behaviour

- Exit status `0` allows a `PreToolUse` command to continue.
- Exit status `2` blocks the tool call.
- JSON output containing `{"decision":"block","reason":"..."}` blocks the tool call and supplies the reason.
- Plain-text output is not a structured decision.
- Timeouts and invalid responses fail open, except for a valid exit status `2`.
- Hook standard error is forwarded to the extension diagnostics.

## How it works

The extension is a JSONL protocol process described by `extension.json`. When zot starts it, the process sends a `hello` frame, receives `hello_ack`, loads hook files using the working directory, subscribes to the events available in the current zot protocol, and reports `ready`.

For a tool call, zot sends an interception frame. The extension selects `PreToolUse` commands whose matcher matches the tool name, passes each command the Claude-style payload, and returns an interception response. Other subscribed events are forwarded asynchronously to matching commands.

This repository implements the protocol client, hook discovery, synchronous
`PreToolUse`, and command-hook forwarding for the lifecycle events listed
above. Claude hook names are a compatibility layer over zot events; they are
not a complete substitute for zot's native interception protocol.

Planned hook and lifecycle support is tracked in [`PLAN.md`](PLAN.md):

| Planned capability | Current status |
| --- | --- |
| `user_prompt_submit` | Supported when zot emits the event. |
| `PostToolUse` / `tool_result` | Supported when zot emits tool-result events. |
| Final tool status | Supported through the event payload status and executed fields. |
| `session_end` | Supported. |
| `pre_compact` and `post_compact` | Supported. |
| `subagent_start` and `subagent_stop` | Supported. |
| `permission_decision` | Supported observationally. |
| Prompt replacement or synchronous prompt blocking | Waiting for zot semantics. |

The plan also includes event-ordering tests and reconsidering fail-open behaviour if zot adds a fail-closed policy mode.

Hook files execute arbitrary shell commands from user and project configuration. Review configuration before enabling it, especially in untrusted repositories. The extension intentionally does not add a trust prompt yet; that remains an open design decision.

## Repository layout

- [`extension.json`](extension.json): zot extension manifest.
- [`main.go`](main.go): Go SDK extension, hook discovery, command runner, and `list` diagnostic command.
- [`fixtures/README.md`](fixtures/README.md): manual fixture walkthroughs.
- [`test/e2e/runner.test.ts`](test/e2e/runner.test.ts): end-to-end coverage.
- [`PLAN.md`](PLAN.md): current capabilities, limitations, and future work.
- [`go.mod`](go.mod): Go module and zot extension SDK dependency.

## Project status

This is an early `0.1.0` extension scaffold. The current behaviour is defined by the implementation and tests in this repository; use [`PLAN.md`](PLAN.md) for the boundary between available functionality and planned work.
