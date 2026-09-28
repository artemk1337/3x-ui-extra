# Architecture

- `internal/database/model` defines persistent panel entities. Register new
  tables in both `internal/database/db.go` and `internal/database/migrate_data.go`.
- `internal/web/service` owns panel mutations; `internal/web/controller` exposes
  authenticated APIs. `frontend` is the React admin panel.
- Telegram panel login uses a personal user ID linked with current credentials,
  not the bot's notification/admin chat list. One-time bot approvals issue the
  ordinary panel session; password login remains available.
