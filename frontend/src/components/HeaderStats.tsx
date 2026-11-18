import { memo } from "react";
import { Box, HStack, Text, VStack, Badge } from "@chakra-ui/react";
import { Agent, AgentStats, Silo } from "../logic/models";
import { Zap, Eye, Star, Megaphone, Users, Clock, Shield, Flame } from 'lucide-react';
import { Tooltip } from "./ui/tooltip";
import { ResourcePanel } from "./ResourcePanel";

interface HeaderStatsProps {
    agent: Agent | null;
    silo: Silo | null;
    agentStats: AgentStats | null;
    // 最近一次结算的死因归因行（后端月度日志），挂在人口读数上。
    deathNote?: string | null;
}

export const HeaderStats = memo(function HeaderStats({ agent, silo, agentStats, deathNote }: HeaderStatsProps) {
    if (!agent || !silo || !agentStats) return null;

    return (
        <Box 
            w="full" 
            bg="white" 
            borderBottom="1px solid" 
            borderColor="gray.200" 
            py={2} 
            px={6} 
            position="sticky" 
            top={0} 
            zIndex={100}
        >
            <HStack justify="space-between" wrap="wrap" gap={4}>
                {/* Time & Silo Info */}
                <HStack gap={4}>
                    <HStack gap={1}>
                        <Clock size={16} color="#4A5568" />
                        <Text fontSize="sm" fontWeight="bold">
                            Year {silo.current_year} Month {silo.current_month}
                        </Text>
                    </HStack>
                    <Badge colorPalette="blue" variant="subtle">{silo.name}</Badge>
                </HStack>

                {/* Global Stats */}
                <HStack gap={4}>
                    <Tooltip content="市长的正统性。随管理决策与事件变化，也会受叛乱侵蚀。">
                        <HStack gap={1} cursor="help">
                            <Shield size={16} color="#3182ce" />
                            <Text fontSize="sm" fontWeight="bold" color="blue.600">
                                {(silo.legitimacy || 0).toFixed(2)}
                            </Text>
                        </HStack>
                    </Tooltip>
                    
                    <Tooltip content="地堡叛乱度。受居民不满情绪和激进阵营影响。">
                        <HStack gap={1} cursor="help">
                            <Flame size={16} color="#e53e3e" />
                            <Text fontSize="sm" fontWeight="bold" color="red.600">
                                {(silo.rebellion || 0).toFixed(2)}
                            </Text>
                        </HStack>
                    </Tooltip>

                    <Tooltip
                        content={
                            deathNote ? (
                                <VStack align="start" gap={1} p={1}>
                                    <Text fontWeight="bold" borderBottom="1px solid" borderColor="whiteAlpha.300" w="full" pb={1}>地堡总人口</Text>
                                    <Text color="red.300">{deathNote}</Text>
                                    <Text fontSize="10px" color="gray.400">死因按死亡率构成逐项拆分；库存耗尽、结构失稳与高叛乱都会抬升死亡。</Text>
                                </VStack>
                            ) : (
                                "地堡总人口。"
                            )
                        }
                    >
                        <HStack gap={1} cursor="help">
                            <Users size={16} color={deathNote ? "#e53e3e" : "#38a169"} />
                            <Text fontSize="sm" fontWeight="bold" color={deathNote ? "red.600" : "green.600"}>
                                {silo.total_population}
                            </Text>
                        </HStack>
                    </Tooltip>

                    <ResourcePanel silo={silo} />
                </HStack>

                {/* Agent Stats */}
                <HStack gap={6} wrap="wrap">
                    <Tooltip 
                        content={
                            <VStack align="start" gap={1} p={1}>
                                <Text fontWeight="bold" borderBottom="1px solid" borderColor="whiteAlpha.300" w="full" pb={1}>行动点 (AP)</Text>
                                <HStack justify="space-between" w="full">
                                    <Text>基础恢复/年:</Text>
                                    <Text color="green.300">+{agentStats.ap_base_recovery.toFixed(2)}</Text>
                                </HStack>
                                <HStack justify="space-between" w="full">
                                    <Text>威望加成/年:</Text>
                                    <Text color="green.300">+{agentStats.ap_prestige_bonus.toFixed(2)}</Text>
                                </HStack>
                                <Text fontWeight="bold" pt={1} color="blue.300">年度总恢复: {agentStats.ap_total_recovery.toFixed(2)} AP</Text>
                                <Text fontSize="10px" color="gray.400" mt={2} borderTop="1px solid" borderColor="whiteAlpha.200" pt={1}>
                                    上限: {agentStats.ap_max.toFixed(0)}
                                </Text>
                            </VStack>
                        }
                    >
                        <HStack gap={1} cursor="help">
                            <Zap size={16} color="#3182ce" />
                            <Text fontSize="sm" fontWeight="bold" color="blue.600">
                                {Math.floor(agent.action_points)} AP
                            </Text>
                        </HStack>
                    </Tooltip>

                    <Tooltip 
                        content={
                            <VStack align="start" gap={1} p={1}>
                                <Text fontWeight="bold" borderBottom="1px solid" borderColor="whiteAlpha.300" w="full" pb={1}>怀疑度</Text>
                                <Text>当前等级: {agent.suspicion_level.toFixed(2)}</Text>
                                <Text fontSize="10px" color="gray.400">达到 1.00 会触发身份暴露结局。怀疑度随时间下降，每年减少 0.05。</Text>
                                <VStack align="start" gap={0} mt={1}>
                                    <Text fontSize="10px">累积来源</Text>
                                    <Text fontSize="10px" color="red.300">· 留下痕迹的探查</Text>
                                    <Text fontSize="10px" color="red.300">· 涉及怀疑度的职业行动与行动反查</Text>
                                </VStack>
                            </VStack>
                        }
                    >
                        <HStack gap={1} cursor="help">
                            <Eye size={16} color={agent.suspicion_level >= 1 ? "#e53e3e" : "#38a169"} />
                            <Text fontSize="sm" fontWeight="bold" color={agent.suspicion_level >= 1 ? "red.600" : "green.600"}>
                                {agent.suspicion_level.toFixed(2)}
                            </Text>
                        </HStack>
                    </Tooltip>

                    <Tooltip 
                        content={
                            <VStack align="start" gap={1} p={1}>
                                <Text fontWeight="bold" borderBottom="1px solid" borderColor="whiteAlpha.300" w="full" pb={1}>政治威望</Text>
                                <HStack justify="space-between" w="full">
                                    <Text>野心 ×50:</Text>
                                    <Text>{(agentStats.ambition * 50).toFixed(1)}</Text>
                                </HStack>
                                <HStack justify="space-between" w="full">
                                    <Text>影响力 ×50:</Text>
                                    <Text>{(agentStats.influence * 50).toFixed(1)}</Text>
                                </HStack>
                                <Text fontSize="10px" color="gray.400">威望 = 50×(野心+影响力)。野心开局定死，仅事件可改；影响力随时间缓慢增长。</Text>
                                <Text fontWeight="bold" pt={1} color="orange.300">最终威望值: {Math.floor(agentStats.political_prestige)}</Text>
                            </VStack>
                        }
                    >
                        <HStack gap={1} cursor="help">
                            <Star size={16} color="#d69e2e" />
                            <Text fontSize="sm" fontWeight="bold" color="orange.600">
                                {Math.floor(agentStats.political_prestige)}
                            </Text>
                        </HStack>
                    </Tooltip>

                    <Tooltip 
                        content={
                            <VStack align="start" gap={1} p={1}>
                                <Text fontWeight="bold" borderBottom="1px solid" borderColor="whiteAlpha.300" w="full" pb={1}>宣传力度</Text>
                                <Text>当前等级: {agentStats.propaganda_level.toFixed(1)}</Text>
                                <Text fontSize="10px" color="gray.400">会放大 `Incite Rebellion` 的最终效果。</Text>
                                <Text fontSize="10px" color="blue.300" mt={1}>宣传倍率: ×{agentStats.propaganda_multiplier.toFixed(2)}</Text>
                                <Text fontSize="10px" color="blue.300">叛乱基础强度: {agentStats.rebellion_base_effect.toFixed(3)}</Text>
                            </VStack>
                        }
                    >
                        <HStack gap={1} cursor="help">
                            <Megaphone size={16} color="#d53f8c" />
                            <Text fontSize="sm" fontWeight="bold" color="pink.600">
                                {agentStats.propaganda_level.toFixed(1)}
                            </Text>
                        </HStack>
                    </Tooltip>
                </HStack>

                {/* Identity */}
                <Tooltip 
                    content={
                        <VStack align="start" gap={2}>
                            <Box>
                                <Text fontWeight="bold" fontSize="xs" mb={1}>特质 (Traits)</Text>
                                <HStack wrap="wrap" gap={1}>
                                    {agent.traits?.map(t => (
                                        <Badge key={t} colorPalette="yellow" size="xs">{t}</Badge>
                                    ))}
                                </HStack>
                            </Box>
                            <Box>
                                <Text fontWeight="bold" fontSize="xs" mb={1}>已知碎片 (Known Fragments)</Text>
                                <HStack wrap="wrap" gap={1}>
                                    {agent.known_fragments?.map(f => (
                                        <Badge key={f} colorPalette="purple" size="xs" variant="solid">{f}</Badge>
                                    ))}
                                </HStack>
                            </Box>
                        </VStack>
                    }
                >
                    <HStack gap={2} cursor="help">
                        <Text fontSize="xs" color="gray.500">{agent.name}</Text>
                        <Badge colorPalette="purple" size="sm">{agent.profession}</Badge>
                    </HStack>
                </Tooltip>
            </HStack>
        </Box>
    );
});
