# 06: Document and verify environment compatibility

**What to build:** Provide a clear user-facing compatibility contract and a complete test matrix for inherited variables, `ZOT_*` values, Claude aliases, extension values, and persistence behavior.

**Blocked by:** 01: Add the hook environment builder and project variables; 02: Align hook directory and event context; 03: Add verified session and runtime variables; 04: Add extension source environment variables; 05: Add safe hook environment persistence.

**Status:** ready-for-agent

- [ ] Document supported zot variables.
- [ ] Document matching Claude aliases.
- [ ] Document variables that are preserved but not synthesized.
- [ ] Document plugin and extension source boundaries.
- [ ] Document shell expansion and stdin JSON separately.
- [ ] Document unset-value behavior and security limits.
- [ ] Add unit tests for the complete environment matrix.
- [ ] Add end-to-end tests for all current hook events.
- [ ] Verify that diagnostics never print complete environments or secrets.
- [ ] Run the complete test suite and confirm that existing hook behavior remains valid.
