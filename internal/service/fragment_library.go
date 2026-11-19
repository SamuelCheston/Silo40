package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// 信息碎片内容库 (events/fragments.json)：为既有 26 个碎片键提供只读的
// 标题/正文/来源呈现层 (11-information-fragments-design §2.2 的
// FragmentTemplate 展示子集)。纯内容与展示：不进入任何判定、不消费 RNG、
// 不影响同种子复现。加载/校验复刻 relic catalog 的 fail-closed 纪律：
// 缺失、缺条、多余条目、空字段、越出枚举都在 Bootstrap 期显式报错。

// fragmentTiers 信息源层级枚举 (11 分册 §3.1：基层/中层/决策层)。
var fragmentTiers = map[string]bool{"基层": true, "中层": true, "决策": true}

// fragmentThemes 主题枚举 (11 分册 §3.1 六主题)。
var fragmentThemes = map[string]bool{
	"资源与基础设施":  true,
	"高危处置与外部":  true,
	"Covenant 与执法": true,
	"档案与历史清洗":  true,
	"IT 控制与继承":   true,
	"控制网络与终止链": true,
}

// fragmentDocumentForms 来源口吻枚举：传闻/维修单/证词/档案/私人记录。
var fragmentDocumentForms = map[string]bool{
	"传闻":   true,
	"维修单": true,
	"证词":   true,
	"档案":   true,
	"私人记录": true,
}

// fragmentTemplate 单条碎片模板。字段即文件契约：DisallowUnknownFields
// 保证新属性必须先升版本加代码支持。
type fragmentTemplate struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Department   string   `json:"department"`
	Tier         string   `json:"tier"`
	Theme        string   `json:"theme"`
	DocumentForm string   `json:"document_form"`
	BodyTemplate string   `json:"body_template"`
	LockedClaims []string `json:"locked_claims"`
	NextLeads    string   `json:"next_leads"`
}

type fragmentFile struct {
	Version   int                `json:"version"`
	Fragments []fragmentTemplate `json:"fragments"`
}

// fragmentLibraryRuntime 编译后的碎片内容库，按文件声明顺序保留。
type fragmentLibraryRuntime struct {
	Enabled   bool
	Fragments []fragmentTemplate
	byID      map[string]fragmentTemplate
}

func fragmentLibraryConfigPath(dirs map[string]string) (string, error) {
	return rootConfigPath(dirs, "fragments.json")
}

// loadFragmentLibrary 生产路径的碎片内容库装载：缺失或非法一律显式失败。
// 与 loadRelicCatalog 相同，仅 Bootstrap 调用；不走该路径的旧机制测试
// 保持内容库关闭，既有行为逐位不变。
func (s *GameService) loadFragmentLibrary(dirs map[string]string) error {
	path, err := fragmentLibraryConfigPath(dirs)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		s.fragmentLibrary = fragmentLibraryRuntime{}
		return fmt.Errorf("required fragment library is missing: %s", path)
	}
	if err != nil {
		return fmt.Errorf("read fragment library: %w", err)
	}
	var file fragmentFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return fmt.Errorf("parse fragment library %s: %w", path, err)
	}
	runtime, err := compileFragmentLibrary(file)
	if err != nil {
		return fmt.Errorf("validate fragment library %s: %w", path, err)
	}
	s.fragmentLibrary = runtime
	// 标题登记进引擎注册表：mechanic 脚本的结果消息用它把裸键可读化。
	// 纯展示层，不参与判定。
	titles := make(map[string]string, len(runtime.Fragments))
	for _, fragment := range runtime.Fragments {
		titles[fragment.ID] = fragment.Title
	}
	engine.RegisterFragmentTitles(titles)
	return nil
}

// compileFragmentLibrary 校验并编译碎片内容库。错误聚合后一次返回。
// 覆盖契约：条目必须与 model.ALL_FRAGMENTS 精确一一对应——缺条或多余
// 条目都显式失败，避免界面出现裸键或幽灵条目。
func compileFragmentLibrary(file fragmentFile) (fragmentLibraryRuntime, error) {
	if file.Version != 1 {
		return fragmentLibraryRuntime{}, fmt.Errorf("unsupported version %d", file.Version)
	}

	var problems []string

	expected := map[string]bool{}
	for _, id := range model.ALL_FRAGMENTS {
		expected[id] = true
	}

	seen := map[string]bool{}
	byID := map[string]fragmentTemplate{}
	for i, fragment := range file.Fragments {
		where := fmt.Sprintf("fragments[%d]", i)
		id := strings.TrimSpace(fragment.ID)
		if id == "" {
			problems = append(problems, where+": empty id")
			continue
		}
		where = fmt.Sprintf("fragment %q", id)
		if seen[id] {
			problems = append(problems, where+": duplicate fragment id")
			continue
		}
		seen[id] = true
		if !expected[id] {
			problems = append(problems, where+": id not in ALL_FRAGMENTS")
			continue
		}
		if strings.TrimSpace(fragment.Title) == "" {
			problems = append(problems, where+": empty title")
		}
		if !relicDepts[fragment.Department] {
			problems = append(problems, fmt.Sprintf("%s: unknown department %q", where, fragment.Department))
		}
		if !fragmentTiers[fragment.Tier] {
			problems = append(problems, fmt.Sprintf("%s: invalid tier %q (want 基层/中层/决策)", where, fragment.Tier))
		}
		if !fragmentThemes[fragment.Theme] {
			problems = append(problems, fmt.Sprintf("%s: unknown theme %q", where, fragment.Theme))
		}
		if !fragmentDocumentForms[fragment.DocumentForm] {
			problems = append(problems, fmt.Sprintf("%s: unknown document form %q (want 传闻/维修单/证词/档案/私人记录)", where, fragment.DocumentForm))
		}
		if strings.TrimSpace(fragment.BodyTemplate) == "" {
			problems = append(problems, where+": empty body_template")
		}
		if len(fragment.LockedClaims) == 0 {
			problems = append(problems, where+": no locked_claims")
		}
		for j, claim := range fragment.LockedClaims {
			if strings.TrimSpace(claim) == "" {
				problems = append(problems, fmt.Sprintf("%s: empty locked_claims[%d]", where, j))
			}
		}
		if strings.TrimSpace(fragment.NextLeads) == "" {
			problems = append(problems, where+": empty next_leads")
		}
		byID[id] = fragment
	}

	for _, id := range model.ALL_FRAGMENTS {
		if !seen[id] {
			problems = append(problems, fmt.Sprintf("fragment %q: missing from library", id))
		}
	}

	if len(problems) > 0 {
		return fragmentLibraryRuntime{}, errors.New(strings.Join(problems, "; "))
	}

	return fragmentLibraryRuntime{
		Enabled:   true,
		Fragments: file.Fragments,
		byID:      byID,
	}, nil
}

// FragmentLibraryMeta 返回面向前端的碎片内容清单 (只读展示)。
// 内容库未加载时返回空切片——生产路径 Bootstrap fail-closed，这里只是
// 旧机制测试路径的兜底。
func (s *GameService) FragmentLibraryMeta() []model.FragmentTemplateMeta {
	library := s.fragmentLibrary
	out := make([]model.FragmentTemplateMeta, 0, len(library.Fragments))
	for _, fragment := range library.Fragments {
		out = append(out, model.FragmentTemplateMeta{
			ID:           fragment.ID,
			Title:        fragment.Title,
			Department:   fragment.Department,
			Tier:         fragment.Tier,
			Theme:        fragment.Theme,
			DocumentForm: fragment.DocumentForm,
			Body:         fragment.BodyTemplate,
			LockedClaims: append([]string(nil), fragment.LockedClaims...),
			NextLeads:    fragment.NextLeads,
		})
	}
	return out
}
