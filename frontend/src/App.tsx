import { useState } from 'react';
import logo from './assets/images/logo-universal.png';
import './App.css';
import { Greet } from "../wailsjs/go/main/App";
import { Box, Button, Center, Heading, Image, Input, Text, VStack, HStack, Badge, SimpleGrid, NativeSelect } from "@chakra-ui/react";
import { ProgressBar, ProgressRoot } from './components/ui/progress';
import { Slider } from './components/ui/slider';
import { TimeWheel } from './components/TimeWheel';
import { createInitialSilo, createInitialAgent } from './logic/initializer';
import { GameEngine } from './logic/engine';
import { Silo, Agent, AgentAction, AgentActionType, ACTION_COSTS } from './logic/models';

function App() {
    const [resultText, setResultText] = useState("Please enter your name below 👇");
    const [name, setName] = useState('');
    const [gameStarted, setGameStarted] = useState(false);
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

    async function handleStartGame(year: number) {
        const agentName = name || "Juliette";
        const initialSilo = createInitialSilo(agentName, year);
        const initialAgent = createInitialAgent(agentName);
        
        try {
            // 这里为了演示，跳过了后端的 CreateSilo 直接在前端驱动
            // const savedSilo = await CreateSilo(initialSilo);
            setSilo(initialSilo);
            setAgent(initialAgent);
            updateResultText(`Welcome to ${initialSilo.name}. The year is ${initialSilo.current_year}.`);
            setGameStarted(true);
        } catch (err) {
            updateResultText(`Failed to initialize silo: ${err}`);
        }
    }

    // 模拟时间推移 (跳过1年)
    const handlePassTime = () => {
        if (!silo || !agent) return;
        
        // Deep copy state for React updates
        const nextSilo = JSON.parse(JSON.stringify(silo));
        const nextAgent = JSON.parse(JSON.stringify(agent));
        
        engine.updateAgentState(nextAgent, 1);
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
        <Center minH="100vh" bg="gray.900" color="white" py={8}>
            <VStack gap={8} p={8} bg="gray.800" borderRadius="xl" boxShadow="2xl" maxW="800px" w="full">
                <Image src={logo} h="80px" alt="logo" />
                
                <Heading size="md" textAlign="center" color="blue.300">{resultText}</Heading>
                
                {!gameStarted ? (
                    <VStack gap={6} w="full" maxW="400px">
                        <Input 
                            placeholder="Enter agent name (e.g. Juliette)" 
                            value={name} 
                            onChange={updateName} 
                            size="md"
                            bg="gray.700"
                            border="none"
                            _focus={{ border: "1px solid", borderColor: "blue.500" }}
                        />
                        <TimeWheel onSelect={handleStartGame} />
                    </VStack>
                ) : (
                    <VStack gap={6} w="full">
                        {/* Agent Status Panel */}
                        <Box w="full" p={4} bg="gray.700" borderRadius="md" borderLeft="4px solid" borderColor="blue.400">
                            <Heading size="sm" mb={4}>Agent Profile: {agent?.name}</Heading>
                            <SimpleGrid columns={2} gap={4} mb={4}>
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.400">Profession</Text>
                                    <Badge colorPalette="blue" variant="solid">{agent?.profession}</Badge>
                                </VStack>
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.400">Traits</Text>
                                    <HStack wrap="wrap">
                                        {agent?.traits?.map(t => (
                                            <Badge key={t} colorPalette="yellow">{t}</Badge>
                                        ))}
                                    </HStack>
                                </VStack>
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.400">Action Points (AP)</Text>
                                    <HStack w="full">
                                        <Text fontWeight="bold">{Math.floor(agent?.action_points || 0)}</Text>
                                        <ProgressRoot value={(agent?.action_points || 0)} max={100} w="full" size="sm" colorPalette="blue">
                                            <ProgressBar />
                                        </ProgressRoot>
                                    </HStack>
                                </VStack>
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.400">Suspicion Level</Text>
                                    <HStack w="full">
                                        <Text fontWeight="bold" color={agent && agent.suspicion_level > 0.5 ? "red.400" : "white"}>
                                            {(agent?.suspicion_level || 0).toFixed(2)}
                                        </Text>
                                        <ProgressRoot value={(agent?.suspicion_level || 0) * 100} max={100} w="full" size="sm" colorPalette={agent && agent.suspicion_level > 0.5 ? "red" : "green"}>
                                            <ProgressBar />
                                        </ProgressRoot>
                                    </HStack>
                                </VStack>
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.400">Known Fragments</Text>
                                    <HStack wrap="wrap">
                                        {agent?.known_fragments?.map(f => (
                                            <Badge key={f} colorPalette="purple" variant="solid">{f}</Badge>
                                        ))}
                                    </HStack>
                                </VStack>
                                <VStack align="start" gap={1}>
                                    <Text fontSize="sm" color="gray.400">Political Prestige</Text>
                                    <Text fontWeight="bold" color="yellow.400">{Math.floor(agent?.political_prestige || 0)}</Text>
                                </VStack>
                            </SimpleGrid>

                            {/* Connections breakdown */}
                            <Box borderTop="1px solid" borderColor="gray.600" pt={3}>
                                <Text fontSize="sm" color="gray.400" mb={2}>Department Connections (Network)</Text>
                                <SimpleGrid columns={{ base: 2, md: 3 }} gap={2}>
                                    {silo?.professions?.map(prof => {
                                        const conn = agent?.connections?.find(c => c.profession_id === prof.id);
                                        const val = conn ? conn.value : 0;
                                        return (
                                            <HStack key={prof.id} justify="space-between" bg="gray.800" p={1} px={2} borderRadius="sm">
                                                <Text fontSize="xs" color={val > 0.5 ? "green.300" : "gray.300"}>{prof.name}</Text>
                                                <Text fontSize="xs" fontWeight="bold">{(val * 100).toFixed(0)}%</Text>
                                            </HStack>
                                        );
                                    })}
                                </SimpleGrid>
                            </Box>
                        </Box>

                        {/* Silo State Panel */}
                        <Box w="full" p={4} bg="gray.700" borderRadius="md">
                            <Heading size="sm" mb={4}>Silo State Overview (Year: {silo?.current_year})</Heading>
                            <HStack justify="space-between" mb={4} p={2} bg="gray.800" borderRadius="md">
                                <Text fontSize="sm">Legitimacy: {(silo?.legitimacy || 0).toFixed(2)}</Text>
                                <Text fontSize="sm">Rebellion: {(silo?.rebellion || 0).toFixed(2)}</Text>
                                <Text fontSize="sm">Population: {silo?.total_population}</Text>
                            </HStack>

                            <Heading size="xs" mb={3} color="gray.400">Departments Overview</Heading>
                            <SimpleGrid columns={{ base: 1, md: 2 }} gap={4} maxH="400px" overflowY="auto" pr={2}>
                                {silo?.professions?.map(dept => (
                                    <Box key={dept.id + dept.name} p={3} bg="gray.800" borderRadius="md" borderLeft="3px solid" borderColor="teal.400">
                                        <HStack justify="space-between" mb={2}>
                                            <Text fontWeight="bold" fontSize="sm">{dept.name}</Text>
                                            <Badge colorPalette="gray" size="sm">Pop: {dept.population}</Badge>
                                        </HStack>
                                        <SimpleGrid columns={2} gap={2} mb={2}>
                                            <VStack align="start" gap={0}>
                                                <Text fontSize="xs" color="gray.400">Pro-Foreign (Ideology)</Text>
                                                <HStack w="full">
                                                    <Text fontSize="xs" w="30px">{(dept.ideology_value * 100).toFixed(0)}%</Text>
                                                    <ProgressRoot value={dept.ideology_value * 100} max={100} w="full" size="xs" colorPalette="teal">
                                                        <ProgressBar />
                                                    </ProgressRoot>
                                                </HStack>
                                            </VStack>
                                            <VStack align="start" gap={0}>
                                                <Text fontSize="xs" color="gray.400">Panic Level</Text>
                                                <HStack w="full">
                                                    <Text fontSize="xs" w="30px">{(dept.panic_value * 100).toFixed(0)}%</Text>
                                                    <ProgressRoot value={dept.panic_value * 100} max={100} w="full" size="xs" colorPalette={dept.panic_value > 0.5 ? "red" : "orange"}>
                                                        <ProgressBar />
                                                    </ProgressRoot>
                                                </HStack>
                                            </VStack>
                                        </SimpleGrid>
                                        <Box>
                                            <Text fontSize="xs" color="gray.400" mb={1}>Known Fragments ({dept.known_fragments?.length || 0}/10):</Text>
                                            <HStack wrap="wrap" gap={1}>
                                                {dept.known_fragments?.map(f => (
                                                    <Badge key={f} colorPalette="cyan" size="xs">{f}</Badge>
                                                )) || <Text fontSize="xs" color="gray.500">None</Text>}
                                            </HStack>
                                        </Box>
                                    </Box>
                                ))}
                            </SimpleGrid>
                        </Box>

                        {/* Actions Panel */}
                        <Box w="full" p={4} bg="gray.700" borderRadius="md">
                            <Heading size="sm" mb={4}>Agent Action Interface</Heading>
                            <VStack gap={4} align="stretch">
                                <HStack justify="space-between">
                                    <Text fontSize="sm">Action Type:</Text>
                                    <NativeSelect.Root size="sm" w="200px">
                                        <NativeSelect.Field value={actionType} onChange={(e) => setActionType(e.target.value as AgentActionType)}>
                                            <option value="GATHER_INFO">Gather Info (10 AP)</option>
                                            <option value="SHARE_INFO">Share Info (20 AP)</option>
                                            <option value="BUILD_CONNECTION">Build Network (15 AP)</option>
                                            <option value="INCITE_REBELLION">Incite Rebellion (30 AP)</option>
                                        </NativeSelect.Field>
                                        <NativeSelect.Indicator />
                                    </NativeSelect.Root>
                                </HStack>

                                <HStack justify="space-between">
                                    <Text fontSize="sm">Target Dept:</Text>
                                    <NativeSelect.Root size="sm" w="200px">
                                        <NativeSelect.Field value={targetDept} onChange={(e) => setTargetDept(e.target.value)}>
                                            <option value="" disabled>Select Department...</option>
                                            {silo?.professions?.map(p => (
                                                <option key={p.id} value={p.name}>{p.name}</option>
                                            ))}
                                        </NativeSelect.Field>
                                        <NativeSelect.Indicator />
                                    </NativeSelect.Root>
                                </HStack>

                                {actionType === 'SHARE_INFO' && (
                                    <>
                                        <HStack justify="space-between">
                                            <Text fontSize="sm">Fragment to Share:</Text>
                                            <NativeSelect.Root size="sm" w="200px">
                                                <NativeSelect.Field value={fragmentId} onChange={(e) => setFragmentId(e.target.value)}>
                                                    <option value="" disabled>Select Fragment...</option>
                                                    {agent?.known_fragments?.map(f => (
                                                        <option key={f} value={f}>{f}</option>
                                                    ))}
                                                </NativeSelect.Field>
                                                <NativeSelect.Indicator />
                                            </NativeSelect.Root>
                                        </HStack>
                                        <VStack align="stretch" gap={1}>
                                            <HStack justify="space-between">
                                                <Text fontSize="sm">Adulteration Level (Risk):</Text>
                                                <Text fontSize="sm" fontWeight="bold" color={adulteration > 50 ? "red.400" : "orange.300"}>
                                                    {adulteration}%
                                                </Text>
                                            </HStack>
                                            <Slider 
                                                value={[adulteration]} 
                                                onValueChange={(e: any) => setAdulteration(e.value[0])} 
                                                min={0} max={100} step={5}
                                                colorPalette={adulteration > 50 ? "red" : "orange"}
                                            />
                                            <Text fontSize="xs" color="gray.500">
                                                Higher adulteration reduces AP cost & boosts ideology spread, but drastically increases Suspicion.
                                            </Text>
                                        </VStack>
                                    </>
                                )}

                                <HStack gap={4} w="full" justify="space-between" mt={4}>
                                    <Button colorPalette="teal" variant="outline" onClick={handlePassTime} w="full">
                                        Pass 1 Year
                                    </Button>
                                    <Button colorPalette="blue" onClick={handleExecuteAction} w="full">
                                        Execute Action
                                    </Button>
                                </HStack>
                            </VStack>
                        </Box>
                    </VStack>
                )}

                <Box w="full" pt={4} borderTop="1px solid" borderColor="gray.700">
                    <Text color="gray.400" fontSize="sm" textAlign="center">
                        Silo40 Control Panel - Agent Operations
                    </Text>
                </Box>
            </VStack>
        </Center>
    )
}

export default App;
