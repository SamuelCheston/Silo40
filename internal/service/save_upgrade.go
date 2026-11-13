package service

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"silo40/internal/engine"
	"silo40/internal/model"

	"gorm.io/gorm"
)

// UpgradeLegacySave 检测无版本标记的旧存档并执行一次性显式升级 (legacy → v1)。
// 注意：它只补写到 v1；v1 → v2 (世界模型) 由 UpgradeV1ToWorldSave 接力，
// v2 → v3 (双地堡) 由 UpgradeV2ToTwoSiloSave 接力，v3 → v4 (派系持久身份)
// 由 UpgradeV3ToFactionIdentitySave 接力完成，启动链路依次调用四者，
// 每一步都独立留痕、可独立回滚。
//
// 流程：先对数据库做一致性备份（SQLite VACUUM INTO，文件名带时间戳），再在
// 单个事务里补写版本标记并回读校验，校验通过才提交。任何一步失败：事务回滚、
// 数据库保持原样（不存在半迁移状态），已生成的备份文件留在原地可人工还原。
//
// 升级会留痕：SaveMeta.LegacyUpgraded/UpgradedAt 写入档内，并打印日志。
//
// 已有版本标记（任何版本）或根本没有存档的情况一律原样跳过（返回 false, nil），
// 由后续升级步骤与 Resume 的版本闸门按既有语义处理。
func UpgradeLegacySave(db *gorm.DB, dbPath string) (bool, error) {
	legacy, err := isLegacySave(db)
	if err != nil {
		return false, fmt.Errorf("legacy upgrade: %w", err)
	}
	if !legacy {
		return false, nil
	}

	// 1. 一致性备份（VACUUM INTO 对打开中的库安全，生成独立快照文件）。
	backupPath := fmt.Sprintf("%s.bak-%s", dbPath, time.Now().Format("20060102-150405"))
	if err := backupSaveFile(db, backupPath); err != nil {
		return false, fmt.Errorf("legacy upgrade: backup to %s failed: %w（数据库未改动）", backupPath, err)
	}

	// 2. 事务内补写标记 + 回读校验，通过才提交；失败自动回滚，库保持原样。
	if err := db.Transaction(writeLegacyVersionMarker); err != nil {
		return false, fmt.Errorf("legacy upgrade: %w（数据库未改动，备份保留在 %s，可用它还原）", err, backupPath)
	}

	log.Printf("legacy save upgraded to v1 (backup: %s)——本局存档由无版本标记的旧档一次性显式升级", backupPath)
	return true, nil
}

// UpgradeV1ToWorldSave 检测 v1 存档 (单堡、无 World 层) 并执行一次性显式
// 升级 (v1 → v2)：把单堡包成只含玩家堡的 World (空邻堡配置)。
//
// 与 UpgradeLegacySave 同构的安全语义：先一致性备份，再在单个事务里建
// World 行、回填 Silo.WorldID、提升版本标记并回读校验；任何一步失败则
// 事务回滚、库保持原样、备份留存。升级后空邻堡配置的行为与 v1 逐位一致。
//
// 已是 v2、版本更高、无存档的情况原样跳过（返回 false, nil）。
func UpgradeV1ToWorldSave(db *gorm.DB, dbPath string) (bool, error) {
	v1, err := isV1Save(db)
	if err != nil {
		return false, fmt.Errorf("v1 world upgrade: %w", err)
	}
	if !v1 {
		return false, nil
	}

	// 阶段后缀避免与同一秒内 legacy 升级的备份文件名冲突。
	backupPath := fmt.Sprintf("%s.bak-%s-v2world", dbPath, time.Now().Format("20060102-150405"))
	if err := backupSaveFile(db, backupPath); err != nil {
		return false, fmt.Errorf("v1 world upgrade: backup to %s failed: %w（数据库未改动）", backupPath, err)
	}

	if err := db.Transaction(upgradeV1ToWorld); err != nil {
		return false, fmt.Errorf("v1 world upgrade: %w（数据库未改动，备份保留在 %s，可用它还原）", err, backupPath)
	}

	log.Printf("v1 save upgraded to v%d world model (backup: %s)——单堡存档已包装为只含玩家堡的世界",
		worldModelSaveVersion, backupPath)
	return true, nil
}

// isV1Save 判定是否为带 v1 版本标记的存档。无存档 / 其他版本都返回 false。
func isV1Save(db *gorm.DB) (bool, error) {
	var meta model.SaveMeta
	err := db.First(&meta, "`key` = ?", saveMetaKey).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read save meta: %w", err)
	}
	return meta.SaveVersion == 1, nil
}

// upgradeV1ToWorld 在事务内完成 v1 → v2 迁移并回读校验；校验不过返回
// 错误触发回滚，保证不出现半迁移状态。
func upgradeV1ToWorld(tx *gorm.DB) error {
	var silo model.Silo
	if err := tx.First(&silo, ActiveSiloID).Error; err != nil {
		return fmt.Errorf("load silo %d: %w", ActiveSiloID, err)
	}

	world := model.World{
		ID:           ActiveWorldID,
		Seed:         silo.RNGSeed,
		CurrentYear:  silo.CurrentYear,
		CurrentMonth: silo.CurrentMonth,
		PlayerSiloID: ActiveSiloID,
		Pacing:       model.DefaultWorldPacing(),
		Silo1: model.Silo1State{
			Alive:      !silo.Silo1Destroyed,
			OrderPhase: "DORMANT",
			Attention:  map[uint]float64{},
		},
		Topology:   map[uint][]uint{},
		CrossQueue: []model.CrossEvent{},
		Flags:      map[string]bool{},
	}
	if err := tx.Create(&world).Error; err != nil {
		return fmt.Errorf("create world: %w", err)
	}
	if err := tx.Model(&model.Silo{}).Where("id = ?", ActiveSiloID).Update("world_id", ActiveWorldID).Error; err != nil {
		return fmt.Errorf("link silo to world: %w", err)
	}
	// v1 → v2 只升到世界模型版本 (2)；v2 → v3 (双地堡) 由
	// UpgradeV2ToTwoSiloSave 接力，保证每步独立留痕、可独立回滚。
	if err := tx.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: worldModelSaveVersion}).Error; err != nil {
		return fmt.Errorf("bump save meta: %w", err)
	}

	// 回读校验：World 存在、Silo 已关联、版本标记正确。
	var checkWorld model.World
	if err := tx.First(&checkWorld, ActiveWorldID).Error; err != nil {
		return fmt.Errorf("post-write verification failed: world missing: %w", err)
	}
	var checkSilo model.Silo
	if err := tx.First(&checkSilo, ActiveSiloID).Error; err != nil {
		return fmt.Errorf("post-write verification failed: silo missing: %w", err)
	}
	if checkSilo.WorldID != ActiveWorldID {
		return fmt.Errorf("post-write verification failed: silo.WorldID = %d, want %d", checkSilo.WorldID, ActiveWorldID)
	}
	var checkMeta model.SaveMeta
	if err := tx.First(&checkMeta, "`key` = ?", saveMetaKey).Error; err != nil {
		return fmt.Errorf("post-write verification failed: meta missing: %w", err)
	}
	if checkMeta.SaveVersion != worldModelSaveVersion {
		return fmt.Errorf("post-write verification failed: version = %d, want %d", checkMeta.SaveVersion, worldModelSaveVersion)
	}
	return nil
}

// UpgradeV2ToTwoSiloSave 检测 v2 存档 (世界层已存在但无跨堡队列/世界 flag/
// 特工堡归属) 并执行一次性显式升级 (v2 → v3)。新列由 AutoMigrate 在启动时
// 补齐，本升级器只负责语义回填：玩家特工 SiloID = 玩家堡、世界 flag 表与
// 跨堡队列置空、版本标记提升。单堡 (空邻堡) 配置的行为与 v2 逐位一致。
//
// 与前两档升级同构的安全语义：先一致性备份，再在单个事务里回填并回读校验；
// 任何一步失败则事务回滚、库保持原样、备份留存。
//
// 已是 v3、版本更高、无存档的情况原样跳过（返回 false, nil）。
func UpgradeV2ToTwoSiloSave(db *gorm.DB, dbPath string) (bool, error) {
	v2, err := isV2Save(db)
	if err != nil {
		return false, fmt.Errorf("v3 two-silo upgrade: %w", err)
	}
	if !v2 {
		return false, nil
	}

	backupPath := fmt.Sprintf("%s.bak-%s-v3twosilo", dbPath, time.Now().Format("20060102-150405"))
	if err := backupSaveFile(db, backupPath); err != nil {
		return false, fmt.Errorf("v3 two-silo upgrade: backup to %s failed: %w（数据库未改动）", backupPath, err)
	}

	if err := db.Transaction(upgradeV2ToTwoSilo); err != nil {
		return false, fmt.Errorf("v3 two-silo upgrade: %w（数据库未改动，备份保留在 %s，可用它还原）", err, backupPath)
	}

	log.Printf("v2 save upgraded to v%d two-silo model (backup: %s)——世界层已具备跨堡队列与世界 flag",
		twoSiloSaveVersion, backupPath)
	return true, nil
}

// isV2Save 判定是否为带 v2 版本标记的存档。无存档 / 其他版本都返回 false。
func isV2Save(db *gorm.DB) (bool, error) {
	var meta model.SaveMeta
	err := db.First(&meta, "`key` = ?", saveMetaKey).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read save meta: %w", err)
	}
	return meta.SaveVersion == 2, nil
}

// upgradeV2ToTwoSilo 在事务内完成 v2 → v3 回填并回读校验；校验不过返回错误
// 触发回滚，保证不出现半迁移状态。
func upgradeV2ToTwoSilo(tx *gorm.DB) error {
	// 玩家特工回填堡归属 (v2 存档只有一个特工，固定服务玩家堡)。
	if err := tx.Model(&model.Agent{}).Where("id = ?", ActiveAgentID).Update("silo_id", ActiveSiloID).Error; err != nil {
		return fmt.Errorf("backfill agent silo: %w", err)
	}

	// 世界行必须存在 (v2 的不变量)；跨堡队列/序号/flag 表初始化为空。
	var world model.World
	if err := tx.First(&world, ActiveWorldID).Error; err != nil {
		return fmt.Errorf("load world %d: %w", ActiveWorldID, err)
	}
	if world.CrossQueue == nil {
		world.CrossQueue = []model.CrossEvent{}
	}
	if world.Flags == nil {
		world.Flags = map[string]bool{}
	}
	if err := tx.Save(&world).Error; err != nil {
		return fmt.Errorf("persist world cross-queue state: %w", err)
	}

	// v2 → v3 只升到双地堡版本 (3)；v3 → v4 (派系持久身份) 由
	// UpgradeV3ToFactionIdentitySave 接力，保证每步独立留痕、可独立回滚。
	if err := tx.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: twoSiloSaveVersion}).Error; err != nil {
		return fmt.Errorf("bump save meta: %w", err)
	}

	// 回读校验。
	var checkAgent model.Agent
	if err := tx.First(&checkAgent, ActiveAgentID).Error; err != nil {
		return fmt.Errorf("post-write verification failed: agent missing: %w", err)
	}
	if checkAgent.SiloID != ActiveSiloID {
		return fmt.Errorf("post-write verification failed: agent.SiloID = %d, want %d", checkAgent.SiloID, ActiveSiloID)
	}
	var checkMeta model.SaveMeta
	if err := tx.First(&checkMeta, "`key` = ?", saveMetaKey).Error; err != nil {
		return fmt.Errorf("post-write verification failed: meta missing: %w", err)
	}
	if checkMeta.SaveVersion != twoSiloSaveVersion {
		return fmt.Errorf("post-write verification failed: version = %d, want %d", checkMeta.SaveVersion, twoSiloSaveVersion)
	}
	return nil
}

// UpgradeV3ToFactionIdentitySave 检测 v3 存档 (双地堡/18 号链条，派系无持久
// 身份字段) 并执行一次性显式升级 (v3 → v4)：派系既有 ID 原样保留为持久
// 身份，回填 Stance/Memory/Demands (由旧签名派生)，版本标记提升。
//
// 与前几档升级同构的安全语义：先一致性备份，再在单个事务里回填并回读校验；
// 任何一步失败则事务回滚、库保持原样、备份留存。
//
// 已是 v4、版本更高、无存档的情况原样跳过（返回 false, nil）。
func UpgradeV3ToFactionIdentitySave(db *gorm.DB, dbPath string) (bool, error) {
	v3, err := isV3Save(db)
	if err != nil {
		return false, fmt.Errorf("v4 faction identity upgrade: %w", err)
	}
	if !v3 {
		return false, nil
	}

	backupPath := fmt.Sprintf("%s.bak-%s-v4faction", dbPath, time.Now().Format("20060102-150405"))
	if err := backupSaveFile(db, backupPath); err != nil {
		return false, fmt.Errorf("v4 faction identity upgrade: backup to %s failed: %w（数据库未改动）", backupPath, err)
	}

	if err := db.Transaction(upgradeV3ToFactionIdentity); err != nil {
		return false, fmt.Errorf("v4 faction identity upgrade: %w（数据库未改动，备份保留在 %s，可用它还原）", err, backupPath)
	}

	log.Printf("v3 save upgraded to v%d faction identity (backup: %s)——派系获得持久身份 (Stance/Memory/Demands 已回填)",
		factionIdentitySaveVersion, backupPath)
	return true, nil
}

// isV3Save 判定是否为带 v3 版本标记的存档。无存档 / 其他版本都返回 false。
func isV3Save(db *gorm.DB) (bool, error) {
	var meta model.SaveMeta
	err := db.First(&meta, "`key` = ?", saveMetaKey).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read save meta: %w", err)
	}
	return meta.SaveVersion == 3, nil
}

// upgradeV3ToFactionIdentity 在事务内完成 v3 → v4 回填并回读校验；校验不过
// 返回错误触发回滚，保证不出现半迁移状态。
func upgradeV3ToFactionIdentity(tx *gorm.DB) error {
	var factions []model.Faction
	if err := tx.Find(&factions).Error; err != nil {
		return fmt.Errorf("load factions: %w", err)
	}
	for i := range factions {
		f := &factions[i]
		if f.Signature == legacyUnaffiliatedSignature {
			f.Memory = []string{"upgraded:from_v3"}
		} else {
			f.Stance, f.Demands = parseLegacyFactionSignature(f.Signature)
			f.Memory = []string{"upgraded:from_v3"}
		}
		if err := tx.Save(f).Error; err != nil {
			return fmt.Errorf("backfill faction %d: %w", f.ID, err)
		}
	}

	if err := tx.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: factionIdentitySaveVersion}).Error; err != nil {
		return fmt.Errorf("bump save meta: %w", err)
	}

	// 回读校验。
	var checkMeta model.SaveMeta
	if err := tx.First(&checkMeta, "`key` = ?", saveMetaKey).Error; err != nil {
		return fmt.Errorf("post-write verification failed: meta missing: %w", err)
	}
	if checkMeta.SaveVersion != factionIdentitySaveVersion {
		return fmt.Errorf("post-write verification failed: version = %d, want %d", checkMeta.SaveVersion, model.CurrentSaveVersion)
	}
	var checkFactions []model.Faction
	if err := tx.Find(&checkFactions).Error; err != nil {
		return fmt.Errorf("post-write verification failed: reload factions: %w", err)
	}
	for _, f := range checkFactions {
		if len(f.Memory) == 0 {
			return fmt.Errorf("post-write verification failed: faction %d missing memory backfill", f.ID)
		}
	}
	return nil
}

// UpgradeV4ToPlayableWorldSave upgrades only the new playable-world JSON
// fields. It deliberately does not inject Silo 1 or alter faction rows, so an
// existing v4 timeline keeps its original simulation semantics.
func UpgradeV4ToPlayableWorldSave(db *gorm.DB, dbPath string) (bool, error) {
	v4, err := isV4Save(db)
	if err != nil {
		return false, fmt.Errorf("v5 playable world upgrade: %w", err)
	}
	if !v4 {
		return false, nil
	}

	backupPath := fmt.Sprintf("%s.bak-%s-v5playable", dbPath, time.Now().Format("20060102-150405"))
	if err := backupSaveFile(db, backupPath); err != nil {
		return false, fmt.Errorf("v5 playable world upgrade: backup to %s failed: %w（数据库未改动）", backupPath, err)
	}
	if err := db.Transaction(upgradeV4ToPlayableWorld); err != nil {
		return false, fmt.Errorf("v5 playable world upgrade: %w（数据库未改动，备份保留在 %s，可用它还原）", err, backupPath)
	}
	log.Printf("v4 save upgraded to v%d playable world (backup: %s)——operation 与历史评分字段已初始化，既有派系和堡构成不变",
		playableWorldSaveVersion, backupPath)
	return true, nil
}

func isV4Save(db *gorm.DB) (bool, error) {
	var meta model.SaveMeta
	err := db.First(&meta, "`key` = ?", saveMetaKey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read save meta: %w", err)
	}
	return meta.SaveVersion == factionIdentitySaveVersion, nil
}

func upgradeV4ToPlayableWorld(tx *gorm.DB) error {
	var world model.World
	if err := tx.First(&world, ActiveWorldID).Error; err != nil {
		return fmt.Errorf("load world %d: %w", ActiveWorldID, err)
	}
	if world.Silo1.Operations == nil {
		world.Silo1.Operations = []model.Silo1Operation{}
	}
	if world.StationOps.Operations == nil {
		world.StationOps.Operations = []model.Silo1Operation{}
	}
	if world.HistoryImpacts == nil {
		world.HistoryImpacts = []model.HistoryImpact{}
	}
	if err := tx.Save(&world).Error; err != nil {
		return fmt.Errorf("initialize playable world fields: %w", err)
	}
	if err := tx.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: playableWorldSaveVersion}).Error; err != nil {
		return fmt.Errorf("bump save meta: %w", err)
	}

	var checkWorld model.World
	if err := tx.First(&checkWorld, ActiveWorldID).Error; err != nil {
		return fmt.Errorf("post-write verification failed: world missing: %w", err)
	}
	if checkWorld.Silo1.Operations == nil || checkWorld.StationOps.Operations == nil || checkWorld.HistoryImpacts == nil {
		return fmt.Errorf("post-write verification failed: playable fields were not initialized")
	}
	var checkMeta model.SaveMeta
	if err := tx.First(&checkMeta, "`key` = ?", saveMetaKey).Error; err != nil {
		return fmt.Errorf("post-write verification failed: meta missing: %w", err)
	}
	if checkMeta.SaveVersion != playableWorldSaveVersion {
		return fmt.Errorf("post-write verification failed: version = %d, want %d", checkMeta.SaveVersion, playableWorldSaveVersion)
	}
	return nil
}

// UpgradeV7ToAuthoritySave adds independent Mayor/IT/Judicial authority state.
// It uses the same backup + one transaction + readback contract as earlier upgrades.
func UpgradeV7ToAuthoritySave(db *gorm.DB, dbPath string) (bool, error) {
	var meta model.SaveMeta
	err := db.First(&meta, "`key` = ?", saveMetaKey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("v8 authority upgrade: read save meta: %w", err)
	}
	if meta.SaveVersion != 7 {
		return false, nil
	}
	backupPath := fmt.Sprintf("%s.bak-%s-v8authority", dbPath, time.Now().Format("20060102-150405"))
	if err := backupSaveFile(db, backupPath); err != nil {
		return false, fmt.Errorf("v8 authority upgrade: backup to %s failed: %w（数据库未改动）", backupPath, err)
	}
	if err := db.Transaction(upgradeV7ToAuthority); err != nil {
		return false, fmt.Errorf("v8 authority upgrade: %w（数据库未改动，备份保留在 %s，可用它还原）", err, backupPath)
	}
	log.Printf("v7 save upgraded to v8 authority model (backup: %s)——Mayor 正统性继承旧值，W 使用保守初值 50", backupPath)
	return true, nil
}

func upgradeV7ToAuthority(tx *gorm.DB) error {
	var silos []model.Silo
	if err := tx.Order("id ASC").Find(&silos).Error; err != nil {
		return fmt.Errorf("load silos: %w", err)
	}
	legacyLegitimacy := make(map[uint]float64, len(silos))
	for i := range silos {
		silo := &silos[i]
		legacyLegitimacy[silo.ID] = silo.Legitimacy
		engine.InitializeAuthorityState(silo, true)
		if err := tx.Save(silo).Error; err != nil {
			return fmt.Errorf("backfill silo %d authority: %w", silo.ID, err)
		}
	}
	if err := tx.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: authoritySaveVersion}).Error; err != nil {
		return fmt.Errorf("bump save meta: %w", err)
	}

	var checkSilos []model.Silo
	if err := tx.Order("id ASC").Find(&checkSilos).Error; err != nil {
		return fmt.Errorf("post-write verification failed: reload silos: %w", err)
	}
	if len(checkSilos) != len(silos) {
		return fmt.Errorf("post-write verification failed: silo count changed")
	}
	for i := range checkSilos {
		silo := &checkSilos[i]
		if !silo.Mayor.Initialized || !silo.Authority.Initialized {
			return fmt.Errorf("post-write verification failed: silo %d authority is uninitialized", silo.ID)
		}
		if silo.MayorLegitimacy() != legacyLegitimacy[silo.ID] || silo.Legitimacy != legacyLegitimacy[silo.ID] {
			return fmt.Errorf("post-write verification failed: silo %d Mayor legitimacy mismatch", silo.ID)
		}
		if silo.Authority.W != engine.DefaultAuthorityConfig().LegacyConservativeW || silo.Authority.InitializationSource != "v7_conservative_w50" {
			return fmt.Errorf("post-write verification failed: silo %d conservative W mismatch", silo.ID)
		}
	}
	var checkMeta model.SaveMeta
	if err := tx.First(&checkMeta, "`key` = ?", saveMetaKey).Error; err != nil {
		return fmt.Errorf("post-write verification failed: meta missing: %w", err)
	}
	if checkMeta.SaveVersion != authoritySaveVersion {
		return fmt.Errorf("post-write verification failed: version = %d, want %d", checkMeta.SaveVersion, authoritySaveVersion)
	}
	return nil
}

const authoritySaveVersion = 8

// legacyUnaffiliatedSignature 与 engine.unaffiliatedSignature 一致 (跨包不导出，
// 升级器按字面值对齐)。
const legacyUnaffiliatedSignature = "special:unaffiliated"

// parseLegacyFactionSignature 从 v3 时代签名 ("pro_foreign:High|drive:care")
// 派生立场与诉求；空档位 → status_quo。
func parseLegacyFactionSignature(signature string) (string, []string) {
	profile := []string{}
	var demands []string
	for _, part := range strings.Split(signature, "|") {
		if strings.HasPrefix(part, "drive:") {
			demands = append(demands, "demand:"+strings.TrimPrefix(part, "drive:"))
			continue
		}
		if part != "" {
			profile = append(profile, part)
		}
	}
	if len(profile) == 0 {
		return "status_quo", demands
	}
	return strings.Join(profile, "|"), demands
}

// isLegacySave 判定是否为"有游戏数据但无版本标记"的旧档。
// 无存档 / 已有标记（任何版本）都返回 false，交由 Resume 闸门处理。
func isLegacySave(db *gorm.DB) (bool, error) {
	var siloCount int64
	if err := db.Model(&model.Silo{}).Count(&siloCount).Error; err != nil {
		return false, fmt.Errorf("count silos: %w", err)
	}
	if siloCount == 0 {
		return false, nil
	}
	var meta model.SaveMeta
	err := db.First(&meta, "`key` = ?", saveMetaKey).Error
	if err == nil {
		return false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return false, fmt.Errorf("read save meta: %w", err)
	}
	return true, nil
}

// backupSaveFile 用 VACUUM INTO 生成一致性备份（对打开中的库安全）。
func backupSaveFile(db *gorm.DB, backupPath string) error {
	return db.Exec("VACUUM INTO ?", backupPath).Error
}

// legacyMarkerVersion 是 legacy 升级补写的版本：固定为 1 (legacy → v1 这一
// 步只补历史标记，v1 → 当前版本的迁移由后续升级步骤接力)。
const legacyMarkerVersion = 1

// worldModelSaveVersion 是 v1 → v2 升级写入的版本：固定为 2 (世界模型)。
// v2 → v3 (双地堡) 由 UpgradeV2ToTwoSiloSave 接力，避免一步跳到当前版本而
// 跳过中间回填 (agent.SiloID / 跨堡队列 / 世界 flag)。
const worldModelSaveVersion = 2

// twoSiloSaveVersion 是 v2 → v3 升级写入的版本：固定为 3 (双地堡)。
// v3 → v4 (派系持久身份) 由 UpgradeV3ToFactionIdentitySave 接力，保证升级链
// 每一步独立留痕、可独立回滚。
const twoSiloSaveVersion = 3

// factionIdentitySaveVersion is the fixed v3 -> v4 destination. Later
// upgrades must not make the v4 migration skip its own verification by
// writing the newest save version directly.
const factionIdentitySaveVersion = 4

// playableWorldSaveVersion is the fixed v4 -> v5 destination.
const playableWorldSaveVersion = 5

// writeLegacyVersionMarker 在事务内补写版本标记并回读校验；校验不过返回错误
// 触发事务回滚，保证不出现半迁移状态。
func writeLegacyVersionMarker(tx *gorm.DB) error {
	row := model.SaveMeta{
		Key:            saveMetaKey,
		SaveVersion:    legacyMarkerVersion,
		LegacyUpgraded: true,
		UpgradedAt:     time.Now(),
	}
	if err := tx.Create(&row).Error; err != nil {
		return err
	}
	var check model.SaveMeta
	if err := tx.First(&check, "`key` = ?", saveMetaKey).Error; err != nil {
		return fmt.Errorf("post-write verification failed: %w", err)
	}
	if check.SaveVersion != legacyMarkerVersion {
		return fmt.Errorf("post-write verification failed: version = %d, want %d",
			check.SaveVersion, legacyMarkerVersion)
	}
	return nil
}
