package engine

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
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

	// 预编译 Prelude
	if prog, err := goja.Compile("mechanic_prelude.js", mechanicPrelude, true); err == nil {
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
	runtime.SetRandSource(rand.Float64)
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
`

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
	runtime.SetRandSource(rand.Float64)
	if _, err := runtime.RunString(mechanicPrelude); err != nil {
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
	runtime.SetRandSource(rand.Float64)
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
			"traits":           append([]string(nil), ctx.Silo.Traits...),
			"metrics": map[string]float64{
				"legitimacy":          ctx.Silo.Legitimacy,
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
			professions = append(professions, map[string]interface{}{
				"id":              prof.ID,
				"name":            prof.Name,
				"class_type":      prof.ClassType,
				"population":      prof.Population,
				"panic_value":     prof.PanicValue,
				"productivity":    prof.Productivity,
				"power_level":     prof.PowerLevel,
				"action_points":   prof.ActionPoints,
				"known_fragments": append([]string(nil), prof.KnownFragments...),
				"all_fragments":   allFragments,
				"ideologies":      cloneFloatMap(prof.Ideologies),
				"relations":       cloneFloatMap(prof.Relations),
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
			"political_points":   ctx.Actor.PoliticalPoints(),
			"propaganda_level":   ctx.Actor.PropagandaLevel(),
			"traits":             append([]string(nil), ctx.Actor.Traits()...),
			"known_fragments":    append([]string(nil), ctx.Actor.KnownFragments()...),
			"is_representative":  ctx.Actor.IsRepresentative(),
			"is_faction_leader":  actorLeadsFaction(ctx.Silo, ctx.Actor),
			"connections":        []map[string]interface{}{},
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
		applyPopulationDeathsToCohorts(silo, mutation.IntValue)
	case "refresh_profession_population_from_cohorts":
		refreshProfessionPopulationFromCohorts(silo)
	case "faction_metric_delta_actor":
		e.applyActorFactionMetricDelta(silo, actor, mutation.Field, mutation.Value)
	case "faction_is_public_actor":
		e.applyActorFactionPublic(silo, actor, mutation.BoolValue)
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
		actor.SetPoliticalPrestige(math.Max(0, actor.PoliticalPrestige()+delta))
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
		actor.SetPoliticalPrestige(math.Max(0, value))
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
		prof.Productivity = math.Max(0, prof.Productivity+delta)
	case "action_points":
		prof.ActionPoints = math.Max(0, prof.ActionPoints+delta)
	}
}

func (e *GameEngine) applyProfessionMetricSet(prof *model.Profession, field string, value float64) {
	switch field {
	case "panic_value":
		prof.PanicValue = clampUnit(value)
	case "productivity":
		prof.Productivity = math.Max(0, value)
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
		silo.Legitimacy = clampUnit(value)
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
	statusType := mutation.Field
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
