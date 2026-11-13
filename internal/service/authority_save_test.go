package service

import (
	"path/filepath"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

func TestUpgradeV7ToAuthoritySaveReadbackAndResume(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "v7-authority.db")
	svc := newVersionTestService(t, dbPath)
	if _, err := svc.CreateGame(threeSiloRequest(207)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	legacy := downgradeToV7AuthorityFixture(t, svc)

	upgraded, err := UpgradeV7ToAuthoritySave(svc.db, dbPath)
	if err != nil || !upgraded {
		t.Fatalf("v7 authority upgrade failed: upgraded=%v err=%v", upgraded, err)
	}
	backups, _ := filepath.Glob(dbPath + ".bak-*-v8authority")
	if len(backups) != 1 {
		t.Fatalf("expected one authority backup, got %v", backups)
	}
	assertAuthorityMigrationReadback(t, svc, legacy)

	var meta model.SaveMeta
	if err := svc.db.First(&meta, "`key` = ?", saveMetaKey).Error; err != nil {
		t.Fatalf("read migrated meta: %v", err)
	}
	if meta.SaveVersion != authoritySaveVersion {
		t.Fatalf("meta version = %d, want %d", meta.SaveVersion, authoritySaveVersion)
	}

	fresh := newVersionTestService(t, dbPath)
	if err := fresh.Resume(); err != nil {
		t.Fatalf("resume migrated v8 save: %v", err)
	}
	if len(fresh.silos) != 3 || fresh.silo.MayorLegitimacy() != legacy[fresh.silo.ID] {
		t.Fatalf("resumed authority state mismatch: silos=%d Mayor=%+v", len(fresh.silos), fresh.silo.Mayor)
	}

	upgradedAgain, err := UpgradeV7ToAuthoritySave(svc.db, dbPath)
	if err != nil || upgradedAgain {
		t.Fatalf("v8 save must skip v7 migration: upgraded=%v err=%v", upgradedAgain, err)
	}
}

func TestUpgradeV7ToAuthoritySaveRollsBackAllSilos(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "v7-authority-rollback.db")
	svc := newVersionTestService(t, dbPath)
	createVersionTestGame(t, svc)
	legacy := downgradeToV7AuthorityFixture(t, svc)

	if err := svc.db.Migrator().DropTable(&model.SaveMeta{}); err != nil {
		t.Fatalf("drop meta table: %v", err)
	}
	if err := svc.db.Transaction(upgradeV7ToAuthority); err == nil {
		t.Fatal("migration must fail when its version write cannot complete")
	}

	var silos []model.Silo
	if err := svc.db.Order("id ASC").Find(&silos).Error; err != nil {
		t.Fatalf("reload rollback fixture: %v", err)
	}
	for i := range silos {
		silo := &silos[i]
		if silo.Mayor.Initialized || silo.Authority.Initialized || silo.Authority.W != 0 {
			t.Fatalf("failed transaction leaked authority state into silo %d: %+v", silo.ID, silo.Authority)
		}
		if silo.Legitimacy != legacy[silo.ID] {
			t.Fatalf("rollback changed legacy legitimacy for silo %d", silo.ID)
		}
	}
}

func downgradeToV7AuthorityFixture(t *testing.T, svc *GameService) map[uint]float64 {
	t.Helper()
	var silos []model.Silo
	if err := svc.db.Order("id ASC").Find(&silos).Error; err != nil {
		t.Fatalf("load fixture silos: %v", err)
	}
	legacy := make(map[uint]float64, len(silos))
	for i := range silos {
		silo := &silos[i]
		legacy[silo.ID] = 0.61 + float64(i)*0.11
		silo.Legitimacy = legacy[silo.ID]
		silo.Mayor = model.MayorOffice{}
		silo.Authority = model.ShadowAuthority{}
		if err := svc.db.Save(silo).Error; err != nil {
			t.Fatalf("downgrade silo %d: %v", silo.ID, err)
		}
	}
	if err := svc.db.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: 7}).Error; err != nil {
		t.Fatalf("downgrade meta: %v", err)
	}
	return legacy
}

func assertAuthorityMigrationReadback(t *testing.T, svc *GameService, legacy map[uint]float64) {
	t.Helper()
	var silos []model.Silo
	if err := svc.db.Order("id ASC").Find(&silos).Error; err != nil {
		t.Fatalf("reload migrated silos: %v", err)
	}
	if len(silos) != len(legacy) {
		t.Fatalf("migrated silo count = %d, want %d", len(silos), len(legacy))
	}
	for i := range silos {
		silo := &silos[i]
		if !silo.Mayor.Initialized || silo.Mayor.HolderName == "" || silo.MayorLegitimacy() != legacy[silo.ID] {
			t.Fatalf("Mayor migration mismatch for silo %d: %+v", silo.ID, silo.Mayor)
		}
		if silo.Legitimacy != legacy[silo.ID] {
			t.Fatalf("compatibility mirror mismatch for silo %d", silo.ID)
		}
		if !silo.Authority.Initialized || silo.Authority.W != engine.DefaultAuthorityConfig().LegacyConservativeW || silo.Authority.InitializationSource != "v7_conservative_w50" {
			t.Fatalf("conservative W mismatch for silo %d: %+v", silo.ID, silo.Authority)
		}
		if silo.Authority.Judicial.PactQualified || silo.Authority.Judicial.PactLocked {
			t.Fatalf("migration must not qualify or lock pact content for silo %d", silo.ID)
		}
	}
}
