# Release process

This repository uses Release Please with the same branch and channel model as ClankPipe.

- `master` uses `release-please-config--release.json` and minor bumps.
- `release/*` uses `release-please-config--hotfix.json` and patch bumps.
- Release Please creates and merges the stable release PR.
- Ordinary pushes publish `next` binaries.
- A merged Release Please release publishes `latest` binaries.

Releases are built for Linux, macOS, and Windows on amd64 and arm64 where supported. Every publish run checks out and validates an immutable source commit before building.

Use Conventional Commit messages (`feat:`, `fix:`, `docs:`, and so on). The manifest version and `extension.json` version must remain aligned.
