package tgbot

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/mymmrac/telego"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/telegramauth"
)

func TestTelegramLoginNeedsPrivateCallback(t *testing.T) {
	mock, calls := staleButtonServer(t, map[string]any{
		"answerCallbackQuery": map[string]any{"ok": true, "result": true},
		"sendMessage": map[string]any{"ok": true, "result": map[string]any{
			"message_id": 1,
			"date":       0,
			"chat":       map[string]any{"id": 777, "type": "private"},
		}},
	})
	swapTestBot(t, mock.URL)
	t.Cleanup(mock.Close)
	if err := database.InitDB(filepath.Join(t.TempDir(), "x-ui.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.CloseDB() })
	previous := telegramauth.Default
	telegramauth.Default = telegramauth.NewService()
	t.Cleanup(func() { telegramauth.Default = previous })
	wasRunning := isRunning
	isRunning = true
	t.Cleanup(func() { isRunning = wasRunning })

	user := model.User{Username: "panel-admin"}
	if err := database.GetDB().Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	wantID := user.Id
	link, _, err := telegramauth.Default.StartLink(user.Id)
	if err != nil {
		t.Fatal(err)
	}
	bot := &Tgbot{}
	command := &telego.Message{From: &telego.User{ID: 777}, Chat: telego.Chat{ID: 777, Type: telego.ChatTypePrivate}, Text: "/start link_" + link}
	bot.answerCommand(command, 777, false)
	code, _, err := telegramauth.Default.StartLogin("browser-csrf", "192.0.2.1")
	if err != nil {
		t.Fatal(err)
	}
	command.Text = "/start login_" + code
	bot.answerCommand(command, 777, false)
	if len(telegramAuthApprovePrefix+code) > 64 || len(telegramAuthDenyPrefix+code) > 64 {
		t.Fatal("Telegram callback data exceeds 64 bytes")
	}
	if calls("sendMessage") < 2 {
		t.Fatal("login request was not sent to the private chat")
	}
	if user, pending, err := telegramauth.Default.CompleteLogin(code, "browser-csrf"); err != nil || !pending || user != nil {
		t.Fatalf("login before callback = %v, %v, %v", user, pending, err)
	}
	callback := &telego.CallbackQuery{
		ID: "confirm", From: telego.User{ID: 777}, Data: telegramAuthApprovePrefix + code,
		Message: &telego.Message{Chat: telego.Chat{ID: -777, Type: telego.ChatTypeSupergroup}},
	}
	bot.answerCallback(callback, false)
	if user, pending, err := telegramauth.Default.CompleteLogin(code, "browser-csrf"); err != nil || !pending || user != nil {
		t.Fatalf("group callback approved login = %v, %v, %v", user, pending, err)
	}
	callback.Message = &telego.Message{Chat: telego.Chat{ID: 777, Type: telego.ChatTypePrivate}}
	bot.answerCallback(callback, false)
	if user, pending, err := telegramauth.Default.CompleteLogin(code, "browser-csrf"); err != nil || pending || user == nil || user.Id != wantID {
		t.Fatalf("confirmed login = %v, %v, %v", user, pending, err)
	}

	denied, _, err := telegramauth.Default.StartLogin("second-browser", "192.0.2.2")
	if err != nil {
		t.Fatal(err)
	}
	command.Text = "/login " + denied
	bot.answerCommand(command, 777, false)
	callback.Data = telegramAuthDenyPrefix + denied
	bot.answerCallback(callback, false)
	if _, _, err := telegramauth.Default.CompleteLogin(denied, "second-browser"); !errors.Is(err, telegramauth.ErrInvalidCode) {
		t.Fatalf("denied login completed: %v", err)
	}
}
