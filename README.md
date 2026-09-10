# zot-cluade-hooks

`zot-cluade-hooks` is a [zot](https://github.com/patriceckhart/zot) extension that runs command hooks defined in Claude-style JSON settings files. It lets existing `PreToolUse` hooks participate in zot tool calls and forwards the lifecycle events that zot currently exposes.

> The repository and extension name intentionally use `cluade` for compatibility with the existing package and manifest names.

## Start here: run a hook in five minutes

### 1. Prerequisites

You need:

- Go 1.25 or a prebuilt `zot-cluade-hooks` binary.
- `zot`, available on your `PATH`.
- A project directory in which zot can run.

Build the extension from this repository:

```sh
go build -o zot-cluade-hooks .
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

Set `ZOT_HOOKS_PATH` to a JSON file or a path relative to the project directory:

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

### Run the test suite

Run the Go tests:

```sh
go test ./...
```

The existing zot integration tests can be run against the compiled extension:

```sh
bun test test/e2e
```

The end-to-end tests launch zot with temporary configuration and a fake provider. They verify allowing a matching hook and blocking with either exit status `2` or a JSON decision.

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

Relative command paths and relative configuration paths resolve from the project directory. Invalid regular expressions are logged and do not match. A malformed or unreadable configuration file is logged and skipped.

### Discovery paths

Existing files are checked in this order:

1. `~/.claude/settings.json`
2. `$ZOT_HOME/zot-cluade-hooks.json`
3. `.claude/settings.json`
4. `.claude/settings.local.json`
5. `.zot/zot-cluade-hooks.json`
6. `.zot/zot-cluade-hooks.local.json`
7. `$ZOT_HOOKS_PATH` (when `ZOT_HOOKS_PATH` is set)
8. `$ZOT_HOME/extensions/*/hooks/*.json` (excluding `zot-cluade-hooks` itself)

All valid definitions found at these paths are loaded. Later files do not automatically replace earlier files, so use matchers and commands that make multiple matching hooks safe.

### Hook events

The compatibility matrix below shows how zot lifecycle events map to hook names and whether this extension currently supports them:

| zot event | Claude hook | zot-claude-hooks support |
| --- | --- | --- |
| `session_start` | `SessionStart` | Supported |
| `user_prompt_submit` | `UserPromptSubmit` | Not supported: zot does not currently expose this event to extensions. |
| `turn_start` | — | Available in zot, but there is no direct hook equivalent currently implemented. |
| `tool_call` | `PreToolUse` | Supported synchronously; exit `2` or JSON `decision: "block"` prevents the call. |
| `tool_result` | `PostToolUse` | Not supported: zot does not currently expose post-tool results to extensions. |
| `tool_confirmation_requested` | `PermissionRequest` | Observable in zot, but not currently mapped to a hook. |
| `permission_decision` | `PermissionRequest` | Not supported: zot does not currently expose the final permission decision. |
| `turn_end` | `Stop` | Supported. Runs when a turn ends. |
| `assistant_message` | `Notification` | Partially supported through the currently available notification-like events. |
| `session_end` | `SessionEnd` | Not supported: zot does not currently expose session shutdown to extensions. |
| `pre_compact` | `PreCompact` | Not supported. |
| `post_compact` | — | Not supported. |
| `subagent_start` | `SubagentStart` | Not supported. |
| `subagent_stop` | `SubagentStop` | Not supported. |

The current implementation supports these Claude-style event names:

| Hook event | zot source | Behaviour |
| --- | --- | --- |
| `PreToolUse` | synchronous `tool_call` interception | Runs before the tool. Exit `2` or JSON `decision: "block"` prevents the call. |
| `SessionStart` | `session_start` | Runs when the extension session starts. |
| `Stop` | `turn_end` | Runs when a turn ends. |
| `Notification` | `tool_call` and `assistant_message` events | Runs for the currently available notification-like events. |

`PreToolUse` receives a payload such as:

```json
{
  "hook_event_name": "PreToolUse",
  "cwd": "/path/to/project",
  "tool_name": "bash",
  "tool_input": { "command": "printf hello" }
}
```

Asynchronous event hooks receive the zot event frame with `hook_event_name` added. Their output does not change the event; use them for auditing, notifications, or other side effects.

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

This repository currently implements the protocol client, hook discovery, `PreToolUse`, and the event forwarding listed above. It does not yet implement every Claude hook event.

Planned hook and lifecycle support is tracked in [`PLAN.md`](PLAN.md):

| Planned capability | Current status |
| --- | --- |
| `user_prompt_submit` | Waiting for zot to expose the event. |
| `PostToolUse` / `tool_result` | Waiting for zot tool-result events. |
| Final tool status | Planned distinction between completed, failed, blocked, cancelled, and timed-out calls. |
| `session_end` | Planned. |
| `pre_compact` and `post_compact` | Planned. |
| `subagent_start` and `subagent_stop` | Planned. |
| `permission_decision` | Planned. |
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
