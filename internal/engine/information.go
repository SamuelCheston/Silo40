package engine

import (
	"math"
	"sort"
	"strings"

	"silo40/internal/model"
)

// 检测分类 (10-it-control-credibility §8.2)。
const (
	// DetectUnsampled：身份×位置不在监控采样框 (矿工普通行为默认此类)。
	// 不进入检测流程：无采样 RNG、无普通怀疑增量；AP 仍独立计算。
	DetectUnsampled = "UNSAMPLED"
	// DetectSampled：监控范围内的普通身份与行为，走 §8.3 数值公式。
	DetectSampled = "SAMPLED"
	// DetectDirectProtocol：行为触及 Protocol 条款；不生成普通怀疑惩罚，
	// 直接进入偏离后果 (子项 4 接入条款判定)。
	DetectDirectProtocol = "DIRECT_PROTOCOL"
)

// InformationConfig 集中碎片获取、信念吸收、行动资格与检测参数
// (10-it-control-credibility §3/§8/§9)。除注明外均为 sim 起点。
type InformationConfig struct {
	Version int

	// §3.4 单次获取成功率 p_get。
	PGetBase        float64
	PGetMin         float64
	PGetMax         float64
	MismatchPenalty float64 // 1-0.45·mismatch
	AlertPenalty    float64 // 1-0.60·alert_d
	RelatedBonus    float64 // 1+0.35·related_f
	FProfMatch      float64 // 本职业域材料
	FProfMismatch   float64 // 跨域材料 (不提供全知)

	// §3.4 渠道表：access_c / reliability_c。
	ChannelInternalAccess  float64 // 本职业内部记录
	ChannelInternalRel     float64
	ChannelArchiveAccess   float64 // 原始档案/实物
	ChannelArchiveRel      float64
	ChannelInformantAccess float64 // 可信线人
	ChannelInformantRel    float64
	ChannelTestimonyAccess float64 // 签署证词
	ChannelTestimonyRel    float64
	ChannelRumorAccess     float64 // 市面传闻
	ChannelRumorRel        float64

	// §3.5 多次吸收。
	AbsorbBase      float64 // s_i = rel·(0.85+0.15·related_f)·independence
	AbsorbRelated   float64
	DiminishPerN    float64 // B 更新里的 1/(1+0.25·n)
	IndependenceNew float64 // 独立来源
	IndependenceRe  float64 // 同源转述

	// §3.6 确信度行动资格门槛。
	QualRumor       float64 // ≥0.30 可作为(低置信)材料分享
	QualInvestigate float64 // ≥0.60 定向调查/私下对质
	QualReport      float64 // ≥0.85 正式举报/申请归档
	QualPublish     float64 // ≥0.90 核验后公开

	// §8.1 AP 与怀疑独立：AP_cost = ceil(base·(1+0.5·mismatch))。
	APMismatchSlope float64

	// §8.3 SAMPLED 因子。
	WitnessPrivate     float64 // 私下
	WitnessCircle      float64 // 小圈子
	WitnessDepartment  float64 // 部门
	WitnessPublic      float64 // 公开
	RecordNone         float64
	RecordAnon         float64
	RecordPersonal     float64
	RecordSurveillance float64

	// 探查动作的 sim 起点 (TODO(balance))。
	GatherDeviation    float64 // 跨部门探查的行为偏离强度 a
	ProbeBaseSuspicion float64 // 探查的 base_action
	ProbeFitCross      float64 // 跨部门探查的 F_fit
	ProbeFitOwn        float64 // 本部门内查的 F_fit
	AlertOnFailedProbe float64 // 失败探查抬高的目标部门戒备
	ProbeWitness       float64 // 探查的目击档 (私下)
	ProbeRecord        float64 // 探查的记录档 (匿名日志)
	ProbeVisibility    float64 // 探查者的 V_identity
	LegacySeedB        float64 // 旧档已知碎片的保守补种确信
}

// DefaultInformationConfig 的参数除结构常量外全部是 sim 起点；调整须按
// §7.4 的 TODO(balance) 目标走 sim 实测。
func DefaultInformationConfig() InformationConfig {
	return InformationConfig{
		Version: 1,

		PGetBase:        0.55,
		PGetMin:         0.05,
		PGetMax:         0.95,
		MismatchPenalty: 0.45,
		AlertPenalty:    0.60,
		RelatedBonus:    0.35,
		// TODO(balance): 跨域适配下限取设计区间最低档，待 sim 校准。
		FProfMatch:    1.25,
		FProfMismatch: 0.75,

		ChannelInternalAccess:  1.15,
		ChannelInternalRel:     0.90,
		ChannelArchiveAccess:   0.85,
		ChannelArchiveRel:      0.95,
		ChannelInformantAccess: 1.00,
		ChannelInformantRel:    0.70,
		ChannelTestimonyAccess: 0.90,
		ChannelTestimonyRel:    0.75,
		ChannelRumorAccess:     0.60,
		ChannelRumorRel:        0.35,

		AbsorbBase:      0.85,
		AbsorbRelated:   0.15,
		DiminishPerN:    0.25,
		IndependenceNew: 1.0,
		IndependenceRe:  0.25,

		QualRumor:       0.30,
		QualInvestigate: 0.60,
		QualReport:      0.85,
		QualPublish:     0.90,

		APMismatchSlope: 0.5,

		WitnessPrivate:     0.35,
		WitnessCircle:      1.0,
		WitnessDepartment:  1.6,
		WitnessPublic:      2.2,
		RecordNone:         0.7,
		RecordAnon:         1.0,
		RecordPersonal:     1.35,
		RecordSurveillance: 1.6,

		// TODO(balance): 探查档参数为 sim 起点，量级对齐旧 0.005×职业系数。
		GatherDeviation:    0.4,
		ProbeBaseSuspicion: 0.01,
		ProbeFitCross:      1.2,
		ProbeFitOwn:        0.6,
		AlertOnFailedProbe: 0.10,
		ProbeWitness:       0.35,
		ProbeRecord:        1.0,
		ProbeVisibility:    1.0,
		LegacySeedB:        0.65,
	}
}

// 版本 ID：解释分支的稳定键 (子项 3 的 E 归因读 it_directed 的 attribution)。
const (
	VersionITDirected = "it_directed"
	VersionMundane    = "mundane"
)

// FragmentVersion 一个碎片的一种解释版本。
type FragmentVersion struct {
	ID          string
	Prior       float64 // 初始 π
	Likelihood  float64 // 观察到该碎片时本版本的似然
	Attribution float64 // a_e,v：IT 归因强度 (子项 3 的 E 聚合用)
}

// FragmentSpec 碎片的静态目录项。
type FragmentSpec struct {
	ID       string
	Domain   string
	Versions []FragmentVersion // 按 ID 升序
}

// fragmentCatalog 由 ALL_FRAGMENTS 确定性派生：IT/Judicial 域碎片带
// it_directed/mundane 两版本 (归因可辨)，其余域只有 mundane 单版本
// (10-it-control-credibility §3.3：普通事实不自动形成 IT 归因)。
// TODO(balance): 先验与似然为 sim 起点。
var fragmentCatalog = buildFragmentCatalog()

func buildFragmentCatalog() map[string]FragmentSpec {
	specs := map[string]FragmentSpec{}
	for _, fragment := range model.ALL_FRAGMENTS {
		domain := fragment
		if idx := strings.Index(fragment, "_"); idx > 0 {
			domain = fragment[:idx]
		}
		var versions []FragmentVersion
		switch domain {
		case "IT", "Judicial":
			versions = []FragmentVersion{
				{ID: VersionITDirected, Prior: 0.5, Likelihood: 0.75, Attribution: 1},
				{ID: VersionMundane, Prior: 0.5, Likelihood: 0.25, Attribution: 0},
			}
		default:
			versions = []FragmentVersion{
				{ID: VersionMundane, Prior: 1.0, Likelihood: 1.0, Attribution: 0},
			}
		}
		sort.Slice(versions, func(i, j int) bool { return versions[i].ID < versions[j].ID })
		specs[fragment] = FragmentSpec{ID: fragment, Domain: domain, Versions: versions}
	}
	return specs
}

// FragmentSpecFor 返回碎片的目录项；未知碎片给单版本 mundane 兜底。
func FragmentSpecFor(fragmentID string) FragmentSpec {
	if spec, ok := fragmentCatalog[fragmentID]; ok {
		return spec
	}
	domain := fragmentID
	if idx := strings.Index(fragmentID, "_"); idx > 0 {
		domain = fragmentID[:idx]
	}
	return FragmentSpec{
		ID:     fragmentID,
		Domain: domain,
		Versions: []FragmentVersion{
			{ID: VersionMundane, Prior: 1.0, Likelihood: 1.0, Attribution: 0},
		},
	}
}

// InitialVersions 目录先验 π 的副本。
func InitialVersions(spec FragmentSpec) map[string]float64 {
	out := make(map[string]float64, len(spec.Versions))
	for _, version := range spec.Versions {
		out[version.ID] = version.Prior
	}
	return out
}

// PGet §3.4 单次获取成功率。
func PGet(cfg InformationConfig, fProf, mismatch, alert, relatedF, accessC float64) float64 {
	p := cfg.PGetBase * fProf *
		(1 - cfg.MismatchPenalty*clamp01(mismatch)) *
		(1 - cfg.AlertPenalty*clamp01(alert)) *
		(1 + cfg.RelatedBonus*clamp01(relatedF)) *
		accessC
	return clamp(p, cfg.PGetMin, cfg.PGetMax)
}

// AbsorptionStrength §3.5 单次观察强度 s_i (未含 1/(1+0.25n) 衰减)。
func AbsorptionStrength(cfg InformationConfig, reliability, relatedF, independence float64) float64 {
	return clamp01(reliability) * (cfg.AbsorbBase + cfg.AbsorbRelated*clamp01(relatedF)) * independence
}

// ObserveFact 把一次观察吸收进信念：observation_id 去重、B 更新、π 贝叶斯
// 更新。返回是否实际吸收 (重复观察返回 false，不重放收益)。
func ObserveFact(cfg InformationConfig, belief *model.FactBelief, spec FragmentSpec, observationID string, s float64) bool {
	if belief == nil || observationID == "" {
		return false
	}
	for _, existing := range belief.Observations {
		if existing == observationID {
			return false
		}
	}
	if belief.Versions == nil || len(belief.Versions) == 0 {
		belief.Versions = InitialVersions(spec)
	}
	s = clamp01(s)
	denom := 1 + cfg.DiminishPerN*float64(belief.ObsCount)
	belief.B = clamp01(1 - (1-belief.B)*(1-s/denom))

	// π 贝叶斯更新 (按目录版本序确定性累加)。
	weighted := make([]float64, len(spec.Versions))
	total := 0.0
	for i, version := range spec.Versions {
		prior := belief.Versions[version.ID]
		if prior <= 0 {
			prior = version.Prior
		}
		weighted[i] = prior * version.Likelihood
		total += weighted[i]
	}
	if total > 0 {
		for i, version := range spec.Versions {
			belief.Versions[version.ID] = weighted[i] / total
		}
	}

	belief.Observations = append(belief.Observations, observationID)
	belief.ObsCount++
	return true
}

// BeliefFor 按事实 ID 查信念；不存在返回 nil。
func BeliefFor(agent *model.Agent, factID string) *model.FactBelief {
	if agent == nil {
		return nil
	}
	for i := range agent.Beliefs {
		if agent.Beliefs[i].FactID == factID {
			return &agent.Beliefs[i]
		}
	}
	return nil
}

// EnsureBelief 查或建信念条目 (新建时按目录先验初始化 π)。
func EnsureBelief(agent *model.Agent, factID string) *model.FactBelief {
	if agent == nil || factID == "" {
		return nil
	}
	if belief := BeliefFor(agent, factID); belief != nil {
		return belief
	}
	agent.Beliefs = append(agent.Beliefs, model.FactBelief{
		FactID:   factID,
		Versions: InitialVersions(FragmentSpecFor(factID)),
	})
	return &agent.Beliefs[len(agent.Beliefs)-1]
}

// SeedLegacyBeliefs 旧档兼容：KnownFragments 里没有信念条目的碎片按
// "一次有效观察" 保守补种 (B=0.65，π=先验)。不升存档版本。
func SeedLegacyBeliefs(agent *model.Agent) {
	if agent == nil {
		return
	}
	cfg := DefaultInformationConfig()
	for _, fragment := range agent.KnownFragments {
		if BeliefFor(agent, fragment) != nil {
			continue
		}
		belief := EnsureBelief(agent, fragment)
		belief.B = cfg.LegacySeedB
		belief.Observations = []string{"legacy:" + fragment}
		belief.ObsCount = 1
	}
}

// RelatedF §3.4：max_k related(f,k)·B_k,p，只取最强相关项。相关度取同域 1、
// 跨域 0 (TODO(balance): 相关图之后可细化)。
func RelatedF(agent *model.Agent, fragmentID string) float64 {
	if agent == nil {
		return 0
	}
	domain := FragmentSpecFor(fragmentID).Domain
	best := 0.0
	for i := range agent.Beliefs {
		belief := &agent.Beliefs[i]
		if belief.FactID == fragmentID || FragmentSpecFor(belief.FactID).Domain != domain {
			continue
		}
		if belief.B > best {
			best = belief.B
		}
	}
	return best
}

// FProf 职业与材料域适配。
func FProf(cfg InformationConfig, actorProfession, fragmentID string) float64 {
	if FragmentSpecFor(fragmentID).Domain == actorProfession {
		return cfg.FProfMatch
	}
	return cfg.FProfMismatch
}

// Mismatch §8.1：|a|·max(0,-sign(a)·d)。a 为行为偏离方向/强度，d 为部门
// 规范倾向 (正=开放，负=排外)。
func Mismatch(actionDeviation, deptNorm float64) float64 {
	a := clamp(actionDeviation, -1, 1)
	d := clamp(deptNorm, -1, 1)
	if a == 0 {
		return 0
	}
	sign := 1.0
	if a < 0 {
		sign = -1.0
	}
	return math.Abs(a) * math.Max(0, -sign*d)
}

// DeptNorm 部门规范倾向：用亲外思潮映射到 [-1,1] (TODO(balance))。
func DeptNorm(prof *model.Profession) float64 {
	if prof == nil {
		return 0
	}
	return clamp(2*clamp01(prof.Ideologies[model.IdeologyProForeign])-1, -1, 1)
}

// APCostWithMismatch §8.1：ceil(base·(1+0.5·mismatch))。AP 与怀疑独立结算。
func APCostWithMismatch(cfg InformationConfig, base, mismatch float64) float64 {
	return math.Ceil(base * (1 + cfg.APMismatchSlope*clamp01(mismatch)))
}

// SampledSuspicion §8.3：base_action·V_identity·F_fit·W_witness·R_record·(1+mismatch)。
func SampledSuspicion(cfg InformationConfig, base, vIdentity, fFit, wWitness, rRecord, mismatch float64) float64 {
	return base * vIdentity * fFit * wWitness * rRecord * (1 + clamp01(mismatch))
}

// ClassifyDetection §8.2 三类检测。protocolTouch 在 Protocol 条款落地 (子项 4)
// 前恒为 false；Miners 结构性不采样。
func ClassifyDetection(actorProfession, targetDept string, protocolTouch bool) string {
	if protocolTouch {
		return DetectDirectProtocol
	}
	if actorProfession == "Mines" || targetDept == "Mines" {
		return DetectUnsampled
	}
	return DetectSampled
}

// QualificationTier §3.6 确信度行动资格 (返回可支撑的最高档)。
func QualificationTier(cfg InformationConfig, b float64) string {
	switch {
	case b >= cfg.QualPublish:
		return "publish"
	case b >= cfg.QualReport:
		return "report"
	case b >= cfg.QualInvestigate:
		return "investigate"
	case b >= cfg.QualRumor:
		return "rumor"
	default:
		return "private"
	}
}
