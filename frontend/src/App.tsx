// @ts-nocheck
import { useState, useEffect } from 'react';
import './App.css';
import { CreateGame, GetGameState, HasActiveGame, PassTime, ExecuteAction } from "../wailsjs/go/main/App";
import { Box, Button, Heading, Input, Text, VStack, HStack, Badge, SimpleGrid, NativeSelect } from "@chakra-ui/react";
import { ProgressBar, ProgressRoot } from './components/ui/progress';
import { Tooltip } from './components/ui/tooltip';
import { TimeWheel } from './components/TimeWheel';
import { SiloWheel } from './components/SiloWheel';
import { SetupPanel } from './components/SetupPanel';
import { BunkerMap } from './components/BunkerMap';
import { FactionView } from './components/FactionView';
import { HeaderStats } from './components/HeaderStats';
import { Silo, Agent, AgentAction, AgentActionType, ACTION_COSTS, ACTION_DURATIONS, ALL_FRAGMENTS, GameState, AgentStats, PlayerActionMeta } from './logic/models';
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
    const [agentStats, setAgentStats] = useState<AgentStats | null>(null);
    const [availableActions, setAvailableActions] = useState<PlayerActionMeta[]>([]);
    const [activeView, setActiveView] = useState<'map' | 'factions'>('map');

    // Action Form State
    const [actionType, setActionType] = useState<AgentActionType>('GATHER_INFO');
    const [targetDept, setTargetDept] = useState<string>('');
    const [selectedFragments, setSelectedFragments] = useState<string[]>([]);
    const [professionActionId, setProfessionActionId] = useState<string>('');
    const [playerActionId, setPlayerActionId] = useState<string>('');
    const [resourceTarget, setResourceTarget] = useState<string>('');

    const toggleFragment = (f: string) => {
        setSelectedFragments(prev => prev.includes(f) ? prev.filter(x => x !== f) : [...prev, f]);
    };

    const getIdeologyLabel = (type: string, val: number) => {
        const v = val * 100;
        if (type === 'loyalty') {
            if (v <= 30) return "异见者";
            if (v <= 70) return "中立";
            return "亲信";
        }
        if (v <= 10) return "排外";
        if (v <= 40) return "中立排外";
        return "亲外";
    };

    const groupedActions = availableActions.reduce((acc, action) => {
        if (!acc[action.group]) {
            acc[action.group] = [];
        }
        acc[action.group].push(action);
        return acc;
    }, {} as Record<string, PlayerActionMeta[]>);

    const selectedActionMeta = availableActions.find((action) => {
        if (action.action_type !== actionType) return false;
        if (action.action_type === 'PROFESSION_ACTION') return action.id === professionActionId;
        if (action.action_type === 'PLAYER_EVENT') return action.id === playerActionId;
        return action.id === actionType;
    });
    const showDeptSelector = selectedActionMeta?.target_type === 'DEPT';
    const showResourceSelector = selectedActionMeta?.target_type === 'RESOURCE';

    // 应用后端返回的游戏状态快照
    const applyGameState = (state: GameState) => {
        setSilo(state.silo);
        setAgent(state.agent);
        setAgentStats(state.agent_stats);
        setAvailableActions(state.available_actions || []);
        setGameStarted(true);
        setShowSetup(false);
    };

    useEffect(() => {
        if (!availableActions.some((action) => {
            if (action.action_type !== actionType) return false;
            if (actionType === 'PROFESSION_ACTION') return action.id === professionActionId;
            if (actionType === 'PLAYER_EVENT') return action.id === playerActionId;
            return action.id === actionType;
        })) {
            const fallback = availableActions.find(action => action.enabled) || availableActions[0];
            if (!fallback) return;
            setActionType(fallback.action_type as AgentActionType);
            setProfessionActionId(fallback.action_type === 'PROFESSION_ACTION' ? fallback.id : '');
            setPlayerActionId(fallback.action_type === 'PLAYER_EVENT' ? fallback.id : '');
        }
    }, [availableActions, actionType, professionActionId, playerActionId]);

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

    const handleDebugStart = async () => {
        const agentName = name || "Juliette";
        const siloName = "Silo 20";
        const debugYear = 122; // Just now
        const debugTraits = ['charismatic', 'native', 'leak'];
        const debugProfession = 'Mechanical';

        try {
            const state = await CreateGame({
                silo_name: siloName,
                start_year: debugYear,
                trait_ids: debugTraits,
                agent_name: agentName,
                profession: debugProfession,
            });
            applyGameState(state);
            updateResultText(`[DEBUG] 快速开始于 ${state.silo.name}。当前年份：${state.silo.current_year}。`);
        } catch (err) {
            updateResultText(`Debug 启动失败: ${err}`);
        }
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
            setAgentStats(result.agent_stats);
            setAvailableActions(result.available_actions || []);

            if (result.game_over) {
                updateResultText(`Game Over: ${result.silo.victory_status?.description || result.ending_narrative}`);
            } else if (result.stories.length > 0) {
                const story = result.stories[0];
                updateResultText(`Event Occurred [${story.category || "uncategorized"}]: ${story.title}`);
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

        const selected = selectedActionMeta;
        const isGlobalAction = selected?.target_type === 'NONE';

        // 目标校验统一以后端下发的 action meta 为准
        let actionTarget: string | undefined = targetDept;
        if (selected?.target_type === 'DEPT') {
            if (!targetDept) {
                updateResultText("Please select a target department.");
                return;
            }
        } else if (selected?.target_type === 'RESOURCE') {
            if (!resourceTarget) {
                updateResultText("Please select a target resource.");
                return;
            }
            actionTarget = resourceTarget;
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
            action_id: actionType === 'PLAYER_EVENT' ? playerActionId : undefined,
            target_dept: isGlobalAction ? undefined : actionTarget,
            fragment_ids: actionType === 'SHARE_INFO' ? selectedFragments : undefined,
            profession_action: actionType === 'PROFESSION_ACTION' ? professionActionId : undefined,
            resource_target: selected?.target_type === 'RESOURCE' ? resourceTarget : undefined,
            cost: selected?.ap_cost ?? ACTION_COSTS[actionType]
        };

        try {
            const outcome = await ExecuteAction(action);

            setSilo(outcome.silo);
            setAgent(outcome.agent);
            setAgentStats(outcome.agent_stats);
            setAvailableActions(outcome.available_actions || []);

            if (outcome.game_over) {
                updateResultText(`Game Over: ${outcome.silo.victory_status?.description || outcome.ending_narrative}`);
            } else if (!outcome.result.executed) {
                updateResultText(`Action failed: ${outcome.result.message}`);
            } else {
                let msg = outcome.result.message;
                if (outcome.stories.length > 0) {
                    msg += ` | Event [${outcome.stories[0].category || "uncategorized"}]: ${outcome.stories[0].title}`;
                }
                const duration = selected?.duration_months ?? ACTION_DURATIONS[actionType] ?? 0;
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

    const groupOrder = ['common', 'profession', 'profession_group', 'faction_member', 'faction_leader'];
    const groupTitle: Record<string, string> = {
        common: 'Common Actions',
        profession: `Profession Actions: ${agent?.profession || ''}`,
        profession_group: 'Profession Group Actions',
        faction_member: 'Faction Member Actions',
        faction_leader: 'Faction Leader Actions',
    };
    const groupColor: Record<string, string> = {
        common: 'blue',
        profession: 'purple',
        profession_group: 'orange',
        faction_member: 'teal',
        faction_leader: 'red',
    };
    const orderedGroups = groupOrder.filter((group) => (groupedActions[group] || []).length > 0);
    const selectAction = (action: PlayerActionMeta) => {
        setActionType(action.action_type as AgentActionType);
        setProfessionActionId(action.action_type === 'PROFESSION_ACTION' ? action.id : '');
        setPlayerActionId(action.action_type === 'PLAYER_EVENT' ? action.id : '');
    };

    return (
        <Box minH="100vh" bg="white" color="gray.800" display="flex" flexDirection="column">
            {gameStarted && <HeaderStats agent={agent} silo={silo} agentStats={agentStats} />}
            
            <HStack align="stretch" flex={1} w="full" gap={0} overflow="hidden">
                {/* Sidebar */}
                {gameStarted && (
                    <VStack 
                        w="70px" 
                        bg="gray.50" 
                        borderRight="1px solid" 
                        borderColor="gray.200" 
                        py={6} 
                        gap={6} 
                        align="center"
                    >
                        <Tooltip content="地堡地图" placement="right">
                            <Button 
                                variant={activeView === 'map' ? "solid" : "ghost"} 
                                colorPalette="blue"
                                onClick={() => setActiveView('map')}
                                w="50px"
                                h="50px"
                                borderRadius="lg"
                                p={0}
                            >
                                <LayoutGrid size={24} />
                            </Button>
                        </Tooltip>

                        <Tooltip content="阵营概览" placement="right">
                            <Button 
                                variant={activeView === 'factions' ? "solid" : "ghost"} 
                                colorPalette="blue"
                                onClick={() => setActiveView('factions')}
                                w="50px"
                                h="50px"
                                borderRadius="lg"
                                p={0}
                            >
                                <Users size={24} />
                            </Button>
                        </Tooltip>
                    </VStack>
                )}

                {/* Main Content Area */}
                <Box flex={1} overflowY="auto" p={8}>
                    <VStack gap={8} w="full">
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
                                <Button 
                                    colorPalette="red" 
                                    variant="outline" 
                                    size="sm" 
                                    w="full" 
                                    onClick={handleDebugStart}
                                >
                                    DEBUG: 快速开始 (Silo 20, Just now)
                                </Button>
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
                                {/* Left Side: View Content */}
                                <VStack gap={6} flex={{ base: "1 1 100%", lg: 2 }} w="full">
                                    {activeView === 'map' ? (
                                        silo && <BunkerMap silo={silo} agent={agent} />
                                    ) : (
                                        silo && <FactionView silo={silo} />
                                    )}
                                </VStack>

                                {/* Right Side: Operations */}
                                <VStack gap={6} flex={{ base: "1 1 100%", lg: 1 }} w="full" position="sticky" top="0px">
                                    {/* Actions Panel */}
                                    <Box w="full" p={5} bg="gray.50" borderRadius="md" border="1px solid" borderColor="gray.200" boxShadow="sm">
                                        <Heading size="sm" mb={4} color="gray.800" borderBottom="1px solid" borderColor="gray.200" pb={2}>Agent Action Interface</Heading>
                                        <VStack gap={5} align="stretch">
                                            {orderedGroups.map((group) => (
                                                <VStack key={group} align="start" gap={2}>
                                                    <Text fontSize="sm" fontWeight="bold" color={`${groupColor[group] || 'gray'}.700`}>
                                                        {groupTitle[group] || group}
                                                    </Text>
                                                    <SimpleGrid columns={2} gap={3} w="full">
                                                        {(groupedActions[group] || []).map((action) => {
                                                            const isSelected = selectedActionMeta?.id === action.id && selectedActionMeta?.action_type === action.action_type;
                                                            const palette = groupColor[group] || 'gray';
                                                            return (
                                                                <Button
                                                                    key={`${action.action_type}:${action.id}`}
                                                                    variant={isSelected ? "solid" : "outline"}
                                                                    colorPalette={isSelected ? palette : "gray"}
                                                                    onClick={() => selectAction(action)}
                                                                    h="80px"
                                                                    display="flex"
                                                                    flexDirection="column"
                                                                    justifyContent="center"
                                                                    alignItems="center"
                                                                    whiteSpace="normal"
                                                                    lineHeight="1.2"
                                                                    title={action.description}
                                                                    disabled={!action.enabled}
                                                                    opacity={action.enabled ? 1 : 0.55}
                                                                    bg={isSelected ? `${palette}.500` : "white"}
                                                                    _hover={{ bg: isSelected ? `${palette}.600` : "gray.50" }}
                                                                >
                                                                    <Text fontWeight="bold">{action.label}</Text>
                                                                    <Text fontSize="xs" mt={1}>({action.ap_cost} AP)</Text>
                                                                </Button>
                                                            );
                                                        })}
                                                    </SimpleGrid>
                                                </VStack>
                                            ))}

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
                                                <Button colorPalette="blue" w="full" size="lg" onClick={handleExecuteAction} boxShadow="md" disabled={!selectedActionMeta || !selectedActionMeta.enabled}>
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
                </Box>
            </HStack>
        </Box>
    )
}

export default App;
