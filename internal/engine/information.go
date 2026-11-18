package engine

import (
	"math"
	"sort"
	"strconv"
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
	AbsorbBase    float64 // s_i = rel·(0.85+0.15·related_f)·independence
	AbsorbRelated float64
	DiminishPerN  float64 // B 更新里的 1/(1+0.25·n)
	// 递减分母计入的观察数上限 (TODO(balance)，见 18-time-permeation)：ObsCount 超过该值
	// 时按该值算分母。恒定淡忘与无上限的 1/(1+0.25n) 数学上不相容——任何正淡
	// 忘速率下都存在足够大的 n 使转述增量 < 淡忘量，流通结构性跑不赢淡忘；
	// 封顶让维持成本有界。仅改 n>cap 的尾部，早期递减 (n≤cap) 逐位不变。
	DiminishObsCap  int
	IndependenceNew float64 // 独立来源
	IndependenceRe  float64 // 同源转述

	// §3.6 确信度行动资格门槛。
	QualRumor       float64 // ≥0.30 可作为(低置信)材料分享
	QualInvestigate float64 // ≥0.60 定向调查/私下对质
	QualReport      float64 // ≥0.85 正式举报/申请归档
	QualPublish     float64 // ≥0.90 核验后公开

	// 恒定淡忘系数 (25-information-economy §修复A)：部门信念层每个事实的 B
	// 每月减去该系数 (下限 0)。玩家的流通 (Share 吸收) 抵住淡忘——情报不会
	// 自己站住，链必须被持续推动。仅部门层衰减：特工层不衰减 (见证者记得)，
	// 具名人物层暂不衰减 (公开传播的再评估留给后续批次)。挂钩点在
	// runMonthlyBeliefDispatch (真实月度路径)。
	// TODO(balance): sim 起点，见 17/18 号设计文档与本配置处的推导。
	ForgetRateMonthly float64

	// 时间渗透月数 (18-time-permeation §4)：(部门, 事实) 的 B 连续不低于
	// 分享地板 (QualRumor) 满该月数后，该事实对该部门成熟，成为结局资格门
	// 读取的"可用证据"。成熟粘性不回退；渗透只读 B 与日历，不改 B/π、不耗
	// RNG。仅部门层渗透：特工层保住开局第一次分享的手感，具名层同淡忘口径
	// 暂不渗透 (§3.2)。
	// TODO(balance): 推荐起点 30，区间 [24, 36]；判据为重调后保守结算 ≤ 94
	// 的最大值 (18 号设计 §4 夹逼与 §7 公平重调)。
	PermeationMonths int

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

	// §3.3 E 聚合的组权重分配 (TODO(balance))：部门层按人口份额分得
	// DepartmentWeightMass，具名人物层按影响力份额分得 ResidentWeightMass。
	DepartmentWeightMass float64
	ResidentWeightMass   float64

	// §3.7 Q 与独立公开事实资格门槛。
	PactFactBelief     float64 // B_Judicial ≥ 0.85
	PactFactProvenance float64 // provenance ≥ 0.8
	PactFactCoverage   float64 // coverage ≥ 0.60

	// 事实证据权重 (TODO(balance)：sim 起点；§4 核心事实 qE=0.20–0.30，
	// §3.7 核心法警调度事实 qP=0.60)。
	QECore    float64
	QESupport float64
	QPCore    float64
	QPSupport float64
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

		AbsorbBase:    0.85,
		AbsorbRelated: 0.15,
		DiminishPerN:  0.25,
		// TODO(balance): 递减分母封顶 4 (分母上限 2.0)。不封顶时任何正淡忘速率下
		// 转述增量都渐近归零，流通结构性跑不赢淡忘；cap=4 保留 n≤4 的早期递减
		// 手感，只把尾部维持成本压到有界。
		DiminishObsCap:  4,
		IndependenceNew: 1.0,
		// TODO(balance): 同源转述折减 0.25 → 0.5。淡忘启用后转述是维持信念的
		// 唯一常态手段；0.5 与 cap=4 使典型分享强度下持续流通能托住《公议》
		// 的实时 0.85 证据水位，同时每次吸收仍有递减收益。
		IndependenceRe: 0.5,

		QualRumor:       0.30,
		QualInvestigate: 0.60,
		QualReport:      0.85,
		QualPublish:     0.90,

		// TODO(balance): 淡忘速率 0.005/月。INFORMATION 门需 27 槽 (9 部门×3
		// 事实) 各连续 30 个月过地板，深填目标为 B₀ = 0.30 + 30r = 0.45；相比
		// r=0.01 所需的 B₀=0.60，深填与补流通成本回到生涯 AP 预算内。0.005 下
		// 深填事实约 30 个月滑回地板，《公议》的实时高水位也必须靠再流通维持。
		ForgetRateMonthly: 0.005,

		// TODO(balance): 渗透月数 sim 起点 (18-time-permeation §4)，推荐起点 30、
		// 区间 [24, 36]，判据为"重调后保守结算 ≤ 94 的最大值"。恒定淡忘在真实
		// 月度路径先生效，B 过地板后仍会回落；只有当月衰减后仍被流通托在地板
		// 以上的事实才累计渗透计数。渗透窗口压力上升，资格深度与 30 月窗不变。
		PermeationMonths: 30,

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

		DepartmentWeightMass: 0.75,
		ResidentWeightMass:   0.25,

		PactFactBelief:     0.85,
		PactFactProvenance: 0.8,
		PactFactCoverage:   0.60,

		QECore:    0.30,
		QESupport: 0.10,
		QPCore:    0.60,
		QPSupport: 0.20,
	}
}

// 公开证据来源等级 (§3.7)。
const (
	ProvenancePrivate   = 0.0 // 私人猜疑
	ProvenanceRumor     = 0.3 // 匿名传闻
	ProvenanceTestimony = 0.6 // 公开证词/未核验记录
	ProvenanceVerified  = 1.0 // 已核验档案或市面可查原始记录
)

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
	QE       float64           // 曝光聚合权重 (§3.3/§4)
	QP       float64           // 公开证据聚合权重 (§3.7)
}

// fragmentCatalog 由 ALL_FRAGMENTS 确定性派生：IT/Judicial 域碎片带
// it_directed/mundane 两版本 (归因可辨)，其余域只有 mundane 单版本
// (10-it-control-credibility §3.3：普通事实不自动形成 IT 归因)。
// TODO(balance): 先验与似然为 sim 起点。
var fragmentCatalog = buildFragmentCatalog()

func buildFragmentCatalog() map[string]FragmentSpec {
	cfg := DefaultInformationConfig()
	specs := map[string]FragmentSpec{}
	for _, fragment := range model.ALL_FRAGMENTS {
		domain := fragment
		if idx := strings.Index(fragment, "_"); idx > 0 {
			domain = fragment[:idx]
		}
		var versions []FragmentVersion
		qe, qp := 0.0, 0.0
		switch domain {
		case "IT", "Judicial":
			versions = []FragmentVersion{
				{ID: VersionITDirected, Prior: 0.5, Likelihood: 0.75, Attribution: 1},
				{ID: VersionMundane, Prior: 0.5, Likelihood: 0.25, Attribution: 0},
			}
			qe, qp = cfg.QESupport, cfg.QPSupport
			// 两份核心"法警受 IT 直接调度"证据事实 (§4/§7.4)。
			// TODO(content): 子项 6 的完整证据链内容落地后以专门事实 ID 替换。
			if fragment == "Judicial_1" || fragment == "Judicial_2" {
				qe, qp = cfg.QECore, cfg.QPCore
			}
		default:
			versions = []FragmentVersion{
				{ID: VersionMundane, Prior: 1.0, Likelihood: 1.0, Attribution: 0},
			}
		}
		sort.Slice(versions, func(i, j int) bool { return versions[i].ID < versions[j].ID })
		specs[fragment] = FragmentSpec{ID: fragment, Domain: domain, Versions: versions, QE: qe, QP: qp}
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
	obsN := belief.ObsCount
	if cfg.DiminishObsCap > 0 && obsN > cfg.DiminishObsCap {
		obsN = cfg.DiminishObsCap
	}
	denom := 1 + cfg.DiminishPerN*float64(obsN)
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

// BeliefIn 在信念列表里按事实 ID 查找；不存在返回 nil。
func BeliefIn(beliefs *[]model.FactBelief, factID string) *model.FactBelief {
	if beliefs == nil {
		return nil
	}
	for i := range *beliefs {
		if (*beliefs)[i].FactID == factID {
			return &(*beliefs)[i]
		}
	}
	return nil
}

// EnsureBeliefIn 查或建信念条目 (新建时按目录先验初始化 π)。
func EnsureBeliefIn(beliefs *[]model.FactBelief, factID string) *model.FactBelief {
	if beliefs == nil || factID == "" {
		return nil
	}
	if belief := BeliefIn(beliefs, factID); belief != nil {
		return belief
	}
	*beliefs = append(*beliefs, model.FactBelief{
		FactID:   factID,
		Versions: InitialVersions(FragmentSpecFor(factID)),
	})
	return &(*beliefs)[len(*beliefs)-1]
}

// BeliefFor 按事实 ID 查玩家信念；不存在返回 nil。
func BeliefFor(agent *model.Agent, factID string) *model.FactBelief {
	if agent == nil {
		return nil
	}
	return BeliefIn(&agent.Beliefs, factID)
}

// EnsureBelief 查或建玩家信念条目。
func EnsureBelief(agent *model.Agent, factID string) *model.FactBelief {
	if agent == nil {
		return nil
	}
	return EnsureBeliefIn(&agent.Beliefs, factID)
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

// RelatedFIn §3.4：max_k related(f,k)·B_k，只取最强相关项。相关度取同域 1、
// 跨域 0 (TODO(balance): 相关图之后可细化)。
func RelatedFIn(beliefs *[]model.FactBelief, fragmentID string) float64 {
	if beliefs == nil {
		return 0
	}
	domain := FragmentSpecFor(fragmentID).Domain
	best := 0.0
	for i := range *beliefs {
		belief := &(*beliefs)[i]
		if belief.FactID == fragmentID || FragmentSpecFor(belief.FactID).Domain != domain {
			continue
		}
		if belief.B > best {
			best = belief.B
		}
	}
	return best
}

// RelatedF 玩家信念层的相关事实强度。
func RelatedF(agent *model.Agent, fragmentID string) float64 {
	if agent == nil {
		return 0
	}
	return RelatedFIn(&agent.Beliefs, fragmentID)
}

// AbsorbSharedFact §3.6 分享接收：分享强度作为观察强度进入接收方信念。
// 同一来源首次分享按独立来源计，其后同源转述按 IndependenceRe 折减；观察
// ID 按计数编号，重复分享仍有衰减收益而非完全无效。递减分母按
// DiminishObsCap 封顶——转述是淡忘启用后维持信念的常态手段，尾部递减必须
// 有界 (见 18-time-permeation)。
func AbsorbSharedFact(cfg InformationConfig, beliefs *[]model.FactBelief, factID, sourceTag string, strength float64) bool {
	belief := EnsureBeliefIn(beliefs, factID)
	if belief == nil {
		return false
	}
	prefix := "share:" + sourceTag + ":" + factID + ":"
	k := 0
	for _, obs := range belief.Observations {
		if strings.HasPrefix(obs, prefix) {
			k++
		}
	}
	independence := cfg.IndependenceNew
	if k > 0 {
		independence = cfg.IndependenceRe
	}
	obsID := prefix + strconv.Itoa(k+1)
	spec := FragmentSpecFor(factID)
	return ObserveFact(cfg, belief, spec, obsID, clamp01(strength)*independence)
}

// ApplyBeliefForgetting 部门信念层恒定淡忘 (25-information-economy §修复A)。
// 确定性：每月每事实 B 减去 ForgetRateMonthly (下限 0)，无随机源；部门切片
// 与信念切片均按既有持久化顺序遍历。仅部门层衰减：特工层不衰减 (见证者
// 记得)，具名人物层暂不衰减。零系数或空堡为无操作。
func ApplyBeliefForgetting(silo *model.Silo, cfg InformationConfig) {
	if silo == nil || cfg.ForgetRateMonthly <= 0 {
		return
	}
	for i := range silo.Professions {
		for j := range silo.Professions[i].Beliefs {
			belief := &silo.Professions[i].Beliefs[j]
			belief.B -= cfg.ForgetRateMonthly
			if belief.B < 0 {
				belief.B = 0
			}
		}
	}
}

// LeakFactPrefix 20 分册 §3.2：反制方对"有人在传某事实"的观察事实前缀。
// 泄漏事实只在反制方 (IT/Judicial) 的信念层累积、会淡忘，但永不成熟，也不
// 计入《共证》资格门的成熟事实数。
const LeakFactPrefix = "leak:"

// IsLeakFact 判断一条部门信念是否为反制用的泄漏观察。
func IsLeakFact(factID string) bool {
	return strings.HasPrefix(factID, LeakFactPrefix)
}

// FactionFactPrefix 24 分册 §1.2 (C8)：派系感知事实前缀 faction:<派系ID>:founded。
// 感知事实是政治情报而非证据：只淡忘、永不成熟，也不计入《共证》资格门的
// 成熟事实数 (与 leak: 同一豁免)。B 即感知度/可靠度，C9 加入掷骰读取。
const FactionFactPrefix = "faction:"

// IsFactionFact 判断一条信念是否为派系感知事实。
func IsFactionFact(factID string) bool {
	return strings.HasPrefix(factID, FactionFactPrefix)
}

// ApplyPermeation 部门信念层时间渗透 (18-time-permeation §3)。事实变成"可用
// 证据"需要日历时间：当月淡忘结算后 B 仍不低于分享地板 (QualRumor) 则计数
// +1，否则计数清零 (清零而非暂停——"断了就要重新渗透")；计数达到
// PermeationMonths 当月置 Matured = true。成熟是粘性的：一旦成熟不回退，
// 即使 B 此后跌破地板、淡忘到 0。边界说明：地板比较发生在淡忘之后，故 B
// 恰好为 QualRumor 的事实当月先衰减到地板下、不计入渗透数。
//
// 只读派生状态：只读 B 与日历，不改写 B/π，不引入随机源；部门切片与信念
// 切片均按既有持久化顺序遍历 (确定性)。挂钩点与 ApplyBeliefForgetting 同一
// 月度 dispatch (EVENT_METRICS_UPDATE)，顺序固定为先淡忘、再计数、再判成熟。
// 仅部门层渗透：特工层与具名人物层不写入 (§3.2)。PermeationMonths <= 0 或
// 空堡为无操作。泄漏观察 (leak:) 与派系感知事实 (faction:) 只淡忘、不计入渗透。
func ApplyPermeation(silo *model.Silo, cfg InformationConfig) {
	if silo == nil || cfg.PermeationMonths <= 0 {
		return
	}
	for i := range silo.Professions {
		for j := range silo.Professions[i].Beliefs {
			belief := &silo.Professions[i].Beliefs[j]
			if belief.Matured || IsLeakFact(belief.FactID) || IsFactionFact(belief.FactID) {
				continue
			}
			if belief.B >= cfg.QualRumor {
				belief.PermeationCounter++
			} else {
				belief.PermeationCounter = 0
			}
			if belief.PermeationCounter >= cfg.PermeationMonths {
				belief.Matured = true
			}
		}
	}
}

// MaturedFactCount 一个部门已完成渗透 (成熟) 的事实数——《共证》资格门
// 改读的量 (18-time-permeation §3.1)。
func MaturedFactCount(prof *model.Profession) int {
	if prof == nil {
		return 0
	}
	count := 0
	for _, belief := range prof.Beliefs {
		if belief.Matured && !IsLeakFact(belief.FactID) && !IsFactionFact(belief.FactID) {
			count++
		}
	}
	return count
}

// PermeationMonthsLeft 一条部门信念距成熟还差几个月 (18-time-permeation §7.2
// 反馈面)：已成熟返回 0；未成熟返回 PermeationMonths - 当前计数 (下限 1)。
// 计数为 0 (尚未过地板或刚被清零) 时即完整一个渗透期。
func PermeationMonthsLeft(belief model.FactBelief, cfg InformationConfig) int {
	if belief.Matured || cfg.PermeationMonths <= 0 {
		return 0
	}
	left := cfg.PermeationMonths - belief.PermeationCounter
	if left < 1 {
		left = 1
	}
	return left
}

// RegisterPublicEvidence §3.7：登记/更新一份公开证据。同一事实只登记一次，
// 再次公开只能抬高 provenance/coverage (确定性取大)。
func RegisterPublicEvidence(silo *model.Silo, record model.PublicEvidenceRecord) {
	if silo == nil || strings.TrimSpace(record.FactID) == "" {
		return
	}
	record.FactID = strings.TrimSpace(record.FactID)
	record.Provenance = clamp01(record.Provenance)
	record.Coverage = clamp01(record.Coverage)
	for i := range silo.PublicEvidence {
		if silo.PublicEvidence[i].FactID == record.FactID {
			if record.Provenance > silo.PublicEvidence[i].Provenance {
				silo.PublicEvidence[i].Provenance = record.Provenance
			}
			if record.Coverage > silo.PublicEvidence[i].Coverage {
				silo.PublicEvidence[i].Coverage = record.Coverage
			}
			return
		}
	}
	silo.PublicEvidence = append(silo.PublicEvidence, record)
}

// InformationEnvironment 一次月度结算里的信息环境快照。
type InformationEnvironment struct {
	ExposureNow       float64  // E_now (§3.3 聚合，未取高水位)
	Exposure          float64  // max(上月 E, E_now)：曝光高水位不自动消退
	PublicEvidenceQ   float64  // Q (§3.7)
	QualifyingFacts   int      // 满足 covenant 独立事实门槛的数量
	QualifyingFactIDs []string // 满足门槛的事实 ID (升序)
}

// ComputeInformationEnvironment §3.3/§3.7：E 只在部门层与具名人物层聚合
// (玩家自己的 B 不直接进入)；Q 只读 Judicial 层信念与已登记公开证据。
// 部门按人口份额、具名人物按影响力份额分组权重 (TODO(balance))；全部按
// 稳定顺序遍历，确定性。
func ComputeInformationEnvironment(silo *model.Silo) InformationEnvironment {
	cfg := DefaultInformationConfig()
	env := InformationEnvironment{}
	if silo == nil {
		return env
	}

	type beliefGroup struct {
		weight  float64
		beliefs *[]model.FactBelief
	}
	var groups []beliefGroup

	totalPopulation := 0
	for i := range silo.Professions {
		totalPopulation += silo.Professions[i].Population
	}
	if totalPopulation > 0 {
		for i := range silo.Professions {
			prof := &silo.Professions[i]
			weight := cfg.DepartmentWeightMass * float64(prof.Population) / float64(totalPopulation)
			if weight > 0 {
				groups = append(groups, beliefGroup{weight: weight, beliefs: &prof.Beliefs})
			}
		}
	}

	totalInfluence := 0.0
	for i := range silo.Residents {
		if silo.Residents[i].Alive && silo.Residents[i].Influence > 0 {
			totalInfluence += silo.Residents[i].Influence
		}
	}
	if totalInfluence > 0 {
		for i := range silo.Residents {
			resident := &silo.Residents[i]
			if !resident.Alive || resident.Influence <= 0 {
				continue
			}
			weight := cfg.ResidentWeightMass * resident.Influence / totalInfluence
			groups = append(groups, beliefGroup{weight: weight, beliefs: &resident.Beliefs})
		}
	}

	// E_now：按 ALL_FRAGMENTS 的目录序逐事实聚合。
	eNow := 0.0
	for _, factID := range model.ALL_FRAGMENTS {
		spec := FragmentSpecFor(factID)
		if spec.QE <= 0 {
			continue
		}
		groupSum := 0.0
		for _, group := range groups {
			belief := BeliefIn(group.beliefs, factID)
			if belief == nil {
				continue
			}
			attribution := 0.0
			for _, version := range spec.Versions {
				attribution += belief.Versions[version.ID] * version.Attribution
			}
			groupSum += group.weight * belief.B * attribution
		}
		eNow += spec.QE * groupSum
	}
	env.ExposureNow = clamp01(eNow)
	env.Exposure = math.Max(clamp01(silo.Authority.E), env.ExposureNow)

	// Q：只读 Judicial 层信念 × 已登记公开证据。
	var judicialBeliefs *[]model.FactBelief
	for i := range silo.Professions {
		if silo.Professions[i].Name == "Judicial" {
			judicialBeliefs = &silo.Professions[i].Beliefs
			break
		}
	}
	q := 0.0
	for _, record := range silo.PublicEvidence {
		spec := FragmentSpecFor(record.FactID)
		if spec.QP <= 0 {
			continue
		}
		bJudicial, piDirected := 0.0, 0.0
		if belief := BeliefIn(judicialBeliefs, record.FactID); belief != nil {
			bJudicial = belief.B
			piDirected = belief.Versions[VersionITDirected]
		}
		q += spec.QP * bJudicial * piDirected * clamp01(record.Provenance) * clamp01(record.Coverage)
		if bJudicial >= cfg.PactFactBelief && record.Provenance >= cfg.PactFactProvenance && record.Coverage >= cfg.PactFactCoverage {
			env.QualifyingFacts++
			env.QualifyingFactIDs = append(env.QualifyingFactIDs, record.FactID)
		}
	}
	env.PublicEvidenceQ = clamp01(q)
	sort.Strings(env.QualifyingFactIDs)
	return env
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
