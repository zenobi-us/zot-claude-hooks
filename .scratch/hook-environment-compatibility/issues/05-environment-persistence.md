# 05: Add safe hook environment persistence

**What to build:** Let supported lifecycle hooks persist safe environment assignments for later hook processes without changing the zot process environment or executing arbitrary shell code.

**Blocked by:** 01: Add the hook environment builder and project variables; 02: Align hook directory and event context.

**Status:** ready-for-agent

- [ ] Define which zot hook events can write environment state.
- [ ] Provide `ZOT_ENV_FILE` only for supported persistence events.
- [ ] Provide `CLAUDE_ENV_FILE` only when the zot behavior matches the Claude contract.
- [ ] Keep environment state scoped to the session and project.
- [ ] Accept only the documented safe assignment form.
- [ ] Reject command substitution, redirects, pipelines, and other executable shell syntax.
- [ ] Apply accepted values only to later hook processes.
- [ ] Never change the zot process environment.
- [ ] Define behavior for malformed entries, resets, session end, and restart.
- [ ] Add tests for safe assignments, rejected syntax, lifecycle resets, and later-hook visibility.
