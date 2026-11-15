package engine

// 盲区经营对黑箱观测管线的注入系数 (23 系列 05 卷 §3.1/§3.2)。全部
// TODO(balance)：推荐初值，等待 sim 校准。
//
// 纪律 (23-05 §2.1)：
//   - 注入是零 RNG 的确定性映射，只改分布参数、不改抽样结构——4.65 仍是每信号
//     1 次伪装判定骰 + 2 次噪声取样，DrawFloat64 消耗次数与顺序逐位不变 (§3.3)；
//   - 零状态 (BlindSpot 未启用/全零) 退化为 22 基线参数 (恒等映射，§8)；
//   - 多通道加权求和一律按固定通道序 equipment→schedule→reports→ledger→
//     census→timing 累加，禁止 map 遍历序 (避免 ULP 漂移)。
//
// 本文件属观测核心格 (W2)。命名独立于 blackbox_constants.go，避免与物理层格
// (W1) 的常量文件冲突。

// 注入系数 (23-05 §3.1 参数表)。
const (
	// bbInjectBP 伪装概率经营放大系数 b_p：满有效经营量 (ῑ_k=1) 使系统性粉饰
	// 概率翻倍 (clamp 前)——经营使「粉饰」从临机决定变成流程默认。
	bbInjectBP = 1.0 // TODO(balance)
	// bbInjectBD 伪装偏移经营增量系数 b_d：与 22 基线最主观信号 line_tone 的
	// d_k=0.25 同量级，防止增量项压倒基线。
	bbInjectBD = 0.15 // TODO(balance)
	// bbInjectBSigma 噪声膨胀系数 b_σ：盲区通道经手改写、层层转抄，报告失去
	// 真值锚定后口径噪声上升 (噪声膨胀只挂 Cov_c，§2.1 纪律 6)。
	bbInjectBSigma = 0.5 // TODO(balance)
	// bbInjectSMax 噪声膨胀上限 s_max：σ 至多 ×1.5——噪声过大则 1 号侧似然权重
	// 趋零、观测被事实丢弃，那是「放弃通道」而不是「经营通道」。
	bbInjectSMax = 0.5 // TODO(balance)
)

// 伪装偏移上限 d_k^cap (receipt/gap/tone)：约为 22 基线 (0.05/0.15/0.25) 的
// 1.6 倍，保证深经营下 y 仍保留 σ 尺度的噪声外观——偏置藏在噪声里，单次看仍像
// 一条普通的好消息 (23-05 §3.1)。
const (
	bbInjectDCapReceiptDelay = 0.20 // TODO(balance)
	bbInjectDCapExecutionGap = 0.30 // TODO(balance)
	bbInjectDCapLineTone     = 0.40 // TODO(balance)
)

// 盲区六通道索引：固定通道序 = models.go BlindSpotChannel* 声明序 = 存档数组序。
const (
	bbChannelEquipment = iota // equipment 监控设备盲区
	bbChannelSchedule         // schedule  排班/迎检安排
	bbChannelReports          // reports   报表口径
	bbChannelLedger           // ledger    台账/回执
	bbChannelCensus           // census    人口底册
	bbChannelTiming           // timing    上报时点
	bbChannelCount
)

// 逐信号通道权重 w_{k,c} (23-05 §3.2)。数组下标 = 固定通道序；每行权重和为 1
// (ῑ_k ≤ 1，保证各上限语义稳定)；零项 = 该通道在该信号无注入点。
var (
	// receipt_delay：timing 0.7 / schedule 0.3——回执时延是 timing 通道主场，
	// 排班决定「谁在什么时候回执」。
	bbWeightsReceiptDelay = [bbChannelCount]float64{0, 0.3, 0, 0, 0, 0.7} // TODO(balance)
	// execution_gap：ledger 0.5 / reports 0.3 / equipment 0.2——台账虚报直接收窄
	// 表观差距，报告粉饰次之，设备状态造假再次之。
	bbWeightsExecutionGap = [bbChannelCount]float64{0.2, 0, 0.3, 0.5, 0, 0} // TODO(balance)
	// line_tone：reports 0.6 / schedule 0.4——汇报文本口径是 reports 核心业务，
	// 迎检排班安排「表演性汇报」影响态度读数。
	bbWeightsLineTone = [bbChannelCount]float64{0, 0.4, 0.6, 0, 0, 0} // TODO(balance)
)
