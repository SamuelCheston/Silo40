package service

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"silo40/internal/model"

	"gorm.io/gorm"
)

// setupLegacyDB 造一份"旧档"：完整游戏数据 + 无版本标记 + 无世界层
// (v1 之前的单堡形态)。
func setupLegacyDB(t *testing.T) (string, *GameService) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	svc := newVersionTestService(t, dbPath)
	createVersionTestGame(t, svc)
	// 模拟旧版本写入的存档：删掉版本标记，并把世界层还原成 v1 形态。
	if err := svc.db.Where("`key` = ?", saveMetaKey).Delete(&model.SaveMeta{}).Error; err != nil {
		t.Fatalf("strip save meta: %v", err)
	}
	if err := svc.db.Where("1 = 1").Delete(&model.World{}).Error; err != nil {
		t.Fatalf("strip worlds: %v", err)
	}
	if err := svc.db.Model(&model.Silo{}).Where("id = ?", ActiveSiloID).Update("world_id", 0).Error; err != nil {
		t.Fatalf("strip silo world link: %v", err)
	}
	return dbPath, svc
}

func assertSiloIntact(t *testing.T, svc *GameService) {
	t.Helper()
	var siloCount int64
	if err := svc.db.Model(&model.Silo{}).Count(&siloCount).Error; err != nil {
		t.Fatalf("count silos: %v", err)
	}
	if siloCount != 1 {
		t.Fatalf("silo rows changed: %d", siloCount)
	}
}

func assertV5IdentitySaveRejected(t *testing.T, dbPath string) {
	t.Helper()
	fresh := newVersionTestService(t, dbPath)
	err := fresh.Resume()
	if !errors.Is(err, ErrIncompatibleSave) {
		t.Fatalf("upgraded v5 identity save must be rejected, got: %v", err)
	}
	if !strings.Contains(err.Error(), "旧档基于旧身份模型，无法继续，请开新局") {
		t.Fatalf("v5 rejection must explain the identity break, got: %v", err)
	}
	if fresh.hasSession() {
		t.Fatal("rejected v5 save must not restore a session")
	}
}

func TestUpgradeLegacySaveSuccess(t *testing.T) {
	dbPath, svc := setupLegacyDB(t)

	upgraded, err := UpgradeLegacySave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}
	if !upgraded {
		t.Fatalf("expected upgrade to run for legacy save")
	}

	// 标记补写到 v1 + 升级留痕（v1→v2 由 UpgradeV1ToWorldSave 接力）。
	var meta model.SaveMeta
	if err := svc.db.First(&meta, "`key` = ?", saveMetaKey).Error; err != nil {
		t.Fatalf("meta missing after upgrade: %v", err)
	}
	if meta.SaveVersion != legacyMarkerVersion {
		t.Fatalf("meta version = %d, want %d", meta.SaveVersion, legacyMarkerVersion)
	}
	if !meta.LegacyUpgraded {
		t.Fatalf("meta must record the legacy upgrade (LegacyUpgraded)")
	}
	if meta.UpgradedAt.IsZero() {
		t.Fatalf("meta must record the upgrade time (UpgradedAt)")
	}

	// 备份文件存在（带时间戳后缀）。
	matches, _ := filepath.Glob(dbPath + ".bak-*")
	if len(matches) != 1 {
		t.Fatalf("expected exactly one timestamped backup, got %v", matches)
	}

	// After the legacy chain reaches v5, Resume must reject the old identity model.
	upgradedV2, err := UpgradeV1ToWorldSave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("v1 world upgrade failed: %v", err)
	}
	if !upgradedV2 {
		t.Fatalf("expected v1 world upgrade to run after legacy upgrade")
	}
	upgradedV3, err := UpgradeV2ToTwoSiloSave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("v2 two-silo upgrade failed: %v", err)
	}
	if !upgradedV3 {
		t.Fatalf("expected v2 two-silo upgrade to run after v1 world upgrade")
	}
	upgradedV4, err := UpgradeV3ToFactionIdentitySave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("v3 faction identity upgrade failed: %v", err)
	}
	if !upgradedV4 {
		t.Fatalf("expected v3 faction identity upgrade to run after v2 two-silo upgrade")
	}
	upgradedV5, err := UpgradeV4ToPlayableWorldSave(svc.db, dbPath)
	if err != nil || !upgradedV5 {
		t.Fatalf("v4 playable world upgrade failed: upgraded=%v err=%v", upgradedV5, err)
	}
	assertV5IdentitySaveRejected(t, dbPath)
}

// TestUpgradeLegacySaveBackupFailureLeavesDBUntouched：备份阶段失败时，
// 不得写任何标记、不得产生备份，游戏数据原样。
func TestUpgradeLegacySaveBackupFailureLeavesDBUntouched(t *testing.T) {
	dbPath, svc := setupLegacyDB(t)

	// 目录变只读 → VACUUM INTO 无法落盘。测试结束恢复权限以便清理。
	dir := filepath.Dir(dbPath)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	upgraded, err := UpgradeLegacySave(svc.db, dbPath)
	if upgraded || err == nil {
		t.Fatalf("expected backup-stage failure, got upgraded=%v err=%v", upgraded, err)
	}

	var meta model.SaveMeta
	if err := svc.db.First(&meta, "`key` = ?", saveMetaKey).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("no meta may be written on backup failure, got err=%v meta=%+v", err, meta)
	}
	assertSiloIntact(t, svc)
	matches, _ := filepath.Glob(dbPath + ".bak-*")
	if len(matches) != 0 {
		t.Fatalf("no backup may exist after backup-stage failure, got %v", matches)
	}
}

// TestUpgradeLegacySaveTransactionRollsBack：事务阶段失败必须回滚，
// 不留半迁移状态（备份已生成，可人工还原）。
func TestUpgradeLegacySaveTransactionRollsBack(t *testing.T) {
	dbPath, svc := setupLegacyDB(t)

	// 先确认备份步骤可独立成功。
	backupPath := dbPath + ".bak-test"
	if err := backupSaveFile(svc.db, backupPath); err != nil {
		t.Fatalf("backup: %v", err)
	}

	// 破坏写入目标表，让事务必然失败并回滚。
	if err := svc.db.Migrator().DropTable(&model.SaveMeta{}); err != nil {
		t.Fatalf("drop save meta table: %v", err)
	}
	if err := svc.db.Transaction(writeLegacyVersionMarker); err == nil {
		t.Fatalf("transaction must fail with meta table dropped")
	}

	// 半迁移检查：无标记残留、游戏数据原样。
	if svc.db.Migrator().HasTable(&model.SaveMeta{}) {
		var n int64
		svc.db.Table("save_metas").Count(&n)
		if n > 0 {
			t.Fatalf("failed transaction left %d meta rows behind", n)
		}
	}
	assertSiloIntact(t, svc)
}

func TestUpgradeSkipsMarkedSave(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "marked.db")
	svc := newVersionTestService(t, dbPath)
	createVersionTestGame(t, svc) // persist 已写入当前版本标记

	upgraded, err := UpgradeLegacySave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("marked save must be skipped without error, got: %v", err)
	}
	if upgraded {
		t.Fatalf("marked save must not be legacy-upgraded")
	}
	upgradedV2, err := UpgradeV1ToWorldSave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("current-version save must be skipped without error, got: %v", err)
	}
	if upgradedV2 {
		t.Fatalf("current-version save must not be v1-upgraded")
	}
	upgradedV3, err := UpgradeV2ToTwoSiloSave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("current-version save must be skipped without error, got: %v", err)
	}
	if upgradedV3 {
		t.Fatalf("current-version save must not be v2-upgraded")
	}
	upgradedV4, err := UpgradeV3ToFactionIdentitySave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("current-version save must be skipped without error, got: %v", err)
	}
	if upgradedV4 {
		t.Fatalf("current-version save must not be v3-upgraded")
	}
	upgradedV5, err := UpgradeV4ToPlayableWorldSave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("current-version save must be skipped without error, got: %v", err)
	}
	if upgradedV5 {
		t.Fatalf("current-version save must not be v4-upgraded")
	}
	upgradedV8, err := UpgradeV7ToAuthoritySave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("current-version save must be skipped without error, got: %v", err)
	}
	if upgradedV8 {
		t.Fatalf("current-version save must not be v7-upgraded")
	}

	var meta model.SaveMeta
	if err := svc.db.First(&meta, "`key` = ?", saveMetaKey).Error; err != nil {
		t.Fatalf("meta missing: %v", err)
	}
	if meta.LegacyUpgraded || !meta.UpgradedAt.IsZero() {
		t.Fatalf("marked save must not carry upgrade traces: %+v", meta)
	}
	matches, _ := filepath.Glob(dbPath + ".bak-*")
	if len(matches) != 0 {
		t.Fatalf("marked save must not produce backups, got %v", matches)
	}

	fresh := newVersionTestService(t, dbPath)
	if err := fresh.Resume(); err != nil {
		t.Fatalf("resume of current-version save must succeed, got: %v", err)
	}
}

// TestUpgradeV1ToWorldSave：v1 单堡档 → v2 世界模型。升级必须：建 World 行
// (种子/时钟来自玩家堡、默认节奏、Silo 1 存活)、回填 Silo.WorldID、
// The v5 endpoint is then rejected explicitly rather than treated as resumable.
func TestUpgradeV1ToWorldSave(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "v1world.db")
	svc := newVersionTestService(t, dbPath)
	createVersionTestGame(t, svc)

	// 把 v2 存档降级回 v1 形态：删 World 行、清 Silo.WorldID、版本标记回 1。
	if err := svc.db.Where("1 = 1").Delete(&model.World{}).Error; err != nil {
		t.Fatalf("strip worlds: %v", err)
	}
	if err := svc.db.Model(&model.Silo{}).Where("id = ?", ActiveSiloID).Update("world_id", 0).Error; err != nil {
		t.Fatalf("strip silo world link: %v", err)
	}
	if err := svc.db.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: 1}).Error; err != nil {
		t.Fatalf("downgrade meta: %v", err)
	}

	upgraded, err := UpgradeV1ToWorldSave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("v1 world upgrade failed: %v", err)
	}
	if !upgraded {
		t.Fatalf("expected v1 world upgrade to run")
	}

	var world model.World
	if err := svc.db.First(&world, ActiveWorldID).Error; err != nil {
		t.Fatalf("world row missing after upgrade: %v", err)
	}
	var silo model.Silo
	if err := svc.db.First(&silo, ActiveSiloID).Error; err != nil {
		t.Fatalf("silo missing after upgrade: %v", err)
	}
	if world.Seed != silo.RNGSeed {
		t.Errorf("world seed %d != silo rng seed %d", world.Seed, silo.RNGSeed)
	}
	if world.CurrentYear != silo.CurrentYear || world.CurrentMonth != silo.CurrentMonth {
		t.Errorf("world clock %d-%d != silo clock %d-%d",
			world.CurrentYear, world.CurrentMonth, silo.CurrentYear, silo.CurrentMonth)
	}
	if world.PlayerSiloID != ActiveSiloID {
		t.Errorf("world player silo = %d, want %d", world.PlayerSiloID, ActiveSiloID)
	}
	if world.Pacing != model.DefaultWorldPacing() {
		t.Errorf("world pacing = %+v, want defaults", world.Pacing)
	}
	if !world.Silo1.Alive || world.Silo1.OrderPhase != "DORMANT" {
		t.Errorf("world silo1 state = %+v, want alive DORMANT", world.Silo1)
	}
	if len(world.Topology) != 0 {
		t.Errorf("world topology must start empty, got %v", world.Topology)
	}
	if silo.WorldID != ActiveWorldID {
		t.Errorf("silo.WorldID = %d, want %d", silo.WorldID, ActiveWorldID)
	}

	var meta model.SaveMeta
	if err := svc.db.First(&meta, "`key` = ?", saveMetaKey).Error; err != nil {
		t.Fatalf("meta missing: %v", err)
	}
	if meta.SaveVersion != worldModelSaveVersion {
		t.Fatalf("meta version after v1→v2 = %d, want %d", meta.SaveVersion, worldModelSaveVersion)
	}
	matches, _ := filepath.Glob(dbPath + ".bak-*")
	if len(matches) != 1 {
		t.Fatalf("expected exactly one timestamped backup after v1→v2, got %v", matches)
	}

	// 接力 v2 → v3 (双地堡)：回填特工堡归属与跨堡队列/世界 flag，版本升到 3。
	upgradedV3, err := UpgradeV2ToTwoSiloSave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("v2 two-silo upgrade failed: %v", err)
	}
	if !upgradedV3 {
		t.Fatalf("expected v2 two-silo upgrade to run")
	}
	if err := svc.db.First(&meta, "`key` = ?", saveMetaKey).Error; err != nil {
		t.Fatalf("meta missing after v3: %v", err)
	}
	if meta.SaveVersion != twoSiloSaveVersion {
		t.Fatalf("meta version after v2→v3 = %d, want %d", meta.SaveVersion, twoSiloSaveVersion)
	}
	var agent model.Agent
	if err := svc.db.First(&agent, ActiveAgentID).Error; err != nil {
		t.Fatalf("agent missing: %v", err)
	}
	if agent.SiloID != ActiveSiloID {
		t.Fatalf("agent.SiloID = %d, want %d (v2→v3 backfill)", agent.SiloID, ActiveSiloID)
	}

	// 接力 v3 → v4 (派系持久身份)：回填 Stance/Memory/Demands，版本升到当前。
	upgradedV4, err := UpgradeV3ToFactionIdentitySave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("v3 faction identity upgrade failed: %v", err)
	}
	if !upgradedV4 {
		t.Fatalf("expected v3 faction identity upgrade to run")
	}
	if err := svc.db.First(&meta, "`key` = ?", saveMetaKey).Error; err != nil {
		t.Fatalf("meta missing after v4: %v", err)
	}
	if meta.SaveVersion != factionIdentitySaveVersion {
		t.Fatalf("meta version after v3→v4 = %d, want %d", meta.SaveVersion, factionIdentitySaveVersion)
	}
	upgradedV5, err := UpgradeV4ToPlayableWorldSave(svc.db, dbPath)
	if err != nil || !upgradedV5 {
		t.Fatalf("v4 playable world upgrade failed: upgraded=%v err=%v", upgradedV5, err)
	}

	assertV5IdentitySaveRejected(t, dbPath)
}

// TestUpgradeV2ToTwoSiloSave：v2 世界模型档 → v3 双地堡。升级必须回填玩家
// 特工 SiloID、初始化跨堡队列与世界 flag、版本升到当前、生成备份；升级后
// The final v5 save must be rejected by v6 Resume.
func TestUpgradeV2ToTwoSiloSave(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "v2two.db")
	svc := newVersionTestService(t, dbPath)
	createVersionTestGame(t, svc)

	// 把 v3 存档降级回 v2 形态：版本标记回 2、清特工堡归属。
	if err := svc.db.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: worldModelSaveVersion}).Error; err != nil {
		t.Fatalf("downgrade meta: %v", err)
	}
	if err := svc.db.Model(&model.Agent{}).Where("id = ?", ActiveAgentID).Update("silo_id", 0).Error; err != nil {
		t.Fatalf("strip agent silo link: %v", err)
	}

	upgraded, err := UpgradeV2ToTwoSiloSave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("v2 two-silo upgrade failed: %v", err)
	}
	if !upgraded {
		t.Fatalf("expected v2 two-silo upgrade to run")
	}

	var meta model.SaveMeta
	if err := svc.db.First(&meta, "`key` = ?", saveMetaKey).Error; err != nil {
		t.Fatalf("meta missing: %v", err)
	}
	if meta.SaveVersion != twoSiloSaveVersion {
		t.Fatalf("meta version = %d, want %d", meta.SaveVersion, twoSiloSaveVersion)
	}
	var agent model.Agent
	if err := svc.db.First(&agent, ActiveAgentID).Error; err != nil {
		t.Fatalf("agent missing: %v", err)
	}
	if agent.SiloID != ActiveSiloID {
		t.Fatalf("agent.SiloID = %d, want %d", agent.SiloID, ActiveSiloID)
	}
	var world model.World
	if err := svc.db.First(&world, ActiveWorldID).Error; err != nil {
		t.Fatalf("world missing: %v", err)
	}
	if world.CrossQueue == nil || world.Flags == nil {
		t.Fatalf("world cross-queue/flags must be initialized, got %+v", world)
	}
	matches, _ := filepath.Glob(dbPath + ".bak-*v3twosilo")
	if len(matches) != 1 {
		t.Fatalf("expected exactly one v3 backup, got %v", matches)
	}

	// Chain through v4 and v5, then assert the explicit v6 rejection.
	upgradedV4, err := UpgradeV3ToFactionIdentitySave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("v3 faction identity upgrade failed: %v", err)
	}
	if !upgradedV4 {
		t.Fatalf("expected v3 faction identity upgrade to run")
	}
	upgradedV5, err := UpgradeV4ToPlayableWorldSave(svc.db, dbPath)
	if err != nil || !upgradedV5 {
		t.Fatalf("v4 playable world upgrade failed: upgraded=%v err=%v", upgradedV5, err)
	}

	assertV5IdentitySaveRejected(t, dbPath)
}

// TestUpgradeV3ToFactionIdentitySave：v3 双地堡档 → v4 派系持久身份。升级必须
// 回填 Stance/Memory/Demands (由旧签名派生)、保留既有派系 ID、版本升到当前、
// Backups and faction fields remain intact, but the resulting v5 save is not resumable.
func TestUpgradeV3ToFactionIdentitySave(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "v3faction.db")
	svc := newVersionTestService(t, dbPath)
	createVersionTestGame(t, svc)
	// 推进一个月，确保至少存在无阵营桶以外的派系状态可观察。
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("pass time: %v", err)
	}
	if err := svc.persist(); err != nil {
		t.Fatalf("persist: %v", err)
	}
	// 确定性插入一个政治派系行 (v3 形态：无身份字段)，保证回填路径被覆盖。
	// 必须在 persist 之后插入：persist 是整图替换式落库。
	politicalFaction := model.Faction{
		SiloID:      ActiveSiloID,
		Name:        "High Proforeign Circle",
		Signature:   "pro_foreign:High|drive:care",
		Tags:        []string{"pro_foreign:High", "drive:care"},
		MemberCount: 500,
	}
	if err := svc.db.Create(&politicalFaction).Error; err != nil {
		t.Fatalf("seed political faction: %v", err)
	}

	// 降级回 v3 形态：版本标记回 3，清空派系身份字段 (模拟旧档无这些列值)。
	if err := svc.db.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: twoSiloSaveVersion}).Error; err != nil {
		t.Fatalf("downgrade meta: %v", err)
	}
	if err := svc.db.Model(&model.Faction{}).Where("1 = 1").
		Updates(map[string]interface{}{"stance": "", "memory": nil, "demands": nil}).Error; err != nil {
		t.Fatalf("strip faction identity fields: %v", err)
	}

	upgraded, err := UpgradeV3ToFactionIdentitySave(svc.db, dbPath)
	if err != nil {
		t.Fatalf("v3 faction identity upgrade failed: %v", err)
	}
	if !upgraded {
		t.Fatalf("expected v3 faction identity upgrade to run")
	}

	var meta model.SaveMeta
	if err := svc.db.First(&meta, "`key` = ?", saveMetaKey).Error; err != nil {
		t.Fatalf("meta missing: %v", err)
	}
	if meta.SaveVersion != factionIdentitySaveVersion {
		t.Fatalf("meta version = %d, want %d", meta.SaveVersion, factionIdentitySaveVersion)
	}
	var factions []model.Faction
	if err := svc.db.Find(&factions).Error; err != nil {
		t.Fatalf("load factions: %v", err)
	}
	if len(factions) == 0 {
		t.Fatalf("expected faction rows")
	}
	for _, f := range factions {
		if len(f.Memory) == 0 || f.Memory[0] != "upgraded:from_v3" {
			t.Fatalf("faction %d memory not backfilled: %v", f.ID, f.Memory)
		}
		if f.Signature != "special:unaffiliated" {
			if f.Stance == "" {
				t.Fatalf("political faction %d stance not backfilled (signature %q)", f.ID, f.Signature)
			}
			if len(f.Demands) == 0 {
				t.Fatalf("political faction %d demands not backfilled (signature %q)", f.ID, f.Signature)
			}
		}
	}
	matches, _ := filepath.Glob(dbPath + ".bak-*v4faction")
	if len(matches) != 1 {
		t.Fatalf("expected exactly one v4 backup, got %v", matches)
	}
	upgradedV5, err := UpgradeV4ToPlayableWorldSave(svc.db, dbPath)
	if err != nil || !upgradedV5 {
		t.Fatalf("v4 playable world upgrade failed: upgraded=%v err=%v", upgradedV5, err)
	}

	assertV5IdentitySaveRejected(t, dbPath)
}

func TestUpgradeV4ToPlayableWorldSave(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "v4playable.db")
	svc := newVersionTestService(t, dbPath)
	createVersionTestGame(t, svc)
	var factionsBefore []model.Faction
	if err := svc.db.Find(&factionsBefore).Error; err != nil {
		t.Fatalf("load factions: %v", err)
	}
	var world model.World
	if err := svc.db.First(&world, ActiveWorldID).Error; err != nil {
		t.Fatalf("load world: %v", err)
	}
	world.Silo1.Operations = nil
	world.StationOps.Operations = nil
	world.HistoryImpacts = nil
	if err := svc.db.Save(&world).Error; err != nil {
		t.Fatalf("strip playable fields: %v", err)
	}
	if err := svc.db.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: factionIdentitySaveVersion}).Error; err != nil {
		t.Fatalf("downgrade meta: %v", err)
	}

	upgraded, err := UpgradeV4ToPlayableWorldSave(svc.db, dbPath)
	if err != nil || !upgraded {
		t.Fatalf("v4 playable world upgrade failed: upgraded=%v err=%v", upgraded, err)
	}
	var meta model.SaveMeta
	if err := svc.db.First(&meta, "`key` = ?", saveMetaKey).Error; err != nil {
		t.Fatalf("load upgraded meta: %v", err)
	}
	if meta.SaveVersion != playableWorldSaveVersion {
		t.Fatalf("meta version = %d, want %d", meta.SaveVersion, playableWorldSaveVersion)
	}
	if err := svc.db.First(&world, ActiveWorldID).Error; err != nil {
		t.Fatalf("reload world: %v", err)
	}
	if world.Silo1.Operations == nil || world.StationOps.Operations == nil || world.HistoryImpacts == nil {
		t.Fatal("v5 playable fields were not initialized")
	}
	var factionsAfter []model.Faction
	if err := svc.db.Find(&factionsAfter).Error; err != nil {
		t.Fatalf("reload factions: %v", err)
	}
	if !reflect.DeepEqual(factionsBefore, factionsAfter) {
		t.Fatal("v4 -> v5 changed faction rows")
	}
	var siloCount int64
	if err := svc.db.Model(&model.Silo{}).Count(&siloCount).Error; err != nil || siloCount != 1 {
		t.Fatalf("v4 -> v5 changed silo composition: count=%d err=%v", siloCount, err)
	}
	matches, _ := filepath.Glob(dbPath + ".bak-*v5playable")
	if len(matches) != 1 {
		t.Fatalf("expected one v5 backup, got %v", matches)
	}
	assertV5IdentitySaveRejected(t, dbPath)
}

func TestUpgradeV4ToPlayableWorldTransactionRollsBack(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "v5rollback.db")
	svc := newVersionTestService(t, dbPath)
	createVersionTestGame(t, svc)
	var world model.World
	if err := svc.db.First(&world, ActiveWorldID).Error; err != nil {
		t.Fatalf("load world: %v", err)
	}
	world.Silo1.Operations = nil
	world.StationOps.Operations = nil
	world.HistoryImpacts = nil
	if err := svc.db.Save(&world).Error; err != nil {
		t.Fatalf("strip playable fields: %v", err)
	}
	if err := svc.db.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: factionIdentitySaveVersion}).Error; err != nil {
		t.Fatalf("downgrade meta: %v", err)
	}
	if err := backupSaveFile(svc.db, dbPath+".bak-test"); err != nil {
		t.Fatalf("backup: %v", err)
	}
	if err := svc.db.Migrator().DropTable(&model.SaveMeta{}); err != nil {
		t.Fatalf("drop save meta: %v", err)
	}
	if err := svc.db.Transaction(upgradeV4ToPlayableWorld); err == nil {
		t.Fatal("transaction must fail with save meta table dropped")
	}
	if err := svc.db.First(&world, ActiveWorldID).Error; err != nil {
		t.Fatalf("reload world: %v", err)
	}
	if world.Silo1.Operations != nil || world.StationOps.Operations != nil || world.HistoryImpacts != nil {
		t.Fatal("failed v5 transaction left partial world changes")
	}
}

func TestResumeStillRejectsHigherVersion(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "future.db")
	svc := newVersionTestService(t, dbPath)
	createVersionTestGame(t, svc)
	if err := svc.db.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: model.CurrentSaveVersion + 1}).Error; err != nil {
		t.Fatalf("bump meta: %v", err)
	}

	// 更高版本不触发升级（原样跳过），仍由 Resume 显式拒绝。
	upgraded, err := UpgradeLegacySave(svc.db, dbPath)
	if upgraded || err != nil {
		t.Fatalf("higher-version save must be skipped as-is, got upgraded=%v err=%v", upgraded, err)
	}
	fresh := newVersionTestService(t, dbPath)
	if err := fresh.Resume(); !errors.Is(err, ErrIncompatibleSave) {
		t.Fatalf("expected ErrIncompatibleSave for higher version, got: %v", err)
	}
}
