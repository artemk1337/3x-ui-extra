# Panel APIs and orchestration

Parent: [backend instructions](../AGENTS.md).

`web.go` wires Gin, sessions, WebSocket delivery and cron. `controller` owns HTTP
validation/routes, `service` owns mutations and runtime coordination, `entity`
defines transport shapes, `job` schedules service/runtime work, and `websocket`
delivers live events. [Node transport](runtime/AGENTS.md) and
[Telegram login](service/telegramauth/AGENTS.md) have their own contracts.

## Authorization and state

- API middleware order is authentication → token scope → config envelope → CSRF.
  Keep scope enforcement on new routes; scopes use route-pattern/method allowlists.
- Verified mTLS authenticates as `node-sync`, not admin. `monitor` only permits
  allowlisted GET/HEAD operations; unmatched `node-sync` operations are denied.
- Local API tokens store SHA-256 hashes; creation/recreation shows plaintext once,
  listing only returns metadata. API `createdAt` is seconds; `expiresAt` is unix-ms.
  Delete/toggle validates `expectedScope` to avoid cross-scope management.
- Sessions contain user ID and `LoginEpoch`, with the user reread from DB;
  credential changes and Telegram unlink revoke existing sessions via the epoch.
- Keep client/inbound mutations in services and coordinate persisted desired state
  with the selected local/remote runtime; controllers and cron must not bypass that layer.

## Background lifecycle

- Cadences are centralized in `web.go`. Cron uses `SkipIfStillRunning` and `Recover`;
  preserve non-overlap for stateful traffic/statistics jobs and shutdown ownership.
- Node heartbeat/traffic fanout is bounded with request timeouts. Dashboard traffic
  deltas are normalized to the shared five-second interval.
- Client IP limits use online-stats API; unavailable API makes the job a no-op.
  Do not restore an access-log fallback without changing that explicit contract.
- `StopPanelOnly` preserves proxy runtimes during listener reload; full `Stop`
  shuts down managers, jobs and traffic writing. See `main.go` signal orchestration.

## Checks and sources

Use `make dist-stub` before backend-only checks. Run affected package tests under
`./internal/web/controller`, `./internal/web/middleware`, `./internal/web/session`,
`./internal/web/service/...`, `./internal/web/job` and `./internal/web/runtime`.
Use race tests for cron/lifecycle/concurrent state changes.
Sources: `web.go`, `controller/api.go`, `controller/setting.go`, `session/session.go`,
`service/panel/api_token.go`, `service/panel/user.go`, `job/node_traffic_sync_job.go`,
`job/node_heartbeat_job.go`, `job/check_client_ip_job.go`.
