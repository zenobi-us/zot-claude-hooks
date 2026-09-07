# E2E fixtures

These fixtures provide small projects for manual zot extension tests.
Run zot from the fixture directory so hook commands resolve against the fixture.

## Allow fixture

This fixture records `SessionStart` and matching `PreToolUse` payloads, then allows the tool.

```sh
cd ~/Projects/zot-claude-hooks/fixtures/allow
zot --ext ../..
```

In zot, ask:

> Use the `bash` tool to run `printf 'allow fixture\n'`. Report the output.

Inspect the captured payloads:

```sh
cat .zot/hook-events.jsonl
```

## Block fixture

This fixture records matching `bash` calls and blocks them with exit status `2`.

```sh
cd ~/Projects/zot-claude-hooks/fixtures/block
zot --ext ../..
```

In zot, ask:

> Use the `bash` tool to run `printf 'this must not run\n'`. Report what happened.

The command should not execute. Inspect the captured payloads with:

```sh
cat .zot/hook-events.jsonl
```

Delete `.zot/hook-events.jsonl` before each run if you want a clean capture.
