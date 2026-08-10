// @ts-nocheck
import React from 'react';
import {
    Box,
    VStack,
    HStack,
    Text,
    Heading,
    Badge,
    SimpleGrid,
    Icon,
    Separator,
    Collapsible,
} from "@chakra-ui/react";
import { Users, Zap, Target, ShieldCheck, ChevronDown, ChevronUp } from "lucide-react";
import { Silo, Faction, PopulationCohort } from '../logic/models';
import { ProgressBar, ProgressRoot } from './ui/progress';

interface FactionViewProps {
    silo: Silo;
}

export const FactionView: React.FC<FactionViewProps> = ({ silo }) => {
    const factions = silo.factions || [];
    const cohorts = silo.cohorts || [];

    if (factions.length === 0) {
        return (
            <Box p={8} textAlign="center">
                <Text color="gray.500">暂无活跃阵营信息。</Text>
            </Box>
        );
    }

    return (
        <VStack gap={4} align="stretch" w="full">
            {factions.map(faction => (
                <FactionCard key={faction.id} faction={faction} cohorts={cohorts} />
            ))}
        </VStack>
    );
};

interface FactionCardProps {
    faction: Faction;
    cohorts: PopulationCohort[];
}

const FactionCard: React.FC<FactionCardProps> = ({ faction, cohorts }) => {
    const [isOpen, setIsOpen] = React.useState(false);
    const factionCohorts = cohorts.filter(c => c.faction_id === faction.id);

    // 计算主要意识形态标签的颜色
    const getTagColor = (tag: string) => {
        if (tag.includes('极端')) return 'red';
        if (tag.includes('中立')) return 'gray';
        if (tag.includes('民主')) return 'blue';
        if (tag.includes('排外')) return 'orange';
        if (tag.includes('亲外')) return 'teal';
        return 'purple';
    };

    return (
        <Box
            bg="white"
            p={4}
            borderRadius="md"
            border="1px solid"
            borderColor="gray.200"
            boxShadow="sm"
            _hover={{ borderColor: "blue.300", boxShadow: "md" }}
            transition="all 0.2s"
        >
            <VStack align="stretch" gap={3}>
                <HStack justify="space-between">
                    <HStack gap={3}>
                        <Icon as={Users} color="blue.500" />
                        <Heading size="sm" color="gray.800">{faction.name}</Heading>
                        <Text fontSize="xs" color="gray.400" fontStyle="italic">#{faction.signature}</Text>
                    </HStack>
                    <Badge colorPalette="blue" variant="outline">
                        {faction.member_count} 人
                    </Badge>
                </HStack>

                <HStack wrap="wrap" gap={2}>
                    {faction.tags.map(tag => (
                        <Badge key={tag} colorPalette={getTagColor(tag)} variant="subtle" size="sm">
                            {tag}
                        </Badge>
                    ))}
                </HStack>

                <SimpleGrid columns={2} gap={4} py={2}>
                    <VStack align="start" gap={1}>
                        <HStack w="full" justify="space-between">
                            <HStack gap={1}>
                                <Icon as={Zap} size={12} color="orange.500" />
                                <Text fontSize="xs" color="gray.500">影响力</Text>
                            </HStack>
                            <Text fontSize="xs" fontWeight="bold">{(faction.influence * 100).toFixed(0)}%</Text>
                        </HStack>
                        <ProgressRoot value={faction.influence * 100} max={100} w="full" size="xs" colorPalette="orange">
                            <ProgressBar />
                        </ProgressRoot>
                    </VStack>

                    <VStack align="start" gap={1}>
                        <HStack w="full" justify="space-between">
                            <HStack gap={1}>
                                <Icon as={Target} size={12} color="purple.500" />
                                <Text fontSize="xs" color="gray.500">凝聚力</Text>
                            </HStack>
                            <Text fontSize="xs" fontWeight="bold">{(faction.cohesion * 100).toFixed(0)}%</Text>
                        </HStack>
                        <ProgressRoot value={faction.cohesion * 100} max={100} w="full" size="xs" colorPalette="purple">
                            <ProgressBar />
                        </ProgressRoot>
                    </VStack>
                </SimpleGrid>

                <Separator />

                <Box>
                    <HStack
                        justify="space-between"
                        cursor="pointer"
                        onClick={() => setIsOpen(!isOpen)}
                        _hover={{ color: "blue.500" }}
                    >
                        <Text fontSize="xs" fontWeight="bold" color="gray.600">
                            成员组成 ({factionCohorts.length} 个单元)
                        </Text>
                        <Icon as={isOpen ? ChevronUp : ChevronDown} size={14} />
                    </HStack>

                    <Collapsible.Root open={isOpen}>
                        <Collapsible.Content>
                            <VStack align="stretch" mt={3} gap={2} pl={2} borderLeft="2px solid" borderColor="gray.100">
                                {factionCohorts.map(cohort => (
                                    <HStack key={cohort.id} justify="space-between" bg="gray.50" p={2} borderRadius="sm">
                                        <VStack align="start" gap={0}>
                                            <Text fontSize="xs" fontWeight="bold">{cohort.name}</Text>
                                            <HStack gap={1}>
                                                {cohort.tags?.map(t => (
                                                    <Text key={t} fontSize="10px" color="gray.400">· {t}</Text>
                                                ))}
                                            </HStack>
                                        </VStack>
                                        <Text fontSize="xs" color="gray.600">{cohort.count} 人</Text>
                                    </HStack>
                                ))}
                            </VStack>
                        </Collapsible.Content>
                    </Collapsible.Root>
                </Box>
            </VStack>
        </Box>
    );
};
