# Embedded React frontend

Parent: [project instructions](../AGENTS.md). See [README](README.md) for orientation;
use current code/package scripts where older README details differ.

React/Ant Design/Vite build three entries: admin, login and subscription page.
`src/pages` owns feature UI, `src/api` owns HTTP/query/realtime integration,
`src/schemas` owns validation and `src/lib/xray/*-form-adapter.ts` converts form/wire
shapes. `vite.config.js` emits `../internal/web/dist` for Go embedding.

## Invariants

- Server state uses TanStack Query; realtime updates use the shared WebSocket bridge.
  Local view/form state stays in React. Preserve existing feature query keys/invalidation.
- All runtime routes/assets respect `X_UI_BASE_PATH`. Shared HTTP initialization
  sends CSRF on unsafe methods, refreshes/retries after CSRF 403 and handles 401 redirects.
- Follow the existing form pattern of the area (including React Hook Form); do not
  impose the older README's AntD-only pattern on current forms.
- Keep form-to-wire conversions in adapters/schemas so generated config matches
  backend protocol validation. Preserve distinct subscription/admin/login bootstraps.
- Never edit `src/generated` directly; `npm run gen:zod` uses `tools/openapigen`.
  `npm run gen:api` produces the API specification. Current `npm run build` runs
  API generation, but does not regenerate Zod; use `make gen-check` for freshness.
- `parseMsg` warns on schema failures by default; only `strict: true` throws.
  Do not assume development mode turns validation into an exception.

## Checks and sources

Run relevant `npm run typecheck`, `npm run lint`, `npm run format:check`, `npm test`
and `npm run build` here. Vitest separates Node unit tests, jsdom components and
Chromium Storybook tests; UI behavior also needs rendered interaction checks.
Sources: `package.json`, `vite.config.js`, `vitest.config.ts`, `src/api/http-init.ts`,
`src/api/websocketBridge.ts`, `src/routes.tsx`, `src/utils/zodValidate.ts`.
