import { Box, HStack, Text, VStack, Badge } from "@chakra-ui/react";
import { Agent, Silo } from "../logic/models";
import { Zap, Eye, Star, Megaphone, Network, Users, Clock, Shield, Flame } from 'lucide-react';

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
                    <HStack gap={1} title="Legitimacy">
                        <Shield size={16} color="#3182ce" />
                        <Text fontSize="sm" fontWeight="bold" color="blue.600">
                            {(silo.legitimacy || 0).toFixed(2)}
                        </Text>
                    </HStack>
                    <HStack gap={1} title="Rebellion">
                        <Flame size={16} color="#e53e3e" />
                        <Text fontSize="sm" fontWeight="bold" color="red.600">
                            {(silo.rebellion || 0).toFixed(2)}
                        </Text>
                    </HStack>
                </HStack>

                {/* Agent Stats */}
                <HStack gap={6} wrap="wrap">
                    <HStack gap={1} title="Action Points (AP)">
                        <Zap size={16} color="#3182ce" />
                        <Text fontSize="sm" fontWeight="bold" color="blue.600">
                            {Math.floor(agent.action_points)} AP
                        </Text>
                    </HStack>

                    <HStack gap={1} title="Suspicion Level">
                        <Eye size={16} color={agent.suspicion_level > 0.5 ? "#e53e3e" : "#38a169"} />
                        <Text fontSize="sm" fontWeight="bold" color={agent.suspicion_level > 0.5 ? "red.600" : "green.600"}>
                            {agent.suspicion_level.toFixed(2)}
                        </Text>
                    </HStack>

                    <HStack gap={1} title="Political Prestige">
                        <Star size={16} color="#d69e2e" />
                        <Text fontSize="sm" fontWeight="bold" color="orange.600">
                            {Math.floor(agent.political_prestige)}
                        </Text>
                    </HStack>

                    <HStack gap={1} title="Propaganda Level">
                        <Megaphone size={16} color="#d53f8c" />
                        <Text fontSize="sm" fontWeight="bold" color="pink.600">
                            {agent.propaganda_level.toFixed(1)}
                        </Text>
                    </HStack>

                    <HStack gap={1} title="Organization Factor">
                        <Network size={16} color="#319795" />
                        <Text fontSize="sm" fontWeight="bold" color="teal.600">
                            {agent.organization_factor.toFixed(1)}x
                        </Text>
                    </HStack>

                    <HStack gap={1} title="Organized Population">
                        <Users size={16} color="#00b5d8" />
                        <Text fontSize="sm" fontWeight="bold" color="cyan.700">
                            {organizedPopulation} / {silo.total_population}
                        </Text>
                    </HStack>
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
