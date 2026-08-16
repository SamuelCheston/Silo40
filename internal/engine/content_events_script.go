package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
)

const contentEventJSPrelude = `
var __contentEventCaptured = null;
function defineEvent(def) {
  __contentEventCaptured = def;
  return def;
}
function all() {
  return { type: "all", conditions: Array.prototype.slice.call(arguments) };
}
function any() {
  return { type: "any", conditions: Array.prototype.slice.call(arguments) };
}
function timestampAtOrAfter(year, month) {
  return { type: "timestamp_at_or_after", year: year, month: month };
}
function siloMetricGte(metric, value) {
  return { type: "silo_metric_gte", metric: metric, value: value };
}
function siloMetricLte(metric, value) {
  return { type: "silo_metric_lte", metric: metric, value: value };
}
function siloFlagTrue(flag) {
  return { type: "silo_flag_true", flag: flag };
}
function siloFlagFalse(flag) {
  return { type: "silo_flag_false", flag: flag };
}
function eventTriggered(category, eventId) {
  return { type: "event_triggered", category: category, event_id: eventId };
}
function playerActionIs(action) {
  return { type: "player_action_is", action: action };
}
function professionMetricGteCount(metric, value, count, profession) {
  return { type: "profession_metric_gte_count", metric: metric, value: value, count: count, profession: profession };
}
function professionIdeologyGteCount(ideology, value, count, profession) {
  return { type: "profession_ideology_gte_count", ideology: ideology, value: value, count: count, profession: profession };
}
function siloMetricDelta(metric, value) {
  return { type: "silo_metric_delta", metric: metric, value: value };
}
function professionMetricDeltaAll(metric, value) {
  return { type: "profession_metric_delta_all", metric: metric, value: value };
}
function professionIdeologyDeltaAll(ideology, value) {
  return { type: "profession_ideology_delta_all", ideology: ideology, value: value };
}
function siloFlagSet(flag, boolValue) {
  return { type: "silo_flag_set", flag: flag, bool_value: !!boolValue };
}
function __contentStrip(value) {
  if (Array.isArray(value)) {
    return value.map(__contentStrip);
  }
  if (!value || typeof value !== "object") {
    return value;
  }
  var out = {};
  for (var key in value) {
    if (!Object.prototype.hasOwnProperty.call(value, key)) {
      continue;
    }
    if (typeof value[key] === "function") {
      continue;
    }
    out[key] = __contentStrip(value[key]);
  }
  return out;
}
function __contentEnsureCaptured() {
  if (!__contentEventCaptured || typeof __contentEventCaptured !== "object" || Array.isArray(__contentEventCaptured)) {
    throw new Error("content event files must call defineEvent({...}) exactly once");
  }
  return __contentEventCaptured;
}
function __contentExportEvent() {
  return JSON.stringify(__contentStrip(__contentEnsureCaptured()));
}
function __contentHookFlags() {
  var eventDef = __contentEnsureCaptured();
  var script = eventDef.script || {};
  return JSON.stringify({
    canTrigger: typeof script.canTrigger === "function",
    apply: typeof script.apply === "function"
  });
}
`

type ContentScriptApplyResult struct {
	Effects []ContentEffect `json:"effects,omitempty"`
	Emit    []string        `json:"emit,omitempty"`
}

func loadContentEventJSFile(sourceGroup, baseDir, path string) ([]ContentEventDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read content event file %s: %w", path, err)
	}

	def, err := parseContentEventJSDefinition(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse content event file %s: %w", path, err)
	}

	relPath, relErr := filepath.Rel(baseDir, path)
	if relErr != nil {
		relPath = filepath.Base(path)
	}
	baseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if def.EventID == "" {
		def.EventID = baseName
	}
	def.SourceGroup = sourceGroup
	def.SourceFile = filepath.ToSlash(relPath)
	def.SourceFormat = "javascript"
	def.ScriptSource = string(data)
	def.Key = sourceGroup + ":" + def.EventID
	normalizeContentEventDefinition(&def)
	if err := ValidateContentEventDefinition(def); err != nil {
		return nil, fmt.Errorf("invalid content event in %s: %w", path, err)
	}
	return []ContentEventDefinition{def}, nil
}

func parseContentEventJSDefinition(source string) (ContentEventDefinition, error) {
	runtime, _, err := loadContentEventScriptRuntime(source)
	if err != nil {
		return ContentEventDefinition{}, err
	}

	metadataJSON, err := runtime.RunString("__contentExportEvent()")
	if err != nil {
		return ContentEventDefinition{}, err
	}
	hookJSON, err := runtime.RunString("__contentHookFlags()")
	if err != nil {
		return ContentEventDefinition{}, err
	}

	var def ContentEventDefinition
	if err := json.Unmarshal([]byte(metadataJSON.String()), &def); err != nil {
		return ContentEventDefinition{}, err
	}
	if err := json.Unmarshal([]byte(hookJSON.String()), &def.ScriptHooks); err != nil {
		return ContentEventDefinition{}, err
	}
	return def, nil
}

func RunContentEventCanTriggerScript(def ContentEventDefinition, ctx ContentEvaluationContext, ignoreActionMatch bool) (bool, error) {
	if !def.ScriptHooks.CanTrigger {
		return true, nil
	}
	runtime, eventObj, err := loadContentEventScriptRuntime(def.ScriptSource)
	if err != nil {
		return false, err
	}
	callable, scriptObj, err := contentEventScriptHook(runtime, eventObj, "canTrigger")
	if err != nil {
		return false, err
	}
	value, err := callable(scriptObj, runtime.ToValue(contentScriptContext(ctx, ignoreActionMatch)))
	if err != nil {
		return false, err
	}
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return false, nil
	}
	return value.ToBoolean(), nil
}

func RunContentEventApplyScript(def ContentEventDefinition, ctx ContentEvaluationContext) (ContentScriptApplyResult, error) {
	if !def.ScriptHooks.Apply {
		return ContentScriptApplyResult{}, nil
	}
	runtime, eventObj, err := loadContentEventScriptRuntime(def.ScriptSource)
	if err != nil {
		return ContentScriptApplyResult{}, err
	}
	callable, scriptObj, err := contentEventScriptHook(runtime, eventObj, "apply")
	if err != nil {
		return ContentScriptApplyResult{}, err
	}
	value, err := callable(scriptObj, runtime.ToValue(contentScriptContext(ctx, false)))
	if err != nil {
		return ContentScriptApplyResult{}, err
	}
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return ContentScriptApplyResult{}, nil
	}
	payload, err := normalizeContentScriptApplyResult(value.Export())
	if err != nil {
		return ContentScriptApplyResult{}, err
	}
	return payload, nil
}

func loadContentEventScriptRuntime(source string) (*goja.Runtime, *goja.Object, error) {
	runtime := goja.New()
	if _, err := runtime.RunString(contentEventJSPrelude); err != nil {
		return nil, nil, err
	}
	if _, err := runtime.RunString(source); err != nil {
		return nil, nil, err
	}
	value := runtime.Get("__contentEventCaptured")
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, nil, fmt.Errorf("content event file did not call defineEvent")
	}
	return runtime, value.ToObject(runtime), nil
}

func contentEventScriptHook(runtime *goja.Runtime, eventObj *goja.Object, hookName string) (goja.Callable, goja.Value, error) {
	scriptValue := eventObj.Get("script")
	if goja.IsUndefined(scriptValue) || goja.IsNull(scriptValue) {
		return nil, nil, fmt.Errorf("content event script hook %q is missing", hookName)
	}
	scriptObject := scriptValue.ToObject(runtime)
	hookValue := scriptObject.Get(hookName)
	callable, ok := goja.AssertFunction(hookValue)
	if !ok {
		return nil, nil, fmt.Errorf("content event script hook %q is not a function", hookName)
	}
	return callable, scriptObject, nil
}

func contentScriptContext(ctx ContentEvaluationContext, ignoreActionMatch bool) map[string]interface{} {
	result := map[string]interface{}{
		"year":                0,
		"month":               0,
		"ignore_action_match": ignoreActionMatch,
		"silo_metrics":        map[string]float64{},
		"silo_flags":          map[string]bool{},
		"professions":         []map[string]interface{}{},
		"runtime": map[string]interface{}{
			"triggered":                ctx.Runtime.Triggered,
			"trigger_count":            ctx.Runtime.TriggerCount,
			"last_triggered_timestamp": ctx.Runtime.LastTriggeredTimestamp,
		},
		"states": map[string]map[string]interface{}{},
	}
	if ctx.Silo != nil {
		result["year"] = ctx.Silo.CurrentYear
		result["month"] = ctx.Silo.CurrentMonth
		result["silo_metrics"] = map[string]float64{
			"legitimacy":          ctx.Silo.Legitimacy,
			"cohesion":            ctx.Silo.Cohesion,
			"rebellion":           ctx.Silo.Rebellion,
			"dept_tension":        ctx.Silo.DeptTension,
			"class_fragmentation": ctx.Silo.ClassFragmentation,
			"history_burden":      ctx.Silo.HistoryBurden,
			"event_trigger":       ctx.Silo.EventTrigger,
			"countdown":           ctx.Silo.Countdown,
		}
		result["silo_flags"] = map[string]bool{
			"silo1_destroyed": ctx.Silo.Silo1Destroyed,
		}
		professions := make([]map[string]interface{}, 0, len(ctx.Silo.Professions))
		for _, profession := range ctx.Silo.Professions {
			ideologies := map[string]float64{}
			for ideology, value := range profession.Ideologies {
				ideologies[ideology] = value
			}
			professions = append(professions, map[string]interface{}{
				"name":         profession.Name,
				"class_type":   profession.ClassType,
				"panic_value":  profession.PanicValue,
				"productivity": profession.Productivity,
				"ideologies":   ideologies,
			})
		}
		result["professions"] = professions
	}
	if ctx.Action != nil {
		result["action"] = map[string]interface{}{
			"type":              string(ctx.Action.Type),
			"action_id":         ctx.Action.ActionID,
			"target_dept":       ctx.Action.TargetDept,
			"fragment_ids":      append([]string(nil), ctx.Action.FragmentIds...),
			"profession_action": ctx.Action.ProfessionAction,
			"resource_target":   ctx.Action.ResourceTarget,
			"cost":              ctx.Action.Cost,
		}
	}
	states := map[string]map[string]interface{}{}
	for key, runtime := range ctx.States {
		states[key] = map[string]interface{}{
			"triggered":                runtime.Triggered,
			"trigger_count":            runtime.TriggerCount,
			"last_triggered_timestamp": runtime.LastTriggeredTimestamp,
		}
	}
	result["states"] = states
	return result
}

func normalizeContentScriptApplyResult(exported any) (ContentScriptApplyResult, error) {
	raw, err := json.Marshal(exported)
	if err != nil {
		return ContentScriptApplyResult{}, err
	}
	var payload ContentScriptApplyResult
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ContentScriptApplyResult{}, err
	}
	for i := range payload.Effects {
		payload.Effects[i].Type = strings.ToLower(strings.TrimSpace(payload.Effects[i].Type))
		payload.Effects[i].Metric = strings.ToLower(strings.TrimSpace(payload.Effects[i].Metric))
		payload.Effects[i].Flag = strings.ToLower(strings.TrimSpace(payload.Effects[i].Flag))
		payload.Effects[i].Profession = strings.TrimSpace(payload.Effects[i].Profession)
		if err := validateEffect(payload.Effects[i]); err != nil {
			return ContentScriptApplyResult{}, err
		}
	}
	for i := range payload.Emit {
		payload.Emit[i] = strings.TrimSpace(payload.Emit[i])
		if payload.Emit[i] == "" {
			return ContentScriptApplyResult{}, fmt.Errorf("script.apply emit contains an empty event name")
		}
	}
	return payload, nil
}
