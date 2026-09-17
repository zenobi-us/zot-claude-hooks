# Hook environment compatibility

Status: ready-for-agent

## Problem Statement

Claude Code command hooks can read the parent process environment and several Claude-specific variables. They can also use path placeholders in hook commands.

The zot extension currently lets hook commands inherit the zot process environment, but it does not define a native `ZOT_*` environment contract. It does not replace stale project variables, add zot-specific values, or document which Claude values it can reproduce.

This creates several problems:

- A hook can see a stale `CLAUDE_PROJECT_DIR` from the caller environment.
- A hook cannot depend on a stable zot-native variable namespace.
- The extension may invent values that do not have the same meaning as Claude values.
- Plugin, remote-session, messaging, and environment-persistence variables have no clear boundary.
- Tests cannot verify environment behavior at one stable execution seam.

The extension needs a compatibility contract that preserves arbitrary inherited variables, adds verified `ZOT_*` values, and adds `CLAUDE_*` aliases only when zot has the same semantic value.

## Solution

Build the environment at the hook execution seam.

The command runner will call one environment builder before it starts a hook process. The builder will:

- copy the complete parent environment;
- remove duplicate entries for variables owned by the extension;
- add verified `ZOT_*` variables;
- add matching `CLAUDE_*` aliases only when their meanings match;
- preserve all other inherited variables, including empty values;
- avoid logging secrets or full environment contents.

The first owned values will be:

- `ZOT_PROJECT_DIR`;
- `CLAUDE_PROJECT_DIR`.

Both values will use the effective zot project directory. The command working directory and hook input `cwd` will follow the effective event directory when zot provides one. The implementation must document the difference between the session project directory and the active event directory when zot exposes both.

The design will define a source and lifecycle for every later `ZOT_*` value before the extension exports it. The extension will not set a guessed value. It will preserve an inherited Claude variable when zot cannot verify an equivalent value.

The implementation will use one primary test seam:

```text
runHook -> buildHookEnvironment -> exec.Cmd.Env
```

The tests will observe hook behavior. They will not test private map layout or helper call order.

## User Stories

1. As a hook author, I want my hook to keep access to inherited environment variables, so that existing scripts continue to work.
2. As a hook author, I want `ZOT_PROJECT_DIR`, so that I can find the zot project directory without relying on a Claude-specific name.
3. As a hook author, I want `CLAUDE_PROJECT_DIR` when it has the same meaning as `ZOT_PROJECT_DIR`, so that existing Claude hook scripts work under zot.
4. As a hook author, I want the project variable to contain the effective project directory, so that a hook does not use a stale caller value.
5. As a hook author, I want one value for each owned environment key, so that shell behavior does not depend on duplicate environment entries.
6. As a hook author, I want `$ZOT_PROJECT_DIR` and `${ZOT_PROJECT_DIR}` to work through the selected shell, so that normal shell scripts need no special zot syntax.
7. As a hook author, I want `$CLAUDE_PROJECT_DIR` and `${CLAUDE_PROJECT_DIR}` to work through the selected shell, so that Claude hook scripts remain portable to zot.
8. As a hook author, I want custom variables such as `CI`, `PATH`, and `MY_HOOK_MODE` to remain available, so that my hook keeps its existing configuration.
9. As a hook author, I want inherited `ZOT_*` values to remain available when zot does not own them, so that wrapper scripts can provide custom values.
10. As a hook author, I want inherited Claude variables to remain available when zot cannot reproduce them, so that the extension does not remove caller-provided compatibility values.
11. As a hook author, I want unset values to stay unset, so that a hook can distinguish unknown state from a false state.
12. As a hook author, I want empty inherited values to remain empty unless zot owns the key, so that existing shell checks keep their meaning.
13. As a hook author, I want `ZOT_SESSION_ID` when zot exposes a session identifier with a defined lifecycle, so that I can correlate hook calls.
14. As a hook author, I want a matching `CLAUDE_CODE_SESSION_ID` alias only when the zot session identifier has the same meaning, so that the alias does not create false compatibility.
15. As a hook author, I want `ZOT_EFFORT` when the zot event exposes a matching effort value, so that hooks can adapt to the active effort level.
16. As a hook author, I want `ZOT_CHILD_SESSION` when zot identifies a child or agent session, so that hooks can distinguish child work.
17. As a hook author, I want remote-session variables only when zot exposes remote state, so that a missing value does not become a false local value.
18. As a hook author, I want messaging variables only when zot exposes a matching socket and token contract, so that hooks do not receive unusable or unsafe values.
19. As a hook author, I want messaging tokens to stay out of logs, so that hook diagnostics do not disclose credentials.
20. As an extension author, I want `ZOT_EXTENSION_ROOT` for extension-owned hook files, so that a hook can locate immutable extension files.
21. As an extension author, I want `ZOT_EXTENSION_DATA` for extension-owned hook files, so that a hook can store data outside the extension installation directory.
22. As an extension author, I want `ZOT_EXTENSION_OPTION_<KEY>` for explicit extension options, so that shell hooks can read validated configuration values.
23. As an extension author, I want extension variables to be source-scoped, so that project hooks do not receive values that belong to another extension.
24. As a hook author, I want the extension to preserve `TRACEPARENT`, so that standard trace propagation can continue.
25. As a hook author, I want the extension to avoid renaming standard variables, so that `TRACEPARENT` remains compatible with tracing tools.
26. As a hook author, I want `ZOT_ENV_FILE` only when zot implements environment persistence, so that the variable does not point to an inactive or unsafe file.
27. As a hook author, I want environment persistence to apply only to later hook processes, so that a hook cannot silently change the zot process environment.
28. As a hook author, I want environment persistence to reject shell execution syntax, so that a persisted environment file cannot execute arbitrary commands.
29. As a hook author, I want the same environment rules for `PreToolUse`, `SessionStart`, `Stop`, and `Notification`, so that hook behavior does not depend on the event type.
30. As a maintainer, I want one environment builder, so that future hook paths use the same compatibility rules.
31. As a maintainer, I want tests at the command execution seam, so that tests verify behavior rather than implementation details.
32. As a maintainer, I want the README to list supported, preserved, and unsupported variables, so that users know the compatibility boundary.
33. As a maintainer, I want the implementation to tolerate new Claude variables, so that future Claude changes do not remove unrelated inherited values.
34. As a maintainer, I want the implementation to avoid blind `CLAUDE_*` to `ZOT_*` conversion, so that names do not imply unsupported semantics.
35. As a maintainer, I want the environment builder to replace stale owned values, so that wrapper environments cannot override values that zot owns.

## Implementation Decisions

- Use one environment-building boundary before command execution.
- Start from the parent process environment. Do not use a narrow allowlist.
- Model owned values separately from inherited values.
- Remove all existing entries for an owned key before adding the current value.
- Export `ZOT_PROJECT_DIR` as the native project variable.
- Export `CLAUDE_PROJECT_DIR` as a compatibility alias because both names refer to the same value in the first implementation.
- Do not export `CLAUDE_CODE_REMOTE=false`, `ZOT_REMOTE=false`, or another guessed state.
- Preserve inherited values for variables that zot does not own.
- Treat an unset value as unknown. Do not convert unknown state to an empty or false value.
- Keep stdin JSON separate from process environment variables. Event fields such as session data, tool data, and active `cwd` belong in the JSON payload.
- Use the effective host or event directory. Do not use the extension process current directory as a substitute.
- Keep shell expansion in the selected shell. Do not pre-expand `$NAME`, `${NAME}`, quoting, pipes, globbing, or command substitution in Go.
- Add native variables only after the zot SDK provides a source with the same meaning and lifecycle.
- Use these planned native names when zot supports the matching concepts:
  - `ZOT_SESSION_ID`
  - `ZOT_EFFORT`
  - `ZOT_REMOTE`
  - `ZOT_REMOTE_SESSION_ID`
  - `ZOT_BRIDGE_SESSION_ID`
  - `ZOT_MESSAGING_SOCKET`
  - `ZOT_MESSAGING_TOKEN`
  - `ZOT_PID`
  - `ZOTCODE`
  - `ZOT_CHILD_SESSION`
  - `ZOT_SHELL_PREFIX`
  - `ZOT_SUBPROCESS_ENV_SCRUB`
  - `ZOT_SESSIONEND_HOOKS_TIMEOUT_MS`
  - `ZOT_DEBUG_LOG_LEVEL`
  - `ZOT_PROPAGATE_TRACEPARENT`
  - `ZOT_SAFE_MODE`
  - `ZOT_SIMPLE`
- Add a matching Claude alias only when the zot value has the same meaning, lifecycle, and format.
- Do not create a `ZOT_*` alias for `TRACEPARENT`. Preserve the W3C variable unchanged.
- Keep extension variables separate from project variables:
  - `ZOT_EXTENSION_ROOT`
  - `ZOT_EXTENSION_DATA`
  - `ZOT_EXTENSION_OPTION_<KEY>`
- Set extension variables only for hook files that have an explicit extension source context.
- Do not treat an ordinary project settings file as an extension or plugin source.
- Do not implement `${user_config.*}` until a validated extension option source exists.
- Do not implement Claude plugin variables by guessing from an extension directory name.
- Treat `ZOT_ENV_FILE` and `CLAUDE_ENV_FILE` as a separate feature. Do not create or source a file in this work unless the lifecycle and parser rules are defined.
- Never print complete environments or sensitive values such as messaging tokens.
- Keep the existing command runner behavior separate from this environment work. Do not add `args`, `if`, `async`, `asyncRewake`, or `shell` execution support as an implicit side effect.

## Testing Decisions

Tests must observe external hook behavior at the highest practical seam. They must verify what the hook process receives. They must not inspect private environment-builder maps or require a specific helper call order.

Test the environment builder through the command runner or a direct public execution seam with these cases:

- Preserve a custom inherited variable.
- Preserve `PATH`, `CI`, and `ZOT_TEST_VALUE`.
- Preserve an inherited Claude variable that zot does not own.
- Replace an inherited stale `ZOT_PROJECT_DIR`.
- Replace an inherited stale `CLAUDE_PROJECT_DIR`.
- Emit one entry for each owned key.
- Emit matching values for `ZOT_PROJECT_DIR` and `CLAUDE_PROJECT_DIR`.
- Preserve empty values for non-owned keys.
- Leave unsupported variables unset when the parent does not provide them.
- Preserve unsupported variables when the parent does provide them.
- Confirm that `pwd`, hook input `cwd`, and the defined project directory agree for the current zot directory model.
- Expand `$ZOT_PROJECT_DIR` through the Unix shell.
- Expand `${ZOT_PROJECT_DIR}` through the Unix shell.
- Expand `$CLAUDE_PROJECT_DIR` through the Unix shell.
- Keep ordinary shell syntax unchanged.
- Do not log secret values.

Add end-to-end coverage for the current hook events:

- `PreToolUse`.
- `SessionStart`.
- `Stop`.
- `Notification`.

Use a hook fixture that records selected values to the existing test log. Do not record the complete environment.

When zot exposes session or event fields, add tests for each `ZOT_*` value and its Claude alias. Test both the value-present and value-absent cases.

When extension source context exists, test that project hooks do not receive extension variables and extension-owned hooks receive only their own source values.

When environment persistence is implemented, test safe assignment parsing, malformed input, lifecycle reset, later-hook visibility, and the absence of changes to the zot process environment.

Use the existing Go unit-test and end-to-end test patterns in the repository. Run the complete test suite after each implementation slice and at the end of the work.

## Out of Scope

- Implementing every Claude lifecycle event.
- Implementing HTTP hooks or HTTP header interpolation.
- Implementing MCP, prompt, or agent hook handlers.
- Implementing command `args`, `if`, `async`, `asyncRewake`, or `shell` execution behavior.
- Implementing Claude plugin option resolution without a source option model.
- Creating fake remote, messaging, session, PID, or effort values.
- Mapping `CLAUDE_PID` to the hook process PID or to the zot PID without a documented contract.
- Sourcing arbitrary environment files.
- Executing shell syntax from `ZOT_ENV_FILE` or `CLAUDE_ENV_FILE`.
- Renaming or replacing standard variables such as `PATH` or `TRACEPARENT`.
- Printing or persisting complete process environments.
- Changing the existing hook JSON schema parser beyond the environment data required by this spec.
- Pushing or merging the implementation.

## Further Notes

The main source for Claude behavior is the official Claude Code Hooks reference, Environment Variables reference, Plugin Reference, Settings reference, Skills reference, and Subagents reference. The research document records the source links and version caveats.

Claude behavior changes over time. The implementation should record only stable semantic mappings. It should not copy every new Claude variable into zot without checking the zot source, lifecycle, security impact, and test surface.

The recommended first implementation exports only `ZOT_PROJECT_DIR` and `CLAUDE_PROJECT_DIR`, preserves the parent environment, and adds tests at the command execution seam. Later changes can add verified `ZOT_*` values without changing the core environment builder contract.
