package tgbot

import (
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/telegramauth"
	"github.com/mymmrac/telego"
)

func (t *Tgbot) telegramAuthCommand(message *telego.Message, command, code string) string {
	if message.Chat.Type != "private" || message.From == nil || message.Chat.ID != message.From.ID {
		return t.I18nBot("tgbot.authPrivateOnly")
	}
	if code == "" {
		return t.I18nBot("tgbot.authInvalidCode")
	}
	var err error
	switch command {
	case "link":
		err = telegramauth.Default.ConsumeLink(code, message.From.ID)
		if err == nil {
			logger.Infof("Telegram account linked to panel admin: telegram_id=%d", message.From.ID)
			return t.I18nBot("tgbot.authLinked")
		}
	case "login":
		err = telegramauth.Default.ApproveLogin(code, message.From.ID)
		if err == nil {
			logger.Infof("Telegram panel login approved: telegram_id=%d", message.From.ID)
			return t.I18nBot("tgbot.authApproved")
		}
	}
	return t.I18nBot("tgbot.authInvalidCode")
}
