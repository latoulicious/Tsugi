# internal/version

Build identity served at `GET /version`: the release tag, commit, and build
time stamped in by the linker.

## Symbols

- `Version`, `Commit`, `Date` (vars) — build identity, set at link time via
  `-X` (see `Makefile` / `Dockerfile`). Defaults `dev` / `none` / `unknown`
  cover a plain `go build` with no `-ldflags`.
- `Info` (struct) — JSON payload `{version, commit, deployed_at}`.
- `Get() Info` — returns the build identity. When the linker vars are unset
  (local `go build`), falls back to the VCS stamp Go embeds (`vcs.revision`,
  `vcs.time`) via `debug.ReadBuildInfo`.

## Notes

- `deployed_at` = **build time**. `deploy.sh` rebuilds on every deploy, so build
  time ≈ deploy time, and it stays stable across restarts of the same image.
- `VERSION` comes from `git describe --tags --always`; with no tags it falls
  back to the short SHA (so version == commit until the first tag).
