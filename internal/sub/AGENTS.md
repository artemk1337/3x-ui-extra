# Public subscriptions

Parent: [backend instructions](../AGENTS.md).

`sub.go` owns the separate HTTP server; `controller.go` selects formats and maps
errors. `service.go` resolves clients/hosts and share links; `json_*`/`clash_*` export
client configs, `external_*`/`remote_routing.go` consume external data. `dist.go`
serves the shared embedded subscription page. Panel mutations stay in `web/service`.

## Invariants

- Request handlers use `SubService.ForRequest`; do not mutate the shared base service
  via `PrepareForRequest`. Host/client/node/settings/stat caches are request-scoped.
- UUID share exports prefer normalized client records; WireGuard/AmneziaWG tunnel
  fields come from inbound settings. Traffic identity is global by email and can
  belong to another inbound; do not filter that traffic only by the current inbound ID.
- Preserve enabled/expiry/quota filtering, host exclusions, stable ordering and
  deduplication across raw, Xray JSON and Clash/Mihomo exports.
- Empty subscriptions map to 404; service failures map to 500. Public routes do not
  use panel session authentication.
- Forwarded-header trust follows `forwarded_trust.go`: explicit CIDRs check
  `RemoteAddr`, while empty/default settings and settings-read failures retain
  legacy trust. Do not assume forwarded headers are always trusted or always denied.
- Retain bounded fetching/validation for external subscriptions/routing and keep
  HWID/access policy consistent across every supported format.

Checks: `go test ./internal/sub`; use existing format/property tests for exporter
changes. Scale tests are opt-in with `XUI_SCALE_TEST=1` or PostgreSQL env configuration
(`XUI_DB_TYPE`/`XUI_DB_DSN`), and require an appropriate test database.
Sources: `service.go`, `controller.go`, `forwarded_trust.go`, `sub_scale_test.go`,
`external_subscription.go` and adjacent format/identity tests.
