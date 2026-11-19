import React, { useState, useEffect, useCallback } from 'react';
import { Box, Button, Text, VStack, HStack, Heading, SimpleGrid, Badge, Input } from '@chakra-ui/react';
import { GetLlmStatus, SetLlmContext } from "../../wailsjs/go/main/App";

export interface TraitOption {
    id: string;
    name: string;
    description: string;
    cost: number;
    type: 'AGENT' | 'SILO';
}

export const AVAILABLE_TRAITS: TraitOption[] = [
    { id: 'shadowy', name: '隐秘行事 (Shadowy)', description: '解锁专属动作「无声探查」：与普通探查同效，但不进入监控抽样、不产生怀疑增量，代价是 2 倍 AP。此外行动产生的怀疑度降低 20%。', cost: 3, type: 'AGENT' },
    { id: 'charismatic', name: '魅力非凡 (Charismatic)', description: '增加初始威望和人脉建立速度，并在建立组织化力量时获得额外增益。', cost: 4, type: 'AGENT' },
    { id: 'native', name: '地堡土著 (Silo Native)', description: '同阶层免引荐：以关系阈值为门的动作与接触卡，目标部门与你同阶层时关系门视为已满足。此外开局获得自己所在阶层各部门随机 10-15% 的人脉值。', cost: 3, type: 'AGENT' },
    { id: 'abundant', name: '资源充沛 (Abundant Resources)', description: '初始地堡食物与能源资源翻倍。', cost: 3, type: 'SILO' },
    { id: 'leak', name: '信息泄漏 (Information Leak)', description: '平民阶层初始亲外度 (Pro-Foreign) 增加。', cost: 2, type: 'SILO' },
    { id: 'psychoactive_meds', name: '精神药物常态化投放 (Psychoactive Meds)', description: '所有部门的思潮会随时间推移缓慢偏向 IT 部门的思潮。', cost: 3, type: 'SILO' },
];

export const AVAILABLE_DEPARTMENTS = [
    { id: 'Mayor', name: 'Mayor', description: '身份醒目，部分行动更容易积累怀疑度。开局威望高，具备各阶层人脉。' },
    { id: 'Judicial', name: 'Judicial', description: '可签发搜查令和执行逮捕。开局威望低，具备各阶层人脉。' },
    // 描述必须与实际开局一致：玩家知识上限由 limitPlayerKnowledge 统一为
    // 本部门 2 份碎片，"掌握全碎片"是旧文案，会误导职业选择。
    { id: 'IT', name: 'IT', description: '部分行动留下的怀疑度较低。掩盖记录会增加 Safeguard 风险。开局与司法部满人脉，与本部门有 40% 人脉。' },
    { id: 'Police', name: 'Police', description: '部分行动留下的怀疑度较低。开局与市长和本部门满人脉，其余精英 15%、平民 10%。' },
    { id: 'Medical', name: 'Medical', description: '开局极低威望，获得精英15%、平民10%人脉。随时间推移会随机获得信息碎片。' },
    { id: 'Supply', name: 'Supply', description: '可调拨库存，或利用供应网络安抚部门并建立人脉。开局与其他平民部门有 15% 人脉。' },
    { id: 'Mechanical', name: 'Mechanical', description: '可检修设备、补充能源与材料，也能通过管道打听消息。开局与其他平民部门有 15% 人脉。' },
    { id: 'Mines', name: 'Mines', description: '开局威望低。常规行动可避开身份采样，与其他平民部门有 15% 人脉。' },
    { id: 'Agricultural', name: 'Agricultural', description: '可组织集中收割补充库存，也能在田间打听消息。开局与其他平民部门有 15% 人脉。' },
];

interface SetupPanelProps {
    onComplete: (selectedTraitIds: string[], profession: string) => void;
}

// ============ LLM 顾问上下文（可选） ============
// 参与度决定"什么时候问模型"，上下文档位决定"问的时候给多少料"，两者正交。
// 未启用 LLM（SILO40_LLM_MODE=off，默认）时整区只读：设置只在启用后生效。

const CONTEXT_LEVELS = [
    { id: 'minimal', name: '极简', desc: '仅当前 NPC、候选动作、输出契约。最省 token。' },
    { id: 'standard', name: '标准', desc: '极简 + 地堡大局指标、相关派系人物、近期事件摘要。' },
    { id: 'rich', name: '丰富', desc: '标准 + 本堡主线经过（已发生的主线事件）。' },
    { id: 'world', name: '世界观', desc: '丰富 + silo 设定、其他地堡关系、长期剧情线。' },
];

const CONTEXT_SWITCHES = [
    { id: 'personal_memory', name: '个人记忆', desc: '该 NPC 的特质、已知信息碎片与人脉。关闭后各档都只给基础状态。' },
    { id: 'protocol_clauses', name: '条文法规', desc: 'Order 条款清单。关闭时模型看不到条款原文。' },
    { id: 'peer_silo_intel', name: '其他地堡情报', desc: '邻堡的谣言档动态（只给档位不给精确值）。默认关闭。' },
];

const LlmContextSection: React.FC = () => {
    const [enabled, setEnabled] = useState(false);
    const [mode, setMode] = useState('off');
    const [participation, setParticipation] = useState('off');
    const [level, setLevel] = useState('standard');
    const [personalMemory, setPersonalMemory] = useState(true);
    const [protocolClauses, setProtocolClauses] = useState(false);
    const [peerIntel, setPeerIntel] = useState(false);
    const [eventLimit, setEventLimit] = useState(5);
    const [feedback, setFeedback] = useState('');
    const [busy, setBusy] = useState(false);

    useEffect(() => {
        GetLlmStatus().then(st => {
            setEnabled(st.enabled);
            setMode(st.mode || 'off');
            setParticipation(st.level || 'off');
            const ctx = st.context;
            if (ctx) {
                setLevel(ctx.level || 'standard');
                setPersonalMemory(!!ctx.personal_memory);
                setProtocolClauses(!!ctx.protocol_clauses);
                setPeerIntel(!!ctx.peer_silo_intel);
                setEventLimit(typeof ctx.event_summary_limit === 'number' ? ctx.event_summary_limit : 5);
            }
        }).catch(() => setFeedback('LLM 状态读取失败。'));
    }, []);

    const apply = useCallback(async () => {
        setBusy(true);
        try {
            const applied = await SetLlmContext({
                level,
                personal_memory: personalMemory,
                protocol_clauses: protocolClauses,
                peer_silo_intel: peerIntel,
                event_summary_limit: eventLimit,
            });
            setFeedback(`已应用：${CONTEXT_LEVELS.find(l => l.id === applied.level)?.name || applied.level} 档。`);
        } catch (err) {
            setFeedback(`应用失败：${err}`);
        } finally {
            setBusy(false);
        }
    }, [level, personalMemory, protocolClauses, peerIntel, eventLimit]);

    const switchOf = (id: string): [boolean, (v: boolean) => void] => {
        if (id === 'personal_memory') return [personalMemory, setPersonalMemory];
        if (id === 'protocol_clauses') return [protocolClauses, setProtocolClauses];
        return [peerIntel, setPeerIntel];
    };

    return (
        <Box w="full" p={4} bg="gray.50" borderRadius="md" border="1px solid" borderColor="gray.200">
            <HStack justify="space-between" w="full" mb={2}>
                <Heading size="sm" color="gray.700">LLM 顾问上下文（可选）</Heading>
                <Badge colorPalette={enabled ? "green" : "gray"}>
                    {enabled ? `已启用 · 模式 ${mode} · 参与度 ${participation}` : '未启用'}
                </Badge>
            </HStack>
            <Text fontSize="xs" color="gray.500" mb={3}>
                参与度决定「什么时候问模型」，上下文档位决定「问的时候给多少料」，两者互不影响。
                {!enabled && ' 当前未启用 LLM（环境变量 SILO40_LLM_MODE=off）：以下设置仅在启用后生效。'}
            </Text>

            <Text fontSize="sm" fontWeight="bold" color="gray.700" mb={2}>上下文档位</Text>
            <SimpleGrid columns={{ base: 1, md: 2 }} gap={3} w="full" mb={4}>
                {CONTEXT_LEVELS.map(lv => {
                    const isSelected = level === lv.id;
                    return (
                        <Box
                            key={lv.id}
                            p={3}
                            borderWidth="2px"
                            borderRadius="md"
                            cursor={enabled ? "pointer" : "not-allowed"}
                            borderColor={isSelected ? "purple.500" : "gray.200"}
                            bg={isSelected ? "purple.50" : "white"}
                            opacity={enabled ? 1 : 0.6}
                            onClick={() => enabled && setLevel(lv.id)}
                            transition="all 0.2s"
                            _hover={enabled ? { borderColor: "purple.300" } : {}}
                        >
                            <Text fontWeight="bold" color={isSelected ? "purple.800" : "gray.700"} fontSize="sm">{lv.name}</Text>
                            <Text fontSize="xs" color="gray.500">{lv.desc}</Text>
                        </Box>
                    );
                })}
            </SimpleGrid>

            <Text fontSize="sm" fontWeight="bold" color="gray.700" mb={2}>独立开关（与档位正交）</Text>
            <VStack align="stretch" gap={2} w="full">
                {CONTEXT_SWITCHES.map(sw => {
                    const [on, setOn] = switchOf(sw.id);
                    return (
                        <HStack
                            key={sw.id}
                            justify="space-between"
                            p={3}
                            bg="white"
                            borderRadius="md"
                            border="1px solid"
                            borderColor="gray.200"
                            cursor={enabled ? "pointer" : "not-allowed"}
                            opacity={enabled ? 1 : 0.6}
                            onClick={() => enabled && setOn(!on)}
                        >
                            <Box flex={1}>
                                <Text fontWeight="bold" fontSize="sm" color="gray.700">{sw.name}</Text>
                                <Text fontSize="xs" color="gray.500">{sw.desc}</Text>
                            </Box>
                            <Badge colorPalette={on ? "green" : "gray"}>{on ? '开' : '关'}</Badge>
                        </HStack>
                    );
                })}
                <HStack justify="space-between" p={3} bg="white" borderRadius="md" border="1px solid" borderColor="gray.200">
                    <Box flex={1}>
                        <Text fontWeight="bold" fontSize="sm" color="gray.700">事件摘要条数</Text>
                        <Text fontSize="xs" color="gray.500">标准档及以上携带的近期事件摘要条数；0 表示不带事件摘要。</Text>
                    </Box>
                    <Input
                        type="number"
                        min={0}
                        max={50}
                        w="80px"
                        size="sm"
                        value={eventLimit}
                        disabled={!enabled}
                        onChange={e => {
                            const v = parseInt(e.target.value, 10);
                            setEventLimit(Number.isFinite(v) ? Math.max(0, Math.min(50, v)) : 0);
                        }}
                    />
                </HStack>
            </VStack>

            <HStack w="full" mt={3} gap={3}>
                <Button size="sm" colorPalette="purple" variant="outline" bg="white" onClick={apply} disabled={!enabled || busy}>
                    {busy ? '应用中…' : '应用上下文设置'}
                </Button>
                {feedback && <Text fontSize="xs" color={feedback.startsWith('应用失败') ? "red.600" : "green.600"}>{feedback}</Text>}
            </HStack>
        </Box>
    );
};

export const SetupPanel: React.FC<SetupPanelProps> = ({ onComplete }) => {
    const [selected, setSelected] = useState<string[]>([]);
    const [profession, setProfession] = useState<string>('Mechanical');
    const MAX_POINTS = 10;

    const currentPoints = selected.reduce((acc, id) => {
        const trait = AVAILABLE_TRAITS.find(t => t.id === id);
        return acc + (trait ? trait.cost : 0);
    }, 0);

    const remainingPoints = MAX_POINTS - currentPoints;

    const toggleTrait = (id: string) => {
        if (selected.includes(id)) {
            setSelected(selected.filter(t => t !== id));
        } else {
            const trait = AVAILABLE_TRAITS.find(t => t.id === id);
            if (trait && currentPoints + trait.cost <= MAX_POINTS) {
                setSelected([...selected, id]);
            }
        }
    };

    const agentTraits = AVAILABLE_TRAITS.filter(t => t.type === 'AGENT');
    const siloTraits = AVAILABLE_TRAITS.filter(t => t.type === 'SILO');

    return (
        <VStack gap={6} w="full" p={6} bg="white" borderRadius="lg" boxShadow="sm" border="1px solid" borderColor="gray.200">
            <Heading size="md" color="blue.700">定制你的 Silo 40 特工</Heading>
            <HStack w="full" justify="space-between" bg="blue.50" p={4} borderRadius="md" border="1px solid" borderColor="blue.100">
                <Text fontSize="lg" fontWeight="bold" color="blue.800">
                    可用选择点数 (Selection Points):
                </Text>
                <Badge colorPalette={remainingPoints > 0 ? "blue" : "red"} fontSize="xl" px={3} py={1} borderRadius="full">
                    {remainingPoints} / {MAX_POINTS}
                </Badge>
            </HStack>
            <Text fontSize="xs" color="gray.500" w="full" mt={-3}>
                点数用于购买特质；未花完的点数开局即作废，不会转化为其他优势。
            </Text>

            <Box w="full">
                <Heading size="sm" mb={4} color="gray.700" borderBottom="2px solid" borderColor="gray.100" pb={2}>
                    选择特工所属部门 (Select Agent Profession)
                </Heading>
                <SimpleGrid columns={{ base: 1, md: 2 }} gap={3}>
                    {AVAILABLE_DEPARTMENTS.map(dept => {
                        const isSelected = profession === dept.id;
                        return (
                            <Box 
                                key={dept.id} 
                                p={3} 
                                borderWidth="2px" 
                                borderRadius="md" 
                                cursor="pointer"
                                borderColor={isSelected ? "blue.500" : "gray.200"}
                                bg={isSelected ? "blue.50" : "white"}
                                onClick={() => setProfession(dept.id)}
                                transition="all 0.2s"
                                _hover={{ borderColor: isSelected ? "blue.500" : "blue.300", transform: "translateY(-1px)" }}
                            >
                                <Text fontWeight="bold" color={isSelected ? "blue.800" : "gray.700"} fontSize="sm" mb={1}>
                                    {dept.name}
                                </Text>
                                <Text fontSize="xs" color="gray.500">
                                    {dept.description}
                                </Text>
                            </Box>
                        );
                    })}
                </SimpleGrid>
            </Box>

            <Box w="full">
                <Heading size="sm" mb={4} color="gray.700" borderBottom="2px solid" borderColor="gray.100" pb={2}>
                    特工特质 (Agent Traits)
                </Heading>
                <SimpleGrid columns={{ base: 1, md: 2 }} gap={4}>
                    {agentTraits.map(trait => {
                        const isSelected = selected.includes(trait.id);
                        const canSelect = isSelected || currentPoints + trait.cost <= MAX_POINTS;
                        return (
                            <Box 
                                key={trait.id} 
                                p={4} 
                                borderWidth="2px" 
                                borderRadius="md" 
                                cursor={canSelect ? "pointer" : "not-allowed"}
                                borderColor={isSelected ? "blue.500" : "gray.200"}
                                bg={isSelected ? "blue.50" : (canSelect ? "white" : "gray.50")}
                                opacity={canSelect ? 1 : 0.6}
                                onClick={() => canSelect && toggleTrait(trait.id)}
                                transition="all 0.2s"
                                _hover={canSelect ? { borderColor: "blue.400", transform: "translateY(-2px)" } : {}}
                            >
                                <HStack justify="space-between" mb={2}>
                                    <Text fontWeight="bold" color="gray.800">{trait.name}</Text>
                                    <Badge colorPalette={trait.cost > 0 ? "purple" : "green"}>
                                        {trait.cost > 0 ? `Cost: ${trait.cost}` : `Gain: ${-trait.cost}`}
                                    </Badge>
                                </HStack>
                                <Text fontSize="sm" color="gray.600">{trait.description}</Text>
                            </Box>
                        );
                    })}
                </SimpleGrid>
            </Box>

            <Box w="full">
                <Heading size="sm" mb={4} color="gray.700" borderBottom="2px solid" borderColor="gray.100" pb={2}>
                    地堡初始状况 (Silo Conditions)
                </Heading>
                <SimpleGrid columns={{ base: 1, md: 2 }} gap={4}>
                    {siloTraits.map(trait => {
                        const isSelected = selected.includes(trait.id);
                        const canSelect = isSelected || currentPoints + trait.cost <= MAX_POINTS;
                        return (
                            <Box 
                                key={trait.id} 
                                p={4} 
                                borderWidth="2px" 
                                borderRadius="md" 
                                cursor={canSelect ? "pointer" : "not-allowed"}
                                borderColor={isSelected ? "green.500" : "gray.200"}
                                bg={isSelected ? "green.50" : (canSelect ? "white" : "gray.50")}
                                opacity={canSelect ? 1 : 0.6}
                                onClick={() => canSelect && toggleTrait(trait.id)}
                                transition="all 0.2s"
                                _hover={canSelect ? { borderColor: "green.400", transform: "translateY(-2px)" } : {}}
                            >
                                <HStack justify="space-between" mb={2}>
                                    <Text fontWeight="bold" color="gray.800">{trait.name}</Text>
                                    <Badge colorPalette={trait.cost > 0 ? "purple" : "green"}>
                                        {trait.cost > 0 ? `Cost: ${trait.cost}` : `Gain: ${-trait.cost}`}
                                    </Badge>
                                </HStack>
                                <Text fontSize="sm" color="gray.600">{trait.description}</Text>
                            </Box>
                        );
                    })}
                </SimpleGrid>
            </Box>

            <LlmContextSection />

            <Button 
                colorPalette="blue" 
                size="lg" 
                w="full" 
                mt={4} 
                onClick={() => onComplete(selected, profession)}
            >
                确认选择并开始游戏 (Confirm & Start Game)
            </Button>
        </VStack>
    );
};
