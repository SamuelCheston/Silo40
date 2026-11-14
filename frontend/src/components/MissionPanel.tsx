import { memo } from "react";
import { Box, Heading, Text, VStack, HStack, Badge } from "@chakra-ui/react";

// 开局自解释目标卡：把《见证》教义给特工的目标与完成判据翻译成界面语言。
// 不做教程弹窗——目标、判据与"现在走到哪了"直接长在常驻界面里。
// 全部读数来自既有 state 通道：world.progress_stage、agent.known_fragments、
// information_progress（链节点、部门覆盖、终止链读数）。

interface Props {
    world: any;
    agent: any;
    progress: any;
}

// 阶段是后端权威进度 (action_gates.json)，这里只翻译成人话并指出下一步。
const stageBriefing: Record<string, { label: string; brief: string }> = {
    arrival: {
        label: "初到驻地",
        brief: "没有人认识你，你手里也还没有能独立成立的事实。第一步：用「Gather Info」取得第一份可核验的材料。",
    },
    first_signal: {
        label: "第一次本地核验",
        brief: "你拿到了能独立成立的材料。「Share Info」已解锁——把事实交给能独立查账的人，让它在你的叙述之外成立。",
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

export const MissionPanel = memo(function MissionPanel({ world, agent, progress }: Props) {
    const stage = world?.progress_stage || "arrival";
    const briefing = stageBriefing[stage] || stageBriefing.arrival;

    const knownCount = agent?.known_fragments?.length || 0;
    const absorbedTotal = (progress?.departments || []).reduce(
        (sum: number, dept: any) => sum + (dept.above_floor || 0), 0);

    const nodeLit = (key: string) => (progress?.nodes || []).some((n: any) => n.key === key && n.lit);
    const replicationCount = (progress?.nodes || []).find(
        (n: any) => n.key === "special:replication_progress")?.count || 0;
    const infected = nodeLit("player_actions:receiver_verify_accompany");
    const replicating = nodeLit("special:replication_progress");
    const living = nodeLit("special:living_chain_confirmed");

    // 判据一：事实脱离自述
    let factLine: string;
    if (knownCount === 0) {
        factLine = "你还没有任何可核验的材料——「Gather Info」是起点。";
    } else if (absorbedTotal === 0) {
        factLine = `在手材料 ${knownCount} 份。它们还只是你的私人确信，尚未进入任何部门的账本。`;
    } else {
        factLine = `在手材料 ${knownCount} 份 · ${absorbedTotal} 份事实已进入部门信念层，开始不依赖你而存在。`;
    }

    // 判据二：本地承接者
    let receiverLine: string;
    if (living) {
        receiverLine = "活链已成立：承接者们在没有你逐日维持的情况下继续核验与复制。";
    } else if (replicating) {
        receiverLine = `第一位承接者已独立核验，复制正在推进（×${replicationCount}）。`;
    } else if (infected) {
        receiverLine = "第一位承接者已完成独立核验。下一个承接者要由他们自己长出来。";
    } else {
        receiverLine = "还没有人从你手里接过材料并独立核验。你第一次分享的对象，决定承接者是谁。";
    }

    // 判据三：终止链（读数来自 information_progress 的只读下发）
    let chainLine: string;
    let chainTone: "gray" | "orange" | "green" = "gray";
    if (progress?.reset_countermeasure_done) {
        chainLine = "反制已成立：终止链的副本分散在多人手中，单方命令不再好使。";
        chainTone = "green";
    } else if (progress?.reset_omen_seen) {
        chainLine = "先兆已现：终止链正在启动。把反制做成可核验的事实——分发托管是成立之路。";
        chainTone = "orange";
    } else {
        chainLine = "终止链仍在正常运转：1 号可以经本地线路单方重置驻地。它的先兆会以危机的形式出现。";
    }

    return (
        <Box w="full" p={4} bg="gray.50" border="1px solid" borderColor="gray.300" borderRadius="md">
            <Heading size="sm" color="gray.800" mb={2}>任务目标 ·《见证》</Heading>
            <Text fontSize="sm" color="gray.700" mb={3}>
                让 Silo 40 的覆灭与 1 号的控制方式，不再只能由控制者单方叙述：
                使真相能被独立核验、被驻地的人保有——即使你沉默、离开或失忆，事实仍然成立。
            </Text>

            <VStack align="stretch" gap={2} mb={3}>
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
            </VStack>

            <HStack align="start" gap={2} mb={2}>
                <Badge colorPalette="blue" mt={0.5}>{briefing.label}</Badge>
                <Text fontSize="xs" color="gray.700">{briefing.brief}</Text>
            </HStack>

            <Text fontSize="xs" color="gray.500">
                你的身份在几十年前就已预置，纸面自洽——它是盾，不是桥；纸面与现实之间的缝，要靠真实的关系填。
                边界：不以任务之名重建任何总部；你被忘记没关系，目标必须能在你缺席时继续成立。
            </Text>
        </Box>
    );
});
