# Xray execution and configuration

Parent: [backend instructions](../AGENTS.md).

`process.go` owns the Xray child process and configuration snapshot; `api.go` exposes
gRPC mutations/statistics. `config.go`/`inbound.go` and `hot_diff.go` translate desired
config and choose live changes versus restart. `dnsconf` and `geodata` own DNS shapes
and bounded geodata reading/querying. Panel services own DB state and orchestration.

## Invariants

- Resolve platform binary/config paths through existing helpers. Process lifecycle
  and online state have separate locking; keep local and remote online snapshots distinct.
- Advance the applied config snapshot only after successful hot apply. Unsupported
  changes must fall back to restart instead of presenting desired state as applied.
- `ComputeHotDiff` supports inbounds/outbounds/routing; static sections require
  restart. Preserve special handling for API inbounds, reverse-tagged clients,
  REALITY, TPROXY and password SOCKS listener replacement.
- Stats/client identity must agree with service persistence; use existing traffic
  readers and mutation APIs rather than manipulating process configuration directly.
- Keep output/log writers safe during restart/shutdown; OS-specific child lifetime
  management remains in the platform files.

Checks: `go test ./internal/xray/...`; race tests for lifecycle/log writer changes.
Binary E2E tests are opt-in via `XRAY_E2E_BINARY`; passing unit tests alone does not
validate the packaged Xray binary. Sources: `process.go`, `hot_diff.go`, `api.go`,
`traffic.go`, `api_e2e_test.go` and adjacent mutation/race tests.
