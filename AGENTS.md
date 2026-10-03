# Architecture

3x-ui-extra manages local and remote proxy nodes, clients, traffic and public
subscriptions. The Go panel serves an embedded React admin application.

- `main.go` composes configuration, node-token crypto, database, panel and
  subscription servers; it also exposes administration/migration CLI commands.
- `internal/web/controller` handles HTTP; `internal/web/service` owns mutations,
  runtime orchestration and persistence. `internal/web/runtime` talks to nodes.
- `internal/database` owns GORM models, SQLite/PostgreSQL setup and migration.
- `internal/xray` owns Xray child-process execution/configuration; AmneziaWG runs
  in-process, while TUIC and MTProto use separately managed processes.
- `internal/sub` serves public subscriptions and produces client configurations.
  `main.go` registers its link provider with panel services to avoid a package cycle.
- `frontend` builds into `internal/web/dist`; `docs` is a separate documentation
  site. `tools/openapigen` derives frontend schemas from selected Go types.

# Global invariants

- Keep database models, HTTP contracts, generated frontend schemas and runtime
  configuration consistent when changing a protocol or persistent entity.
- `SIGHUP` recreates panel/subscription listeners without restarting proxy cores;
  `SIGUSR1` restarts Xray. Full shutdown stops the bot and runtime processes.
- Panel login, scoped API-token access and public subscriptions have distinct
  authorization boundaries; do not make public endpoints depend on panel sessions.
- Build the frontend before a production Go build: `web.go` embeds `all:dist`.
  `make dist-stub` supports backend-only checks without a frontend bundle.

# Local instructions

- [Backend and shared runtime](internal/AGENTS.md)
- [Frontend](frontend/AGENTS.md)
- [Documentation site](docs/AGENTS.md)
- [Installation and delivery](deploy/AGENTS.md)

# Work and verification

Read the nearest instructions and tests before changing an area. Preserve unrelated
worktree changes. Use `Makefile` targets for focused checks; `make verify` mirrors
the CI gate. For documentation-only changes, check local links and `git diff --check`;
Go tests are unnecessary.

# Useful sources

[README](README.md) covers operations and environment variables; [Makefile](Makefile)
and `.github/workflows/` define current checks, packaging and release behavior.
