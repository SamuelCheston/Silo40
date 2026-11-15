import { memo } from "react";
import { Box, Badge, Heading, HStack, Text, SimpleGrid } from "@chakra-ui/react";

// 《共证》线进度面板 (修复3)：链节点点亮状态与部门事实覆盖。
// 数据全部来自后端权威快照 (information_progress)，前端不复算引擎公式。
// 线信息架构起，登记簿与 Q 是跨线共享事实，只在 LinesPanel 的共享事实带
// 说一次，本面板不再重复。

interface ChainNode {
    key: string;
    title: string;
    lit: boolean;
    count?: number;
}

interface DepartmentCoverage {
    name: string;
    above_floor: number;
}

interface InformationProgress {
    nodes: ChainNode[];
    departments: DepartmentCoverage[];
    gate_fact_count: number;
    belief_floor: number;
}

interface Props {
    progress: InformationProgress | null;
    // 结局已定：进度定格，不再给"下一步"指引。
    finished?: boolean;
}

// 链节点的点亮条件说明：空状态与"下一步"都靠它，避免面板只剩裸 0/3。
const nodeHint: Record<string, string> = {
    "player_actions:receiver_verify_accompany": "点亮条件：第一位接收者独立完成对你所分享事实的核验——陪同 TA 完成回查。",
    "special:replication_progress": "点亮条件：接收者开始把核验能力传下去——链每推进一环，这里跳动一次。",
    "special:independent_receivers_met": "点亮条件：出现不止一位不依赖你逐日授意的接收者。",
    "special:living_chain_confirmed": "点亮条件：感染、复制、隐匿齐备——链不需要你逐日维持也能复制。",
    "special:public_fact_cross_dept": "点亮条件：核心真相成为跨部门、可独立复核的公共事实。",
    "special:info_vigil_eve": "点亮条件：结局临近——剩下的事是守住宿主活到结算。",
};

export const InformationProgressPanel = memo(function InformationProgressPanel({ progress, finished }: Props) {
    if (!progress) return null;
    const need = progress.gate_fact_count;
    const nothingLit = progress.nodes.every(node => !node.lit);
    const absorbedTotal = progress.departments.reduce((sum, dept) => sum + (dept.above_floor || 0), 0);
    const freshStart = nothingLit && absorbedTotal === 0;
    const nextNode = progress.nodes.find(node => !node.lit);

    return (
        <Box w="full" p={3} bg="blue.50" border="1px solid" borderColor="blue.200" borderRadius="md">
            <Heading size="xs" color="blue.800" mb={2}>《共证》进度</Heading>
            <Text fontSize="xs" color="gray.600" mb={3}>
                结局门槛：每个部门持有 {need} 份浓度 ≥ {progress.belief_floor.toFixed(2)} 的事实；情报会淡忘，流通跑赢淡忘才守得住。
            </Text>

            {freshStart && (
                <Box p={2} mb={3} bg="white" borderRadius="sm" border="1px dashed" borderColor="blue.300">
                    <Text fontSize="xs" color="gray.700">
                        链还没有开始。路径：用「Gather Info」取得可核验的材料（确信 B ≥ {progress.belief_floor.toFixed(2)} 才可分享）→
                        用「Share Info」把它交给能独立查账的人 → 对方吸收后，本面板开始点亮。
                    </Text>
                </Box>
            )}

            <Text fontSize="xs" fontWeight="bold" color="blue.700" mb={1}>链节点</Text>
            <HStack wrap="wrap" gap={2} mb={1}>
                {progress.nodes.map(node => (
                    <Badge
                        key={node.key}
                        colorPalette={node.lit ? "green" : "gray"}
                        variant={node.lit ? "solid" : "outline"}
                    >
                        {node.title}{node.count ? ` ×${node.count}` : ""}
                    </Badge>
                ))}
            </HStack>
            {finished && (
                <Text fontSize="xs" color="gray.600" mb={3}>
                    本局已结算——链的状态在此定格，不再有下一步。
                </Text>
            )}
            {!finished && nextNode && (
                <Text fontSize="xs" color="gray.600" mb={3}>
                    下一步 ·「{nextNode.title}」：{nodeHint[nextNode.key] || "继续推进《共证》线。"}
                </Text>
            )}
            {!nextNode && <Box mb={3} />}

            <Text fontSize="xs" fontWeight="bold" color="blue.700" mb={1}>部门事实覆盖（达标事实数）</Text>
            {absorbedTotal === 0 && (
                <Text fontSize="xs" color="gray.600" mb={1}>
                    还没有任何部门开始持有事实——第一次「Share Info」被吸收后，这里出现第一个计数。
                </Text>
            )}
            <SimpleGrid columns={3} gap={2} mb={3}>
                {progress.departments.map(dept => {
                    const done = dept.above_floor >= need;
                    return (
                        <Box key={dept.name} p={1.5} bg="white" borderRadius="sm" border="1px solid"
                            borderColor={done ? "green.300" : "gray.200"}>
                            <Text fontSize="xs" fontWeight="bold" color={done ? "green.700" : "gray.700"}>
                                {dept.name}
                            </Text>
                            <Text fontSize="xs" color={done ? "green.600" : "gray.500"}>
                                {Math.min(dept.above_floor, need)}/{need}{dept.above_floor > need ? ` (+${dept.above_floor - need})` : ""}
                            </Text>
                        </Box>
                    );
                })}
            </SimpleGrid>
        </Box>
    );
});
