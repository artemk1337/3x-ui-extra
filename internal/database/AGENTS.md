# Database and migrations

Parent: [backend instructions](../AGENTS.md).

`model` defines persistent panel entities. `db.go` initializes GORM, schema repairs
and seeders; `dialect.go` contains SQL differences. `migrate_data.go` copies panel
data between SQLite and PostgreSQL; `dump_sqlite.go` handles portable SQLite dumps.
Panel business mutations remain in `web/service`.

## Invariants

- Register new top-level models in both `allModels()` (`db.go`) and the FK-aware
  `migrationModels()` list (`migrate_data.go`); `migration_models_test.go` checks parity.
- Changes must work for SQLite and PostgreSQL. Reuse dialect helpers for JSON,
  boolean/number expressions and counter updates instead of inserting SQLite-only SQL.
- Migration/repair code runs on existing databases at startup: preserve user data
  and make upgrades repeatable. Seeders record completion in `HistoryOfSeeders`
  after successful transactional work.
- Traffic counter additions must stay below `TrafficMax`; SQLite integer overflow
  promotes values to REAL and breaks subsequent scans.
- Inbound ports are not globally unique: different nodes may reuse a port.
  Keep client/host/link identity and orphan handling consistent with service logic.
- SQLite-to-PostgreSQL migration preserves the source; destination clearing/copying
  occurs in one transaction. Resync PostgreSQL sequences after commit.
- Use the SQLite backup API rather than copying a live WAL database file; retain
  integrity validation, timeout, private file permissions and failure cleanup.

## Checks and sources

Use `make dist-stub` then focused tests for `./internal/database/...`; add a fixture
for the old schema when changing migration behavior. Existing tests cover model
list parity, indexes, backup/restore, counter repairs and repeatable startup.
Sources: `db.go`, `dialect.go`, `migrate_data.go`, `model/model.go` and adjacent tests.
