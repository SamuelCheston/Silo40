// @ts-nocheck
import { useState, useEffect } from 'react';
import logo from './assets/images/logo-universal.png';
import './App.css';
import { CreateGame, GetGameState, HasActiveGame, PassTime, ExecuteAction } from "../wailsjs/go/main/App";
import { Box, Button, Center, Heading, Image, Input, Text, VStack, HStack, Badge, SimpleGrid, NativeSelect } from "@chakra-ui/react";
import { ProgressBar, ProgressRoot } from './components/ui/progress';
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from './components/ui/tabs';
import { TimeWheel } from './components/TimeWheel';
import { SiloWheel } from './components/SiloWheel';
import { SetupPanel } from './components/SetupPanel';
import { BunkerMap } from './components/BunkerMap';
import { FactionView } from './components/FactionView';
import { HeaderStats } from './components/HeaderStats';
import { Silo, Agent, AgentAction, AgentActionType, ACTION_COSTS, ACTION_DURATIONS, ALL_FRAGMENTS, ProfessionActionMeta, GameState } from './logic/models';
import { LayoutGrid, Users } from 'lucide-react';

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
    const [professionActions, setProfessionActions] = useState<ProfessionActionMeta[]>([]);

    // Action Form State
    const [actionType, setActionType] = useState<AgentActionType>('GATHER_INFO');
    const [targetDept, setTargetDept] = useState<string>('');
    const [selectedFragments, setSelectedFragments] = useState<string[]>([]);
    // Profession Action State
    const [professionActionId, setProfessionActionId] = useState<string>('');
    const [resourceTarget, setResourceTarget] = useState<string>('');

    const toggleFragment = (f: string) => {
        setSelectedFragments(prev => prev.includes(f) ? prev.filter(x => x !== f) : [...prev, f]);
    };

    const getIdeologyLabel = (type: string, val: number) => {
        const v = val * 100;
        if (type === 'democracy') {
            if (v <= 30) return "顺民";
            if (v <= 60) return "臣民";
            if (v <= 90) return "民主";
            return "积极民主";
        }
        if (type === 'loyalty') {
            if (v <= 30) return "异见者";
            if (v <= 70) return "中立";
            return "亲信";
        }
        if (v <= 10) return "排外";
        if (v <= 40) return "中立排外";
        return "亲外";
    };

    // 当前选中的职业专属行动元数据 (由 Go 后端下发，actionType === 'PROFESSION_ACTION' 时非空)
    const selectedProfessionAction = actionType === 'PROFESSION_ACTION'
        ? professionActions.find(a => a.id === professionActionId)
        : undefined;
    // 目标选择器显示条件：职业行动按 targetType 决定；通用行动排除全局操作
    const showDeptSelector = actionType === 'PROFESSION_ACTION'
        ? selectedProfessionAction?.target_type === 'DEPT'
        : !(actionType === 'CONDUCT_PROPAGANDA' || actionType === 'INCITE_REBELLION');
    const showResourceSelector = actionType === 'PROFESSION_ACTION' && selectedProfessionAction?.target_type === 'RESOURCE';

    // 应用后端返回的游戏状态快照
    const applyGameState = (state: GameState) => {
        setSilo(state.silo);
        setAgent(state.agent);
        setProfessionActions(state.profession_actions || []);
        setGameStarted(true);
        setShowSetup(false);
    };

    // 启动时恢复进行中的游戏会话 (后端内存/缓存/SQLite 为唯一事实来源)
    useEffect(() => {
        HasActiveGame()
            .then(active => {
                if (!active) return;
                return GetGameState();
            })
            .then(state => {
                if (!state) return;
                applyGameState(state);
                setResultText(`Resumed ${state.silo.name}. The year is ${state.silo.current_year}, Month ${state.silo.current_month}.`);
            })
            .catch(err => updateResultText(`Failed to load saved game: ${err}`));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

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

    // 新建游戏：初始化/落库/缓存全部由 Go 后端完成
    const handleSetupComplete = async (selectedTraitIds: string[], profession: string) => {
        const agentName = name || "Juliette";
        const siloName = `Silo ${siloNumber}`;

        try {
            const state = await CreateGame({
                silo_name: siloName,
                start_year: startYear,
                trait_ids: selectedTraitIds,
                agent_name: agentName,
                profession,
            });
            applyGameState(state);
            updateResultText(`Welcome to ${state.silo.name}. The year is ${state.silo.current_year}, Month ${state.silo.current_month}.`);
        } catch (err) {
            updateResultText(`Failed to initialize silo: ${err}`);
        }
    };

    const handlePassTime = async () => {
        if (!silo || !agent) return;

        try {
            const result = await PassTime(1); // 每次点击过 1 个月，结算在 Go 完成
            setSilo(result.silo);
            setAgent(result.agent);

            if (result.game_over) {
                updateResultText(`Game Over: ${result.silo.victory_status?.description || result.ending_narrative}`);
            } else if (result.stories.length > 0) {
                updateResultText(`Event Occurred: ${result.stories[0].title}`);
            } else {
                const prefix = `Year ${result.silo.current_year} Month ${result.silo.current_month}. `;
                if (result.logs.length > 0) {
                    updateResultText(`${prefix}Rumors: ${result.logs.join(' | ')}`);
                } else {
                    updateResultText(`${prefix}The silo was relatively quiet.`);
                }
            }
        } catch (err) {
            updateResultText(`Failed to advance time: ${err}`);
        }
    };

    // 表单提交：执行动作 (执行/结算/NPC 回合均在 Go)
    const handleExecuteAction = async () => {
        if (!silo || !agent) return;

        const isGlobalAction = actionType === 'CONDUCT_PROPAGANDA' || actionType === 'INCITE_REBELLION';
        const profDef = selectedProfessionAction;

        // 目标校验：职业行动按 targetType；通用行动必须选目标部门
        let actionTarget: string | undefined = targetDept;
        if (profDef) {
            if (profDef.target_type === 'DEPT' && !targetDept) {
                updateResultText("Please select a target department.");
                return;
            }
            if (profDef.target_type === 'RESOURCE' && !resourceTarget) {
                updateResultText("Please select a target resource.");
                return;
            }
            actionTarget = profDef.target_type === 'RESOURCE' ? resourceTarget : targetDept;
        } else if (!isGlobalAction && !targetDept) {
            updateResultText("Please select a target department.");
            return;
        }
        if (actionType === 'SHARE_INFO' && selectedFragments.length === 0) {
            updateResultText("Please select at least one fragment to share.");
            return;
        }

        const action: AgentAction = {
            type: actionType,
            // 针对全局操作，不传递 target_dept，避免引擎校验失败
            target_dept: isGlobalAction ? undefined : actionTarget,
            fragment_ids: actionType === 'SHARE_INFO' ? selectedFragments : undefined,
            profession_action: profDef ? profDef.id : undefined,
            resource_target: profDef?.target_type === 'RESOURCE' ? resourceTarget : undefined,
            cost: profDef ? profDef.ap_cost : ACTION_COSTS[actionType]
        };

        try {
            const outcome = await ExecuteAction(action);

            setSilo(outcome.silo);
            setAgent(outcome.agent);

            if (outcome.game_over) {
                updateResultText(`Game Over: ${outcome.silo.victory_status?.description || outcome.ending_narrative}`);
            } else if (!outcome.result.executed) {
                updateResultText(`Action failed: ${outcome.result.message}`);
            } else {
                let msg = outcome.result.message;
                const duration = ACTION_DURATIONS[actionType] || 0;
                if (duration > 0) {
                    msg += ` (Time passed: ${duration} months)`;
                }
                if (outcome.logs.length > 0) {
                    msg += ` | NPC Activity: ${outcome.logs.join(', ')}`;
                }
                updateResultText(msg);
            }
        } catch (err) {
            updateResultText(`Failed to execute action: ${err}`);
        }
    };

    return (
        <Box minH="100vh" bg="gray.50" color="gray.800">
            {gameStarted && <HeaderStats agent={agent} silo={silo} />}
            
            <Center py={8}>
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
                                    {agent?.profession === 'IT' && (
                                        <VStack align="start" gap={0} minW="100px">
                                            <Text fontSize="xs" color="gray.500">Safeguard Risk</Text>
                                            <Text fontSize="md" fontWeight="bold" color={silo?.safeguard_risk && silo.safeguard_risk > 0.7 ? "red.500" : "purple.600"}>
                                                {((silo?.safeguard_risk || 0) * 100).toFixed(0)}%
                                            </Text>
                                        </VStack>
                                    )}
                                </HStack>

                                <TabsRoot defaultValue="map" variant="enclosed" size="sm" w="full">
                                     <TabsList mb={4} borderBottom="1px solid" borderColor="gray.200">
                                         <TabsTrigger value="map" _selected={{ color: "blue.600", borderBottom: "2px solid", borderColor: "blue.600" }}>
                                             <HStack gap={2}>
                                                 <LayoutGrid size={14} />
                                                 <Text fontWeight="bold">地堡地图</Text>
                                             </HStack>
                                         </TabsTrigger>
                                         <TabsTrigger value="factions" _selected={{ color: "blue.600", borderBottom: "2px solid", borderColor: "blue.600" }}>
                                             <HStack gap={2}>
                                                 <Users size={14} />
                                                 <Text fontWeight="bold">阵营概览</Text>
                                             </HStack>
                                         </TabsTrigger>
                                     </TabsList>
                                     <TabsContent value="map">
                                         {silo && <BunkerMap silo={silo} agent={agent} />}
                                     </TabsContent>
                                     <TabsContent value="factions">
                                         {silo && <FactionView silo={silo} />}
                                     </TabsContent>
                                 </TabsRoot>
                            </Box>
                        </VStack>

                    {/* Right Side: Operations */}
                        <VStack gap={6} flex={{ base: "1 1 100%", lg: 1 }} w="full" position="sticky" top="20px">
                        {/* Profession Actions Frame */}
                        <Box w="full" p={5} bg="purple.50" borderRadius="md" border="1px solid" borderColor="purple.200" boxShadow="sm">
                            <Heading size="sm" mb={2} color="purple.800" borderBottom="1px solid" borderColor="purple.200" pb={2}>
                                Profession Actions: {agent?.profession}
                            </Heading>
                            <Text fontSize="xs" color="gray.600" mb={3}>
                                Unique actions of your profession. Hover for details.
                            </Text>
                            <SimpleGrid columns={2} gap={3} w="full">
                                {professionActions.map(def => {
                                    const isSelected = professionActionId === def.id && actionType === 'PROFESSION_ACTION';
                                    return (
                                        <Button
                                            key={def.id}
                                            variant={isSelected ? "solid" : "outline"}
                                            colorPalette={isSelected ? "purple" : "gray"}
                                            onClick={() => { setActionType('PROFESSION_ACTION'); setProfessionActionId(def.id); }}
                                            h="80px"
                                            display="flex"
                                            flexDirection="column"
                                            justifyContent="center"
                                            alignItems="center"
                                            whiteSpace="normal"
                                            lineHeight="1.2"
                                            title={def.description}
                                            bg={isSelected ? "purple.500" : "white"}
                                            _hover={{ bg: isSelected ? "purple.600" : "gray.50" }}
                                        >
                                            <Text fontWeight="bold">{def.label}</Text>
                                            <Text fontSize="xs" mt={1}>({def.ap_cost} AP)</Text>
                                        </Button>
                                    );
                                })}
                            </SimpleGrid>
                        </Box>

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

                                {showDeptSelector && (
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

                                {showResourceSelector && (
                                    <VStack align="start" gap={1}>
                                        <Text fontSize="sm" fontWeight="bold" color="gray.700">Target Resource:</Text>
                                        <NativeSelect.Root size="md" w="full" bg="white">
                                            <NativeSelect.Field value={resourceTarget} onChange={(e) => setResourceTarget(e.target.value)}>
                                                <option value="" disabled>Select Resource...</option>
                                                {['Energy', 'Materials', 'Supplies'].map(r => (
                                                    <option key={r} value={r}>{r}</option>
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
    </Box>
    )
}

export default App;
