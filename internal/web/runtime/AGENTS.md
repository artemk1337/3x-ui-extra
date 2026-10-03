# Local and remote node transport

Parent: [panel instructions](../AGENTS.md).

`Runtime` abstracts local Xray APIs and remote panel RPCs. `Manager` resolves/caches
node runtimes; `remote.go` implements bounded HTTP calls, `tls_client.go` manages TLS
credentials/pools. Services own policy/persistence; this package must not import them.
`web.go` injects local API/restart hooks, egress resolution and master certificates.

## Invariants

- `DeleteUser` detaches a user from an inbound; `DeleteClient` removes the remote
  client record, attachments and traffic (a local no-op). Do not interchange them.
- Runtime identity includes address, bearer token, TLS policy/pin and outbound tag.
  Invalidation clears both the runtime cache and pooled HTTP client.
- mTLS rotation closes old idle pools and prevents new requests using stale
  credentials while allowing inflight requests to finish.
- Remote calls validate HTTP status and the JSON message envelope. Keep bounded
  decoding (64 MiB) and small error snippets (8 KiB).
- Preserve node-specific egress/TLS verification instead of using a generic default
  HTTP client that bypasses the manager's injected policy.

Checks: `go test ./internal/web/runtime`; use race tests for manager/cache rotation.
Sources: `runtime.go`, `manager.go`, `remote.go`, `tls_client.go`, `../web.go`.
