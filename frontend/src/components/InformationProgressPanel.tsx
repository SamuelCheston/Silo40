import { memo } from "react";
import { Box, Badge, Heading, HStack, Text, VStack, SimpleGrid } from "@chakra-ui/react";

// 《共证》线进度面板 (修复3)：链节点点亮状态、部门事实覆盖与公共证据登记。
// 数据全部来自后端权威快照 (information_progress + silo.public_evidence +
// silo.authority.judicial.public_evidence_q)，前端不复算引擎公式。

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
    silo: any;
}

export const InformationProgressPanel = memo(function InformationProgressPanel({ progress, silo }: Props) {
    if (!progress) return null;
    const evidence = silo?.public_evidence || [];
    const q = silo?.authority?.judicial?.public_evidence_q || 0;
    const need = progress.gate_fact_count;

    return (
        <Box w="full" p={4} bg="blue.50" border="1px solid" borderColor="blue.200" borderRadius="md">
            <Heading size="sm" color="blue.800" mb={2}>《共证》进度</Heading>
            <Text fontSize="xs" color="gray.600" mb={3}>
                结局门槛：每个部门持有 {need} 份浓度 ≥ {progress.belief_floor.toFixed(2)} 的事实；情报会淡忘，流通跑赢淡忘才守得住。
            </Text>

            <Text fontSize="xs" fontWeight="bold" color="blue.700" mb={1}>链节点</Text>
            <HStack wrap="wrap" gap={2} mb={3}>
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

            <Text fontSize="xs" fontWeight="bold" color="blue.700" mb={1}>部门事实覆盖（达标事实数）</Text>
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

            <Text fontSize="xs" fontWeight="bold" color="blue.700" mb={1}>公共证据登记</Text>
            <VStack align="stretch" gap={1}>
                {evidence.length === 0 && <Text fontSize="xs" color="gray.500">尚无已登记证据。</Text>}
                {evidence.map((record: any) => (
                    <HStack key={record.fact_id} justify="space-between">
                        <Text fontSize="xs" color="gray.700">{record.fact_id}</Text>
                        <Text fontSize="xs" color="gray.500">
                            来源 {Number(record.provenance).toFixed(1)} · 覆盖 {Number(record.coverage).toFixed(1)}
                        </Text>
                    </HStack>
                ))}
                <Text fontSize="xs" color="gray.600" mt={1}>
                    司法侧可读性 Q：{q.toFixed(2)}{q > 0 ? "（公共事实可被引用）" : "（尚未可读——司法侧信念仍在门槛下）"}
                </Text>
            </VStack>
        </Box>
    );
});
