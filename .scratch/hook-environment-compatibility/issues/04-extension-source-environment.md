# 04: Add extension source environment variables

**What to build:** Give extension-owned hook files a scoped zot environment that identifies the owning extension and its persistent data location without exposing extension values to project hooks.

**Blocked by:** 01: Add the hook environment builder and project variables.

**Status:** ready-for-agent

- [ ] Define an explicit source context for extension-owned hook files.
- [ ] Set `ZOT_EXTENSION_ROOT` only for hooks owned by an extension.
- [ ] Set `ZOT_EXTENSION_DATA` only when the persistent data directory contract exists.
- [ ] Set `ZOT_EXTENSION_OPTION_<KEY>` only for validated extension options.
- [ ] Keep extension variables out of ordinary project hook processes.
- [ ] Add Claude plugin aliases only when the source is a true Claude-compatible plugin source.
- [ ] Preserve inherited values for unowned extension variables.
- [ ] Define safe behavior for missing, invalid, or conflicting source metadata.
- [ ] Add tests for project hooks, extension hooks, multiple owners, and stale values.
