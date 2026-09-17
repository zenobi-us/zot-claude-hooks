---
backend: local-markdown
issue-root: .scratch
---

# Issue tracker: Local Markdown

Issues and specifications live in `.scratch/` under the active alignment root.

## Conventions

- Use one feature directory for each feature: `.scratch/<feature-slug>/`.
- Store the feature specification at `.scratch/<feature-slug>/spec.md`.
- Store implementation issues in `.scratch/<feature-slug>/issues/`.
- Use one file for each issue. Start numbering at `01`.
- Add a `Status:` line near the top of each issue.
- Add comments under a `## Comments` heading at the end of the issue file.

When a skill says to publish to the issue tracker, create a file under `.scratch/<feature-slug>/`.
