# Backend and shared runtime

Parent: [project instructions](../AGENTS.md).

`config` owns paths/environment and build identity; `logger` and `util` provide
shared helpers. `web` orchestrates persistent state and protocol runtimes;
lower-level runtime packages should not depend on HTTP controllers.

## Boundaries and invariants

- `amneziawg` translates panel settings into devices; `amneziawgnet` owns the
  userspace network stack, relays, SOCKS bridge and outbound devices.
- `tuic` and `mtproto` own sidecar config/process/manager lifecycle. Keep platform
  process handling in their OS-specific files and reconcile removed inbounds.
- `pia` owns provider authentication, signed region catalogs and peer registration;
  panel settings and persistence belong to `web/service`. Catalog signatures,
  response limits, no-redirect policy and pinned provider trust must remain enforced.
- `crypto/nodetoken` encrypts replayable node credentials with row-bound AES-GCM
  and versioned keys; local API-token hashes are a different credential model.
  Initialize its codec before database startup. Encryption modes require usable
  keys; preserve legacy plaintext migration reads and authenticated ciphertext writes.
- `eventbus` uses bounded, nonblocking queues and serial workers per subscriber;
  consumers must tolerate dropped events. Stop/unsubscribe workers with their owner.
- `tunnelmonitor` is disabled by default and configured through environment.
  Its recovery hook restarts Xray after threshold/cooldown; without a proxy URL,
  probes measure host connectivity. Its context is cancelled on full shutdown.

## Areas and checks

- [Database](database/AGENTS.md)
- [Web](web/AGENTS.md)
- [Xray](xray/AGENTS.md)
- [Subscriptions](sub/AGENTS.md)
- [AmneziaWG networking](amneziawgnet/AGENTS.md)

Run focused package tests for changes to the shared packages (`./internal/pia`,
`./internal/crypto/nodetoken`, `./internal/eventbus`, protocol packages). Use race
checks for lifecycle/worker changes; sidecar and network integration tests have
additional platform/binary requirements documented in their test files.

Sources: `main.go`, `crypto/nodetoken/nodetoken.go`, `eventbus/bus.go`,
`pia/serverlist_client.go`, `pia/serverlist_signature.go`, `tunnelmonitor/monitor.go`.
