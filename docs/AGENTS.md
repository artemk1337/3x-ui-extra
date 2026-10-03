# Documentation site

Parent: [project instructions](../AGENTS.md).
See [README](README.md) and [contribution guide](CONTRIBUTING.md).

This is a separate Next.js/Fumadocs application using pnpm, not the embedded Vite
panel. `app` owns routes/layouts, `content/docs` contains localized MDX, `components`
provides documentation/UI tools, and `lib/xray` contains testable configuration logic.

## Invariants

- English is the canonical content; `lib/i18n.ts` wires `en`, `fa`, `ru`, `zh`,
  English fallback and Persian RTL. Keep locale routing and navigation consistent.
- Keep configuration generators in `lib/xray` independent of React and browser-local.
  The explicit API client/explorer is a separate network feature; preserve its chosen
  endpoint and authentication boundary rather than adding telemetry to generators.
- Both normal hosting and `DEPLOY_TARGET=static` must work; static export enables
  directory-style routes and unoptimized images in `next.config.mjs`.
- API reference generation reads `public/openapi.json` and emits English pages
  under `content/docs/en/reference/api`. Update the spec and regenerate via
  `pnpm gen:api` when the reference changes; do not edit generated pages alone.

## Checks

Run the relevant `pnpm test`, `pnpm typecheck`, `pnpm lint` and `pnpm build` from
this directory. For routing/export changes also run `DEPLOY_TARGET=static pnpm build`.
Documentation-only AGENTS changes need link validation and `git diff --check`.
Sources: `package.json`, `next.config.mjs`, `lib/i18n.ts`, `scripts/gen-openapi.ts`.
