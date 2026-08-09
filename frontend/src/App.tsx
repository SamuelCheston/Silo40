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
import { Silo, Agent, AgentAction, AgentActionType, ACTION_COSTS, ACTION_DURATIONS, ALL_FRAGMENTS } from './logic/models';
import { StoryEvent } from './logic/models';

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
    const [selectedFragments, setSelectedFragments] = useState<string[]>([]);

    const toggleFragment = (f: string) => {
        setSelectedFragments(prev => prev.includes(f) ? prev.filter(x => x !== f) : [...prev, f]);
    };

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
            updateResultText(`Welcome to ${initialSilo.name}. The year is ${initialSilo.current_year}, Month ${initialSilo.current_month}.`);
            setGameStarted(true);
            setShowSetup(false);
        } catch (err) {
            updateResultText(`Failed to initialize silo: ${err}`);
        }
    };

    // 模拟时间推移 (跳过指定月数)
    const passMonths = (months: number, currentSilo: Silo, currentAgent: Agent, logs: string[]): StoryEvent[] => {
        const addLog = (msg: string) => logs.push(msg);
        const events: StoryEvent[] = [];
        
        for (let i = 0; i < months; i++) {
            engine.updateAgentState(currentAgent, 1/12, currentSilo, addLog);
            const monthEvents = engine.updateSiloState(currentSilo, 1/12, currentAgent, addLog);
            events.push(...monthEvents);
            
            currentSilo.current_month = (currentSilo.current_month || 1) + 1;
            if (currentSilo.current_month > 12) {
                currentSilo.current_month = 1;
                currentSilo.current_year += 1;
            }
            
            if (currentSilo.victory_status?.is_won !== undefined) {
                break; // 游戏结束，不再继续推移
            }
        }
        return events;
    };

    const handlePassTime = () => {
        if (!silo || !agent) return;
        
        const nextSilo = JSON.parse(JSON.stringify(silo));
        const nextAgent = JSON.parse(JSON.stringify(agent));
        const logs: string[] = [];

        const events = passMonths(1, nextSilo, nextAgent, logs); // 每次点击过 1 个月
        
        setSilo(nextSilo);
        setAgent(nextAgent);

        if (nextSilo.victory_status?.is_won !== undefined) {
            updateResultText(`Game Over: ${nextSilo.victory_status.description}`);
        } else if (events.length > 0) {
            updateResultText(`Event Occurred: ${events[0].title}`);
        } else {
            const prefix = `Year ${nextSilo.current_year} Month ${nextSilo.current_month}. `;
            if (logs.length > 0) {
                updateResultText(`${prefix}Rumors: ${logs.join(' | ')}`);
            } else {
                updateResultText(`${prefix}The silo was relatively quiet.`);
            }
        }
    };

    // 表单提交：执行动作
    const handleExecuteAction = () => {
        if (!silo || !agent) return;
        
        const isGlobalAction = actionType === 'CONDUCT_PROPAGANDA' || actionType === 'INCITE_REBELLION';
        if (!isGlobalAction && !targetDept) {
            updateResultText("Please select a target department.");
            return;
        }
        if (actionType === 'SHARE_INFO' && selectedFragments.length === 0) {
            updateResultText("Please select at least one fragment to share.");
            return;
        }

        const nextSilo = JSON.parse(JSON.stringify(silo));
        const nextAgent = JSON.parse(JSON.stringify(agent));
        
        const action: AgentAction = {
            type: actionType,
            // 针对全局操作，不传递 target_dept，避免引擎校验失败
            target_dept: isGlobalAction ? undefined : targetDept,
            fragment_ids: actionType === 'SHARE_INFO' ? selectedFragments : undefined,
            cost: ACTION_COSTS[actionType]
        };

        const result = engine.executeAgentAction(nextSilo, nextAgent, action);
        
        if (result.executed) {
            const duration = ACTION_DURATIONS[actionType] || 0;
            const logs: string[] = [];
            
            if (duration > 0) {
                passMonths(duration, nextSilo, nextAgent, logs);
            } else {
                // 即时操作，时间不流逝，但触发一次 NPC 回合 (统一 Actor 管线)
                engine.runNpcTurn(nextSilo, nextAgent, 1/12, (msg) => logs.push(msg));
                engine.updateSiloState(nextSilo, 0, nextAgent); // 检测一下是否有特殊状态更新，不过 deltaYears 为 0 时其实返回了
            }
            
            setSilo(nextSilo);
            setAgent(nextAgent);
            
            if (nextSilo.victory_status?.is_won !== undefined) {
                 updateResultText(`Game Over: ${nextSilo.victory_status.description}`);
            } else {
                 let msg = result.message;
                 if (duration > 0) {
                     msg += ` (Time passed: ${duration} months)`;
                 }
                 if (logs.length > 0) {
                     msg += ` | NPC Activity: ${logs.join(', ')}`;
                 }
                 updateResultText(msg);
            }
        } else {
            updateResultText(`Action failed: ${result.message}`);
        }
    };

    const getIdeologyLabel = (val: number) => {
        const v = val * 100;
        if (v <= 10) return "排外";
        if (v <= 40) return "中立排外";
        return "亲外";
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
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.600">Propaganda Level</Text>
                                    <Text fontWeight="bold" color="pink.500">{(agent?.propaganda_level || 0).toFixed(1)}</Text>
                                </VStack>
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.600">Organization Factor</Text>
                                    <Text fontWeight="bold" color="teal.600">{(agent?.organization_factor || 1).toFixed(1)}x</Text>
                                </VStack>
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.600">Organization Size</Text>
                                    <HStack w="full">
                                        <Text fontWeight="bold" color="cyan.700">
                                            {silo && agent ? engine.getOrganizedPopulation(silo, agent) : 0}
                                        </Text>
                                        <Text fontSize="xs" color="gray.500">
                                            / {silo?.total_population || 0}
                                        </Text>
                                    </HStack>
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
                            <Heading size="sm" mb={4} color="gray.800">Silo State Overview (Year: {silo?.current_year}, Month: {silo?.current_month})</Heading>
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
                                                    <Text fontSize="xs" color="gray.500">Pro-Foreign ({getIdeologyLabel(dept.ideology_value)})</Text>
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
                                                <Text fontSize="xs" color="gray.500" mb={1}>Fragments ({dept.known_fragments?.length || 0}/26):</Text>
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
                                                    <Text fontSize="xs" color="gray.500">Pro-Foreign ({getIdeologyLabel(dept.ideology_value)})</Text>
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
                                                <Text fontSize="xs" color="gray.500" mb={1}>Fragments ({dept.known_fragments?.length || 0}/26):</Text>
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
                                <VStack align="start" gap={2}>
                                    <Text fontSize="sm" fontWeight="bold" color="gray.700">Department Actions:</Text>
                                    <SimpleGrid columns={2} gap={3} w="full">
                                        {[
                                            { value: 'GATHER_INFO', label: 'Gather Info', ap: 10 },
                                            { value: 'SHARE_INFO', label: 'Share Info', ap: 20 },
                                            { value: 'BUILD_CONNECTION', label: 'Build Network', ap: 15 }
                                        ].map((action) => (
                                            <Button
                                                key={action.value}
                                                variant={actionType === action.value ? "solid" : "outline"}
                                                colorPalette={actionType === action.value ? "blue" : "gray"}
                                                onClick={() => setActionType(action.value as AgentActionType)}
                                                h="80px"
                                                display="flex"
                                                flexDirection="column"
                                                justifyContent="center"
                                                alignItems="center"
                                                whiteSpace="normal"
                                                lineHeight="1.2"
                                                bg={actionType === action.value ? "blue.500" : "white"}
                                                _hover={{ bg: actionType === action.value ? "blue.600" : "gray.50" }}
                                            >
                                                <Text fontWeight="bold">{action.label}</Text>
                                                <Text fontSize="xs" mt={1}>({action.ap} AP)</Text>
                                            </Button>
                                        ))}
                                    </SimpleGrid>
                                </VStack>

                                <VStack align="start" gap={2}>
                                    <Text fontSize="sm" fontWeight="bold" color="red.700">Global Actions:</Text>
                                    <SimpleGrid columns={2} gap={3} w="full">
                                        {[
                                            { value: 'CONDUCT_PROPAGANDA', label: 'Propaganda', ap: 20 },
                                            { value: 'INCITE_REBELLION', label: 'Incite Rebellion', ap: 30 }
                                        ].map((action) => (
                                            <Button
                                                key={action.value}
                                                variant={actionType === action.value ? "solid" : "outline"}
                                                colorPalette={actionType === action.value ? "red" : "gray"}
                                                onClick={() => setActionType(action.value as AgentActionType)}
                                                h="80px"
                                                display="flex"
                                                flexDirection="column"
                                                justifyContent="center"
                                                alignItems="center"
                                                whiteSpace="normal"
                                                lineHeight="1.2"
                                                bg={actionType === action.value ? "red.500" : "white"}
                                                _hover={{ bg: actionType === action.value ? "red.600" : "gray.50" }}
                                            >
                                                <Text fontWeight="bold">{action.label}</Text>
                                                <Text fontSize="xs" mt={1}>({action.ap} AP)</Text>
                                            </Button>
                                        ))}
                                    </SimpleGrid>
                                </VStack>

                                {!(actionType === 'CONDUCT_PROPAGANDA' || actionType === 'INCITE_REBELLION') && (
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
                                )}

                                {actionType === 'SHARE_INFO' && (
                                    <>
                                        <VStack align="start" gap={1}>
                                            <Text fontSize="sm" fontWeight="bold" color="gray.700">Fragments to Share (Real & Fake):</Text>
                                            <HStack wrap="wrap" gap={2} maxH="200px" overflowY="auto" p={2} border="1px solid" borderColor="gray.200" borderRadius="md" w="full">
                                                {ALL_FRAGMENTS.map(f => {
                                                    const isSelected = selectedFragments.includes(f);
                                                    const isKnown = agent?.known_fragments?.includes(f);
                                                    return (
                                                        <Badge 
                                                            key={f} 
                                                            colorPalette={isSelected ? (isKnown ? "blue" : "red") : (isKnown ? "gray" : "orange")}
                                                            variant={isSelected ? "solid" : "subtle"}
                                                            cursor="pointer"
                                                            onClick={() => toggleFragment(f)}
                                                            title={isKnown ? "Real Fragment" : "Fake Fragment (Increases Suspicion)"}
                                                        >
                                                            {f} {!isKnown && "(Fake)"}
                                                        </Badge>
                                                    );
                                                })}
                                            </HStack>
                                        </VStack>
                                        <Text fontSize="xs" color="gray.600" mt={1}>
                                            Sharing fake fragments boosts ideology spread but drastically increases Suspicion and lowers acceptance rate.
                                        </Text>
                                    </>
                                )}

                                <VStack gap={3} mt={2}>
                                    <Button colorPalette="blue" w="full" size="lg" onClick={handleExecuteAction} boxShadow="md">
                                        Execute Action
                                    </Button>
                                    <Button colorPalette="teal" variant="outline" w="full" size="md" onClick={handlePassTime} bg="white">
                                        Pass 1 Month
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
