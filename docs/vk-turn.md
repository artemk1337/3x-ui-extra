# VK TURN server

The panel can run `cacggghp/vk-turn-proxy` v1.8.3 beside a local Xray
WireGuard inbound. The proxy accepts UDP on its public port and forwards
decrypted WireGuard packets to the inbound's UDP port. The server does not use
VK call links itself: clients use a call link to obtain a TURN relay.

1. Create and enable a **local** WireGuard inbound with its clients in the
   panel. Give the inbound a UDP port.
2. In **VK TURN**, add one or more `https://vk.ru/call/join/...` or
   `https://vk.com/call/join/...` links. Each
   link can have a maximum number of active configurations; zero means no
   limit. Enable VK TURN for the desired WireGuard clients in their client
   editor. One client attached to two WireGuard inbounds counts as two
   configurations.
3. On the VK TURN page, set and enable a different public UDP port for each
   WireGuard inbound. Open that port in the host firewall. In Docker, publish
   each port as `PORT:PORT/udp` in `docker-compose.yml` and recreate the
   container. The WireGuard inbound port only needs to be reachable from the
   proxy process; bind it to loopback when direct access is unnecessary.
4. Check the process status and the per-call assigned/over-limit counters.
   Disabling or deleting a call moves its configurations to other enabled
   calls. If every enabled call is full, assignments remain active and their
   over-limit counts are displayed. If no call is enabled, configurations
   remain opted in but unassigned.

Linux amd64 and arm64 packages include a pinned proxy binary under the panel's
`bin` directory. The build verifies its SHA-256, and the panel checks it again
before execution. Other platforms can manage the call pool but cannot run the
proxy. A failed proxy appears with its error in the VK TURN page.

Client import links and an Android application are not produced in this phase.
For an end-to-end test, configure a compatible existing VK TURN client with its
assigned call URL, the proxy's public address, and the matching WireGuard keys.
The client's ordinary `wireguard://` share link is not a VK TURN import link.
