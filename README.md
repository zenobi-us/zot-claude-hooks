# zot-cluade-hooks

This extension runs command hooks from JSON files.

The extension uses Bun and `@crustjs/core`.

## Files

- `extension.json` contains the zot manifest.
- `index.ts` contains the JSONL protocol process and the `list` diagnostic command.
- `PLAN.md` contains the current scope and future work.
- `package.json` contains the Bun dependency.

## Install dependencies

```sh
bun install
```

## Run the diagnostic command

```sh
bun run list
```

## Run with zot

```sh
zot --ext ~/Projects/zot-claude-hooks
```

The extension reads hook definitions from these paths, in order:

- `~/.claude/settings.json`
- `.claude/settings.json`
- `.claude/settings.local.json`
- `$ZOT_USER_CONFIG_DIR/zot-cluade-hooks.json` when `ZOT_USER_CONFIG_DIR` is set
- `.zot/zot-cluade-hooks.json`
- `.zot/zot-cluade-hooks.local.json`
- `$ZOT_HOOKS_PATH` when `ZOT_HOOKS_PATH` is set

Each file must contain a top-level `hooks` object. Relative paths resolve from the project directory.
