# 06: Document and verify environment compatibility

**What to build:** Provide a clear user-facing compatibility contract and a complete test matrix for inherited variables, `ZOT_*` values, Claude aliases, extension values, and persistence behavior.

**Blocked by:** 01: Add the hook environment builder and project variables; 02: Align hook directory and event context; 03: Add verified session and runtime variables; 04: Add extension source environment variables; 05: Add safe hook environment persistence.

**Status:** complete

- [x] Document supported zot variables.
- [x] Document matching Claude aliases.
- [x] Document variables that are preserved but not synthesized.
- [x] Document plugin and extension source boundaries.
- [x] Document shell expansion and stdin JSON separately.
- [x] Document unset-value behavior and security limits.
- [x] Add unit tests for the complete environment matrix.
- [x] Add end-to-end tests for all current hook events.
- [x] Verify that diagnostics never print complete environments or secrets.
- [x] Run the complete test suite and confirm that existing hook behavior remains valid.

## Comments

T6 completed the compatibility documentation and added an end-to-end fixture for `SessionStart`, `PreToolUse`, `Notification`, and `Stop`. Unit tests cover inherited, owned, alias, extension, persistence, unset, shell, and security behavior. Diagnostics continue to use event summaries and never print hook environments.
