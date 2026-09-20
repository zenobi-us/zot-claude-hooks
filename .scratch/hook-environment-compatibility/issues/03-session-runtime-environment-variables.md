# 03: Add verified session and runtime variables

**What to build:** Expose zot session and runtime values to hooks when the zot host provides values with a clear meaning and lifecycle. Add Claude aliases only when the values are equivalent.

**Blocked by:** 02: Align hook directory and event context.

**Status:** implemented in T3

- [x] Inventory the installed zot SDK fields that can provide session or runtime values.
- [x] Define the source, scope, format, and unset behavior for each exported value.
- [x] Add `ZOT_SESSION_ID` only when zot provides a matching session identifier.
- [x] Add `ZOT_EFFORT` only when zot provides a matching effort value.
- [x] Add `ZOT_CHILD_SESSION` only when zot identifies a child or agent session.
- [x] Add remote, bridge, messaging, PID, or shell variables only after a matching zot contract exists.
- [x] Add a `CLAUDE_*` alias only when the zot value has the same meaning, lifecycle, and format.
- [x] Keep values unset when the source is unavailable.
- [x] Preserve inherited values when zot cannot derive an equivalent value.
- [x] Never log messaging tokens or other sensitive values.
- [x] Add tests for available, unavailable, inherited, and alias cases.
