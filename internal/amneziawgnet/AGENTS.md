# Embedded AmneziaWG networking

Parent: [backend instructions](../AGENTS.md).

`../amneziawg` defines protocol shapes and derives desired instances; this package
owns embedded amneziawg-go/gVisor devices, relays, forwarding and SOCKS bridging.
`Manager` indexes inbound devices by ID; `OutboundManager` indexes client devices
by outbound tag. Web services reconcile persisted desired state through these managers.

## Invariants

- Unchanged reconcile must not call `IpcSet`: `replace_peers=true` would reset live
  handshakes. Reconfigure peers/key/obfuscation changes; rebuild address/effective-MTU changes.
- Attach forwarders before configuring/bringing up a real relay device;
  `NewDevice` alone is insufficient for relay callers.
- Stop inbound instances in order: port forwarding → device/packet delivery → UDP relay.
- Outbound address/MTU fingerprints control rebuild. Keep `StopAll` locked during
  close so another tick cannot rebind a device during teardown.
- Inbound relay traffic passes through per-peer loopback Xray SOCKS. Outbound
  egress maps tags to netstacks via the loopback listener; preserve that routing boundary.
- Device/register/DNS changes invalidate per-tag tunnel DNS caches. Egress close
  must stop its listener and tracked handlers, avoiding leaked tunnel connections.

Checks: `go test ./internal/amneziawgnet ./internal/amneziawg`; race tests for manager
and relay lifecycle. Preserve regression coverage for unchanged peer reconciliation
and S4-derived MTU. Sources: `manager.go`, `outbound_manager.go`, `device.go`,
`egress.go`, `manager_test.go`, `relay_e2e_test.go`.
