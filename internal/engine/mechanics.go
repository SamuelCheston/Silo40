package engine

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"silo40/internal/model"

	"github.com/dop251/goja"
)

type MechanicDefinition struct {
	ID               string        `json:"id"`
	EventType        string        `json:"event_type,omitempty"`
	ActionType       string        `json:"action_type,omitempty"`
	ProfessionAction string        `json:"profession_action,omitempty"`
	Formula          string        `json:"formula,omitempty"`
	Profession       string        `json:"profession,omitempty"`
	Label            string        `json:"label,omitempty"`
	Description      string        `json:"description,omitempty"`
	TargetType       string        `json:"target_type,omitempty"`
	APCost           int           `json:"ap_cost,omitempty"`
	DurationMonths   int           `json:"duration_months,omitempty"`
	SuspicionPenalty float64       `json:"suspicion_penalty,omitempty"`
	ScriptSource     string        `json:"-"`
	compiledProgram  *goja.Program `json:"-"`
}

type MechanicMutation struct {
	Type           string   `json:"type"`
	Field          string   `json:"field,omitempty"`
	Metric         string   `json:"metric,omitempty"`
	Ideology       string   `json:"ideology,omitempty"`
	Profession     string   `json:"profession,omitempty"`
	ClassType      string   `json:"class_type,omitempty"`
	Resource       string   `json:"resource,omitempty"`
	Fragment       string   `json:"fragment,omitempty"`
	Fragments      []string `json:"fragments,omitempty"`
	FactionName    string   `json:"faction_name,omitempty"`
	ConnectionDept string   `json:"connection_dept,omitempty"`
	Value          float64  `json:"value,omitempty"`
	IntValue       int      `json:"int_value,omitempty"`
	BoolValue      bool     `json:"bool_value,omitempty"`
	Text           string   `json:"text,omitempty"`
	// actor_belief_observe 载荷 (10-it-control-credibility §3.5)：观察 ID 去重，
	// 强度由渠道可靠度、相关事实与独立性在 Go 侧统一计算。
	Observation  string  `json:"observation,omitempty"`
	Reliability  float64 `json:"reliability,omitempty"`
	RelatedF     float64 `json:"related_f,omitempty"`
	Independence float64 `json:"independence,omitempty"`
	// profession_belief_absorb 载荷：分享来源标签 (同源转述折减用)。
	Source string `json:"source,omitempty"`
}

type MechanicResult struct {
	Mutations    []MechanicMutation  `json:"mutations,omitempty"`
	Emit         []string            `json:"emit,omitempty"`
	Logs         []string            `json:"logs,omitempty"`
	ActionResult *model.ActionResult `json:"action_result,omitempty"`
	Stats        *model.AgentStats   `json:"stats,omitempty"`
	Score        *model.GameScore    `json:"score,omitempty"`
}

type MechanicContext struct {
	Event *GameEvent
	Silo  *model.Silo
	Agent *model.Agent
	Actor *ActorView
}

type MechanicEngine struct {
	byEventType        map[string]MechanicDefinition
	byActionType       map[model.AgentActionType]MechanicDefinition
	byProfessionAction map[string]MechanicDefinition
	byFormula          map[string]MechanicDefinition
	preludeProgram     *goja.Program
	loadErr            error
}

func NewMechanicEngine() *MechanicEngine {
	m := &MechanicEngine{
		byEventType:        map[string]MechanicDefinition{},
		byActionType:       map[model.AgentActionType]MechanicDefinition{},
		byProfessionAction: map[string]MechanicDefinition{},
		byFormula:          map[string]MechanicDefinition{},
	}

	// 预编译 Prelude (含 InformationConfig 注入段)
	if prog, err := goja.Compile("mechanic_prelude.js", mechanicPreludeSource(), true); err == nil {
		m.preludeProgram = prog
	} else {
		m.loadErr = fmt.Errorf("failed to compile mechanic prelude: %w", err)
	}

	if dir := findMechanicsDirectory(); dir != "" {
		defs, err := LoadMechanicDefinitions(dir)
		if err != nil {
			m.loadErr = err
		} else {
			m.RegisterMany(defs)
		}
	}
	return m
}

func (m *MechanicEngine) RegisterMany(defs []MechanicDefinition) {
	for _, def := range defs {
		m.Register(def)
	}
}

func (m *MechanicEngine) Register(def MechanicDefinition) {
	if def.EventType != "" {
		m.byEventType[def.EventType] = def
	}
	if def.ActionType != "" {
		m.byActionType[model.AgentActionType(def.ActionType)] = def
	}
	if def.ProfessionAction != "" {
		m.byProfessionAction[def.ProfessionAction] = def
	}
	if def.Formula != "" {
		m.byFormula[def.Formula] = def
	}
}

func (m *MechanicEngine) EventDefinition(eventType string) (MechanicDefinition, bool) {
	def, ok := m.byEventType[eventType]
	return def, ok
}

func (m *MechanicEngine) ActionDefinition(action model.AgentAction) (MechanicDefinition, bool) {
	if action.Type == model.ActionProfession {
		def, ok := m.byProfessionAction[strings.TrimSpace(action.ProfessionAction)]
		return def, ok
	}
	def, ok := m.byActionType[action.Type]
	return def, ok
}

func (m *MechanicEngine) FormulaDefinition(name string) (MechanicDefinition, bool) {
	def, ok := m.byFormula[name]
	return def, ok
}

func (m *MechanicEngine) ProfessionActionDefinitions() []MechanicDefinition {
	out := make([]MechanicDefinition, 0, len(m.byProfessionAction))
	for _, def := range m.byProfessionAction {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProfessionAction < out[j].ProfessionAction })
	return out
}

func (m *MechanicEngine) CommonActionDefinitions() []MechanicDefinition {
	out := make([]MechanicDefinition, 0, len(m.byActionType))
	for _, def := range m.byActionType {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ActionType < out[j].ActionType })
	return out
}

func (m *MechanicEngine) Run(def MechanicDefinition, ctx MechanicContext) (MechanicResult, error) {
	runtime := goja.New()
	runtime.SetRandSource(rng.Float64)
	if m.preludeProgram != nil {
		if _, err := runtime.RunProgram(m.preludeProgram); err != nil {
			return MechanicResult{}, err
		}
	}
	if def.compiledProgram != nil {
		if _, err := runtime.RunProgram(def.compiledProgram); err != nil {
			return MechanicResult{}, err
		}
	} else {
		// 回退到字符串运行 (如果不小心没编译)
		if _, err := runtime.RunString(def.ScriptSource); err != nil {
			return MechanicResult{}, err
		}
	}

	value := runtime.Get("__mechanicCaptured")
	scriptObj, err := mechanicObjectByID(runtime, value, def.ID)
	if err != nil {
		return MechanicResult{}, err
	}
	applyValue := scriptObj.Get("apply")
	callable, ok := goja.AssertFunction(applyValue)
	if !ok {
		return MechanicResult{}, fmt.Errorf("mechanic %s apply is not a function", def.ID)
	}

	res, err := callable(scriptObj, runtime.ToValue(mechanicScriptContext(ctx)))
	if err != nil {
		return MechanicResult{}, err
	}
	if goja.IsUndefined(res) || goja.IsNull(res) {
		return MechanicResult{}, nil
	}

	// 使用 JSON 序列化保证 JSON 标签被正确尊重 (这在单次执行中开销可控)
	raw, err := json.Marshal(res.Export())
	if err != nil {
		return MechanicResult{}, err
	}
	var result MechanicResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return MechanicResult{}, err
	}

	// 统一处理字段格式
	for i := range result.Mutations {
		result.Mutations[i].Type = strings.ToLower(strings.TrimSpace(result.Mutations[i].Type))
		result.Mutations[i].Field = strings.ToLower(strings.TrimSpace(result.Mutations[i].Field))
		result.Mutations[i].Metric = strings.ToLower(strings.TrimSpace(result.Mutations[i].Metric))
		result.Mutations[i].Ideology = strings.TrimSpace(result.Mutations[i].Ideology)
		result.Mutations[i].Profession = strings.TrimSpace(result.Mutations[i].Profession)
		result.Mutations[i].ClassType = strings.ToUpper(strings.TrimSpace(result.Mutations[i].ClassType))
		result.Mutations[i].Resource = strings.TrimSpace(result.Mutations[i].Resource)
		result.Mutations[i].FactionName = strings.TrimSpace(result.Mutations[i].FactionName)
		result.Mutations[i].ConnectionDept = strings.TrimSpace(result.Mutations[i].ConnectionDept)
		result.Mutations[i].Observation = strings.TrimSpace(result.Mutations[i].Observation)
		result.Mutations[i].Source = strings.TrimSpace(result.Mutations[i].Source)
	}

	return result, nil
}

func LoadMechanicDefinitions(dir string) ([]MechanicDefinition, error) {
	var defs []MechanicDefinition
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".js" {
			return nil
		}
		fileDefs, err := loadMechanicFile(path)
		if err != nil {
			return err
		}
		defs = append(defs, fileDefs...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load mechanics from %s: %w", dir, err)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].ID < defs[j].ID })
	return defs, nil
}

const mechanicPrelude = `
var __mechanicCaptured = [];
function defineMechanic(def) {
  if (!def || typeof def !== "object") {
    throw new Error("defineMechanic expects an object");
  }
  __mechanicCaptured.push(def);
  return def;
}
function defineMechanics(defs) {
  if (!Array.isArray(defs)) {
    throw new Error("defineMechanics expects an array");
  }
  defs.forEach(defineMechanic);
  return defs;
}
function clamp(value, min, max) {
  if (value < min) return min;
  if (value > max) return max;
  return value;
}
function clampUnit(value) {
  return clamp(value, 0, 1);
}
function randomFloat() {
  return Math.random();
}
function randomInt(max) {
  if (!max || max <= 0) return 0;
  return Math.floor(Math.random() * max);
}
// ---- 信息与检测 (10-it-control-credibility §3/§8)；常量由 Go 侧 ----
// ---- InformationConfig 注入 (mechanicPreludeSource)，此处仅函数。 ----
function fragmentDomain(fragment) {
  var idx = String(fragment).indexOf("_");
  return idx > 0 ? String(fragment).slice(0, idx) : String(fragment);
}
function beliefOf(ctx, fragment) {
  var list = (ctx.actor && ctx.actor.beliefs) || [];
  for (var i = 0; i < list.length; i++) {
    if (list[i].fact_id === fragment) return list[i];
  }
  return null;
}
function beliefB(ctx, fragment) {
  var belief = beliefOf(ctx, fragment);
  return belief ? (belief.b || 0) : 0;
}
function relatedF(ctx, fragment) {
  var domain = fragmentDomain(fragment);
  var best = 0;
  ((ctx.actor && ctx.actor.beliefs) || []).forEach(function(item) {
    if (item.fact_id === fragment) return;
    if (fragmentDomain(item.fact_id) !== domain) return;
    if ((item.b || 0) > best) best = item.b || 0;
  });
  return best;
}
function pGet(fProf, mismatch, alert, related, accessC) {
  var p = INFO_CFG.p_get_base * fProf
    * (1 - INFO_CFG.mismatch_penalty * clampUnit(mismatch))
    * (1 - INFO_CFG.alert_penalty * clampUnit(alert))
    * (1 + INFO_CFG.related_bonus * clampUnit(related))
    * accessC;
  return clamp(p, INFO_CFG.p_get_min, INFO_CFG.p_get_max);
}
function deptNorm(prof) {
  if (!prof) return 0;
  return clamp(2 * clampUnit((prof.ideologies || {}).pro_foreign || 0) - 1, -1, 1);
}
function mismatchFor(ctx, targetDept, deviation) {
  var prof = professionByName(ctx, targetDept);
  var d = deptNorm(prof);
  var a = clamp(deviation, -1, 1);
  if (a === 0) return 0;
  var sign = a < 0 ? -1 : 1;
  return Math.abs(a) * Math.max(0, -sign * d);
}
function detectionClass(ctx, targetDept, protocolTouch) {
  if (protocolTouch) return "DIRECT_PROTOCOL";
  var profession = ctx.actor ? ctx.actor.profession : "";
  if (profession === "Mines" || targetDept === "Mines") return "UNSAMPLED";
  return "SAMPLED";
}
function sampledSuspicion(base, vIdentity, fFit, wWitness, rRecord, mismatch) {
  return base * vIdentity * fFit * wWitness * rRecord * (1 + clampUnit(mismatch));
}
function vIdentityFor(profession) {
  var table = INFO_CFG.v_identity || {};
  return table[profession] !== undefined ? table[profession] : 1.0;
}
function gatherChannelFor(ctx, targetDept) {
  var channels = INFO_CFG.channels;
  if (ctx.actor && ctx.actor.profession === targetDept) {
    return { id: "internal", access_c: channels.internal.access, reliability_c: channels.internal.reliability };
  }
  if (actorConnectionTo(ctx, targetDept) >= 0.3) {
    return { id: "informant", access_c: channels.informant.access, reliability_c: channels.informant.reliability };
  }
  return { id: "rumor", access_c: channels.rumor.access, reliability_c: channels.rumor.reliability };
}
function __mechanicStrip(value) {
  if (Array.isArray(value)) return value.map(__mechanicStrip);
  if (!value || typeof value !== "object") return value;
  var out = {};
  for (var key in value) {
    if (!Object.prototype.hasOwnProperty.call(value, key)) continue;
    if (typeof value[key] === "function") continue;
    out[key] = __mechanicStrip(value[key]);
  }
  return out;
}
function __mechanicExport() {
  return JSON.stringify(__mechanicCaptured.map(__mechanicStrip));
}
function professionByName(ctx, name) {
  return (ctx.silo.professions || []).find(function(item) { return item.name === name; }) || null;
}
function factionByName(ctx, name) {
  return (ctx.silo.factions || []).find(function(item) { return item.name === name; }) || null;
}
function actorConnectionTo(ctx, name) {
  var entry = (ctx.actor.connections || []).find(function(item) { return item.profession === name; });
  return entry ? entry.value : 0;
}
function actorKnownSet(ctx) {
  var out = {};
  (ctx.actor.known_fragments || []).forEach(function(fragment) { out[fragment] = true; });
  return out;
}
function unknownFragmentsFrom(ctx, profession) {
  var prof = professionByName(ctx, profession);
  if (!prof) return [];
  var known = actorKnownSet(ctx);
  return (prof.all_fragments || []).filter(function(fragment) { return !known[fragment]; });
}
// suspicionDelta 结构化检测 (§8.2/§8.3)：UNSAMPLED (矿工结构性不采样) 与
// DIRECT_PROTOCOL (触及条款) 不产生普通怀疑；SAMPLED 按身份可见度折算。
// 旧的"统一职业倍率"已被设计裁决替换 (10-it-control-credibility §1.3)。
function suspicionDelta(ctx, base) {
  var cls = detectionClass(ctx, (ctx.action && ctx.action.target_dept) || "", false);
  if (cls !== "SAMPLED") {
    return { type: "actor_metric_delta", field: "suspicion_level", value: 0 };
  }
  var trait = ((ctx.actor && ctx.actor.traits) || []).indexOf("隐秘行事") !== -1 ? INFO_CFG.trait_stealth : 1.0;
  var profession = ctx.actor ? ctx.actor.profession : "";
  return { type: "actor_metric_delta", field: "suspicion_level", value: base * vIdentityFor(profession) * trait };
}
`

// mechanicPreludeSource 拼装注入段与基础 prelude。INFO_CFG 由 Go 侧
// InformationConfig 单一来源生成，脚本与 Go 公式不出现两套常量。
func mechanicPreludeSource() string {
	return informationPreludeJS(DefaultInformationConfig()) + mechanicPrelude
}

func informationPreludeJS(cfg InformationConfig) string {
	return "var INFO_CFG = {\n" +
		"  p_get_base: " + strconv.FormatFloat(cfg.PGetBase, 'g', -1, 64) + ",\n" +
		"  p_get_min: " + strconv.FormatFloat(cfg.PGetMin, 'g', -1, 64) + ",\n" +
		"  p_get_max: " + strconv.FormatFloat(cfg.PGetMax, 'g', -1, 64) + ",\n" +
		"  mismatch_penalty: " + strconv.FormatFloat(cfg.MismatchPenalty, 'g', -1, 64) + ",\n" +
		"  alert_penalty: " + strconv.FormatFloat(cfg.AlertPenalty, 'g', -1, 64) + ",\n" +
		"  related_bonus: " + strconv.FormatFloat(cfg.RelatedBonus, 'g', -1, 64) + ",\n" +
		"  f_prof_match: " + strconv.FormatFloat(cfg.FProfMatch, 'g', -1, 64) + ",\n" +
		"  f_prof_mismatch: " + strconv.FormatFloat(cfg.FProfMismatch, 'g', -1, 64) + ",\n" +
		"  qual_rumor: " + strconv.FormatFloat(cfg.QualRumor, 'g', -1, 64) + ",\n" +
		"  gather_deviation: " + strconv.FormatFloat(cfg.GatherDeviation, 'g', -1, 64) + ",\n" +
		"  probe_base_suspicion: " + strconv.FormatFloat(cfg.ProbeBaseSuspicion, 'g', -1, 64) + ",\n" +
		"  probe_fit_cross: " + strconv.FormatFloat(cfg.ProbeFitCross, 'g', -1, 64) + ",\n" +
		"  probe_fit_own: " + strconv.FormatFloat(cfg.ProbeFitOwn, 'g', -1, 64) + ",\n" +
		"  alert_on_failed_probe: " + strconv.FormatFloat(cfg.AlertOnFailedProbe, 'g', -1, 64) + ",\n" +
		"  probe_witness: " + strconv.FormatFloat(cfg.ProbeWitness, 'g', -1, 64) + ",\n" +
		"  probe_record: " + strconv.FormatFloat(cfg.ProbeRecord, 'g', -1, 64) + ",\n" +
		"  probe_visibility: " + strconv.FormatFloat(cfg.ProbeVisibility, 'g', -1, 64) + ",\n" +
		"  trait_stealth: 0.8,\n" +
		"  v_identity: { Mayor: 2.0, IT: 0.3, Police: 0.7 },\n" +
		"  channels: {\n" +
		"    internal: { access: " + strconv.FormatFloat(cfg.ChannelInternalAccess, 'g', -1, 64) + ", reliability: " + strconv.FormatFloat(cfg.ChannelInternalRel, 'g', -1, 64) + " },\n" +
		"    archive: { access: " + strconv.FormatFloat(cfg.ChannelArchiveAccess, 'g', -1, 64) + ", reliability: " + strconv.FormatFloat(cfg.ChannelArchiveRel, 'g', -1, 64) + " },\n" +
		"    informant: { access: " + strconv.FormatFloat(cfg.ChannelInformantAccess, 'g', -1, 64) + ", reliability: " + strconv.FormatFloat(cfg.ChannelInformantRel, 'g', -1, 64) + " },\n" +
		"    testimony: { access: " + strconv.FormatFloat(cfg.ChannelTestimonyAccess, 'g', -1, 64) + ", reliability: " + strconv.FormatFloat(cfg.ChannelTestimonyRel, 'g', -1, 64) + " },\n" +
		"    rumor: { access: " + strconv.FormatFloat(cfg.ChannelRumorAccess, 'g', -1, 64) + ", reliability: " + strconv.FormatFloat(cfg.ChannelRumorRel, 'g', -1, 64) + " }\n" +
		"  }\n" +
		"};\n"
}

func loadMechanicFile(path string) ([]MechanicDefinition, error) {
	sourceBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	source := string(sourceBytes)

	// 预编译脚本
	prog, err := goja.Compile(filepath.Base(path), source, true)
	if err != nil {
		return nil, fmt.Errorf("failed to compile %s: %w", path, err)
	}

	runtime := goja.New()
	runtime.SetRandSource(rng.Float64)
	if _, err := runtime.RunString(mechanicPreludeSource()); err != nil {
		return nil, err
	}
	if _, err := runtime.RunProgram(prog); err != nil {
		return nil, err
	}

	exported, err := runtime.RunString("__mechanicExport()")
	if err != nil {
		return nil, err
	}
	var defs []MechanicDefinition
	if err := json.Unmarshal([]byte(exported.String()), &defs); err != nil {
		return nil, err
	}
	for i := range defs {
		defs[i].ID = strings.TrimSpace(defs[i].ID)
		defs[i].EventType = strings.TrimSpace(defs[i].EventType)
		defs[i].ActionType = strings.TrimSpace(defs[i].ActionType)
		defs[i].ProfessionAction = strings.TrimSpace(defs[i].ProfessionAction)
		defs[i].Formula = strings.TrimSpace(defs[i].Formula)
		defs[i].Profession = strings.TrimSpace(defs[i].Profession)
		defs[i].TargetType = strings.ToUpper(strings.TrimSpace(defs[i].TargetType))
		if defs[i].ID == "" {
			return nil, fmt.Errorf("mechanic in %s is missing id", path)
		}
		routeCount := 0
		if defs[i].EventType != "" {
			routeCount++
		}
		if defs[i].ActionType != "" {
			routeCount++
		}
		if defs[i].ProfessionAction != "" {
			routeCount++
		}
		if defs[i].Formula != "" {
			routeCount++
		}
		if routeCount != 1 {
			return nil, fmt.Errorf("mechanic %s in %s must declare exactly one route key", defs[i].ID, path)
		}
		defs[i].ScriptSource = source
		defs[i].compiledProgram = prog
	}
	return defs, nil
}

func loadMechanicRuntime(source string) (*goja.Runtime, goja.Value, error) {
	runtime := goja.New()
	runtime.SetRandSource(rng.Float64)
	if _, err := runtime.RunString(mechanicPrelude); err != nil {
		return nil, nil, err
	}
	if _, err := runtime.RunString(source); err != nil {
		return nil, nil, err
	}
	value := runtime.Get("__mechanicCaptured")
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, nil, fmt.Errorf("mechanic file did not register any definitions")
	}
	return runtime, value, nil
}

func mechanicObjectByID(runtime *goja.Runtime, defsValue goja.Value, id string) (*goja.Object, error) {
	defsObj := defsValue.ToObject(runtime)
	lengthValue := defsObj.Get("length")
	length := int(lengthValue.ToInteger())
	for i := 0; i < length; i++ {
		value := defsObj.Get(itoa(i))
		if goja.IsUndefined(value) || goja.IsNull(value) {
			continue
		}
		obj := value.ToObject(runtime)
		if strings.TrimSpace(obj.Get("id").String()) == id {
			return obj, nil
		}
	}
	return nil, fmt.Errorf("mechanic definition %s not found in runtime", id)
}

func mechanicScriptContext(ctx MechanicContext) map[string]interface{} {
	result := map[string]interface{}{
		"event":         map[string]interface{}{},
		"all_fragments": append([]string(nil), model.ALL_FRAGMENTS...),
		"silo": map[string]interface{}{
			"metrics":     map[string]float64{},
			"resources":   map[string]float64{},
			"professions": []map[string]interface{}{},
			"cohorts":     []map[string]interface{}{},
			"factions":    []map[string]interface{}{},
			"residents":   []map[string]interface{}{},
			"traits":      []string{},
		},
		"actor": map[string]interface{}{
			"connections":     []map[string]interface{}{},
			"known_fragments": []string{},
			"traits":          []string{},
		},
		"action": map[string]interface{}{},
	}
	if ctx.Event != nil {
		eventPayload := map[string]interface{}{
			"id":     ctx.Event.ID,
			"type":   ctx.Event.Type,
			"source": ctx.Event.Source,
			"target": ctx.Event.Target,
		}
		if deltaYears, ok := ctx.Event.Data["deltaYears"].(float64); ok {
			eventPayload["delta_years"] = deltaYears
		}
		result["event"] = eventPayload
		if action, ok := ctx.Event.Data["action"].(model.AgentAction); ok {
			result["action"] = actionContext(action)
		}
	}
	if ctx.Silo != nil {
		siloPayload := map[string]interface{}{
			"current_year":     ctx.Silo.CurrentYear,
			"current_month":    ctx.Silo.CurrentMonth,
			"total_population": ctx.Silo.TotalPopulation,
			"safeguard_risk":   ctx.Silo.SafeguardRisk,
			"silo1_destroyed":  ctx.Silo.Silo1Destroyed,
			"pact_locked":      ctx.Silo.Authority.Judicial.PactLocked,
			"traits":           append([]string(nil), ctx.Silo.Traits...),
			// G5: victory_check reads the escape group to apply the DEATH exception.
			"escaped_population": ctx.Silo.EscapedGroup.Population,
			"escaped_surviving":  ctx.Silo.EscapedGroup.Initialized && ctx.Silo.EscapedGroup.Surviving,
			"metrics": map[string]float64{
				"legitimacy":          ctx.Silo.MayorLegitimacy(),
				"cohesion":            ctx.Silo.Cohesion,
				"rebellion":           ctx.Silo.Rebellion,
				"dept_tension":        ctx.Silo.DeptTension,
				"class_fragmentation": ctx.Silo.ClassFragmentation,
				"history_burden":      ctx.Silo.HistoryBurden,
				"event_trigger":       ctx.Silo.EventTrigger,
				"countdown":           ctx.Silo.Countdown,
				"safeguard_risk":      ctx.Silo.SafeguardRisk,
			},
		}
		if ctx.Silo.VictoryStatus != nil {
			siloPayload["victory_status"] = map[string]interface{}{
				"is_won":      ctx.Silo.VictoryStatus.IsWon,
				"type":        ctx.Silo.VictoryStatus.Type,
				"description": ctx.Silo.VictoryStatus.Description,
			}
		}
		resources := map[string]float64{}
		for _, res := range ctx.Silo.Resources {
			resources[res.Type] = res.Amount
		}
		siloPayload["resources"] = resources
		professions := make([]map[string]interface{}, 0, len(ctx.Silo.Professions))
		for _, prof := range ctx.Silo.Professions {
			allFragments := make([]string, 0)
			prefix := prof.Name + "_"
			for _, fragment := range model.ALL_FRAGMENTS {
				if strings.HasPrefix(fragment, prefix) {
					allFragments = append(allFragments, fragment)
				}
			}
			// 部门信念层 (§3.3)：暴露 B>0 或已成熟的事实，切片顺序即持久化
			// 顺序 (确定性)。victory_check 的 INFORMATION 结构桥读这一层的
			// 成熟计数 (18-time-permeation §3.1)：成熟粘性，B 可以淡忘到 0，
			// 已成熟事实必须继续可见。
			beliefs := make([]map[string]interface{}, 0, len(prof.Beliefs))
			for _, belief := range prof.Beliefs {
				if IsLeakFact(belief.FactID) {
					continue // 20 分册：反制观察不进 mechanic ctx / UI 信念视图
				}
				if belief.B <= 0 && !belief.Matured {
					continue
				}
				beliefs = append(beliefs, map[string]interface{}{
					"fact_id":            belief.FactID,
					"b":                  belief.B,
					"permeation_counter": belief.PermeationCounter,
					"matured":            belief.Matured,
				})
			}
			professions = append(professions, map[string]interface{}{
				"id":              prof.ID,
				"name":            prof.Name,
				"class_type":      prof.ClassType,
				"zone":            prof.Zone,
				"population":      prof.Population,
				"panic_value":     prof.PanicValue,
				"productivity":    prof.Productivity,
				"power_level":     prof.PowerLevel,
				"action_points":   prof.ActionPoints,
				"alertness":       prof.Alertness,
				"known_fragments": append([]string(nil), prof.KnownFragments...),
				"all_fragments":   allFragments,
				"beliefs":         beliefs,
				"ideologies":      cloneFloatMap(prof.Ideologies),
				"relations":       cloneFloatMap(prof.Relations),
				// T1-3 关系冷却：>0 时 Build 关系增长减半、回执措辞变谨慎。
				"relation_cooldown": ctx.Silo.Countermeasure.RelationCooldown[prof.Name],
			})
		}
		siloPayload["professions"] = professions
		cohorts := make([]map[string]interface{}, 0, len(ctx.Silo.Cohorts))
		for _, cohort := range ctx.Silo.Cohorts {
			cohorts = append(cohorts, map[string]interface{}{
				"id":                cohort.ID,
				"profession_id":     cohort.ProfessionID,
				"count":             cohort.Count,
				"ideologies":        cloneFloatMap(cohort.Ideologies),
				"panic_sensitivity": cohort.PanicSensitivity,
			})
		}
		siloPayload["cohorts"] = cohorts
		factions := make([]map[string]interface{}, 0, len(ctx.Silo.Factions))
		for _, faction := range ctx.Silo.Factions {
			factions = append(factions, map[string]interface{}{
				"id":        faction.ID,
				"name":      faction.Name,
				"influence": faction.Influence,
				"cohesion":  faction.Cohesion,
				"prestige":  faction.Prestige,
				"is_public": faction.IsPublic,
			})
		}
		siloPayload["factions"] = factions
		floors := make([]map[string]interface{}, 0, len(ctx.Silo.Floors))
		for _, floor := range ctx.Silo.Floors {
			floors = append(floors, map[string]interface{}{
				"level":     floor.Level,
				"function":  floor.Function,
				"zone":      floor.Zone,
				"stability": floor.Stability,
			})
		}
		siloPayload["floors"] = floors
		result["silo"] = siloPayload
	}
	if ctx.Actor != nil {
		actorPayload := map[string]interface{}{
			"kind":               string(ctx.Actor.Ref.Kind),
			"profession":         ctx.Actor.Profession(),
			"label":              ctx.Actor.Label(),
			"action_points":      ctx.Actor.ActionPoints(),
			"suspicion_level":    ctx.Actor.SuspicionLevel(),
			"political_prestige": ctx.Actor.PoliticalPrestige(),
			"ambition":           ctx.Actor.Ambition(),
			"influence":          ctx.Actor.Influence(),
			"political_points":   ctx.Actor.PoliticalPoints(),
			"propaganda_level":   ctx.Actor.PropagandaLevel(),
			"traits":             append([]string(nil), ctx.Actor.Traits()...),
			"known_fragments":    append([]string(nil), ctx.Actor.KnownFragments()...),
			"is_representative":  ctx.Actor.IsRepresentative(),
			"is_faction_leader":  actorLeadsFaction(ctx.Silo, ctx.Actor),
			"ideologies":         cloneFloatMap(ctx.Actor.Ideologies()),
			"connections":        []map[string]interface{}{},
			"beliefs":            []map[string]interface{}{},
		}
		if agent := ctx.Actor.Agent(); agent != nil {
			beliefs := make([]map[string]interface{}, 0, len(agent.Beliefs))
			for i := range agent.Beliefs {
				beliefs = append(beliefs, map[string]interface{}{
					"fact_id":   agent.Beliefs[i].FactID,
					"b":         agent.Beliefs[i].B,
					"obs_count": agent.Beliefs[i].ObsCount,
				})
			}
			actorPayload["beliefs"] = beliefs
		}
		if prof := professionByActor(ctx.Silo, ctx.Actor); prof != nil {
			actorPayload["power_level"] = prof.PowerLevel
			actorPayload["class_type"] = prof.ClassType
		}
		connections := make([]map[string]interface{}, 0, len(ctx.Silo.Professions))
		for _, prof := range ctx.Silo.Professions {
			connections = append(connections, map[string]interface{}{
				"profession": prof.Name,
				"value":      ctx.Actor.GetConnection(prof.ID),
			})
		}
		actorPayload["connections"] = connections
		if faction := actorFaction(ctx.Silo, ctx.Actor); faction != nil {
			actorPayload["faction"] = map[string]interface{}{
				"id":        faction.ID,
				"name":      faction.Name,
				"signature": faction.Signature,
				"influence": faction.Influence,
				"cohesion":  faction.Cohesion,
				"prestige":  faction.Prestige,
				"is_public": faction.IsPublic,
			}
		}
		result["actor"] = actorPayload
	}
	return result
}

func actionContext(action model.AgentAction) map[string]interface{} {
	return map[string]interface{}{
		"type":              string(action.Type),
		"action_id":         action.ActionID,
		"target_dept":       action.TargetDept,
		"fragment_ids":      append([]string(nil), action.FragmentIds...),
		"profession_action": action.ProfessionAction,
		"resource_target":   action.ResourceTarget,
		"target_faction_id": action.TargetFactionID,
		"cost":              action.Cost,
	}
}

func cloneFloatMap(in map[string]float64) map[string]float64 {
	if in == nil {
		return map[string]float64{}
	}
	out := make(map[string]float64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func professionByActor(silo *model.Silo, actor *ActorView) *model.Profession {
	if silo == nil || actor == nil {
		return nil
	}
	for i := range silo.Professions {
		if silo.Professions[i].Name == actor.Profession() {
			return &silo.Professions[i]
		}
	}
	return nil
}

func actorFaction(silo *model.Silo, actor *ActorView) *model.Faction {
	if silo == nil || actor == nil {
		return nil
	}
	factionID := actor.FactionID()
	if factionID == nil {
		return nil
	}
	for i := range silo.Factions {
		if silo.Factions[i].ID == *factionID {
			return &silo.Factions[i]
		}
	}
	return nil
}

func actorLeadsFaction(silo *model.Silo, actor *ActorView) bool {
	if silo == nil || actor == nil {
		return false
	}
	if actor.IsRepresentative() {
		return true
	}
	for i := range silo.Residents {
		res := &silo.Residents[i]
		if res.Alive && res.Profession == actor.Profession() && res.IsRepresentative {
			return true
		}
	}
	return false
}

func findMechanicsDirectory() string {
	cwd, _ := os.Getwd()
	exePath, _ := os.Executable()
	exeDir := ""
	if exePath != "" {
		exeDir = filepath.Dir(exePath)
	}
	for _, base := range []string{cwd, exeDir} {
		if dir := findMechanicsDirectoryFrom(base); dir != "" {
			return dir
		}
	}
	return ""
}

func findMechanicsDirectoryFrom(base string) string {
	if base == "" {
		return ""
	}
	current := base
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(current, "events", "mechanics")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

func (e *GameEngine) applyMechanicMutations(event *GameEvent, mutations []MechanicMutation, ctx *EventContext) {
	actorRef, _ := event.Data["actor"].(ActorRef)
	if actorRef.Kind == "" {
		silo, _ := event.Data["silo"].(*model.Silo)
		agent, _ := event.Data["agent"].(*model.Agent)
		if silo != nil && agent != nil {
			actorRef = CreateActorRefForAgent(agent, silo)
		}
	}

	silo, _ := event.Data["silo"].(*model.Silo)
	agent, _ := event.Data["agent"].(*model.Agent)

	// 死因归因快照必须在批前取：同一批里 silo_metric_set rebellion 先于
	// apply_population_deaths 应用，批中状态已不是 JS 计算死亡率时的口径。
	e.pendingDeathParts = nil
	if silo != nil {
		for _, mutation := range mutations {
			if mutation.Type == "apply_population_deaths" {
				e.pendingDeathParts = deathRateBreakdown(silo)
				break
			}
		}
	}

	for _, mutation := range mutations {
		// 直接应用变更
		e.ApplyMutation(silo, agent, actorRef, mutation)
	}
}

func (e *GameEngine) ApplyMutation(silo *model.Silo, agent *model.Agent, actorRef ActorRef, mutation MechanicMutation) {
	var actor *ActorView
	if silo != nil && actorRef.Kind != "" {
		actor, _ = CreateActorView(actorRef, silo, agent)
	}

	switch mutation.Type {
	case "actor_metric_delta":
		e.applyActorMetricDelta(actor, mutation.Field, mutation.Value)
	case "actor_metric_set":
		e.applyActorMetricSet(actor, mutation.Field, mutation.Value)
	case "actor_connection_delta":
		e.applyActorConnectionDelta(silo, actor, mutation.ConnectionDept, mutation.Value)
	case "actor_fragment_add":
		e.applyActorFragmentAdd(actor, mutation.Fragment)
	case "actor_belief_observe":
		e.applyActorBeliefObserve(actor, mutation)
	case "profession_alertness_delta":
		if prof := findDept(silo, mutation.Profession); prof != nil {
			prof.Alertness = clampUnit(prof.Alertness + mutation.Value)
		}
	case "profession_belief_absorb":
		// §3.6 分享接收：强度进入接收部门信念层；玩家私下确信不直接进 E/Q。
		if prof := findDept(silo, mutation.Profession); prof != nil {
			AbsorbSharedFact(DefaultInformationConfig(), &prof.Beliefs, mutation.Fragment, mutation.Source, mutation.Value)
		}
	case "profession_metric_delta":
		if prof := findDept(silo, mutation.Profession); prof != nil {
			e.applyProfessionMetricDelta(prof, mutation.Field, mutation.Value)
		}
	case "profession_metric_set":
		if prof := findDept(silo, mutation.Profession); prof != nil {
			e.applyProfessionMetricSet(prof, mutation.Field, mutation.Value)
		}
	case "profession_ideology_delta":
		if prof := findDept(silo, mutation.Profession); prof != nil {
			if prof.Ideologies == nil {
				prof.Ideologies = map[string]float64{}
			}
			prof.Ideologies[mutation.Ideology] = clampUnit(prof.Ideologies[mutation.Ideology] + mutation.Value)
		}
	case "profession_fragment_remove":
		if prof := findDept(silo, mutation.Profession); prof != nil {
			prof.KnownFragments = removeString(prof.KnownFragments, mutation.Fragment)
		}
	case "profession_fragment_add":
		if prof := findDept(silo, mutation.Profession); prof != nil {
			for _, item := range prof.KnownFragments {
				if item == mutation.Fragment {
					return
				}
			}
			prof.KnownFragments = append(prof.KnownFragments, mutation.Fragment)
		}
	case "silo_metric_delta":
		e.applySiloMetricDeltaWithSupport(silo, mutation.Metric, mutation.Value)
	case "silo_metric_set":
		e.applySiloMetricSet(silo, mutation.Metric, mutation.Value)
	case "resource_delta":
		e.applyResourceDelta(silo, mutation.Resource, mutation.Value)
	case "resource_net_balance_set":
		e.applyResourceNetBalanceSet(silo, mutation.Resource, mutation.Value)
	case "cohort_ideology_delta_all":
		e.applyCohortIdeologyDeltaAll(silo, mutation)
	case "cohort_ideology_delta":
		e.applyCohortIdeologyDelta(silo, mutation)
	case "sync_profession_ideologies_from_cohorts":
		e.syncProfessionIdeologiesFromCohorts(silo)
	case "apply_population_deaths":
		parts := e.pendingDeathParts
		e.pendingDeathParts = nil
		applyPopulationDeathsToCohorts(silo, mutation.IntValue)
		if silo != nil && mutation.IntValue > 0 {
			if parts == nil {
				// 批外直接调用 (测试/兜底路径) 时按当前状态归因。
				parts = deathRateBreakdown(silo)
			}
			e.logf(deathAttributionLine(silo, mutation.IntValue, parts))
		}
	case "refresh_profession_population_from_cohorts":
		refreshProfessionPopulationFromCohorts(silo)
	case "faction_metric_delta_actor":
		e.applyActorFactionMetricDelta(silo, actor, mutation.Field, mutation.Value)
	case "faction_is_public_actor":
		e.applyActorFactionPublic(silo, actor, mutation.BoolValue)
	case "faction_found_actor":
		e.applyFactionFoundActor(silo, actor)
	case "faction_join_actor":
		e.applyFactionJoinActor(silo, actor, agent, uint(mutation.IntValue))
	case "faction_merge_actor":
		e.applyFactionMergeActor(silo, actor, agent, uint(mutation.IntValue))
	case "silo_flag_set":
		writeSiloFlag(silo, mutation.Field, mutation.BoolValue)
	case "victory_status_set":
		e.applyVictoryStatusSet(silo, mutation)
	case "score_set":
		e.applyScoreSet(silo, mutation)
	case "log":
		e.logf(mutation.Text)
	}
}

// ============ 死亡归因 (纯展示层) ============
//
// apply_population_deaths 只携带死亡总数，死因拆分在 Go 侧按
// events/mechanics/core_systems.js metrics_update 的死亡率公式逐项还原：
// 基础 0.001；每种耗尽资源 +0.05；楼层平均稳定度 < 0.8 时 +(0.8-均值)*0.25；
// 叛乱 > 0.8 时 +(叛乱-0.8)*0.2。全部为纯算术，不消耗 RNG、不修改任何
// 状态与数值，只产出一条带死因的月度日志。

// deathCausePart 死因拆分之一：Rate 为该死因在总死亡率中的份额。
type deathCausePart struct {
	Label string
	Rate  float64
}

// deathRateBreakdown 按 JS 侧公式把当前死亡率拆成逐项死因。
// 与 mechanicScriptContext 一致：资源按 Type→Amount 聚合后判定耗尽
// (同类型多条记录后者生效)，输出按类型名排序保证日志确定性。
func deathRateBreakdown(silo *model.Silo) []deathCausePart {
	if silo == nil {
		return nil
	}
	parts := []deathCausePart{{Label: "基础死亡", Rate: 0.001}}

	amounts := make(map[string]float64, len(silo.Resources))
	order := make([]string, 0, len(silo.Resources))
	for _, res := range silo.Resources {
		if _, ok := amounts[res.Type]; !ok {
			order = append(order, res.Type)
		}
		amounts[res.Type] = res.Amount
	}
	sort.Strings(order)
	for _, resType := range order {
		if amounts[resType] <= 0 {
			parts = append(parts, deathCausePart{Label: resourceLabelZH(resType) + "耗尽", Rate: 0.05})
		}
	}

	if len(silo.Floors) > 0 {
		total := 0.0
		for _, floor := range silo.Floors {
			total += floor.Stability
		}
		avg := total / float64(len(silo.Floors))
		if avg < 0.8 {
			parts = append(parts, deathCausePart{Label: "结构失稳", Rate: (0.8 - avg) * 0.25})
		}
	}

	if silo.Rebellion > 0.8 {
		parts = append(parts, deathCausePart{Label: "叛乱冲突", Rate: (silo.Rebellion - 0.8) * 0.2})
	}
	return parts
}

func resourceLabelZH(resType string) string {
	switch resType {
	case "Energy":
		return "能源"
	case "Materials":
		return "材料"
	case "Supplies":
		return "补给"
	default:
		return resType
	}
}

// splitDeathsByCauses 把死亡人数按各死因的死亡率份额拆分。确定性：
// 先逐项取整，余数按小数部分从大到小逐个补齐（并列按死因顺序），
// 保证各项之和恒等于 deaths。
func splitDeathsByCauses(deaths int, parts []deathCausePart) []int {
	counts := make([]int, len(parts))
	if deaths <= 0 || len(parts) == 0 {
		return counts
	}
	total := 0.0
	for _, p := range parts {
		total += p.Rate
	}
	if total <= 0 {
		counts[0] = deaths
		return counts
	}
	type fracEntry struct {
		index int
		frac  float64
	}
	fracs := make([]fracEntry, 0, len(parts))
	used := 0
	for i, p := range parts {
		exact := float64(deaths) * p.Rate / total
		whole := int(math.Floor(exact))
		counts[i] = whole
		used += whole
		fracs = append(fracs, fracEntry{index: i, frac: exact - float64(whole)})
	}
	// 稳定排序：小数部分相同保持原死因顺序，逐位确定。
	sort.SliceStable(fracs, func(i, j int) bool { return fracs[i].frac > fracs[j].frac })
	for i := 0; i < deaths-used && i < len(fracs); i++ {
		counts[fracs[i].index]++
	}
	return counts
}

// deathAttributionLine 生成月度死因归因日志行。
func deathAttributionLine(silo *model.Silo, deaths int, parts []deathCausePart) string {
	counts := splitDeathsByCauses(deaths, parts)
	segments := make([]string, 0, len(parts))
	for i, p := range parts {
		if counts[i] <= 0 {
			continue
		}
		segments = append(segments, fmt.Sprintf("%s %d", p.Label, counts[i]))
	}
	return fmt.Sprintf("[人口] %d年%d月死亡 %d 人：%s。",
		silo.CurrentYear, silo.CurrentMonth, deaths, strings.Join(segments, "、"))
}

func (e *GameEngine) applyActorMetricDelta(actor *ActorView, field string, delta float64) {
	if actor == nil {
		return
	}
	switch field {
	case "action_points":
		actor.SetActionPoints(math.Max(0, actor.ActionPoints()+delta))
	case "suspicion_level":
		actor.SetSuspicionLevel(math.Max(0, actor.SuspicionLevel()+delta))
	case "political_prestige":
		// 威望已全层派生 (24 篇 §2)：旧 prestige mutation 折算为 influence
		// (value/50)，保持内容语义近似。
		actor.SetInfluence(clamp01(actor.Influence() + delta/50))
	case "ambition":
		actor.SetAmbition(clamp01(actor.Ambition() + delta))
	case "influence":
		actor.SetInfluence(clamp01(actor.Influence() + delta))
	case "political_points":
		actor.SetPoliticalPoints(math.Max(0, actor.PoliticalPoints()+delta))
	case "propaganda_level":
		actor.SetPropagandaLevel(math.Max(0, actor.PropagandaLevel()+delta))
	}
}

func (e *GameEngine) applyActorMetricSet(actor *ActorView, field string, value float64) {
	if actor == nil {
		return
	}
	switch field {
	case "action_points":
		actor.SetActionPoints(math.Max(0, value))
	case "suspicion_level":
		actor.SetSuspicionLevel(math.Max(0, value))
	case "political_prestige":
		actor.SetInfluence(clamp01(value / 50))
	case "ambition":
		actor.SetAmbition(clamp01(value))
	case "influence":
		actor.SetInfluence(clamp01(value))
	case "political_points":
		actor.SetPoliticalPoints(math.Max(0, value))
	case "propaganda_level":
		actor.SetPropagandaLevel(math.Max(0, value))
	}
}

func (e *GameEngine) applyActorConnectionDelta(silo *model.Silo, actor *ActorView, dept string, delta float64) {
	if silo == nil || actor == nil || dept == "" {
		return
	}
	target := findDept(silo, dept)
	if target == nil {
		return
	}
	actor.AddConnection(target.ID, delta)
}

// applyActorBeliefObserve 吸收一次观察进玩家信念层 (§3.5)：observation_id
// 去重 (不重放 RNG 与收益)、B 更新、π 贝叶斯更新。信念层本轮只覆盖玩家
// (agent)；部门×阶层观察者属于分层曝光子项。
func (e *GameEngine) applyActorBeliefObserve(actor *ActorView, mutation MechanicMutation) {
	if actor == nil || !actor.IsPlayer() || mutation.Fragment == "" {
		return
	}
	agent := actor.Agent()
	if agent == nil {
		return
	}
	cfg := DefaultInformationConfig()
	spec := FragmentSpecFor(mutation.Fragment)
	belief := EnsureBelief(agent, mutation.Fragment)
	independence := mutation.Independence
	if independence <= 0 {
		independence = cfg.IndependenceNew
	}
	s := AbsorptionStrength(cfg, mutation.Reliability, mutation.RelatedF, independence)
	if !ObserveFact(cfg, belief, spec, mutation.Observation, s) {
		return
	}
	// KnownFragments 维持"已有观察"的记录语义 (UI/门控兼容)。
	e.applyActorFragmentAdd(actor, mutation.Fragment)
}

func (e *GameEngine) applyActorFragmentAdd(actor *ActorView, fragment string) {
	if actor == nil || fragment == "" {
		return
	}
	existing := actor.KnownFragments()
	for _, item := range existing {
		if item == fragment {
			return
		}
	}
	target := actor.agentOrProf()
	switch {
	case target.agent != nil:
		target.agent.KnownFragments = append(target.agent.KnownFragments, fragment)
	case target.resident != nil:
		target.resident.KnownFragments = append(target.resident.KnownFragments, fragment)
	case target.prof != nil:
		target.prof.KnownFragments = append(target.prof.KnownFragments, fragment)
	}
}

func (e *GameEngine) applyProfessionMetricDelta(prof *model.Profession, field string, delta float64) {
	switch field {
	case "panic_value":
		prof.PanicValue = clampUnit(prof.PanicValue + delta)
	case "productivity":
		// Clamp to [0,1]: the Go action fallback always clamped productivity;
		// the mutation path must not diverge (unbounded growth was observed).
		prof.Productivity = clampUnit(prof.Productivity + delta)
	case "action_points":
		prof.ActionPoints = math.Max(0, prof.ActionPoints+delta)
	case "alertness":
		// 内容可留下"被看见"的痕迹；月度冷却 (DecayProfessionAlertness) 负责回落。
		prof.Alertness = clampUnit(prof.Alertness + delta)
	}
}

func (e *GameEngine) applyProfessionMetricSet(prof *model.Profession, field string, value float64) {
	switch field {
	case "panic_value":
		prof.PanicValue = clampUnit(value)
	case "productivity":
		prof.Productivity = clampUnit(value)
	case "action_points":
		prof.ActionPoints = math.Max(0, value)
	}
}

func (e *GameEngine) applySiloMetricDeltaWithSupport(silo *model.Silo, metric string, delta float64) {
	if metric == "safeguard_risk" {
		silo.SafeguardRisk = math.Max(0, silo.SafeguardRisk+delta)
		return
	}
	applySiloMetricDelta(silo, metric, delta)
}

func (e *GameEngine) applySiloMetricSet(silo *model.Silo, metric string, value float64) {
	switch metric {
	case "safeguard_risk":
		silo.SafeguardRisk = math.Max(0, value)
	case "legitimacy":
		silo.SetMayorLegitimacy(value)
	case "cohesion":
		silo.Cohesion = clampUnit(value)
	case "rebellion":
		silo.Rebellion = clampUnit(value)
	case "dept_tension":
		silo.DeptTension = clampUnit(value)
	case "class_fragmentation":
		silo.ClassFragmentation = clampUnit(value)
	case "history_burden":
		silo.HistoryBurden = clampUnit(value)
	case "event_trigger":
		silo.EventTrigger = math.Max(0, value)
	case "countdown":
		silo.Countdown = math.Max(0, value)
	}
}

func (e *GameEngine) applyResourceDelta(silo *model.Silo, resource string, delta float64) {
	for i := range silo.Resources {
		if strings.EqualFold(silo.Resources[i].Type, resource) {
			silo.Resources[i].Amount = math.Max(0, silo.Resources[i].Amount+delta)
			return
		}
	}
}

func (e *GameEngine) applyResourceNetBalanceSet(silo *model.Silo, resource string, value float64) {
	for i := range silo.Resources {
		if strings.EqualFold(silo.Resources[i].Type, resource) {
			silo.Resources[i].NetBalance = value
			return
		}
	}
}

func (e *GameEngine) applyCohortIdeologyDeltaAll(silo *model.Silo, mutation MechanicMutation) {
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		if mutation.ClassType != "" {
			prof := getProfessionByID(silo, cohort.ProfessionID)
			if prof == nil || prof.ClassType != mutation.ClassType {
				continue
			}
		}
		if mutation.Profession != "" {
			prof := getProfessionByID(silo, cohort.ProfessionID)
			if prof == nil || prof.Name != mutation.Profession {
				continue
			}
		}
		if cohort.Ideologies == nil {
			cohort.Ideologies = map[string]float64{}
		}
		cohort.Ideologies[mutation.Ideology] = clampUnit(cohort.Ideologies[mutation.Ideology] + mutation.Value)
	}
}

func (e *GameEngine) applyCohortIdeologyDelta(silo *model.Silo, mutation MechanicMutation) {
	if silo == nil {
		return
	}
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		if mutation.IntValue != 0 && int(cohort.ID) != mutation.IntValue {
			continue
		}
		if mutation.Profession != "" {
			prof := getProfessionByID(silo, cohort.ProfessionID)
			if prof == nil || prof.Name != mutation.Profession {
				continue
			}
		}
		if cohort.Ideologies == nil {
			cohort.Ideologies = map[string]float64{}
		}
		cohort.Ideologies[mutation.Ideology] = clampUnit(cohort.Ideologies[mutation.Ideology] + mutation.Value)
		return
	}
}

func (e *GameEngine) syncProfessionIdeologiesFromCohorts(silo *model.Silo) {
	if silo == nil {
		return
	}
	for i := range silo.Professions {
		p := &silo.Professions[i]
		newIdeologies := make(map[string]float64)
		totalPop := 0.0
		for _, c := range silo.Cohorts {
			if c.ProfessionID == p.ID {
				weight := float64(c.Count)
				for k, v := range c.Ideologies {
					newIdeologies[k] += v * weight
				}
				totalPop += weight
			}
		}
		if totalPop > 0 {
			for k := range newIdeologies {
				newIdeologies[k] /= totalPop
			}
			p.Ideologies = newIdeologies
		}
	}
}

func (e *GameEngine) applyActorFactionMetricDelta(silo *model.Silo, actor *ActorView, field string, delta float64) {
	faction := actorFaction(silo, actor)
	if faction == nil {
		return
	}
	switch field {
	case "prestige":
		faction.Prestige = math.Max(0, faction.Prestige+delta)
	case "influence":
		faction.Influence = math.Max(0, faction.Influence+delta)
	case "cohesion":
		faction.Cohesion = clampUnit(faction.Cohesion + delta)
	}
}

func (e *GameEngine) applyActorFactionPublic(silo *model.Silo, actor *ActorView, value bool) {
	faction := actorFaction(silo, actor)
	if faction == nil {
		return
	}
	faction.IsPublic = value
}

func (e *GameEngine) applyVictoryStatusSet(silo *model.Silo, mutation MechanicMutation) {
	if mutation.Text == "" {
		silo.VictoryStatus = nil
		return
	}
	// Mechanic fields are normalized to lowercase for metric names; ending IDs
	// are enum values and must retain the canonical uppercase used by Go.
	statusType := strings.ToUpper(mutation.Field)
	silo.VictoryStatus = &model.VictoryStatus{
		IsWon:       mutation.BoolValue,
		Type:        statusType,
		Description: mutation.Text,
	}
}

func (e *GameEngine) applyScoreSet(silo *model.Silo, mutation MechanicMutation) {
	if silo.VictoryStatus == nil {
		silo.VictoryStatus = &model.VictoryStatus{}
	}
	var payload struct {
		Total           int     `json:"total"`
		SurvivalPoints  int     `json:"survival_points"`
		DiversityPoints int     `json:"diversity_points"`
		HeritagePoints  int     `json:"heritage_points"`
		IdeologyPoints  int     `json:"ideology_points"`
		Multiplier      float64 `json:"multiplier"`
	}
	if err := json.Unmarshal([]byte(mutation.Text), &payload); err != nil {
		return
	}
	silo.VictoryStatus.Score = &model.GameScore{
		Total:           payload.Total,
		SurvivalPoints:  payload.SurvivalPoints,
		DiversityPoints: payload.DiversityPoints,
		HeritagePoints:  payload.HeritagePoints,
		IdeologyPoints:  payload.IdeologyPoints,
		Multiplier:      payload.Multiplier,
	}
}

func removeString(items []string, target string) []string {
	out := items[:0]
	for _, item := range items {
		if item != target {
			out = append(out, item)
		}
	}
	return out
}

func (e *GameEngine) runEventMechanic(eventType string, event *GameEvent, ctx *EventContext) {
	result, err := e.runMechanicForEvent(event, ctx, nil)
	if err == nil {
		silo, _ := event.Data["silo"].(*model.Silo)
		agent, _ := event.Data["agent"].(*model.Agent)
		e.runMonthlyBeliefDispatch(eventType, event, silo)
		e.runMonthlyPoliticalDispatch(eventType, event, silo)
		e.applyWeightedEnding(eventType, silo, agent)
		if len(result.Logs) > 0 {
			for _, entry := range result.Logs {
				e.logf(entry)
			}
		}
		return
	}

	silo, _ := event.Data["silo"].(*model.Silo)
	agent, _ := event.Data["agent"].(*model.Agent)
	deltaYears, _ := event.Data["deltaYears"].(float64)
	switch eventType {
	case EVENT_AGENT_UPDATE:
		e.UpdateAgentState(agent, deltaYears, silo, e.logf)
	case EVENT_RESOURCE_UPDATE:
		e.updateResources(silo, deltaYears)
	case EVENT_METRICS_UPDATE:
		e.updateSiloMetrics(silo, deltaYears)
	case EVENT_IDEOLOGY_UPDATE:
		e.updateIdeology(silo, deltaYears)
	case EVENT_VICTORY_CHECK:
		e.checkVictoryConditions(silo, agent)
	}
	e.runMonthlyBeliefDispatch(eventType, event, silo)
	e.runMonthlyPoliticalDispatch(eventType, event, silo)
	e.applyWeightedEnding(eventType, silo, agent)
}

// runMonthlyBeliefDispatch 月度信念层结算：恒定淡忘 + 时间渗透
// (25-information-economy §修复A、18-time-permeation §3.1)，确定性。
//
// 每月每部门恰好一次：RunLocalPhase 每个 tick 只在步骤 6 发一次
// EVENT_METRICS_UPDATE；runEventMechanic 的 JS 成功路与 Go 兜底路各调用本
// 函数一次，但两路互斥 (成功路结算后直接 return)，故每个事件恰好一次。
//
// 时序裁决 (显式)：先淡忘、再计数、再判成熟。B 恰好卡在分享地板的部门当月
// 先被衰减到地板下、不计渗透数；只有被持续流通托住 (衰减后仍 ≥ 地板) 的
// 部门才累计。淡忘与渗透都只按既有持久化顺序遍历部门层，不耗 RNG。
//
// deltaYears <= 0 的零月路径不结算 (与 JS 侧守卫一致)。JS metrics_update
// 不写部门信念层，淡忘与渗透都没有 JS 对应物，只在这里执行，不会双跑。
func (e *GameEngine) runMonthlyBeliefDispatch(eventType string, event *GameEvent, silo *model.Silo) {
	if eventType != EVENT_METRICS_UPDATE || silo == nil {
		return
	}
	deltaYears, _ := event.Data["deltaYears"].(float64)
	if deltaYears <= 0 {
		return
	}
	cfg := DefaultInformationConfig()
	ApplyBeliefForgetting(silo, cfg)
	ApplyPermeation(silo, cfg)
}

// runMonthlyPoliticalDispatch 月度政治层结算：叛乱侵蚀市长个人正统性
// (ADR-0007 后果条、18 §4)，确定性。
//
// 与 runMonthlyBeliefDispatch 同型：runEventMechanic 的 JS 成功路与 Go
// 兜底路各调用本函数一次，但两路互斥，故每个 EVENT_METRICS_UPDATE 恰好
// 一次。此前侵蚀只挂在 Go 兜底的 updateSiloMetrics 里，而 JS 的
// metrics_update 总是成功、兜底不可达——真实流程里侵蚀从未发生。现在
// 无论 JS 成败都在同一时点执行（读当月更新后的叛乱值），且
// updateSiloMetrics 不再调用它：全引擎只有这一条月度正统性路径。
//
// deltaYears <= 0 的零月路径不结算 (与 JS 侧守卫一致)；纯算术不耗 RNG，
// 叛乱为 0 时逐位不变。
func (e *GameEngine) runMonthlyPoliticalDispatch(eventType string, event *GameEvent, silo *model.Silo) {
	if eventType != EVENT_METRICS_UPDATE || silo == nil {
		return
	}
	deltaYears, _ := event.Data["deltaYears"].(float64)
	if deltaYears <= 0 {
		return
	}
	ApplyRebellionLegitimacyErosion(silo, deltaYears)
}

// applyWeightedEnding lets the service arbitrate the terminal evaluation
// requested by victory_check without giving the standalone engine access to
// world-level entities. A decision with an empty Selected means "no qualified
// candidate — continue simulating": the request is cleared and the game goes on.
func (e *GameEngine) applyWeightedEnding(eventType string, silo *model.Silo, agent *model.Agent) {
	if eventType != EVENT_VICTORY_CHECK || e.WeightedEndingResolver == nil || silo == nil {
		return
	}
	decision := e.WeightedEndingResolver(silo, agent)
	if decision == nil {
		return
	}
	if decision.Selected == "" {
		silo.VictoryStatus = nil
		return
	}
	if silo.VictoryStatus == nil {
		silo.VictoryStatus = &model.VictoryStatus{}
	}
	silo.VictoryStatus.Type = decision.Selected
	silo.VictoryStatus.IsWon = !isHardFailureEnding(decision.Selected)
	// The arbitrator now writes the settlement narrative into Reason
	// (weighted_endings.go endingDescriptions); the ending screen renders
	// Description, so the story replaces the score formula. Empty Reason
	// falls back to the composed audit line (25-information-endgame §2.2).
	if decision.Reason != "" {
		silo.VictoryStatus.Description = decision.Reason
	} else {
		silo.VictoryStatus.Description = "多主体终局仲裁：" + decision.Selected + "。"
	}
	silo.VictoryStatus.Decision = decision
}

// isHardFailureEnding marks P0 guardrail endings that end the game as failures
// and can never be overridden by weighted scoring.
func isHardFailureEnding(outcome string) bool {
	switch outcome {
	case "DEATH", "RESET", "FRACTURE", "AGENT_COMPROMISED":
		return true
	}
	return false
}

func (e *GameEngine) applyMechanicMetadata() {
	if e == nil || e.MechanicEngine == nil {
		return
	}
	for actionType, def := range e.MechanicEngine.byActionType {
		if def.APCost > 0 {
			model.ACTION_COSTS[actionType] = float64(def.APCost)
		}
		if def.DurationMonths >= 0 {
			model.ACTION_DURATIONS[actionType] = def.DurationMonths
		}
	}
	professionDefs := e.MechanicEngine.ProfessionActionDefinitions()
	if len(professionDefs) == 0 {
		return
	}
	actions := make([]*ProfessionAction, 0, len(professionDefs))
	for _, def := range professionDefs {
		actions = append(actions, &ProfessionAction{
			ID:               def.ProfessionAction,
			Profession:       def.Profession,
			Label:            def.Label,
			Description:      def.Description,
			APCost:           float64(def.APCost),
			TargetType:       ProfessionActionTargetType(def.TargetType),
			SuspicionPenalty: def.SuspicionPenalty,
		})
	}
	PROFESSION_ACTIONS = actions
}

func (e *GameEngine) runMechanicForEvent(event *GameEvent, ctx *EventContext, actor *ActorView) (MechanicResult, error) {
	if e.MechanicEngine == nil || event == nil {
		return MechanicResult{}, fmt.Errorf("mechanic engine unavailable")
	}
	var (
		def MechanicDefinition
		ok  bool
	)
	if action, hasAction := event.Data["action"].(model.AgentAction); hasAction {
		def, ok = e.MechanicEngine.ActionDefinition(action)
	} else {
		def, ok = e.MechanicEngine.EventDefinition(event.Type)
	}
	if !ok {
		return MechanicResult{}, fmt.Errorf("no mechanic definition for %s", event.Type)
	}
	silo, _ := event.Data["silo"].(*model.Silo)
	agent, _ := event.Data["agent"].(*model.Agent)
	if actor == nil && silo != nil && agent != nil {
		actor, _ = CreateActorView(CreateActorRefForAgent(agent, silo), silo, agent)
	}
	result, err := e.MechanicEngine.Run(def, MechanicContext{
		Event: event,
		Silo:  silo,
		Agent: agent,
		Actor: actor,
	})
	if err != nil {
		return MechanicResult{}, err
	}
	e.applyMechanicMutations(event, result.Mutations, ctx)
	for _, entry := range result.Logs {
		e.logf(entry)
	}
	return result, nil
}
