import { memo } from "react";
import { Box, Badge, Heading, HStack, Text, VStack } from "@chakra-ui/react";
import { InformationProgressPanel } from "./InformationProgressPanel";

// 线信息架构面板：一局只有一条主线，其余是"还开着的路"。
//
// 数据全部来自后端权威快照：lines_progress (主线/里程碑/改道日志/征兆) +
// information_progress (《共证》链与部门覆盖) + silo (共享事实的单一来源)。
// 跨线共享的事实 (信念覆盖、登记簿、W、警察服从……) 只在共享事实带出现一次，
// 任何线卡不再各自携带。

interface Props {
    lines: any;      // lines_progress
    progress: any;   // information_progress
    silo: any;
    world: any;
    agent: any;
}

// 阶段是后端权威进度 (action_gates.json)；主线未定时，阶段简报就是"这一局在做什么"。
const stageBriefing: Record<string, { label: string; brief: string }> = {
    arrival: {
        label: "初到驻地",
        brief: "没有人认识你。第一步：取得可核验的材料——「Gather Info」是通用起点；确信提到 B ≥ 0.30，材料才够格作为事实离手。",
    },
    first_signal: {
        label: "第一次本地观察",
        brief: "「Share Info」与职务动作已解锁。把手里确信 B ≥ 0.30 的材料交给能独立查账的人——手里还没有够格的材料，就先「Gather Info」。",
    },
    archive_awakened: {
        label: "档案苏醒",
        brief: "第一次有本地人用自己的职务语言重新提出问题：材料已经离手。下一步：陪同回查、复制，并让流通跑赢淡忘。",
    },
    chain_living: {
        label: "活链成立",
        brief: "链不需要你逐日维持也能复制。剩下的是公共事实，以及守住宿主活到结算。",
    },
    chain_public_fact: {
        label: "公共事实",
        brief: "核心真相已成为跨部门公共事实。守住宿主、跑赢淡忘，直到结算。",
    },
};

const stanceLabel: Record<string, string> = {
    block: "阻挠",
    nonblock: "不阻挠",
    cooperate: "配合",
};

const regimeLabel: Record<string, string> = {
    IT: "IT 控制",
    SWING: "争夺态",
    PACT: "附着约章",
};

const trackingLabel: Record<string, string> = {
    none: "没人注意",
    noticed: "已被察觉",
    trailing: "咬住痕迹",
    hunting: "正在追捕",
};

// 谈判层口径 (18 设计卷)：三方带权决策，只改变谁在阻碍谁，不发放胜利。
const negotiationDispositionLabel: Record<string, string> = {
    hardliner: "强硬派",
    pragmatist: "务实派",
    survivor: "求生派",
};

const negotiationPostureLabel: Record<string, string> = {
    press: "施压",
    probe: "试探",
    offer: "递条款",
};

const negotiationDeciderLabel: Record<string, string> = {
    mayor: "市长",
    rebels: "叛乱方",
    it: "IT",
    channel: "信道",
};

// 决策日志权重档位标签：三方各自的档位合并成一张表。
const negotiationWeightLabel: Record<string, string> = {
    suppress: "镇压",
    talk: "谈判",
    concede: "妥协",
    accept: "接受",
    reject: "拒绝",
    press: "施压",
    probe: "试探",
    offer: "递条款",
    stand_aside: "不干涉",
    undermine: "破坏",
    watch: "观望",
};

function negotiationLeanLabel(lean: number): string {
    if (lean < 0.3) return "强硬";
    if (lean < 0.5) return "动摇";
    if (lean < 0.7) return "倾向妥协";
    return "准备妥协";
}

// 谈判块：信道状态、市长倾向、桌上提案、最近决策日志 (含权重，供复盘)。
function NegotiationDetail({ silo }: { silo: any }) {
    const negotiation = silo?.negotiation || {};
    if (!negotiation.initialized) {
        return (
            <Text fontSize="xs" color="gray.500">
                谈判：信道未建立——叛乱成形后，市长面对的是三条路：镇压、谈判、妥协。
            </Text>
        );
    }
    if (negotiation.closed) {
        const reason = negotiation.close_reason === "it_exposure" ? "被 IT 曝光" : "被镇压打断";
        return (
            <Text fontSize="xs" color="red.700">
                谈判：信道已崩塌（{reason}）· 已交换 {negotiation.rounds || 0} 回合。这一轮谈判到此为止。
            </Text>
        );
    }
    const log = (negotiation.log || []).slice(-4).reverse();
    return (
        <VStack align="stretch" gap={1}>
            <Text fontSize="xs" color="gray.700">
                谈判：信道畅通 · 已交换 {negotiation.rounds || 0} 回合 ·
                市长性情 {negotiationDispositionLabel[negotiation.mayor_disposition] || "未知"} ·
                市长此刻倾向 {negotiationLeanLabel(negotiation.mayor_lean || 0)}
                {negotiation.rebel_posture ? ` · 叛乱方姿态 ${negotiationPostureLabel[negotiation.rebel_posture] || negotiation.rebel_posture}` : ""}
            </Text>
            {negotiation.offer_on_table && (
                <Text fontSize="xs" color="gray.700">
                    桌上提案：{negotiation.offered_by === "mayor" ? "市长开出" : "叛乱方递出"} · {negotiation.offer_terms}
                </Text>
            )}
            {negotiation.compromise_established && (
                <Text fontSize="xs" color="green.700">
                    妥协已成立{negotiation.takeover === "takeover" ? " · 叛乱方已接管市长职位"
                        : negotiation.takeover === "hands_off" ? " · 叛乱方不接管，带组织离开"
                            : " · 接管与否尚未抉择"}
                </Text>
            )}
            {log.length > 0 && (
                <VStack align="stretch" gap={0.5}>
                    {log.map((entry: any, idx: number) => {
                        const weights = entry.weights
                            ? Object.entries(entry.weights).map(([key, value]) =>
                                `${negotiationWeightLabel[key] || key} ${Math.round((value as number) * 100)}%`).join(" · ")
                            : "";
                        return (
                            <Text key={idx} fontSize="xs" color="gray.600">
                                Y{entry.year} M{entry.month} · {negotiationDeciderLabel[entry.decider] || entry.decider}
                                {entry.decision && entry.weights ? ` · ${weights}` : ""} · {entry.outcome}
                            </Text>
                        );
                    })}
                </VStack>
            )}
            <Text fontSize="xs" color="gray.400">
                谈判只改变谁在阻碍谁：拖延与失败都是合法结果，胜负仍只看逃出去多少人。
            </Text>
        </VStack>
    );
}

function fmt(value: number | undefined, digits = 2): string {
    return typeof value === "number" ? value.toFixed(digits) : "—";
}

// 里程碑徽章行：已亮的绿色实心，未亮的灰色描边。
function MilestoneRow({ line }: { line: any }) {
    return (
        <HStack wrap="wrap" gap={2}>
            {(line.milestones || []).map((m: any) => (
                <Badge key={m.key} colorPalette={m.lit ? "green" : "gray"} variant={m.lit ? "solid" : "outline"}>
                    {m.title}{m.count > 1 ? ` ×${m.count}` : ""}
                </Badge>
            ))}
        </HStack>
    );
}

// 线状态徽章：资格成立 / 回摆 / 关闭——状态是事实，不是预测。
function LineStatusBadges({ line }: { line: any }) {
    return (
        <HStack gap={2} wrap="wrap">
            {line.qualified && <Badge colorPalette="green">资格成立</Badge>}
            {line.regressed && <Badge colorPalette="orange">回摆中</Badge>}
            {line.closed && <Badge colorPalette="red">已关闭</Badge>}
        </HStack>
    );
}

function lineNote(line: any): string {
    if (line.closed && line.close_note) return line.close_note;
    if (line.regressed && line.qual_note) return `回摆：${line.qual_note}`;
    if (!line.qualified && line.qual_note && line.depth > 0) return `尚差：${line.qual_note}`;
    return "";
}

// 《越岭》线卡：账本与立场是这条线独有的事实 (仲裁不读它们，面板只报事实)。
function RebellionDetail({ silo }: { silo: any }) {
    const prep = silo?.escape_prep || {};
    const stance = silo?.top_stance || {};
    const group = silo?.escaped_group || {};
    const prepStarted = prep.initialized || prep.suits_functional > 0 || (prep.plan_roles || []).length > 0;
    return (
        <VStack align="stretch" gap={1} mt={2}>
            {prepStarted ? (
                <Text fontSize="xs" color="gray.700">
                    准备账本：防护服 {prep.suits_functional || 0} 套 · 预案 {(prep.plan_roles || []).length}/6 功能位 ·
                    演练 {fmt(prep.drill_level)} · 照护 {fmt(prep.care_ready)} ·
                    补给 {fmt(prep.supplies_packed, 0)} 人月 · 暴露 {fmt(prep.prep_exposure)}
                    {prep.production_start ? " · 生产起点已备" : ""}
                    {prep.reception_stage && prep.reception_stage !== "none" ? ` · 接应：${prep.reception_stage}` : ""}
                </Text>
            ) : (
                <Text fontSize="xs" color="gray.500">准备账本还是空的：防护服、预案、演练、照护——每一项都要用动作与成本换。</Text>
            )}
            <Text fontSize="xs" color="gray.700">
                顶层立场：{stance.initialized
                    ? `市长 ${stanceLabel[stance.mayor] || "阻挠"} · IT ${stanceLabel[stance.it] || "阻挠"}`
                    : "尚未成形——磋商是持续过程，立场随状态漂移，最后一刻妥协也算"}
            </Text>
            <NegotiationDetail silo={silo} />
            {group.initialized && (
                <Text fontSize="xs" color="gray.700">
                    逃出群体：{group.departed || 0} 人出发 · 存续 {group.population || 0} 人
                    {group.outside?.months_out > 0 ? ` · 堡外 ${group.outside.months_out} 月 · 追踪：${trackingLabel[group.outside.tracking] || group.outside.tracking}` : ""}
                </Text>
            )}
        </VStack>
    );
}

// 《公议》线卡：约章附着与锁定状态。
function RuleOfLawDetail({ silo }: { silo: any }) {
    const judicial = silo?.authority?.judicial || {};
    return (
        <Text fontSize="xs" color="gray.700" mt={2}>
            司法部：{regimeLabel[judicial.regime] || judicial.regime || "—"} ·
            约章附着 {fmt(judicial.a_pact)} · {judicial.pact_locked ? "已锁定（不可被单方收回）" : "未锁定"}
        </Text>
    );
}

// 《无人裁决》线卡：覆灭与快照状态。
function TimeDetail({ silo, world }: { silo: any; world: any }) {
    const snap = silo?.pre_fall_snapshot || {};
    const alive = world?.silo1_alive;
    let snapLine = "1 号仍在运转——这条线的前提是它不再给出下一项处置。";
    if (silo?.silo1_destroyed || alive === false) {
        if (!snap.taken) {
            snapLine = "1 号已覆灭：覆灭前夕思潮快照尚未截取。";
        } else if (snap.openQualified) {
            snapLine = "1 号已覆灭：覆灭前夕思潮闸通过——硬条件已在结算瞬间定格。";
        } else {
            snapLine = "1 号已覆灭：覆灭前夕思潮闸未通过（纯种地堡思潮主导）。";
        }
    }
    return <Text fontSize="xs" color="gray.700" mt={2}>{snapLine}</Text>;
}

// 主线未定时的教义卡：《见证》目标与三判据 (原 MissionPanel 的内容)。
function MissionBriefing({ world, agent, progress }: { world: any; agent: any; progress: any }) {
    const stage = world?.progress_stage || "arrival";
    const briefing = stageBriefing[stage] || stageBriefing.arrival;
    const knownCount = agent?.known_fragments?.length || 0;
    const absorbedTotal = (progress?.departments || []).reduce(
        (sum: number, dept: any) => sum + (dept.above_floor || 0), 0);
    const nodeLit = (key: string) => (progress?.nodes || []).some((n: any) => n.key === key && n.lit);
    const replicationCount = (progress?.nodes || []).find(
        (n: any) => n.key === "special:replication_progress")?.count || 0;

    let factLine: string;
    if (knownCount === 0) {
        factLine = "你还没有任何可核验的材料——「Gather Info」是起点。";
    } else if (absorbedTotal === 0) {
        factLine = `在手材料 ${knownCount} 份。它们还只是你的私人确信，尚未进入任何部门的账本。`;
    } else {
        factLine = `在手材料 ${knownCount} 份 · ${absorbedTotal} 份事实已进入部门信念层，开始不依赖你而存在。`;
    }

    let receiverLine: string;
    if (nodeLit("special:living_chain_confirmed")) {
        receiverLine = "活链已成立：承接者们在没有你逐日维持的情况下继续核验与复制。";
    } else if (nodeLit("special:replication_progress")) {
        receiverLine = `第一位承接者已独立核验，复制正在推进（×${replicationCount}）。`;
    } else if (nodeLit("player_actions:receiver_verify_accompany")) {
        receiverLine = "第一位承接者已完成独立核验。下一个承接者要由他们自己长出来。";
    } else {
        receiverLine = "还没有人从你手里接过材料并独立核验。你第一次分享的对象，决定承接者是谁。";
    }

    let chainLine: string;
    if (progress?.reset_countermeasure_done) {
        chainLine = "反制已成立：终止链的副本分散在多人手中，单方命令不再好使。";
    } else if (progress?.reset_omen_seen) {
        chainLine = "先兆已现：终止链正在启动。把反制做成可核验的事实——分发托管是成立之路。";
    } else {
        chainLine = "终止链仍在正常运转：1 号可以经本地线路单方重置驻地。它的先兆会以危机的形式出现。";
    }

    return (
        <VStack align="stretch" gap={2} mb={3}>
            <HStack align="start" gap={2}>
                <Badge colorPalette="blue" mt={0.5}>{briefing.label}</Badge>
                <Text fontSize="xs" color="gray.700">{briefing.brief}</Text>
            </HStack>
            <Box p={2} bg="white" borderRadius="sm" border="1px solid" borderColor="gray.200">
                <Text fontSize="xs" fontWeight="bold" color="gray.700">判据一 · 事实脱离你的自述</Text>
                <Text fontSize="xs" color="gray.600">{factLine}</Text>
            </Box>
            <Box p={2} bg="white" borderRadius="sm" border="1px solid" borderColor="gray.200">
                <Text fontSize="xs" fontWeight="bold" color="gray.700">判据二 · 真相有本地承接者</Text>
                <Text fontSize="xs" color="gray.600">{receiverLine}</Text>
            </Box>
            <Box p={2} bg="white" borderRadius="sm" border="1px solid" borderColor="gray.200">
                <Text fontSize="xs" fontWeight="bold" color="gray.700">判据三 · 驻地不再受单方终止链支配</Text>
                <Text fontSize="xs" color="gray.600">{chainLine}</Text>
            </Box>
            <Text fontSize="xs" color="gray.500">
                三判据是《见证》教义的总账；每条路按自己的里程碑结算——《共证》看部门事实门槛，《无人裁决》看覆灭前夕的思潮快照。
                但终止链若先完成，驻地会被重置（《遗忘》），所有路线都失去结算资格：判据三是任何结局的前提。
            </Text>
        </VStack>
    );
}

// 共享事实带：被多条线消费的事实只在这里说一次。
function SharedFacts({ silo, world, progress }: { silo: any; world: any; progress: any }) {
    const authority = silo?.authority || {};
    const judicial = authority?.judicial || {};
    const departments = progress?.departments || [];
    const gate = progress?.gate_fact_count || 0;
    const beliefFacts = departments.reduce((sum: number, d: any) => sum + (d.above_floor || 0), 0);
    const heldDepts = departments.filter((d: any) => (d.above_floor || 0) >= gate).length;
    const items: { label: string; value: string; tone?: string }[] = [
        { label: "信念覆盖", value: `${beliefFacts} 份（地板 ${fmt(progress?.belief_floor)}）` },
        { label: "达标部门", value: `${heldDepts}/${departments.length}` },
        { label: "登记簿", value: `${(silo?.public_evidence || []).length} 件` },
        { label: "Q", value: fmt(judicial.public_evidence_q) },
        { label: "W", value: fmt(authority.w, 1) },
        { label: "警察服从", value: fmt(authority.police_obedience) },
        { label: "市长正统性", value: fmt(silo?.mayor?.legitimacy) },
        { label: "司法部", value: regimeLabel[judicial.regime] || judicial.regime || "—" },
        { label: "叛乱", value: fmt(silo?.rebellion) },
        { label: "1 号", value: world?.silo1_alive ? "运转中" : "已覆灭", tone: world?.silo1_alive ? undefined : "red" },
    ];
    const evidence = silo?.public_evidence || [];
    return (
        <Box p={2} bg="white" borderRadius="sm" border="1px solid" borderColor="gray.200">
            <Text fontSize="xs" fontWeight="bold" color="gray.700" mb={1}>共享事实 · 各线读同一组数</Text>
            <HStack wrap="wrap" gap={2}>
                {items.map(item => (
                    <Box key={item.label} px={2} py={0.5} bg="gray.50" borderRadius="sm">
                        <Text fontSize="xs" color="gray.500" as="span">{item.label} </Text>
                        <Text fontSize="xs" fontWeight="bold" as="span" color={item.tone === "red" ? "red.600" : "gray.800"}>
                            {item.value}
                        </Text>
                    </Box>
                ))}
            </HStack>
            {evidence.length > 0 && (
                <VStack align="stretch" gap={0.5} mt={1}>
                    {evidence.map((record: any) => (
                        <Text key={record.fact_id} fontSize="xs" color="gray.600">
                            登记 {record.fact_id} · 来源 {Number(record.provenance).toFixed(1)} · 覆盖 {Number(record.coverage).toFixed(1)}
                        </Text>
                    ))}
                </VStack>
            )}
            <Text fontSize="xs" color="gray.400" mt={1}>
                W = 思潮开放度（0 封闭 → 100 开放，人口加权）；Q = 公共证据对司法侧的可读性（&gt;0 才可被引用）。
            </Text>
        </Box>
    );
}

export const LinesPanel = memo(function LinesPanel({ lines, progress, silo, world, agent }: Props) {
    if (!lines) return null;
    const allLines: any[] = lines.lines || [];
    const mainLine = lines.main_line || "";
    const main = allLines.find(l => l.line === mainLine);
    const others = allLines.filter(l => l.line !== mainLine && (l.depth > 0 || l.qualified || l.closed));
    const unstarted = allLines.filter(l => l.line !== mainLine && l.depth === 0 && !l.qualified && !l.closed);
    const shifts = [...(lines.shifts || [])].reverse();
    const omens = lines.omens || [];
    const titleOf = (id: string) => allLines.find(l => l.line === id)?.title || id;

    return (
        <Box w="full" p={4} bg="gray.50" border="1px solid" borderColor="gray.300" borderRadius="md">
            <HStack justify="space-between" align="center" mb={2}>
                <Heading size="sm" color="gray.800">这一局在走什么</Heading>
                {main
                    ? <Badge colorPalette="purple" variant="solid">主线 · {main.title}</Badge>
                    : <Badge colorPalette="gray" variant="outline">主线未定</Badge>}
            </HStack>
            <Text fontSize="xs" color="gray.600" mb={3}>
                让 40 号的覆灭与 1 号的控制方式不再只能由控制者单方叙述——真相要能被独立核验、被驻地的人保有。
                五条路都开着，你的脚印决定哪条是主线。
            </Text>

            {/* 主线未定：教义与阶段简报就是"这一局在做什么" */}
            {!main && <MissionBriefing world={world} agent={agent} progress={progress} />}

            {/* 主线卡：全细节 */}
            {main && (
                <Box p={3} bg="white" borderRadius="md" border="2px solid" borderColor="purple.300" mb={3}>
                    <HStack justify="space-between" align="start" mb={2}>
                        <Text fontWeight="bold" color="purple.800">{main.title}</Text>
                        <LineStatusBadges line={main} />
                    </HStack>
                    {main.line === "information" && (
                        <>
                            <MissionBriefing world={world} agent={agent} progress={progress} />
                            <InformationProgressPanel progress={progress} finished={!!silo?.victory_status} />
                        </>
                    )}
                    {main.line !== "information" && <MilestoneRow line={main} />}
                    {main.line === "rebellion" && <RebellionDetail silo={silo} />}
                    {main.line === "rule_of_law" && <RuleOfLawDetail silo={silo} />}
                    {main.line === "time" && <TimeDetail silo={silo} world={world} />}
                    {lineNote(main) && <Text fontSize="xs" color="orange.700" mt={2}>{lineNote(main)}</Text>}
                </Box>
            )}

            {/* 还开着的路：一行一条，不摊细节 */}
            {others.length > 0 && (
                <Box mb={3}>
                    <Text fontSize="xs" fontWeight="bold" color="gray.700" mb={1}>还开着的路</Text>
                    <VStack align="stretch" gap={1}>
                        {others.map(line => {
                            const lastLit = [...(line.milestones || [])].reverse().find((m: any) => m.lit);
                            return (
                                <Box key={line.line} p={2} bg="white" borderRadius="sm" border="1px solid" borderColor="gray.200">
                                    <HStack justify="space-between" align="center">
                                        <Text fontSize="xs" fontWeight="bold" color="gray.800">
                                            {line.title}
                                            {line.depth > 0 && <Text as="span" color="gray.500" fontWeight="normal"> · 里程碑 {line.depth}/{(line.milestones || []).length}</Text>}
                                            {lastLit && <Text as="span" color="gray.500" fontWeight="normal"> · 最近：{lastLit.title}</Text>}
                                        </Text>
                                        <LineStatusBadges line={line} />
                                    </HStack>
                                    {lineNote(line) && <Text fontSize="xs" color="gray.600" mt={1}>{lineNote(line)}</Text>}
                                </Box>
                            );
                        })}
                    </VStack>
                </Box>
            )}

            {/* 未起步的路：一行带过，不展示机制 */}
            {unstarted.length > 0 && (
                <Text fontSize="xs" color="gray.500" mb={3}>
                    尚未起步：{unstarted.map(l => l.title).join(" · ")}——没有脚印的路不占界面。
                </Text>
            )}

            {/* 共享事实带：跨线事实只说一次 */}
            <Box mb={3}>
                <SharedFacts silo={silo} world={world} progress={progress} />
            </Box>

            {/* 改道日志：路变了要看得见 */}
            {shifts.length > 0 && (
                <Box p={2} mb={3} bg="purple.50" borderRadius="sm" border="1px solid" borderColor="purple.200">
                    <Text fontSize="xs" fontWeight="bold" color="purple.800" mb={1}>改道记录</Text>
                    <VStack align="stretch" gap={0.5}>
                        {shifts.map((shift: any, idx: number) => (
                            <Text key={idx} fontSize="xs" color="gray.700">
                                Y{shift.year} M{shift.month} ·{" "}
                                {shift.from ? `${titleOf(shift.from)} → ${titleOf(shift.to)}` : `主线确立：${titleOf(shift.to)}`}
                            </Text>
                        ))}
                    </VStack>
                </Box>
            )}

            {/* 征兆：只报告已发生的事实，不点名结局、不预测 */}
            {omens.length > 0 && (
                <Box p={2} bg="orange.50" borderRadius="sm" border="1px solid" borderColor="orange.300">
                    <Text fontSize="xs" fontWeight="bold" color="orange.800" mb={1}>征兆</Text>
                    <Text fontSize="xs" color="gray.600" mb={1}>
                        征兆只是已经发生的事实：它指向什么，取决于接下来的选择。
                    </Text>
                    <VStack align="stretch" gap={0.5}>
                        {omens.map((omen: any) => (
                            <Text key={omen.key} fontSize="xs" color="gray.700">
                                {omen.title} · Y{omen.year} M{omen.month}{omen.count > 1 ? ` ×${omen.count}` : ""}
                            </Text>
                        ))}
                    </VStack>
                </Box>
            )}
        </Box>
    );
});
