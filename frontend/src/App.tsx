import { useState } from 'react';
import logo from './assets/images/logo-universal.png';
import './App.css';
import { Greet } from "../wailsjs/go/main/App";
import { Box, Button, Center, Heading, Image, Input, Text, VStack, HStack, Badge, SimpleGrid, NativeSelect } from "@chakra-ui/react";
import { ProgressBar, ProgressRoot } from './components/ui/progress';
import { Slider } from './components/ui/slider';
import { TimeWheel } from './components/TimeWheel';
import { SiloWheel } from './components/SiloWheel';
import { SetupPanel } from './components/SetupPanel';
import { createInitialSilo, createInitialAgent } from './logic/initializer';
import { GameEngine } from './logic/engine';
import { Silo, Agent, AgentAction, AgentActionType, ACTION_COSTS } from './logic/models';

function App() {
    const [resultText, setResultText] = useState("Please enter your name below 👇");
    const [name, setName] = useState('');
    const [gameStarted, setGameStarted] = useState(false);
    const [showSetup, setShowSetup] = useState(false);
    const [showSiloWheel, setShowSiloWheel] = useState(false);
    const [startYear, setStartYear] = useState(122);
    const [siloNumber, setSiloNumber] = useState(40);
    const [silo, setSilo] = useState<Silo | null>(null);
    const [agent, setAgent] = useState<Agent | null>(null);
    
    // Action Form State
    const [actionType, setActionType] = useState<AgentActionType>('GATHER_INFO');
    const [targetDept, setTargetDept] = useState<string>('');
    const [fragmentId, setFragmentId] = useState<string>('');
    const [adulteration, setAdulteration] = useState<number>(0);

    // We only need one instance of the engine
    const [engine] = useState(() => new GameEngine());

    const updateName = (e: any) => setName(e.target.value);
    const updateResultText = (result: string) => setResultText(result);

    const handleWheelSelect = (year: number) => {
        setStartYear(year);
        setShowSiloWheel(true);
        updateResultText("Now, let fate decide which Silo you belong to.");
    };

    const handleSiloSelect = (num: number) => {
        setSiloNumber(num);
        setShowSiloWheel(false);
        setShowSetup(true);
        updateResultText(`Silo ${num} selected. Use Selection Points to customize your Agent and Silo.`);
    };

    const handleSetupComplete = (selectedTraitIds: string[], profession: string) => {
        const agentName = name || "Juliette";
        const siloName = `Silo ${siloNumber}`;
        const initialSilo = createInitialSilo(siloName, startYear, selectedTraitIds);
        const initialAgent = createInitialAgent(agentName, profession, selectedTraitIds, initialSilo);
        
        try {
            setSilo(initialSilo);
            setAgent(initialAgent);
            updateResultText(`Welcome to ${initialSilo.name}. The year is ${initialSilo.current_year}.`);
            setGameStarted(true);
            setShowSetup(false);
        } catch (err) {
            updateResultText(`Failed to initialize silo: ${err}`);
        }
    };

    // 模拟时间推移 (跳过1年)
    const handlePassTime = () => {
        if (!silo || !agent) return;
        
        // Deep copy state for React updates
        const nextSilo = JSON.parse(JSON.stringify(silo));
        const nextAgent = JSON.parse(JSON.stringify(agent));
        
        engine.updateAgentState(nextAgent, 1, nextSilo, updateResultText);
        const events = engine.updateSiloState(nextSilo, 1, nextAgent);
        
        nextSilo.current_year += 1;
        
        setSilo(nextSilo);
        setAgent(nextAgent);

        if (events.length > 0) {
            updateResultText(`Event Occurred: ${events[0].title}`);
        } else if (nextSilo.victory_status?.is_won !== undefined) {
            updateResultText(`Game Over: ${nextSilo.victory_status.description}`);
        } else {
            updateResultText(`Year passed. Current Year: ${nextSilo.current_year}`);
        }
    };

    // 表单提交：执行动作
    const handleExecuteAction = () => {
        if (!silo || !agent) return;
        if (!targetDept) {
            updateResultText("Please select a target department.");
            return;
        }
        if (actionType === 'SHARE_INFO' && !fragmentId) {
            updateResultText("Please select a fragment to share.");
            return;
        }

        const nextSilo = JSON.parse(JSON.stringify(silo));
        const nextAgent = JSON.parse(JSON.stringify(agent));
        
        const action: AgentAction = {
            type: actionType,
            target_dept: targetDept,
            fragment_id: actionType === 'SHARE_INFO' ? fragmentId : undefined,
            adulteration_level: actionType === 'SHARE_INFO' ? adulteration / 100 : undefined,
            cost: ACTION_COSTS[actionType]
        };

        const success = engine.executeAgentAction(nextSilo, nextAgent, action);
        
        if (success) {
            updateResultText(`Action [${actionType}] executed successfully on ${targetDept}!`);
            setSilo(nextSilo);
            setAgent(nextAgent);
            
            // Check if game over after action (e.g. Agent Compromised)
            engine.updateSiloState(nextSilo, 0, nextAgent);
            if (nextSilo.victory_status?.is_won !== undefined) {
                 updateResultText(`Game Over: ${nextSilo.victory_status.description}`);
            }
        } else {
            updateResultText(`Action failed. (Not enough AP, connections, or redundant operation)`);
        }
    };

    return (
        <Center minH="100vh" bg="gray.50" color="gray.800" py={8}>
            <VStack gap={8} p={8} bg="white" borderRadius="xl" boxShadow="2xl" maxW="1200px" w="full" border="1px solid" borderColor="gray.200">
                <Image src={logo} h="80px" alt="logo" />
                
                <Heading size="md" textAlign="center" color="blue.600">{resultText}</Heading>
                
                {!gameStarted && !showSetup && !showSiloWheel && (
                    <VStack gap={6} w="full" maxW="400px">
                        <Input 
                            placeholder="Enter agent name (e.g. Juliette)" 
                            value={name} 
                            onChange={updateName} 
                            size="md"
                            bg="gray.100"
                            border="1px solid"
                            borderColor="gray.300"
                            _focus={{ border: "1px solid", borderColor: "blue.500", bg: "white" }}
                        />
                        <TimeWheel onSelect={handleWheelSelect} />
                    </VStack>
                )}

                {!gameStarted && !showSetup && showSiloWheel && (
                    <VStack gap={6} w="full" maxW="400px">
                        <SiloWheel onSelect={handleSiloSelect} />
                    </VStack>
                )}

                {!gameStarted && showSetup && !showSiloWheel && (
                    <SetupPanel onComplete={handleSetupComplete} />
                )}

                {gameStarted && (
                    <HStack align="start" gap={6} w="full" wrap="wrap">
                        {/* Left Side: Values */}
                        <VStack gap={6} flex={{ base: "1 1 100%", lg: 2 }} w="full">
                            {/* Agent Status Panel */}
                            <Box w="full" p={4} bg="blue.50" borderRadius="md" borderLeft="4px solid" borderColor="blue.500" boxShadow="sm">
                                <Heading size="sm" mb={4} color="blue.800">Agent Profile: {agent?.name}</Heading>
                            <SimpleGrid columns={2} gap={4} mb={4}>
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.600">Profession</Text>
                                    <Badge colorPalette="blue" variant="solid">{agent?.profession}</Badge>
                                </VStack>
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.600">Traits</Text>
                                    <HStack wrap="wrap">
                                        {agent?.traits?.map(t => (
                                            <Badge key={t} colorPalette="yellow">{t}</Badge>
                                        ))}
                                    </HStack>
                                </VStack>
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.600">Action Points (AP)</Text>
                                    <HStack w="full">
                                        <Text fontWeight="bold" color="blue.700">{Math.floor(agent?.action_points || 0)}</Text>
                                        <ProgressRoot value={(agent?.action_points || 0)} max={100} w="full" size="sm" colorPalette="blue">
                                            <ProgressBar />
                                        </ProgressRoot>
                                    </HStack>
                                </VStack>
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.600">Suspicion Level</Text>
                                    <HStack w="full">
                                        <Text fontWeight="bold" color={agent && agent.suspicion_level > 0.5 ? "red.500" : "green.600"}>
                                            {(agent?.suspicion_level || 0).toFixed(2)}
                                        </Text>
                                        <ProgressRoot value={(agent?.suspicion_level || 0) * 100} max={100} w="full" size="sm" colorPalette={agent && agent.suspicion_level > 0.5 ? "red" : "green"}>
                                            <ProgressBar />
                                        </ProgressRoot>
                                    </HStack>
                                </VStack>
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.600">Known Fragments</Text>
                                    <HStack wrap="wrap">
                                        {agent?.known_fragments?.map(f => (
                                            <Badge key={f} colorPalette="purple" variant="solid">{f}</Badge>
                                        ))}
                                    </HStack>
                                </VStack>
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.600">Political Prestige</Text>
                                    <Text fontWeight="bold" color="orange.500">{Math.floor(agent?.political_prestige || 0)}</Text>
                                </VStack>
                                {agent?.profession === 'IT' && (
                                    <VStack align="start" gap={1}>
                                        <Text fontSize="sm" color="gray.600">Safeguard Risk</Text>
                                        <HStack w="full">
                                            <Text fontWeight="bold" color={silo?.safeguard_risk && silo.safeguard_risk > 0.7 ? "red.500" : "purple.600"}>
                                                {((silo?.safeguard_risk || 0) * 100).toFixed(0)}%
                                            </Text>
                                            <ProgressRoot value={(silo?.safeguard_risk || 0) * 100} max={100} w="full" size="sm" colorPalette={silo?.safeguard_risk && silo.safeguard_risk > 0.7 ? "red" : "purple"}>
                                                <ProgressBar />
                                            </ProgressRoot>
                                        </HStack>
                                    </VStack>
                                )}
                            </SimpleGrid>

                            {/* Connections breakdown */}
                            <Box borderTop="1px solid" borderColor="gray.300" pt={3}>
                                <Text fontSize="sm" color="gray.600" mb={2} fontWeight="bold">Department Connections (Network)</Text>
                                <SimpleGrid columns={{ base: 2, md: 3 }} gap={2}>
                                    {silo?.professions?.map(prof => {
                                        const conn = agent?.connections?.find(c => c.profession_id === prof.id);
                                        const val = conn ? conn.value : 0;
                                        return (
                                            <HStack key={prof.id} justify="space-between" bg="white" p={1} px={2} borderRadius="sm" border="1px solid" borderColor="gray.200">
                                                <Text fontSize="xs" color={val > 0.5 ? "green.600" : "gray.500"}>{prof.name}</Text>
                                                <Text fontSize="xs" fontWeight="bold" color="gray.700">{(val * 100).toFixed(0)}%</Text>
                                            </HStack>
                                        );
                                    })}
                                </SimpleGrid>
                            </Box>
                        </Box>

                        {/* Silo State Panel */}
                        <Box w="full" p={4} bg="gray.50" borderRadius="md" border="1px solid" borderColor="gray.200" boxShadow="sm">
                            <Heading size="sm" mb={4} color="gray.800">Silo State Overview (Year: {silo?.current_year})</Heading>
                            <HStack justify="space-between" mb={4} p={3} bg="white" borderRadius="md" border="1px solid" borderColor="gray.200">
                                <VStack align="start" gap={0}>
                                    <Text fontSize="xs" color="gray.500">Legitimacy</Text>
                                    <Text fontSize="md" fontWeight="bold" color="blue.600">{(silo?.legitimacy || 0).toFixed(2)}</Text>
                                </VStack>
                                <VStack align="start" gap={0}>
                                    <Text fontSize="xs" color="gray.500">Rebellion</Text>
                                    <Text fontSize="md" fontWeight="bold" color="red.500">{(silo?.rebellion || 0).toFixed(2)}</Text>
                                </VStack>
                                <VStack align="start" gap={0}>
                                    <Text fontSize="xs" color="gray.500">Population</Text>
                                    <Text fontSize="md" fontWeight="bold" color="green.600">{silo?.total_population}</Text>
                                </VStack>
                            </HStack>

                            <Heading size="xs" mb={3} color="gray.500" textTransform="uppercase">Departments Overview</Heading>
                            
                            {/* ELITE Class Section */}
                            <Box mb={6}>
                                <HStack justify="space-between" mb={2}>
                                    <Heading size="sm" color="purple.600">Elite Class</Heading>
                                    <HStack>
                                        <Text fontSize="xs" color="gray.500">Avg Ideology: {(
                                            (silo?.professions?.filter(p => p.class_type === 'ELITE').reduce((acc, p) => acc + p.ideology_value, 0) || 0) / 
                                            (silo?.professions?.filter(p => p.class_type === 'ELITE').length || 1) * 100
                                        ).toFixed(0)}%</Text>
                                        <Text fontSize="xs" color="gray.500">Avg Panic: {(
                                            (silo?.professions?.filter(p => p.class_type === 'ELITE').reduce((acc, p) => acc + p.panic_value, 0) || 0) / 
                                            (silo?.professions?.filter(p => p.class_type === 'ELITE').length || 1) * 100
                                        ).toFixed(0)}%</Text>
                                    </HStack>
                                </HStack>
                                <SimpleGrid columns={{ base: 1, md: 2 }} gap={4} pr={2}>
                                    {silo?.professions?.filter(dept => dept.class_type === 'ELITE').map(dept => (
                                        <Box key={dept.id + dept.name} p={3} bg="white" borderRadius="md" borderLeft="4px solid" borderColor="purple.500" boxShadow="sm">
                                            <HStack justify="space-between" mb={2}>
                                                <Text fontWeight="bold" fontSize="sm" color="gray.800">{dept.name}</Text>
                                                <HStack gap={2}>
                                                    <Badge colorPalette="purple" size="sm">ELITE</Badge>
                                                    <Badge colorPalette="gray" size="sm">Pop: {dept.population}</Badge>
                                                </HStack>
                                            </HStack>
                                            <SimpleGrid columns={3} gap={2} mb={2}>
                                                <VStack align="start" gap={0}>
                                                    <Text fontSize="xs" color="gray.500">Pro-Foreign</Text>
                                                    <HStack w="full">
                                                        <Text fontSize="xs" w="30px" color="gray.700">{(dept.ideology_value * 100).toFixed(0)}%</Text>
                                                        <ProgressRoot value={dept.ideology_value * 100} max={100} w="full" size="xs" colorPalette="teal">
                                                            <ProgressBar />
                                                        </ProgressRoot>
                                                    </HStack>
                                                </VStack>
                                                <VStack align="start" gap={0}>
                                                    <Text fontSize="xs" color="gray.500">Panic Level</Text>
                                                    <HStack w="full">
                                                        <Text fontSize="xs" w="30px" color="gray.700">{(dept.panic_value * 100).toFixed(0)}%</Text>
                                                        <ProgressRoot value={dept.panic_value * 100} max={100} w="full" size="xs" colorPalette={dept.panic_value > 0.5 ? "red" : "orange"}>
                                                            <ProgressBar />
                                                        </ProgressRoot>
                                                    </HStack>
                                                </VStack>
                                                <VStack align="start" gap={0}>
                                                    <Text fontSize="xs" color="gray.500">Productivity</Text>
                                                    <HStack w="full">
                                                        <Text fontSize="xs" w="30px" color="gray.700">{(dept.productivity * 100).toFixed(0)}%</Text>
                                                        <ProgressRoot value={dept.productivity * 100} max={100} w="full" size="xs" colorPalette="green">
                                                            <ProgressBar />
                                                        </ProgressRoot>
                                                    </HStack>
                                                </VStack>
                                            </SimpleGrid>
                                            <Box>
                                                <Text fontSize="xs" color="gray.500" mb={1}>Fragments ({dept.known_fragments?.length || 0}/10):</Text>
                                                <HStack wrap="wrap" gap={1}>
                                                    {dept.known_fragments?.map(f => (
                                                        <Badge key={f} colorPalette="cyan" size="xs">{f}</Badge>
                                                    )) || <Text fontSize="xs" color="gray.400">None</Text>}
                                                </HStack>
                                            </Box>
                                        </Box>
                                    ))}
                                </SimpleGrid>
                            </Box>

                            {/* COMMONER Class Section */}
                            <Box mb={2}>
                                <HStack justify="space-between" mb={2}>
                                    <Heading size="sm" color="green.600">Commoner Class</Heading>
                                    <HStack>
                                        <Text fontSize="xs" color="gray.500">Avg Ideology: {(
                                            (silo?.professions?.filter(p => p.class_type === 'COMMONER').reduce((acc, p) => acc + p.ideology_value, 0) || 0) / 
                                            (silo?.professions?.filter(p => p.class_type === 'COMMONER').length || 1) * 100
                                        ).toFixed(0)}%</Text>
                                        <Text fontSize="xs" color="gray.500">Avg Panic: {(
                                            (silo?.professions?.filter(p => p.class_type === 'COMMONER').reduce((acc, p) => acc + p.panic_value, 0) || 0) / 
                                            (silo?.professions?.filter(p => p.class_type === 'COMMONER').length || 1) * 100
                                        ).toFixed(0)}%</Text>
                                    </HStack>
                                </HStack>
                                <SimpleGrid columns={{ base: 1, md: 2 }} gap={4} maxH="300px" overflowY="auto" pr={2} css={{
                                    '&::-webkit-scrollbar': { width: '8px' },
                                    '&::-webkit-scrollbar-track': { background: 'gray.100', borderRadius: '8px' },
                                    '&::-webkit-scrollbar-thumb': { background: 'gray.300', borderRadius: '8px' },
                                }}>
                                    {silo?.professions?.filter(dept => dept.class_type === 'COMMONER').map(dept => (
                                        <Box key={dept.id + dept.name} p={3} bg="white" borderRadius="md" borderLeft="4px solid" borderColor="green.500" boxShadow="sm">
                                            <HStack justify="space-between" mb={2}>
                                                <Text fontWeight="bold" fontSize="sm" color="gray.800">{dept.name}</Text>
                                                <HStack gap={2}>
                                                    <Badge colorPalette="green" size="sm">COMMONER</Badge>
                                                    <Badge colorPalette="gray" size="sm">Pop: {dept.population}</Badge>
                                                </HStack>
                                            </HStack>
                                            <SimpleGrid columns={3} gap={2} mb={2}>
                                                <VStack align="start" gap={0}>
                                                    <Text fontSize="xs" color="gray.500">Pro-Foreign</Text>
                                                    <HStack w="full">
                                                        <Text fontSize="xs" w="30px" color="gray.700">{(dept.ideology_value * 100).toFixed(0)}%</Text>
                                                        <ProgressRoot value={dept.ideology_value * 100} max={100} w="full" size="xs" colorPalette="teal">
                                                            <ProgressBar />
                                                        </ProgressRoot>
                                                    </HStack>
                                                </VStack>
                                                <VStack align="start" gap={0}>
                                                    <Text fontSize="xs" color="gray.500">Panic Level</Text>
                                                    <HStack w="full">
                                                        <Text fontSize="xs" w="30px" color="gray.700">{(dept.panic_value * 100).toFixed(0)}%</Text>
                                                        <ProgressRoot value={dept.panic_value * 100} max={100} w="full" size="xs" colorPalette={dept.panic_value > 0.5 ? "red" : "orange"}>
                                                            <ProgressBar />
                                                        </ProgressRoot>
                                                    </HStack>
                                                </VStack>
                                                <VStack align="start" gap={0}>
                                                    <Text fontSize="xs" color="gray.500">Productivity</Text>
                                                    <HStack w="full">
                                                        <Text fontSize="xs" w="30px" color="gray.700">{(dept.productivity * 100).toFixed(0)}%</Text>
                                                        <ProgressRoot value={dept.productivity * 100} max={100} w="full" size="xs" colorPalette="green">
                                                            <ProgressBar />
                                                        </ProgressRoot>
                                                    </HStack>
                                                </VStack>
                                            </SimpleGrid>
                                            <Box>
                                                <Text fontSize="xs" color="gray.500" mb={1}>Fragments ({dept.known_fragments?.length || 0}/10):</Text>
                                                <HStack wrap="wrap" gap={1}>
                                                    {dept.known_fragments?.map(f => (
                                                        <Badge key={f} colorPalette="cyan" size="xs">{f}</Badge>
                                                    )) || <Text fontSize="xs" color="gray.400">None</Text>}
                                                </HStack>
                                            </Box>
                                        </Box>
                                    ))}
                                </SimpleGrid>
                            </Box>
                        </Box>
                        </VStack>

                        {/* Right Side: Operations */}
                        <VStack gap={6} flex={{ base: "1 1 100%", lg: 1 }} w="full" position="sticky" top="20px">
                        {/* Actions Panel */}
                        <Box w="full" p={5} bg="gray.50" borderRadius="md" border="1px solid" borderColor="gray.200" boxShadow="sm">
                            <Heading size="sm" mb={4} color="gray.800" borderBottom="1px solid" borderColor="gray.200" pb={2}>Agent Action Interface</Heading>
                            <VStack gap={5} align="stretch">
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" fontWeight="bold" color="gray.700">Action Type:</Text>
                                    <NativeSelect.Root size="md" w="full" bg="white">
                                        <NativeSelect.Field value={actionType} onChange={(e) => setActionType(e.target.value as AgentActionType)}>
                                            <option value="GATHER_INFO">Gather Info (10 AP)</option>
                                            <option value="SHARE_INFO">Share Info (20 AP)</option>
                                            <option value="BUILD_CONNECTION">Build Network (15 AP)</option>
                                            <option value="INCITE_REBELLION">Incite Rebellion (30 AP)</option>
                                        </NativeSelect.Field>
                                        <NativeSelect.Indicator />
                                    </NativeSelect.Root>
                                </VStack>

                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" fontWeight="bold" color="gray.700">Target Dept:</Text>
                                    <NativeSelect.Root size="md" w="full" bg="white">
                                        <NativeSelect.Field value={targetDept} onChange={(e) => setTargetDept(e.target.value)}>
                                            <option value="" disabled>Select Department...</option>
                                            {silo?.professions?.map(p => (
                                                <option key={p.id} value={p.name}>{p.name}</option>
                                            ))}
                                        </NativeSelect.Field>
                                        <NativeSelect.Indicator />
                                    </NativeSelect.Root>
                                </VStack>

                                {actionType === 'SHARE_INFO' && (
                                    <>
                                        <VStack align="start" gap={1}>
                                            <Text fontSize="sm" fontWeight="bold" color="gray.700">Fragment to Share:</Text>
                                            <NativeSelect.Root size="md" w="full" bg="white">
                                                <NativeSelect.Field value={fragmentId} onChange={(e) => setFragmentId(e.target.value)}>
                                                    <option value="" disabled>Select Fragment...</option>
                                                    {agent?.known_fragments?.map(f => (
                                                        <option key={f} value={f}>{f}</option>
                                                    ))}
                                                </NativeSelect.Field>
                                                <NativeSelect.Indicator />
                                            </NativeSelect.Root>
                                        </VStack>
                                        <VStack align="stretch" gap={2} p={3} bg="orange.50" borderRadius="md" border="1px dashed" borderColor="orange.200">
                                            <HStack justify="space-between">
                                                <Text fontSize="sm" fontWeight="bold" color="gray.700">Adulteration Level (Risk):</Text>
                                                <Text fontSize="md" fontWeight="bold" color={adulteration > 50 ? "red.500" : "orange.500"}>
                                                    {adulteration}%
                                                </Text>
                                            </HStack>
                                            <Slider 
                                                value={[adulteration]} 
                                                onValueChange={(e: any) => setAdulteration(e.value[0])} 
                                                min={0} max={100} step={5}
                                                colorPalette={adulteration > 50 ? "red" : "orange"}
                                            />
                                            <Text fontSize="xs" color="gray.600">
                                                Higher adulteration reduces AP cost & boosts ideology spread, but drastically increases Suspicion.
                                            </Text>
                                        </VStack>
                                    </>
                                )}

                                <VStack gap={3} mt={2}>
                                    <Button colorPalette="blue" w="full" size="lg" onClick={handleExecuteAction} boxShadow="md">
                                        Execute Action
                                    </Button>
                                    <Button colorPalette="teal" variant="outline" w="full" size="md" onClick={handlePassTime} bg="white">
                                        Pass 1 Year
                                    </Button>
                                </VStack>
                            </VStack>
                        </Box>
                        </VStack>
                    </HStack>
                )}

                <Box w="full" pt={4} borderTop="1px solid" borderColor="gray.200">
                    <Text color="gray.400" fontSize="sm" textAlign="center">
                        Silo40 Control Panel - Agent Operations
                    </Text>
                </Box>
            </VStack>
        </Center>
    )
}

export default App;
