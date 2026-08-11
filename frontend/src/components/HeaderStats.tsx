import { Box, HStack, Text, VStack, Badge } from "@chakra-ui/react";
import { Agent, Silo } from "../logic/models";
import { Zap, Eye, Star, Megaphone, Network, Users, Clock, Shield, Flame } from 'lucide-react';
import { Tooltip } from "./ui/tooltip";

interface HeaderStatsProps {
    agent: Agent | null;
    silo: Silo | null;
    organizedPopulation: number;
}

export const HeaderStats = ({ agent, silo, organizedPopulation }: HeaderStatsProps) => {
    if (!agent || !silo) return null;

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
            boxShadow="sm"
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
                    <Tooltip content="地堡正统性。受管理决策和全局事件影响。">
                        <HStack gap={1} cursor="help">
                            <Shield size={16} color="#3182ce" />
                            <Text fontSize="sm" fontWeight="bold" color="blue.600">
                                {(silo.legitimacy || 0).toFixed(2)}
                            </Text>
                        </HStack>
                    </Tooltip>
                    
                    <Tooltip content="地堡叛乱度。受组织化人口规模和居民不满情绪驱动。">
                        <HStack gap={1} cursor="help">
                            <Flame size={16} color="#e53e3e" />
                            <Text fontSize="sm" fontWeight="bold" color="red.600">
                                {(silo.rebellion || 0).toFixed(2)}
                            </Text>
                        </HStack>
                    </Tooltip>
                </HStack>

                {/* Agent Stats */}
                <HStack gap={6} wrap="wrap">
                    <Tooltip 
                        content={
                            <VStack align="start" gap={0}>
                                <Text fontWeight="bold">行动点 (AP)</Text>
                                <Text>恢复：10 + (威望 × 0.05) + (组织系数 × 2) /年</Text>
                                <Text>上限：100 + (组织系数 × 10)</Text>
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
                            <VStack align="start" gap={0}>
                                <Text fontWeight="bold">怀疑度</Text>
                                <Text>来源：非法行动、传播虚假信息</Text>
                                <Text>影响：降低行动成功率，增加暴露风险</Text>
                            </VStack>
                        }
                    >
                        <HStack gap={1} cursor="help">
                            <Eye size={16} color={agent.suspicion_level > 0.5 ? "#e53e3e" : "#38a169"} />
                            <Text fontSize="sm" fontWeight="bold" color={agent.suspicion_level > 0.5 ? "red.600" : "green.600"}>
                                {agent.suspicion_level.toFixed(2)}
                            </Text>
                        </HStack>
                    </Tooltip>

                    <Tooltip 
                        content={
                            <VStack align="start" gap={0}>
                                <Text fontWeight="bold">政治威望</Text>
                                <Text>来源：(平均人脉 × 100) × 职业/特质修正</Text>
                                <Text>影响：AP 恢复、政治点获取、煽动效果</Text>
                            </VStack>
                        }
                    >
                        <HStack gap={1} cursor="help">
                            <Star size={16} color="#d69e2e" />
                            <Text fontSize="sm" fontWeight="bold" color="orange.600">
                                {Math.floor(agent.political_prestige)}
                            </Text>
                        </HStack>
                    </Tooltip>

                    <Tooltip 
                        content={
                            <VStack align="start" gap={0}>
                                <Text fontWeight="bold">宣传力度</Text>
                                <Text>来源：执行宣传行动</Text>
                                <Text>影响：提升特工吸引力 (进而增加人口转化率)</Text>
                            </VStack>
                        }
                    >
                        <HStack gap={1} cursor="help">
                            <Megaphone size={16} color="#d53f8c" />
                            <Text fontSize="sm" fontWeight="bold" color="pink.600">
                                {agent.propaganda_level.toFixed(1)}
                            </Text>
                        </HStack>
                    </Tooltip>

                    <Tooltip 
                        content={
                            <VStack align="start" gap={0}>
                                <Text fontWeight="bold">组织度系数</Text>
                                <Text>影响：AP 恢复速度、AP 上限、人口转化效率</Text>
                            </VStack>
                        }
                    >
                        <HStack gap={1} cursor="help">
                            <Network size={16} color="#319795" />
                            <Text fontSize="sm" fontWeight="bold" color="teal.600">
                                {agent.organization_factor.toFixed(1)}x
                            </Text>
                        </HStack>
                    </Tooltip>

                    <Tooltip 
                        content={
                            <VStack align="start" gap={0}>
                                <Text fontWeight="bold">组织化人口</Text>
                                <Text>转化率：(吸引力 × 0.4 + 人脉 × 0.6) × 组织系数 × 思潮</Text>
                                <Text>目标：达到总人口 3% 触发叛乱胜利</Text>
                            </VStack>
                        }
                    >
                        <HStack gap={1} cursor="help">
                            <Users size={16} color="#00b5d8" />
                            <Text fontSize="sm" fontWeight="bold" color="cyan.700">
                                {organizedPopulation} / {silo.total_population}
                            </Text>
                        </HStack>
                    </Tooltip>
                </HStack>

                {/* Identity */}
                <HStack gap={2}>
                    <Text fontSize="xs" color="gray.500">{agent.name}</Text>
                    <Badge colorPalette="purple" size="sm">{agent.profession}</Badge>
                </HStack>
            </HStack>
        </Box>
    );
};
