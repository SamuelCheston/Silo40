import React, { useState } from 'react';
import { Box, Button, Text, VStack, HStack, Heading, SimpleGrid, Badge } from '@chakra-ui/react';

export interface TraitOption {
    id: string;
    name: string;
    description: string;
    cost: number;
    type: 'AGENT' | 'SILO';
}

export const AVAILABLE_TRAITS: TraitOption[] = [
    { id: 'shadowy', name: '隐秘行事 (Shadowy)', description: '行动产生的怀疑度降低 20%。', cost: 3, type: 'AGENT' },
    { id: 'charismatic', name: '魅力非凡 (Charismatic)', description: '增加初始威望和人脉建立速度，并在建立组织化力量时获得额外增益。', cost: 4, type: 'AGENT' },
    { id: 'native', name: '地堡土著 (Silo Native)', description: '开局获得自己所在阶层各部门随机 10-15% 的人脉值。', cost: 3, type: 'AGENT' },
    { id: 'abundant', name: '资源充沛 (Abundant Resources)', description: '初始地堡食物与能源资源翻倍。', cost: 3, type: 'SILO' },
    { id: 'leak', name: '信息泄漏 (Information Leak)', description: '平民阶层初始亲外度 (Pro-Foreign) 增加。', cost: 2, type: 'SILO' },
    { id: 'psychoactive_meds', name: '精神药物常态化投放 (Psychoactive Meds)', description: '所有部门的思潮会随时间推移缓慢偏向 IT 部门的思潮。', cost: 3, type: 'SILO' },
];

export const AVAILABLE_DEPARTMENTS = [
    { id: 'Mayor', name: 'Mayor', description: '行动增加3倍怀疑度。开局极高威望，获得全阶层人脉。' },
    { id: 'Judicial', name: 'Judicial', description: '有司法审判权。开局极低威望，获得全阶层人脉。' },
    { id: 'IT', name: 'IT', description: '行动不增加怀疑度，但会增加Safeguard风险。开局极低威望，掌握全碎片，司法部满人脉。' },
    { id: 'Police', name: 'Police', description: '行动随机获得5-9折怀疑度减免。与市长互满人脉，其余精英15%、平民10%人脉。' },
    { id: 'Medical', name: 'Medical', description: '开局极低威望，获得精英15%、平民10%人脉。随时间推移会随机获得信息碎片。' },
    { id: 'Supply', name: 'Supply', description: '在清洁刑罚危机中可以保护目标免死。开局与其他平民互有15%人脉。' },
    { id: 'Mechanical', name: 'Mechanical', description: '最高技术平民，组织平民时获得极高额外增益。开局与其他平民互有15%人脉。' },
    { id: 'Mines', name: 'Mines', description: '极低威望，但行动怀疑度打0.05折。开局与其他平民互有15%人脉。' },
    { id: 'Agricultural', name: 'Agricultural', description: '暂无特殊效果（待更新）。开局与其他平民互有15%人脉。' },
];

interface SetupPanelProps {
    onComplete: (selectedTraitIds: string[], profession: string) => void;
}

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
            <Heading size="md" color="blue.700">定制你的开局 (Customize Setup)</Heading>
            <HStack w="full" justify="space-between" bg="blue.50" p={4} borderRadius="md" border="1px solid" borderColor="blue.100">
                <Text fontSize="lg" fontWeight="bold" color="blue.800">
                    可用选择点数 (Selection Points): 
                </Text>
                <Badge colorPalette={remainingPoints > 0 ? "blue" : "red"} fontSize="xl" px={3} py={1} borderRadius="full">
                    {remainingPoints} / {MAX_POINTS}
                </Badge>
            </HStack>

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
