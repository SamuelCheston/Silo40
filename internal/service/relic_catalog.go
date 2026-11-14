package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// 遗物清单 (events/relics.json)：文件驱动遗物定义，开局实例化。
// 加载/校验复刻 action_gates 的 fail-closed 纪律：缺失、重名、位置引用
// 不存在、权重越界、碎片不在白名单都在 Bootstrap 期显式报错，不做跳过
// 坏条目的降级。

// relicSpawnBudget 声明每局开局抽取的遗物数量区间 (内容侧可调)。
type relicSpawnBudget struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// relicLocation 位置词表条目：设计层藏匿地点，向下锚定既有空间结构。
// Zone 锚定楼层分区 (Upper/Mid/Lower)，Dept 锚定部门名，可为空。
type relicLocation struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Zone string `json:"zone"`
	Dept string `json:"dept"`
}

// relicDefinition 单条遗物定义。字段即文件契约：DisallowUnknownFields
// 保证新属性必须先升版本加代码支持，不允许塞无约束字典。
type relicDefinition struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Category    string             `json:"category"`
	Locations   []string           `json:"locations"`
	Weight      int                `json:"weight"`
	Effects     map[string]float64 `json:"effects"`
	Fragments   []string           `json:"fragments"`
	Confiscable bool               `json:"confiscable"`
	Registrable bool               `json:"registrable"`
}

type relicFile struct {
	Version   int               `json:"version"`
	Spawn     relicSpawnBudget  `json:"spawn"`
	Locations []relicLocation   `json:"locations"`
	Relics    []relicDefinition `json:"relics"`
}

// relicCatalogRuntime 编译后的遗物目录。Relics 保持文件声明顺序——
// 开局抽取按该固定顺序消费 RNG，保证同种子同文件可复现。
type relicCatalogRuntime struct {
	Enabled     bool
	Spawn       relicSpawnBudget
	Locations   map[string]relicLocation
	Relics      []relicDefinition
	TotalWeight int
}

// relicCategories 实物类型枚举 (研究文档 10-it-control-credibility §A)。
var relicCategories = map[string]bool{
	"storage_medium": true, // 禁忌电子存储介质
	"personal_item":  true, // 个人随身物品
	"daily_goods":    true, // 日用品与包装
	"imaging_device": true, // 影像记录设备
	"geography_book": true, // 旧世界地理与生活读物
	"old_book":       true, // 旧版书籍
}

// relicZones 位置词表允许的楼层分区锚点 (engine.initFloors 的 Zone 词表)。
var relicZones = map[string]bool{"Upper": true, "Mid": true, "Lower": true}

// relicDepts 位置词表允许的部门锚点 (engine.initProfessions 的部门名)。
var relicDepts = map[string]bool{
	"Mayor": true, "Judicial": true, "IT": true, "Police": true, "Medical": true,
	"Supply": true, "Mechanical": true, "Mines": true, "Agricultural": true,
}

const (
	relicWeightMin = 1
	relicWeightMax = 10
)

// supportedRelicEffectMetric 遗物 effects 允许引用的指标白名单。
// 语义与 action gates 的 supportedActionMetric 一致：越表即校验失败。
func supportedRelicEffectMetric(metric string) bool {
	switch metric {
	case "action_points", "suspicion_level", "political_prestige", "propaganda_level",
		"cohesion", "rebellion", "legitimacy", "history_burden":
		return true
	default:
		return false
	}
}

// rootConfigPath 解析 events/ 根级配置文件路径 (与 action_gates 同款)。
func rootConfigPath(dirs map[string]string, fileName string) (string, error) {
	if dir := dirs["mechanics"]; dir != "" {
		return filepath.Join(filepath.Dir(dir), fileName), nil
	}
	groups := make([]string, 0, len(dirs))
	for group := range dirs {
		groups = append(groups, group)
	}
	sort.Strings(groups)
	if len(groups) == 0 {
		return "", fmt.Errorf("no content directories for %s", fileName)
	}
	return filepath.Join(filepath.Dir(dirs[groups[0]]), fileName), nil
}

func actionGateConfigPath(dirs map[string]string) (string, error) {
	return rootConfigPath(dirs, "action_gates.json")
}

func relicCatalogConfigPath(dirs map[string]string) (string, error) {
	return rootConfigPath(dirs, "relics.json")
}

// loadRelicCatalog 生产路径的遗物目录装载：缺失或非法一律显式失败。
// 与 loadActionGates 相同，仅 Bootstrap 调用；不走该路径的旧机制测试
// 保持目录关闭，开局行为逐位不变。
func (s *GameService) loadRelicCatalog(dirs map[string]string) error {
	path, err := relicCatalogConfigPath(dirs)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		s.relicCatalog = relicCatalogRuntime{}
		return fmt.Errorf("required relic catalog is missing: %s", path)
	}
	if err != nil {
		return fmt.Errorf("read relic catalog: %w", err)
	}
	var file relicFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return fmt.Errorf("parse relic catalog %s: %w", path, err)
	}
	runtime, err := compileRelicCatalog(file)
	if err != nil {
		return fmt.Errorf("validate relic catalog %s: %w", path, err)
	}
	s.relicCatalog = runtime
	return nil
}

// compileRelicCatalog 校验并编译遗物清单。错误聚合后一次返回，
// 每条带条目上下文，让人一眼看出错在哪。
func compileRelicCatalog(file relicFile) (relicCatalogRuntime, error) {
	if file.Version != 1 {
		return relicCatalogRuntime{}, fmt.Errorf("unsupported version %d", file.Version)
	}

	var problems []string

	locations := map[string]relicLocation{}
	for i, loc := range file.Locations {
		where := fmt.Sprintf("locations[%d]", i)
		id := strings.TrimSpace(loc.ID)
		if id == "" {
			problems = append(problems, where+": empty id")
			continue
		}
		where = fmt.Sprintf("location %q", id)
		if _, exists := locations[id]; exists {
			problems = append(problems, where+": duplicate location")
			continue
		}
		if strings.TrimSpace(loc.Name) == "" {
			problems = append(problems, where+": empty name")
		}
		if !relicZones[loc.Zone] {
			problems = append(problems, fmt.Sprintf("%s: invalid zone %q (want Upper/Mid/Lower)", where, loc.Zone))
		}
		if loc.Dept != "" && !relicDepts[loc.Dept] {
			problems = append(problems, fmt.Sprintf("%s: unknown dept anchor %q", where, loc.Dept))
		}
		locations[id] = loc
	}

	fragments := map[string]bool{}
	for _, fragment := range model.ALL_FRAGMENTS {
		fragments[fragment] = true
	}

	seen := map[string]bool{}
	totalWeight := 0
	for i, relic := range file.Relics {
		where := fmt.Sprintf("relics[%d]", i)
		id := strings.TrimSpace(relic.ID)
		if id == "" {
			problems = append(problems, where+": empty id")
			continue
		}
		where = fmt.Sprintf("relic %q", id)
		if seen[id] {
			problems = append(problems, where+": duplicate relic id")
			continue
		}
		seen[id] = true
		if strings.TrimSpace(relic.Name) == "" {
			problems = append(problems, where+": empty name")
		}
		if strings.TrimSpace(relic.Description) == "" {
			problems = append(problems, where+": empty description")
		}
		if !relicCategories[relic.Category] {
			problems = append(problems, fmt.Sprintf("%s: unknown category %q", where, relic.Category))
		}
		if len(relic.Locations) == 0 {
			problems = append(problems, where+": no locations declared")
		}
		for _, ref := range relic.Locations {
			if _, exists := locations[ref]; !exists {
				problems = append(problems, fmt.Sprintf("%s: unknown location %q", where, ref))
			}
		}
		if relic.Weight < relicWeightMin || relic.Weight > relicWeightMax {
			problems = append(problems, fmt.Sprintf("%s: weight %d out of range [%d,%d]", where, relic.Weight, relicWeightMin, relicWeightMax))
		}
		totalWeight += relic.Weight
		for metric, value := range relic.Effects {
			if !supportedRelicEffectMetric(metric) {
				problems = append(problems, fmt.Sprintf("%s: unsupported effect metric %q", where, metric))
			}
			if value < -1 || value > 1 {
				problems = append(problems, fmt.Sprintf("%s: effect %q value %v out of range [-1,1]", where, metric, value))
			}
		}
		for _, fragment := range relic.Fragments {
			if !fragments[fragment] {
				problems = append(problems, fmt.Sprintf("%s: unknown fragment %q", where, fragment))
			}
		}
	}

	if file.Spawn.Min < 0 || file.Spawn.Max < file.Spawn.Min {
		problems = append(problems, fmt.Sprintf("spawn: invalid budget min=%d max=%d", file.Spawn.Min, file.Spawn.Max))
	} else if file.Spawn.Max > 2*len(file.Relics) {
		problems = append(problems, fmt.Sprintf("spawn: max=%d exceeds twice the relic count %d", file.Spawn.Max, len(file.Relics)))
	}

	if len(problems) > 0 {
		return relicCatalogRuntime{}, errors.New(strings.Join(problems, "; "))
	}

	return relicCatalogRuntime{
		Enabled:     true,
		Spawn:       file.Spawn,
		Locations:   locations,
		Relics:      file.Relics,
		TotalWeight: totalWeight,
	}, nil
}

// seedStationRelics 按清单为玩家堡实例化开局遗物。只在 CreateGame 对
// 玩家被分配堡调用一次：一号 / 十八号等受保护堡绝不注入。抽取经引擎
// 单一 RNG 流并按清单文件声明顺序消费，保证同种子同清单可复现。
// 清单未加载（旧机制测试路径）时为空操作，RNG 行为逐位不变。
// 实例化把定义拷贝为独立 Relic 行落库，存档持有行的快照而非文件引用：
// 之后改动清单只影响新局，不会断裂旧档。
func (s *GameService) seedStationRelics(silo *model.Silo, startYear int) {
	catalog := s.relicCatalog
	if !catalog.Enabled || len(catalog.Relics) == 0 || catalog.TotalWeight <= 0 {
		return
	}
	count := catalog.Spawn.Min
	if span := catalog.Spawn.Max - catalog.Spawn.Min; span > 0 {
		count += engine.DrawIntn(span + 1)
	}
	for i := 0; i < count; i++ {
		def := pickRelicDefinition(catalog)
		location := catalog.Locations[def.Locations[engine.DrawIntn(len(def.Locations))]]
		silo.Relics = append(silo.Relics, model.Relic{
			SiloID:      silo.ID,
			Name:        def.Name,
			Description: def.Description,
			SourceDept:  location.Dept,
			// DiscoveryYear 记开局年：遗物随开局入世；未来的发现/
			// 没收流程若改手再覆写实际年份。
			DiscoveryYear: startYear,
			Effects:       copyRelicEffects(def.Effects),
			LocationTag:   location.ID,
		})
	}
}

// pickRelicDefinition 按权重抽取一条定义。沿文件声明顺序线性消耗权重，
// 绝不遍历 map，保证确定性。
func pickRelicDefinition(catalog relicCatalogRuntime) relicDefinition {
	roll := engine.DrawFloat64() * float64(catalog.TotalWeight)
	for _, def := range catalog.Relics {
		roll -= float64(def.Weight)
		if roll < 0 {
			return def
		}
	}
	return catalog.Relics[len(catalog.Relics)-1]
}

// copyRelicEffects 深拷贝 effects，避免多条遗物共享同一底层 map。
func copyRelicEffects(in map[string]float64) map[string]float64 {
	if in == nil {
		return nil
	}
	out := make(map[string]float64, len(in))
	for metric, value := range in {
		out[metric] = value
	}
	return out
}
