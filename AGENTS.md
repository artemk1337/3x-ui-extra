# Architecture

- `internal/database/model` defines persistent panel entities. Register new
  tables in both `internal/database/db.go` and `internal/database/migrate_data.go`.
- `internal/web/service` owns panel mutations; `internal/web/controller` exposes
  authenticated APIs. `frontend` is the React admin panel.
- `internal/vkturnproxy` owns only local proxy child processes. The VK TURN
  service stores call capacity and per-(WireGuard inbound, client) assignments.
  Xray remains the WireGuard server; a call URL is client-side TURN metadata,
  not an Xray share link.
- Keep VK TURN assignment writes serialized with client/inbound mutations.
  Only local WireGuard inbounds can run the proxy. Linux binaries are pinned
  and verified before execution.

See `docs/vk-turn.md` for operator setup.
