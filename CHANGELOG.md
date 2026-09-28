# Changelog

## 3.8.10 - 2026-09-28

- Require explicit confirmation in the Telegram bot before panel login. The bot shows the requesting IP and code and lets the administrator deny the request.
- Prevent public Telegram login requests from blocking account linking or displacing a request already shown in the bot.
- Group IPv6 addresses by /64 for the Telegram login start rate limit.
- Remove VK TURN call pool and proxy support, as already reverted on `main` after the 3.8.9 release.
