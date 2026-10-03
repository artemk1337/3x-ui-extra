# Telegram panel login

Parent: [panel instructions](../../AGENTS.md).

This package owns in-memory link/login challenges. `controller/telegram_auth.go`
handles browser requests and credentials/rate limits; `service/tgbot` handles
private bot commands and explicit approval. It issues the ordinary panel session.

## Invariants

- Identity is the personal Telegram `From.ID` in a private chat. Notification/admin
  chat lists do not authorize panel login; password login remains available.
- Challenges expire after five minutes, are single-use and bounded in memory.
  Issuing `/login` does not approve login; approval is a separate bot action.
- Login completion verifies browser CSRF binding and current persisted Telegram
  binding. Link challenges bind to `LoginEpoch` so old credentials cannot complete them.
- Linking/unlinking requires current password and applicable 2FA; unlink revokes
  sessions through the login epoch. Do not log challenge secrets or login URLs.

Checks: package unit tests, `../tgbot/telegram_auth_router_test.go` and
`../../controller/telegram_auth_rate_test.go`. Sources: `telegramauth.go`,
`../tgbot/telegram_auth.go`, `../../controller/telegram_auth.go`.
