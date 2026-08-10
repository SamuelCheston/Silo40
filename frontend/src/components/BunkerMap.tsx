import React, { useState } from 'react';
import { 
    Box, 
    VStack, 
    HStack, 
    Text, 
    Badge, 
    Heading, 
    SimpleGrid,
    Grid,
    IconButton
} from "@chakra-ui/react";
import { ProgressBar, ProgressRoot } from './ui/progress';
import { Profession, Silo } from '../logic/models';
import { X } from 'lucide-react';
import './BunkerMap.css';

interface BunkerMapProps {
    silo: Silo;
}

const ZONE_DATA = [
    { id: 'Upper', name: '上层区 (Upper Zone)', levels: '1-30', color: 'purple', bg: 'purple.50', borderColor: 'purple.500' },
    { id: 'Mid', name: '中层区 (Mid Zone)', levels: '31-90', color: 'green', bg: 'green.50', borderColor: 'green.500' },
    { id: 'Lower', name: '下层区 (Lower Zone)', levels: '91-144', color: 'orange', bg: 'orange.50', borderColor: 'orange.500' }
];

export const BunkerMap: React.FC<BunkerMapProps> = ({ silo }) => {
    const [selectedDept, setSelectedDept] = useState<Profession | null>(null);

    const getIdeologyLabel = (type: string, val: number) => {
        const v = val * 100;
        if (type === 'democracy') {
            if (v <= 30) return "顺民";
            if (v <= 60) return "臣民";
            if (v <= 90) return "民主";
            return "积极民主";
        }
        if (v <= 10) return "排外";
        if (v <= 40) return "中立排外";
        return "亲外";
    };

    return (
        <VStack align="stretch" gap={4} w="full">
            <HStack align="start" gap={6} w="full" wrap="wrap">
                {/* Silo Map */}
                <Box flex={{ base: "1 1 100%", md: 2 }} className="bunker-map-container" borderRadius="lg" border="1px solid" borderColor="gray.300" overflow="hidden" bg="gray.100">
                    <VStack align="stretch" gap={0} className="bunker-silo-structure">
                        {ZONE_DATA.map(zone => (
                            <Box 
                                key={zone.id} 
                                bg={zone.bg}
                                borderLeft="8px solid"
                                borderColor={zone.borderColor}
                                p={4}
                                borderBottom="2px dashed"
                                className={`bunker-zone zone-${zone.id.toLowerCase()}`}
                            >
                                <HStack justify="space-between" mb={4}>
                                    <VStack align="start" gap={0}>
                                        <Heading size="xs" color={`${zone.color}.700`} textTransform="uppercase" letterSpacing="wider">{zone.name}</Heading>
                                        <Text fontSize="2xs" color="gray.500" fontWeight="bold">LEVELS: {zone.levels}</Text>
                                    </VStack>
                                    <Badge colorPalette={zone.color} variant="subtle" size="sm">
                                        {silo.professions.filter(p => p.zone === zone.id).length} DEPARTMENTS
                                    </Badge>
                                </HStack>

                                <SimpleGrid columns={{ base: 2, sm: 3, lg: 4 }} gap={3}>
                                    {silo.professions
                                        .filter(p => p.zone === zone.id)
                                        .map(dept => (
                                            <Box 
                                                key={dept.id} 
                                                onClick={() => setSelectedDept(dept)}
                                                p={3}
                                                bg={selectedDept?.id === dept.id ? `${zone.color}.200` : "white"}
                                                borderRadius="md"
                                                boxShadow="sm"
                                                cursor="pointer"
                                                border="2px solid"
                                                borderColor={selectedDept?.id === dept.id ? zone.borderColor : "transparent"}
                                                _hover={{ transform: "translateY(-2px)", boxShadow: "md", borderColor: zone.borderColor }}
                                                transition="all 0.2s"
                                                display="flex"
                                                flexDirection="column"
                                                alignItems="center"
                                                justifyContent="center"
                                                minH="60px"
                                            >
                                                <Text fontWeight="bold" fontSize="xs" textAlign="center" color="gray.700">{dept.name}</Text>
                                                <Badge colorPalette={dept.class_type === 'ELITE' ? 'purple' : 'green'} size="xs" mt={1} variant="outline">
                                                    {dept.class_type}
                                                </Badge>
                                            </Box>
                                        ))}
                                </SimpleGrid>
                            </Box>
                        ))}
                    </VStack>
                </Box>

                {/* Detail Panel */}
                <Box flex={{ base: "1 1 100%", md: 1 }} minW="300px">
                    {selectedDept ? (
                        <Box 
                            bg="white" 
                            p={5} 
                            borderRadius="lg" 
                            border="1px solid" 
                            borderColor="gray.200" 
                            boxShadow="lg"
                            position="sticky"
                            top="20px"
                        >
                            <HStack justify="space-between" mb={4} borderBottom="1px solid" borderColor="gray.100" pb={3}>
                                <VStack align="start" gap={0}>
                                    <Heading size="sm" color="gray.800">{selectedDept.name}</Heading>
                                    <Text fontSize="xs" color="gray.500">Department Intelligence Data</Text>
                                </VStack>
                                <IconButton 
                                    aria-label="Close" 
                                    size="xs" 
                                    variant="ghost" 
                                    onClick={() => setSelectedDept(null)}
                                >
                                    <X size={16} />
                                </IconButton>
                            </HStack>

                            <VStack align="stretch" gap={5}>
                                <Grid templateColumns="repeat(2, 1fr)" gap={4}>
                                    <VStack align="start" gap={0}>
                                        <Text fontSize="2xs" color="gray.500" textTransform="uppercase" fontWeight="bold">Class</Text>
                                        <Badge colorPalette={selectedDept.class_type === 'ELITE' ? 'purple' : 'green'} variant="solid">
                                            {selectedDept.class_type}
                                        </Badge>
                                    </VStack>
                                    <VStack align="start" gap={0}>
                                        <Text fontSize="2xs" color="gray.500" textTransform="uppercase" fontWeight="bold">Population</Text>
                                        <Text fontWeight="bold" color="blue.600">{selectedDept.population}</Text>
                                    </VStack>
                                    <VStack align="start" gap={0}>
                                        <Text fontSize="2xs" color="gray.500" textTransform="uppercase" fontWeight="bold">Power Level</Text>
                                        <Text fontWeight="bold" color="orange.600">{selectedDept.power_level}</Text>
                                    </VStack>
                                    <VStack align="start" gap={0}>
                                        <Text fontSize="2xs" color="gray.500" textTransform="uppercase" fontWeight="bold">Productivity</Text>
                                        <Text fontWeight="bold" color="green.600">{(selectedDept.productivity * 100).toFixed(0)}%</Text>
                                    </VStack>
                                </Grid>

                                <VStack align="stretch" gap={1}>
                                    <Text fontSize="2xs" color="gray.500" textTransform="uppercase" fontWeight="bold">Panic Level</Text>
                                    <HStack>
                                        <Text fontSize="xs" fontWeight="bold" w="40px" color={selectedDept.panic_value > 0.5 ? "red.600" : "gray.700"}>
                                            {(selectedDept.panic_value * 100).toFixed(1)}%
                                        </Text>
                                        <ProgressRoot value={selectedDept.panic_value * 100} max={100} w="full" size="xs" colorPalette={selectedDept.panic_value > 0.5 ? "red" : "orange"}>
                                            <ProgressBar />
                                        </ProgressRoot>
                                    </HStack>
                                </VStack>

                                <Box bg="gray.50" p={3} borderRadius="md" border="1px solid" borderColor="gray.100">
                                    <Text fontSize="xs" fontWeight="bold" mb={3} color="gray.700" borderBottom="1px solid" borderColor="gray.200" pb={1}>IDEOLOGIES</Text>
                                    <VStack align="stretch" gap={3}>
                                        {Object.entries(selectedDept.ideologies || {}).map(([type, val]) => (
                                            <VStack key={type} align="stretch" gap={1}>
                                                <HStack justify="space-between">
                                                    <Text fontSize="2xs" textTransform="capitalize" color="gray.600" fontWeight="medium">
                                                        {type.replace('_', ' ')}
                                                    </Text>
                                                    <Badge variant="surface" size="xs" colorPalette={type === 'pro_foreign' ? "teal" : "blue"}>
                                                        {getIdeologyLabel(type, val)}
                                                    </Badge>
                                                </HStack>
                                                <HStack>
                                                    <Text fontSize="2xs" w="35px" color="gray.700">{(val * 100).toFixed(0)}%</Text>
                                                    <ProgressRoot value={val * 100} max={100} w="full" size="xs" colorPalette={type === 'pro_foreign' ? "teal" : "blue"}>
                                                        <ProgressBar />
                                                    </ProgressRoot>
                                                </HStack>
                                            </VStack>
                                        ))}
                                    </VStack>
                                </Box>

                                <Box>
                                    <Text fontSize="xs" fontWeight="bold" mb={2} color="gray.700">INTEL FRAGMENTS ({selectedDept.known_fragments?.length || 0})</Text>
                                    <HStack wrap="wrap" gap={1.5}>
                                        {selectedDept.known_fragments?.map(f => (
                                            <Badge key={f} colorPalette="cyan" variant="solid" size="xs" borderRadius="sm">{f}</Badge>
                                        )) || <Text fontSize="xs" color="gray.400" fontStyle="italic">No fragments known</Text>}
                                    </HStack>
                                </Box>
                            </VStack>
                        </Box>
                    ) : (
                        <Box 
                            h="full" 
                            minH="300px" 
                            display="flex" 
                            alignItems="center" 
                            justifyContent="center" 
                            bg="gray.50" 
                            borderRadius="lg" 
                            border="2px dashed" 
                            borderColor="gray.300"
                            p={8}
                            textAlign="center"
                        >
                            <VStack gap={2}>
                                <Text color="gray.400" fontWeight="bold">NO DEPARTMENT SELECTED</Text>
                                <Text fontSize="xs" color="gray.500">Click a department on the map to view detailed analytics and population metrics.</Text>
                            </VStack>
                        </Box>
                    )}
                </Box>
            </HStack>
        </VStack>
    );
};
