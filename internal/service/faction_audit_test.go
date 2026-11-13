package service

// P1-S1 派系缺陷审计 harness (行为中性：只加测试，不改生产代码)。
//
// 量化签名分桶聚类的三类语义缺陷，固化"改前"基线，供 S2 距离聚类替换后
// 对照 (作者验收口径：双胞胎派系≈0、签名坍缩不再出现、A→B→A 振荡归零)：
//
//  1. 双胞胎派系 (twin)：两个政治派系的签名仅 pro_foreign 出生档位不同
//     (其余标签全同)，且成员人数加权的思潮质心距离已收敛 (<0.20)——
//     同思潮、仅出生档位不同的并立派系。
//  2. 签名坍缩 (collapse)：签名空间过粗导致意识形态相距甚远的 cohort 被
//     并进同一派系。度量：同派系内 cohort 两两质心思潮距离 >0.35 的对数
//     (overmerged)、跨 ≥5 部门的政治派系出现月数 (multi-prof)、最大政治
//     派系人口占比峰值。
//  3. 归属振荡 (oscillation)：cohort 的派系签名逐 tick 变化次数 (flips)
//     与 A→B→A 三步往返次数 (aba)。
//
// 双跑一致性 (硬约束 B) 内建于本测试：每个种子跑两遍，审计摘要必须逐字节
// 一致。"改前"基线通过 FACTION_AUDIT_DUMP=<file> 导出。

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"silo40/internal/model"
)

const (
	auditTwinCentroidRadius     = 0.20 // 双胞胎判定：质心思潮距离上限
	auditOvermergedPairDistance = 0.35 // 坍缩判定：同派系内 cohort 思潮距离下限
	auditMultiProfThreshold     = 5    // 坍缩判定：跨部门派系的部门数下限
	auditClusterHopRadius       = 0.30 // 真实换组判定：新旧组质心距离下限 (与聚类半径同量级)
)

// factionAuditSummary 单局审计汇总 (全部为确定性可复现计数)。
type factionAuditSummary struct {
	Seed   int64 `json:"seed"`
	Months int   `json:"months"`

	TwinPairMonths    int     `json:"twin_pair_months"`    // 存在双胞胎派系对的月数
	OvermergedPairSum int     `json:"overmerged_pair_sum"` // 同派系远距 cohort 对数 (逐月累加)
	MultiProfMonths   int     `json:"multi_prof_months"`   // 存在跨≥5部门政治派系的月数
	PoliticalCountSum int     `json:"political_count_sum"` // 政治派系数逐月累加 (均值=除以月数)
	MaxPopSharePeak   float64 `json:"max_pop_share_peak"`  // 最大政治派系人口占比峰值
	CohortFlips       int     `json:"cohort_flips"`        // cohort 派系签名逐月变化次数 (标签层)
	ABA_Oscillations  int     `json:"aba_oscillations"`    // A→B→A 三步往返次数 (标签层)
	MembershipFlips   int     `json:"membership_flips"`    // cohort 所在组成员构成变化次数 (归属层，含组成放大)
	MembershipABA     int     `json:"membership_aba"`      // 组成员构成 A→B→A 往返次数 (归属层)
	GateFlips         int     `json:"gate_flips"`          // 政治化状态翻转次数 (政治↔无阵营)
	ClusterHops       int     `json:"cluster_hops"`        // 持续政治化的 cohort 真实换组次数 (新旧组质心距离>半径)
	IDFlips           int     `json:"id_flips"`            // cohort.FactionID 原始值变化次数 (S3 后应≈门槛+换组)
}

// collectFactionAudit 单局审计：逐月推进并采集缺陷指标。
func collectFactionAudit(t *testing.T, seed int64, months int) factionAuditSummary {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), fmt.Sprintf("faction-audit-%d.db", seed))
	svc := newParityService(t, dbPath)
	if _, err := svc.CreateGame(model.CreateGameRequest{
		SiloName:   fmt.Sprintf("Audit-%d", seed),
		AgentName:  "AuditAgent",
		Profession: "IT",
		Seed:       seed,
	}); err != nil {
		t.Fatalf("create game seed %d: %v", seed, err)
	}

	sum := factionAuditSummary{Seed: seed}
	prevSig := map[uint]string{}     // cohortID -> 上一月派系签名
	prevPrevSig := map[uint]string{} // cohortID -> 上上月派系签名
	prevGroup := map[uint]string{}     // cohortID -> 上一月组成员指纹
	prevPrevGroup := map[uint]string{} // cohortID -> 上上月组成员指纹
	prevPolitical := map[uint]bool{}   // cohortID -> 上一月是否政治化
	prevCentroid := map[uint][2]float64{} // cohortID -> 上一月所属派系质心 (人数加权 pf, loyalty)
	prevFactionID := map[uint]uint{}   // cohortID -> 上一月 FactionID 原始值 (0 = 无)

	for m := 1; m <= months; m++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("seed %d month %d: %v", seed, m, err)
		}
		if svc.gameOver() {
			break
		}
		sum.Months = m
		auditOneMonth(svc.silo, &sum, prevSig, prevPrevSig, prevGroup, prevPrevGroup, prevPolitical, prevCentroid, prevFactionID)
	}
	return sum
}

// auditOneMonth 采集单个时点的缺陷指标并滚动签名/成员指纹历史。
func auditOneMonth(silo *model.Silo, sum *factionAuditSummary, prevSig, prevPrevSig, prevGroup, prevPrevGroup map[uint]string, prevPolitical map[uint]bool, prevCentroid map[uint][2]float64, prevFactionID map[uint]uint) {
	factionByID := map[uint]*model.Faction{}
	for i := range silo.Factions {
		factionByID[silo.Factions[i].ID] = &silo.Factions[i]
	}

	// ---- 派系级指标 ----
	type polFaction struct {
		signature  string
		rest       string // 签名去掉 pro_foreign 档位后的剩余部分
		bucket     string // pro_foreign 档位标签
		centroidPF float64
		profCount  int
		members    int
	}
	var political []polFaction
	totalPop := silo.TotalPopulation
	for i := range silo.Factions {
		f := &silo.Factions[i]
		if f.Signature == "special:unaffiliated" {
			continue
		}
		political = append(political, polFaction{
			signature:  f.Signature,
			rest:       auditSignatureRest(f.Signature),
			bucket:     auditSignatureBucket(f.Signature),
			centroidPF: auditFactionCentroidProForeign(silo, f),
			profCount:  auditFactionProfCount(silo, f),
			members:    f.MemberCount,
		})
	}
	sum.PoliticalCountSum += len(political)

	// 双胞胎：同 rest、不同档位、质心已收敛。
	twinFound := false
	for i := 0; i < len(political) && !twinFound; i++ {
		for j := i + 1; j < len(political); j++ {
			a, b := political[i], political[j]
			if a.rest == b.rest && a.bucket != b.bucket &&
				a.bucket != "" && b.bucket != "" &&
				math.Abs(a.centroidPF-b.centroidPF) < auditTwinCentroidRadius {
				twinFound = true
				break
			}
		}
	}
	if twinFound {
		sum.TwinPairMonths++
	}

	// 坍缩：跨部门大派系 + 最大派系人口占比。
	for _, pf := range political {
		if pf.profCount >= auditMultiProfThreshold {
			sum.MultiProfMonths++
			break
		}
	}
	for _, pf := range political {
		if totalPop > 0 {
			share := float64(pf.members) / float64(totalPop)
			if share > sum.MaxPopSharePeak {
				sum.MaxPopSharePeak = share
			}
		}
	}

	// 坍缩：同派系内远距 cohort 对。
	cohortSig := map[uint]string{}
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		sig := "special:unaffiliated"
		if cohort.FactionID != nil {
			if f, ok := factionByID[*cohort.FactionID]; ok {
				sig = f.Signature
			}
		}
		cohortSig[cohort.ID] = sig
	}
	for i := range silo.Factions {
		f := &silo.Factions[i]
		if f.Signature == "special:unaffiliated" {
			continue
		}
		var memberPF []float64
		for j := range silo.Cohorts {
			cohort := &silo.Cohorts[j]
			if cohort.FactionID == nil || *cohort.FactionID != f.ID {
				continue
			}
			memberPF = append(memberPF, cohort.Ideologies[model.IdeologyProForeign])
		}
		for a := 0; a < len(memberPF); a++ {
			for b := a + 1; b < len(memberPF); b++ {
				if math.Abs(memberPF[a]-memberPF[b]) > auditOvermergedPairDistance {
					sum.OvermergedPairSum++
				}
			}
		}
	}

	// ---- cohort 归属振荡 (标签层：派系签名变化) ----
	for id, sig := range cohortSig {
		if prev, ok := prevSig[id]; ok && prev != sig {
			sum.CohortFlips++
			if pp, ok := prevPrevSig[id]; ok && pp == sig {
				sum.ABA_Oscillations++
			}
		}
	}

	// ---- cohort 归属振荡 (归属层：组成员构成变化) ----
	// 组成员指纹 = 该派系全部成员 cohort ID 的有序拼接 (无阵营 = "U")。
	memberSets := map[uint][]uint{}
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		if cohort.FactionID == nil {
			continue
		}
		memberSets[*cohort.FactionID] = append(memberSets[*cohort.FactionID], cohort.ID)
	}
	groupFp := map[uint]string{}
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		fp := "U"
		if cohort.FactionID != nil {
			f := factionByID[*cohort.FactionID]
			if f != nil && f.Signature != "special:unaffiliated" {
				ids := append([]uint(nil), memberSets[*cohort.FactionID]...)
				sort.Slice(ids, func(a, b int) bool { return ids[a] < ids[b] })
				parts := make([]string, len(ids))
				for k, id := range ids {
					parts[k] = fmt.Sprintf("%d", id)
				}
				fp = strings.Join(parts, ",")
			}
		}
		groupFp[cohort.ID] = fp
		if prev, ok := prevGroup[cohort.ID]; ok && prev != fp {
			sum.MembershipFlips++
			if pp, ok := prevPrevGroup[cohort.ID]; ok && pp == fp {
				sum.MembershipABA++
			}
		}
	}

	// ---- 真实归属变动 (剔除组成放大) ----
	// 派系质心 (人数加权 pf/loyalty) 与成员政治化状态。
	factionCentroid := map[uint][2]float64{}
	for fid := range memberSets {
		total := 0
		var sumPF, sumLoy float64
		for i := range silo.Cohorts {
			cohort := &silo.Cohorts[i]
			if cohort.FactionID == nil || *cohort.FactionID != fid {
				continue
			}
			total += cohort.Count
			sumPF += cohort.Ideologies[model.IdeologyProForeign] * float64(cohort.Count)
			sumLoy += cohort.Ideologies[model.IdeologyLoyalty] * float64(cohort.Count)
		}
		if total > 0 {
			factionCentroid[fid] = [2]float64{sumPF / float64(total), sumLoy / float64(total)}
		}
	}
	curPolitical := map[uint]bool{}
	curCentroid := map[uint][2]float64{}
	curFactionID := map[uint]uint{}
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		isPol := false
		fid := uint(0)
		if cohort.FactionID != nil {
			fid = *cohort.FactionID
			if f := factionByID[fid]; f != nil && f.Signature != "special:unaffiliated" {
				isPol = true
				curCentroid[cohort.ID] = factionCentroid[fid]
			}
		}
		curFactionID[cohort.ID] = fid
		if prev, ok := prevFactionID[cohort.ID]; ok && prev != fid {
			sum.IDFlips++
		}
		curPolitical[cohort.ID] = isPol
		if was, ok := prevPolitical[cohort.ID]; ok && was != isPol {
			sum.GateFlips++
		} else if isPol && was {
			// 持续政治化：新旧组质心距离超过聚类半径才算真实换组。
			if prevC, ok := prevCentroid[cohort.ID]; ok {
				c := curCentroid[cohort.ID]
				d := math.Sqrt((c[0]-prevC[0])*(c[0]-prevC[0]) + (c[1]-prevC[1])*(c[1]-prevC[1]))
				if d > auditClusterHopRadius {
					sum.ClusterHops++
				}
			}
		}
	}

	// 滚动历史 (确定性：逐条覆盖)。
	for id, sig := range prevSig {
		prevPrevSig[id] = sig
	}
	for id, sig := range cohortSig {
		prevSig[id] = sig
	}
	for id, fp := range prevGroup {
		prevPrevGroup[id] = fp
	}
	for id, fp := range groupFp {
		prevGroup[id] = fp
	}
	for id, pol := range curPolitical {
		prevPolitical[id] = pol
	}
	for id, c := range curCentroid {
		prevCentroid[id] = c
	}
	for id, fid := range curFactionID {
		prevFactionID[id] = fid
	}
}

// auditSignatureRest 去掉 pro_foreign 档位标签后的签名剩余部分。
func auditSignatureRest(signature string) string {
	parts := strings.Split(signature, "|")
	rest := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.HasPrefix(p, model.IdeologyProForeign+":") {
			continue
		}
		rest = append(rest, p)
	}
	sort.Strings(rest)
	return strings.Join(rest, "|")
}

// auditSignatureBucket 签名中的 pro_foreign 档位标签 (无则空)。
func auditSignatureBucket(signature string) string {
	for _, p := range strings.Split(signature, "|") {
		if strings.HasPrefix(p, model.IdeologyProForeign+":") {
			return p
		}
	}
	return ""
}

// auditFactionCentroidProForeign 派系成员人数加权的亲外思潮质心。
func auditFactionCentroidProForeign(silo *model.Silo, f *model.Faction) float64 {
	total := 0
	weighted := 0.0
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		if cohort.FactionID == nil || *cohort.FactionID != f.ID {
			continue
		}
		total += cohort.Count
		weighted += cohort.Ideologies[model.IdeologyProForeign] * float64(cohort.Count)
	}
	if total == 0 {
		return 0
	}
	return weighted / float64(total)
}

// auditFactionProfCount 派系覆盖的不同部门数。
func auditFactionProfCount(silo *model.Silo, f *model.Faction) int {
	profs := map[uint]bool{}
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		if cohort.FactionID != nil && *cohort.FactionID == f.ID {
			profs[cohort.ProfessionID] = true
		}
	}
	return len(profs)
}

// TestFactionAuditDeterministic 硬约束 B：同种子双跑审计摘要逐字节一致。
func TestFactionAuditDeterministic(t *testing.T) {
	for _, seed := range []int64{7, 8} {
		a := collectFactionAudit(t, seed, 120)
		b := collectFactionAudit(t, seed, 120)
		ja, err := json.Marshal(a)
		if err != nil {
			t.Fatalf("marshal a: %v", err)
		}
		jb, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("marshal b: %v", err)
		}
		if string(ja) != string(jb) {
			t.Fatalf("seed %d: audit diverged between runs\nrun1=%s\nrun2=%s", seed, ja, jb)
		}
	}
}

// TestFactionAuditBaselineDump 导出"改前"缺陷基线 (FACTION_AUDIT_DUMP=<file>)，
// 并在日志中打印每种子摘要。无 env 时仅冒烟运行一种子。
func TestFactionAuditBaselineDump(t *testing.T) {
	outPath := os.Getenv("FACTION_AUDIT_DUMP")
	seeds := []int64{7}
	if outPath != "" {
		seeds = []int64{7, 8, 9, 10, 11, 12}
	}
	sums := make([]factionAuditSummary, 0, len(seeds))
	for _, seed := range seeds {
		sum := collectFactionAudit(t, seed, 120)
		sums = append(sums, sum)
		t.Logf("seed=%d months=%d twin_months=%d overmerged=%d multi_prof_months=%d avg_political=%.2f max_share=%.3f flips=%d aba=%d mem_flips=%d mem_aba=%d gate_flips=%d hops=%d id_flips=%d",
			sum.Seed, sum.Months, sum.TwinPairMonths, sum.OvermergedPairSum, sum.MultiProfMonths,
			float64(sum.PoliticalCountSum)/float64(maxInt(sum.Months, 1)), sum.MaxPopSharePeak,
			sum.CohortFlips, sum.ABA_Oscillations, sum.MembershipFlips, sum.MembershipABA,
			sum.GateFlips, sum.ClusterHops, sum.IDFlips)
	}
	if outPath != "" {
		data, err := json.MarshalIndent(sums, "", "  ")
		if err != nil {
			t.Fatalf("marshal sums: %v", err)
		}
		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			t.Fatalf("write dump: %v", err)
		}
		t.Logf("faction audit baseline written to %s", outPath)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
