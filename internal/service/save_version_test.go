package service

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/model"
	"silo40/internal/repository"

	"gorm.io/gorm"
)

func newVersionTestService(t *testing.T, dbPath string) *GameService {
	t.Helper()
	db, err := repository.InitDB(dbPath)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	return NewGameService(db)
}

func createVersionTestGame(t *testing.T, svc *GameService) {
	t.Helper()
	req := model.CreateGameRequest{
		SiloName:   "VersionGate",
		StartYear:  122,
		AgentName:  "Agent",
		Profession: "IT",
		Seed:       7,
	}
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
}

func TestResumeAcceptsCurrentSaveVersion(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "save.db")
	createVersionTestGame(t, newVersionTestService(t, dbPath))

	fresh := newVersionTestService(t, dbPath)
	if err := fresh.Resume(); err != nil {
		t.Fatalf("resume with current save version must succeed, got: %v", err)
	}
	if !fresh.hasSession() {
		t.Fatalf("resume must restore the session")
	}
}

func TestResumeEmptyDatabaseReturnsRecordNotFound(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "empty-save.db")
	svc := newVersionTestService(t, dbPath)
	err := svc.Resume()
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound for a normal empty database, got: %v", err)
	}
}

func TestResumeRejectsMissingVersionMarker(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "save.db")
	svc := newVersionTestService(t, dbPath)
	createVersionTestGame(t, svc)

	// 模拟旧版本写入的存档：有游戏记录但没有版本标记。
	if err := svc.db.Where("`key` = ?", "save").Delete(&model.SaveMeta{}).Error; err != nil {
		t.Fatalf("delete meta: %v", err)
	}

	fresh := newVersionTestService(t, dbPath)
	err := fresh.Resume()
	if !errors.Is(err, ErrIncompatibleSave) {
		t.Fatalf("expected ErrIncompatibleSave for legacy save, got: %v", err)
	}
}

func TestResumeRejectsV5IdentityModel(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "v5-save.db")
	svc := newVersionTestService(t, dbPath)
	createVersionTestGame(t, svc)
	if err := svc.db.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: 5}).Error; err != nil {
		t.Fatalf("downgrade meta to v5: %v", err)
	}

	fresh := newVersionTestService(t, dbPath)
	err := fresh.Resume()
	if !errors.Is(err, ErrIncompatibleSave) {
		t.Fatalf("expected ErrIncompatibleSave for v5 identity model, got: %v", err)
	}
	if !strings.Contains(err.Error(), "旧档基于旧身份模型，无法继续，请开新局") {
		t.Fatalf("v5 rejection lacks the required explanation: %v", err)
	}
	if fresh.hasSession() {
		t.Fatal("rejected v5 save must not restore a session")
	}
}

func TestResumeRejectsV6BeforeProgressionModel(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "v6-save.db")
	svc := newVersionTestService(t, dbPath)
	createVersionTestGame(t, svc)
	if err := svc.db.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: 6}).Error; err != nil {
		t.Fatalf("downgrade meta to v6: %v", err)
	}

	fresh := newVersionTestService(t, dbPath)
	err := fresh.Resume()
	if !errors.Is(err, ErrIncompatibleSave) {
		t.Fatalf("expected ErrIncompatibleSave for v6 progression model, got: %v", err)
	}
	if !strings.Contains(err.Error(), "v6 存档缺少行动门控与多实体结局状态，无法继续，请开新局") {
		t.Fatalf("v6 rejection lacks the progression explanation: %v", err)
	}
	if fresh.hasSession() {
		t.Fatal("rejected v6 save must not restore a session")
	}
}

func TestResumeRejectsWrongVersion(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "save.db")
	svc := newVersionTestService(t, dbPath)
	createVersionTestGame(t, svc)

	if err := svc.db.Save(&model.SaveMeta{Key: "save", SaveVersion: model.CurrentSaveVersion + 98}).Error; err != nil {
		t.Fatalf("bump meta: %v", err)
	}

	fresh := newVersionTestService(t, dbPath)
	err := fresh.Resume()
	if !errors.Is(err, ErrIncompatibleSave) {
		t.Fatalf("expected ErrIncompatibleSave for wrong version, got: %v", err)
	}
}
