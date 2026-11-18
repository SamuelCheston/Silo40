package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"silo40/internal/model"

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
function worldFlagTrue(flag) {
  return { type: "world_flag_true", flag: flag };
}
function worldFlagFalse(flag) {
  return { type: "world_flag_false", flag: flag };
}
function worldFlagNotSet(flag) {
  // 守卫条件：世界事实未成立即通过 (无世界上下文时也算未成立)。
  return { type: "world_flag_not_set", flag: flag };
}
function signalReceived(eventId) {
  return { type: "signal_received", event_id: eventId };
}
function eventTriggered(category, eventId) {
  return { type: "event_triggered", category: category, event_id: eventId };
}
function monthsSinceEventGte(category, eventId, months) {
  // 相对时间门：距事件点亮至少 months 月 (锚定事件而非绝对日历)。
  return { type: "months_since_event_gte", category: category, event_id: eventId, months: months };
}
function actorMetricGte(metric, value) {
  // 操作者指标门 (宣传等级/怀疑度/威望)：反应链用它识别"有组织的煽动"。
  return { type: "actor_metric_gte", metric: metric, value: value };
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
function __withScope(effect, scope) {
  if (scope !== undefined && scope !== null && scope !== "") {
    effect.scope = scope;
  }
  return effect;
}
function siloMetricDelta(metric, value, scope) {
  return __withScope({ type: "silo_metric_delta", metric: metric, value: value }, scope);
}
function professionMetricDeltaAll(metric, value, scope) {
  return __withScope({ type: "profession_metric_delta_all", metric: metric, value: value }, scope);
}
function professionIdeologyDeltaAll(ideology, value, scope) {
  return __withScope({ type: "profession_ideology_delta_all", ideology: ideology, value: value }, scope);
}
function siloResourceDelta(metric, value, scope) {
  return __withScope({ type: "silo_resource_delta", metric: metric, value: value }, scope);
}
function siloPopulationDelta(value, scope) {
  return __withScope({ type: "silo_population_delta", value: value }, scope);
}
function scheduleEvent(eventId, delayMonths) {
  return { type: "schedule_event", event_id: eventId, delay_months: delayMonths };
}
function crossSignal(scope, eventId, delayMonths) {
  return { type: "cross_signal", scope: scope, event_id: eventId, delay_months: delayMonths || 0 };
}
function siloFlagSet(flag, boolValue, scope) {
  return __withScope({ type: "silo_flag_set", flag: flag, bool_value: !!boolValue }, scope);
}
function worldFlagSet(flag, boolValue) {
  return { type: "world_flag_set", flag: flag, bool_value: !!boolValue };
}
function professionFragmentAdd(profession, fragment, scope) {
  return __withScope({ type: "profession_fragment_add", profession: profession, fragment: fragment }, scope);
}
function professionFragmentRemove(profession, fragment, scope) {
  return __withScope({ type: "profession_fragment_remove", profession: profession, fragment: fragment }, scope);
}
function floorStabilityDelta(zone, value, scope) {
  return __withScope({ type: "floor_stability_delta", zone: zone, value: value }, scope);
}
function cohortIdeologyDelta(ideology, value, profession, scope) {
  return __withScope({ type: "cohort_ideology_delta", ideology: ideology, value: value, profession: profession || "" }, scope);
}
function authorityEvidenceDelta(value, scope) {
  return __withScope({ type: "authority_evidence_delta", value: value }, scope);
}
function authorityPublicFact(key, scope) {
  return __withScope({ type: "authority_public_fact", flag: key }, scope);
}
function authorityPactQualify(boolValue, scope) {
  return __withScope({ type: "authority_pact_qualify", bool_value: !!boolValue }, scope);
}
function escapeGroupDepart(count, scope) {
  return __withScope({ type: "escape_group_depart", value: count }, scope);
}
function escapeGroupStatus(boolValue, scope) {
  return __withScope({ type: "escape_group_status", bool_value: !!boolValue }, scope);
}
function topStanceReeval(actor) {
  return { type: "top_stance_reeval", flag: actor || "both" };
}
function topStanceGuarantee(delta) {
  return { type: "top_stance_guarantee", value: delta || 0 };
}
function preparationDelta(field, value) {
  return { type: "preparation_delta", metric: field, value: value || 0 };
}
function preparationSetBool(field, boolValue) {
  return { type: "preparation_delta", metric: field, bool_value: !!boolValue };
}
function preparationStage(stage) {
  return { type: "preparation_delta", metric: "reception_stage", flag: stage };
}
function preparationPlanRole(role, value) {
  return { type: "preparation_delta", metric: "plan_role", profession: role, value: (value === undefined || value === null) ? 1 : value };
}
function preparationTransfer(resource, amount) {
  return { type: "preparation_transfer", metric: resource, value: amount };
}
function spilloverChannelAdd(kind, targetSilo, reliability) {
  return { type: "spillover_channel_add", flag: kind, count: targetSilo, value: reliability || 0 };
}
function spilloverSend(kind) {
  // 16 §3.5 (Layer 3)：请求经一条已建立的受限信道投递外溢包裹。
  // 目标堡、可靠度与暴露代价读建立侧账本；结算在服务层执行。
  return { type: "spillover_send", flag: kind };
}
function orgSuspicionDelta(value) {
  // 20 修订/ADR-0014：组织行为 (动员/通信) 累积组织怀疑度；囤积类动作禁用。
  return { type: "escape_org_suspicion_delta", value: value || 0 };
}
function preparationGte(field, value) {
  return { type: "preparation_gte", metric: field, value: value };
}
function escapeDepartureSettle() {
  return { type: "escape_departure_settle" };
}
function negotiationOpen() {
  return { type: "negotiation_open" };
}
function negotiationRound(kind) {
  return { type: "negotiation_round", flag: kind || "probe" };
}
function negotiationResolve(kind) {
  return { type: "negotiation_resolve", flag: kind || "hands_off" };
}
function completePlayerAction(actionId) {
  return { type: "player_action_complete", action: actionId };
}
function beliefObserve(observer, fragment, reliability) {
  return { type: "belief_observe", observer: observer, fragment: fragment, reliability: reliability || 0.7 };
}
function publicEvidenceRegister(fragment, provenance, coverage) {
  return { type: "public_evidence_register", fragment: fragment, provenance: provenance, coverage: coverage };
}
function evidenceVerify(fragment) {
  return { type: "evidence_verify", fragment: fragment };
}
function authorityWShock(value) {
  return { type: "authority_w_shock", value: value };
}
function decisiveShock(source, value) {
  return { type: "decisive_shock", flag: source || "", value: value || 0 };
}
function allianceLoyaltyDelta(group, value) {
  return { type: "alliance_loyalty_delta", observer: group, value: value };
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
	Effects  []ContentEffect         `json:"effects,omitempty"`
	Emit     []string                `json:"emit,omitempty"`
	Schedule []ContentScheduledEvent `json:"schedule,omitempty"`
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
		"world_flags":         map[string]bool{},
		"professions":         []map[string]interface{}{},
		"factions":            []map[string]interface{}{},
		"runtime": map[string]interface{}{
			"triggered":                ctx.Runtime.Triggered,
			"trigger_count":            ctx.Runtime.TriggerCount,
			"last_triggered_timestamp": ctx.Runtime.LastTriggeredTimestamp,
		},
		"states": map[string]map[string]interface{}{},
	}
	if ctx.World != nil {
		worldFlags := map[string]bool{}
		for flag, value := range ctx.World.Flags {
			worldFlags[flag] = value
		}
		worldFlags[model.WorldFlagSilo1Destroyed] = !ctx.World.Silo1.Alive
		result["world_flags"] = worldFlags
	}
	if ctx.Silo != nil {
		result["year"] = ctx.Silo.CurrentYear
		result["month"] = ctx.Silo.CurrentMonth
		result["total_population"] = ctx.Silo.TotalPopulation
		// E 次消费点读数 (22-00 裁决 1)：反应链触发后的执行烈度调制。
		// 未启用时按 1.0 (全力落实) 暴露，旧行为逐位不变；触发判定不读此值。
		executionIntensity := 1.0
		if ctx.Silo.BlackBox.Initialized {
			executionIntensity = ctx.Silo.BlackBox.CurrentExecution
		}
		result["execution_intensity"] = executionIntensity
		result["silo_metrics"] = map[string]float64{
			"legitimacy":          ctx.Silo.MayorLegitimacy(),
			"cohesion":            ctx.Silo.Cohesion,
			"rebellion":           ctx.Silo.Rebellion,
			"dept_tension":        ctx.Silo.DeptTension,
			"class_fragmentation": ctx.Silo.ClassFragmentation,
			"history_burden":      ctx.Silo.HistoryBurden,
			"event_trigger":       ctx.Silo.EventTrigger,
			"countdown":           ctx.Silo.Countdown,
			"safeguard_risk":      ctx.Silo.SafeguardRisk,
		}
		result["silo_flags"] = map[string]bool{
			"silo1_destroyed": ctx.Silo.Silo1Destroyed,
		}

		// G3: authority state is readable by content (W, judicial attachment,
		// pact qualification, police obedience). Mutation stays on the Go side.
		judicial := ctx.Silo.Authority.Judicial
		result["authority"] = map[string]interface{}{
			"initialized":              ctx.Silo.Authority.Initialized,
			"w":                        ctx.Silo.Authority.W,
			"police_obedience":         ctx.Silo.Authority.PoliceObedience,
			"a_it":                     judicial.IT,
			"a_mayor":                  judicial.Mayor,
			"a_pact":                   judicial.Pact,
			"regime":                   judicial.Regime,
			"pact_qualified":           judicial.PactQualified,
			"pact_locked":              judicial.PactLocked,
			"public_evidence_q":        judicial.PublicEvidenceQ,
			"independent_public_facts": judicial.IndependentPublicFacts,
		}
		result["escaped_group"] = map[string]interface{}{
			"initialized": ctx.Silo.EscapedGroup.Initialized,
			"population":  ctx.Silo.EscapedGroup.Population,
			"surviving":   ctx.Silo.EscapedGroup.Surviving,
		}
		// 16 §3.2 (ADR-0008)：顶层立场只读视图。立场是内部状态：内容可读它
		// 塑形准备动作的可得性 (阻挠下修服被审计、配合下库开启)，但永不作为
		// 玩家交互选项，也不进仲裁。
		result["top_stance"] = map[string]interface{}{
			"initialized": ctx.Silo.TopStance.Initialized,
			"mayor":       ctx.Silo.TopStance.MayorStance(),
			"it":          ctx.Silo.TopStance.ITStance(),
			"guarantee":   ctx.Silo.TopStance.Guarantee,
		}
		// 21 设计卷：谈判/妥协层只读视图。内容可读它塑形动作可得性
		// (信道畅通/妥协成立)，决策与叙事由服务层执行。
		result["negotiation"] = map[string]interface{}{
			"initialized":            ctx.Silo.Negotiation.Initialized,
			"channel_open":           ctx.Silo.Negotiation.ChannelOpen,
			"mayor_disposition":      ctx.Silo.Negotiation.MayorDisposition,
			"rounds":                 ctx.Silo.Negotiation.Rounds,
			"mayor_path":             ctx.Silo.Negotiation.MayorPath,
			"mayor_lean":             ctx.Silo.Negotiation.MayorLean,
			"rebel_posture":          ctx.Silo.Negotiation.RebelPosture,
			"offer_on_table":         ctx.Silo.Negotiation.OfferOnTable,
			"offered_by":             ctx.Silo.Negotiation.OfferedBy,
			"compromise_established": ctx.Silo.Negotiation.CompromiseEstablished,
			"takeover":               ctx.Silo.Negotiation.Takeover,
			"closed":                 ctx.Silo.Negotiation.Closed,
			"close_reason":           ctx.Silo.Negotiation.CloseReason,
		}
		// 16 Layer 1：准备账本只读视图。账本塑形出发结算与动作可得性；
		// 仲裁永不读它 (囤积准备不是胜利进度条)。
		result["escape_prep"] = map[string]interface{}{
			"initialized":       ctx.Silo.EscapePrep.Initialized,
			"suits_functional":  ctx.Silo.EscapePrep.SuitsFunctional,
			"supplies_packed":   ctx.Silo.EscapePrep.SuppliesPacked,
			"materials_packed":  ctx.Silo.EscapePrep.MaterialsPacked,
			"plan_completeness": ctx.Silo.EscapePrep.PlanCompleteness,
			"plan_roles":        append([]string(nil), ctx.Silo.EscapePrep.PlanRoles...),
			"drill_level":       ctx.Silo.EscapePrep.DrillLevel,
			"care_ready":        ctx.Silo.EscapePrep.CareReady,
			"production_start":  ctx.Silo.EscapePrep.ProductionStart,
			"reception_stage":   ctx.Silo.EscapePrep.ReceptionStage,
			"prep_exposure":     ctx.Silo.EscapePrep.PrepExposure,
		}
		spilloverChannels := make([]map[string]interface{}, 0, len(ctx.Silo.SpilloverPrep.Channels))
		for _, channel := range ctx.Silo.SpilloverPrep.Channels {
			spilloverChannels = append(spilloverChannels, map[string]interface{}{
				"kind":          channel.Kind,
				"target_silo":   channel.TargetSilo,
				"reliability":   channel.Reliability,
				"exposure_cost": channel.ExposureCost,
				"year":          channel.Year,
				"month":         channel.Month,
			})
		}
		result["spillover_prep"] = map[string]interface{}{
			"channels":       spilloverChannels,
			"package_quality": ctx.Silo.SpilloverPrep.PackageQuality,
		}

		resources := map[string]float64{}
		for _, r := range ctx.Silo.Resources {
			resources[r.Type] = r.Amount
		}
		result["resources"] = resources

		professions := make([]map[string]interface{}, 0, len(ctx.Silo.Professions))
		for _, profession := range ctx.Silo.Professions {
			ideologies := map[string]float64{}
			for ideology, value := range profession.Ideologies {
				ideologies[ideology] = value
			}
			// 部门信念层可读 (24-spec §3 的 department-belief readability)：
			// 只读快照，形状与 mechanic ctx 一致 (fact_id + b + 渗透状态)。
			// 终局铺垫一类 canTrigger 需要读"覆盖离门槛还差多少"；门槛改读
			// 成熟计数后 (18-time-permeation §3.1)，已成熟事实即使 B 淡忘到 0
			// 也必须继续可见。
			beliefs := make([]map[string]interface{}, 0, len(profession.Beliefs))
			for _, belief := range profession.Beliefs {
				if IsLeakFact(belief.FactID) {
					continue // 20 分册：反制观察不进内容层/UI 信念视图
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
				"name":         profession.Name,
				"class_type":   profession.ClassType,
				"panic_value":  profession.PanicValue,
				"productivity": profession.Productivity,
				"ideologies":   ideologies,
				"beliefs":      beliefs,
			})
		}
		result["professions"] = professions

		// P1-S3: read-only view of persistent factions for event scripts
		// (stable ID, stance, memory, demands). Scripts may condition on it;
		// mutation stays on the Go side.
		factions := make([]map[string]interface{}, 0, len(ctx.Silo.Factions))
		for _, faction := range ctx.Silo.Factions {
			factions = append(factions, map[string]interface{}{
				"id":           faction.ID,
				"name":         faction.Name,
				"stance":       faction.Stance,
				"member_count": faction.MemberCount,
				"influence":    faction.Influence,
				"cohesion":     faction.Cohesion,
				"is_public":    faction.IsPublic,
				"demands":      append([]string(nil), faction.Demands...),
				"memory":       append([]string(nil), faction.Memory...),
			})
		}
		result["factions"] = factions
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
	// G3: actor state (suspicion, prestige, propaganda) is readable by content.
	if ctx.Agent != nil {
		result["actor"] = map[string]interface{}{
			"suspicion_level":    ctx.Agent.SuspicionLevel,
			"political_prestige": ctx.Agent.PoliticalPrestige,
			"propaganda_level":   ctx.Agent.PropagandaLevel,
			"action_points":      ctx.Agent.ActionPoints,
			"known_fragments":    len(ctx.Agent.KnownFragments),
		}
		// 子项 3：玩家信念层可读 (发布门槛一类 canTrigger 用)，只读快照。
		beliefs := make([]map[string]interface{}, 0, len(ctx.Agent.Beliefs))
		for i := range ctx.Agent.Beliefs {
			beliefs = append(beliefs, map[string]interface{}{
				"fact_id":   ctx.Agent.Beliefs[i].FactID,
				"b":         ctx.Agent.Beliefs[i].B,
				"obs_count": ctx.Agent.Beliefs[i].ObsCount,
			})
		}
		result["actor"].(map[string]interface{})["beliefs"] = beliefs
	}
	if ctx.Silo != nil {
		// 已公开证据登记可读 (canTrigger 判断"是否已发布")。
		evidenceIDs := make([]string, 0, len(ctx.Silo.PublicEvidence))
		for _, record := range ctx.Silo.PublicEvidence {
			evidenceIDs = append(evidenceIDs, record.FactID)
		}
		result["public_evidence"] = evidenceIDs
		// 已核验名单可读 (发布前置)。
		result["verified_evidence"] = append([]string(nil), ctx.Silo.VerifiedEvidence...)
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
		payload.Effects[i].Scope = strings.ToLower(strings.TrimSpace(payload.Effects[i].Scope))
		payload.Effects[i].Metric = strings.ToLower(strings.TrimSpace(payload.Effects[i].Metric))
		payload.Effects[i].Flag = strings.ToLower(strings.TrimSpace(payload.Effects[i].Flag))
		payload.Effects[i].Profession = strings.TrimSpace(payload.Effects[i].Profession)
		payload.Effects[i].Fragment = strings.TrimSpace(payload.Effects[i].Fragment)
		payload.Effects[i].Zone = strings.TrimSpace(payload.Effects[i].Zone)
		payload.Effects[i].EventID = strings.TrimSpace(payload.Effects[i].EventID)
		payload.Effects[i].Action = strings.TrimSpace(payload.Effects[i].Action)
		payload.Effects[i].Observer = strings.TrimSpace(payload.Effects[i].Observer)
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
