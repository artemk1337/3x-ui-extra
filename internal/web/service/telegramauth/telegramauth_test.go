package telegramauth

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testService(t *testing.T) (*Service, *gorm.DB, *time.Time) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "auth.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.September, 28, 12, 0, 0, 0, time.UTC)
	s := NewService()
	s.db = func() *gorm.DB { return db }
	s.now = func() time.Time { return now }
	return s, db, &now
}

func createUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()
	user := model.User{Username: "admin"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	return user
}

func TestLinkIsUniqueAndSingleUse(t *testing.T) {
	s, db, _ := testService(t)
	first := createUser(t, db)
	second := createUser(t, db)
	code, _, err := s.StartLink(first.Id)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumeLink(code, 123); err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumeLink(code, 456); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("reused link code: %v", err)
	}
	code, _, err = s.StartLink(second.Id)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumeLink(code, 123); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("duplicate telegram binding: %v", err)
	}
	if err := s.Unlink(first.Id); err != nil {
		t.Fatal(err)
	}
	code, _, err = s.StartLink(second.Id)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumeLink(code, 123); err != nil {
		t.Fatal(err)
	}
}

func TestLoginRequiresMatchingTelegramAndCSRF(t *testing.T) {
	s, db, _ := testService(t)
	user := createUser(t, db)
	link, _, err := s.StartLink(user.Id)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumeLink(link, 123); err != nil {
		t.Fatal(err)
	}
	code, _, err := s.StartLogin("browser-secret")
	if err != nil {
		t.Fatal(err)
	}
	if got, pending, err := s.CompleteLogin(code, "browser-secret"); err != nil || !pending || got != nil {
		t.Fatalf("pending login = %v, %v, %v", got, pending, err)
	}
	if err := s.ApproveLogin(code, 456); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("wrong telegram account approved: %v", err)
	}
	if err := s.ApproveLogin(code, 123); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.CompleteLogin(code, "other-browser"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("wrong csrf completed login: %v", err)
	}
	got, pending, err := s.CompleteLogin(code, "browser-secret")
	if err != nil || pending || got == nil || got.Id != user.Id {
		t.Fatalf("completed login = %v, %v, %v", got, pending, err)
	}
	if _, _, err := s.CompleteLogin(code, "browser-secret"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("reused login code: %v", err)
	}
}

func TestLoginRechecksBindingAndExpiry(t *testing.T) {
	s, db, now := testService(t)
	user := createUser(t, db)
	link, _, err := s.StartLink(user.Id)
	if err != nil {
		t.Fatal(err)
	}
	*now = now.Add(codeTTL)
	if err := s.ConsumeLink(link, 123); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expired link accepted: %v", err)
	}
	link, _, err = s.StartLink(user.Id)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumeLink(link, 123); err != nil {
		t.Fatal(err)
	}
	code, _, err := s.StartLogin("browser")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ApproveLogin(code, 123); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.User{}).Where("id = ?", user.Id).Update("telegram_id", 0).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.CompleteLogin(code, "browser"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("changed binding logged in: %v", err)
	}
	link, _, err = s.StartLink(user.Id)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumeLink(link, 123); err != nil {
		t.Fatal(err)
	}
	code, _, err = s.StartLogin("browser")
	if err != nil {
		t.Fatal(err)
	}
	*now = now.Add(codeTTL)
	if err := s.ApproveLogin(code, 123); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expired login approved: %v", err)
	}
}

func TestLinkRejectedAfterLoginEpochChanges(t *testing.T) {
	s, db, _ := testService(t)
	user := createUser(t, db)
	code, _, err := s.StartLink(user.Id)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.User{}).Where("id = ?", user.Id).
		Update("login_epoch", gorm.Expr("login_epoch + 1")).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumeLink(code, 123); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("link survived account change: %v", err)
	}
	var stored model.User
	if err := db.First(&stored, user.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.TelegramID != 0 {
		t.Fatalf("unexpected telegram binding: %d", stored.TelegramID)
	}
}
