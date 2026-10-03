# Installation and delivery

Parent: [project instructions](../AGENTS.md).
See [deployment README](README.md), [cloud-init](cloud-init/README.md) and
[Hetzner notes](marketplace/hetzner/README.md).

This area owns unattended bootstrap and install smoke verification. Root
`install.sh`, `update.sh`, `x-ui.sh`, Docker scripts and `.github/workflows/` own
installation/update/packaging; keep deployment examples consistent with them.

## Invariants

- Unattended install (`XUI_NONINTERACTIVE=1` or non-TTY stdin) must finish without
  prompts and generate per-instance credentials unless explicitly supplied.
- Preserve the installer result file contract at `/etc/x-ui/install-result.env`
  with mode 600. Its access URL, token, DB and TLS settings are installer outputs;
  do not put fixed shared credentials or encryption/session keys into images.
- Installation/update uses this fork's releases. An explicit tag pins the version;
  the no-argument path resolves stable latest. Verify the release workflow's actual
  tag/dev-channel behavior before changing channel selection.
- Preserve existing configuration and custom `bin/` files during reinstall/update.
- TLS mode and domain/IP prerequisites must match installer options. Cloud-init
  templates must forward `XUI_*` settings instead of reimplementing installer logic.
- Production builds embed the frontend and package required runtime assets;
  release workflow platform/CGO handling and Docker builds are separate paths.

## Checks

For shell changes use `bash -n` on affected scripts and inspect workflow packaging.
`bash deploy/test/smoke-noninteractive.sh <tag>` requires Docker/network and verifies
the pinned version, random credentials, result-file permissions, HTTP access and
reinstall preservation. Validate cloud-init YAML with `cloud-init schema` when available.
Sources: `test/smoke-noninteractive.sh`, root installers/Dockerfile and release workflow.
