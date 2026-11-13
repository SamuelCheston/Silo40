// @ts-nocheck
import { useState, useEffect } from 'react';
import './App.css';
import { CreateGame, GetGameState, HasActiveGame, BeginPassTime, BeginAction, ProcessNextEvent, GetAllFragments, GetEventHistory, SaveGame } from "../wailsjs/go/main/App";
import { Box, Button, Heading, Input, Text, VStack, HStack, Badge, SimpleGrid, NativeSelect } from "@chakra-ui/react";
import { ProgressBar, ProgressRoot } from './components/ui/progress';
import { Tooltip } from './components/ui/tooltip';
import { SetupPanel } from './components/SetupPanel';
import { BunkerMap } from './components/BunkerMap';
import { FactionView } from './components/FactionView';
import { HeaderStats } from './components/HeaderStats';
import { WorldPanel } from './components/WorldPanel';
import { Silo, Agent, AgentAction, AgentActionType, GameState, AgentStats, PlayerActionMeta, StoryEventLog } from './logic/models';
import { LayoutGrid, Users } from 'lucide-react';

function App() {
    const [resultText, setResultText] = useState("Please enter your name below 👇");
    const [name, setName] = useState('');
    const [gameStarted, setGameStarted] = useState(false);
    const [showSetup, setShowSetup] = useState(false);
    const [startYear] = useState(122);
    const [silo, setSilo] = useState<Silo | null>(null);
    const [agent, setAgent] = useState<Agent | null>(null);
    const [agentStats, setAgentStats] = useState<AgentStats | null>(null);
    const [availableActions, setAvailableActions] = useState<PlayerActionMeta[]>([]);
    const [activeView, setActiveView] = useState<'map' | 'factions'>('map');
    const [pendingEvents, setPendingEvents] = useState(0);
    const [pendingOperation, setPendingOperation] = useState('');
    const [eventHistory, setEventHistory] = useState<StoryEventLog[]>([]);
    const [world, setWorld] = useState<any>(null);
    const [busy, setBusy] = useState(false);

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
        if (action.action_type === 'PLAYER_EVENT' || action.action_type === 'SILO1_OPERATION' || action.action_type === 'STATION_OPERATION') return action.id === playerActionId;
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
        setWorld(state.world || null);
        setPendingEvents(0);
        setPendingOperation('');
        setGameStarted(true);
        setShowSetup(false);
    };

    const applyEventState = (state: any) => {
        setSilo(state.silo);
        setAgent(state.agent);
        setAgentStats(state.agent_stats);
        setAvailableActions(state.available_actions || []);
        setWorld(state.world || null);
        setPendingEvents(state.pending_events || 0);
        setPendingOperation(state.pending_operation || '');
        setGameStarted(true);
        setShowSetup(false);
    };

    useEffect(() => {
        if (!availableActions.some((action) => {
            if (action.action_type !== actionType) return false;
            if (actionType === 'PROFESSION_ACTION') return action.id === professionActionId;
            if (actionType === 'PLAYER_EVENT' || actionType === 'SILO1_OPERATION' || actionType === 'STATION_OPERATION') return action.id === playerActionId;
            return action.id === actionType;
        })) {
            const fallback = availableActions.find(action => action.enabled) || availableActions[0];
            if (!fallback) return;
            setActionType(fallback.action_type as AgentActionType);
            setProfessionActionId(fallback.action_type === 'PROFESSION_ACTION' ? fallback.id : '');
            setPlayerActionId(fallback.action_type === 'PLAYER_EVENT' || fallback.action_type === 'SILO1_OPERATION' || fallback.action_type === 'STATION_OPERATION' ? fallback.id : '');
        }
    }, [availableActions, actionType, professionActionId, playerActionId]);

    // 碎片清单从后端权威加载 (不再维护前端静态副本)
    const [allFragments, setAllFragments] = useState<string[]>([]);
    useEffect(() => {
        GetAllFragments()
            .then(setAllFragments)
            .catch(err => updateResultText(`Failed to load fragments: ${err}`));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

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
                refreshEventHistory().catch(err => updateResultText(`Failed to load event history: ${err}`));
                setResultText(`Resumed ${state.silo.name}. The year is ${state.silo.current_year}, Month ${state.silo.current_month}.`);
            })
            .catch(err => updateResultText(`Failed to load saved game: ${err}`));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const updateName = (e: any) => setName(e.target.value);
    const updateResultText = (result: string) => setResultText(result);

    const refreshEventHistory = async () => {
        const history = await GetEventHistory(40);
        setEventHistory(history.events || []);
    };

    const drainOperation = async () => {
        for (let i = 0; i < 256; i++) {
            const step = await ProcessNextEvent();
            applyEventState(step);
            if (step.operation_complete) {
                await refreshEventHistory();
                return step;
            }
        }
        throw new Error('Operation exceeded 256 event steps; save was not attempted.');
    };

    const handleNewGame = () => {
        setGameStarted(false);
        setShowSetup(false);
        setSilo(null);
        setAgent(null);
        setWorld(null);
        setEventHistory([]);
        setResultText('Start a new Silo 40 operation. Your previous save remains until the new game is created.');
    };

    const handleSave = async () => {
        try {
            await SaveGame();
            updateResultText(`Saved ${world?.station_silo?.name || 'station'} at Y${silo?.current_year} M${silo?.current_month}.`);
        } catch (err) {
            updateResultText(`Save failed: ${err}`);
        }
    };

    // 新建游戏：初始化/落库/缓存全部由 Go 后端完成
    const handleSetupComplete = async (selectedTraitIds: string[], profession: string) => {
        const agentName = name || "Operative";
        const siloName = "ASSIGNED_STATION";

        try {
            const state = await CreateGame({
                silo_name: siloName,
                start_year: startYear,
                trait_ids: selectedTraitIds,
                agent_name: agentName,
                profession,
                include_silo1: true,
                neighbor: {},
            });
            applyGameState(state);
            setEventHistory([]);
            updateResultText(`40 号派出，现驻 ${state.world?.station_silo?.name || '未知地堡'}。你的行动只作用于驻地；1 号仅开放有限情报与欺骗通道。`);
        } catch (err) {
            updateResultText(`Failed to initialize silo: ${err}`);
        }
    };

    const handlePassTime = async () => {
        if (!silo || !agent || busy) return;

        setBusy(true);
        try {
            const state = await BeginPassTime(1);
            applyEventState(state);
            const step = await drainOperation();
            const stories = (step.stories || []).map((story: any) => story.title).join(' | ');
            updateResultText(`Advanced to Y${step.silo.current_year} M${step.silo.current_month}.${stories ? ` ${stories}` : ' No new story event.'}`);
        } catch (err) {
            updateResultText(`Failed to advance time: ${err}`);
        } finally {
            setBusy(false);
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
            action_id: actionType === 'PLAYER_EVENT' || actionType === 'SILO1_OPERATION' || actionType === 'STATION_OPERATION' ? playerActionId : undefined,
            target_dept: isGlobalAction ? undefined : actionTarget,
            fragment_ids: actionType === 'SHARE_INFO' ? selectedFragments : undefined,
            profession_action: actionType === 'PROFESSION_ACTION' ? professionActionId : undefined,
            resource_target: selected?.target_type === 'RESOURCE' ? resourceTarget : undefined,
            cost: selected?.ap_cost ?? 0
        };

        setBusy(true);
        try {
            const state = await BeginAction(action);
            applyEventState(state);
            const step = await drainOperation();
            const result = step.action_result?.message || 'Action resolved.';
            const stories = (step.stories || []).map((story: any) => story.title).join(' | ');
            updateResultText(`${result}${stories ? ` | ${stories}` : ''}`);
        } catch (err) {
            updateResultText(`Failed to execute action: ${err}`);
        } finally {
            setBusy(false);
        }
    };

    const groupOrder = ['station', 'silo1', 'common', 'profession', 'profession_group', 'faction_member', 'faction_leader'];
    const groupTitle: Record<string, string> = {
        silo1: 'Cross-Silo Operations: Silo 1',
        station: `Assigned Station Operations: ${world?.station_silo?.name || 'Loading...'}`,
        common: 'Common Actions',
        profession: `Profession Actions: ${agent?.profession || ''}`,
        profession_group: 'Profession Group Actions',
        faction_member: 'Faction Member Actions',
        faction_leader: 'Faction Leader Actions',
    };
    const groupColor: Record<string, string> = {
        silo1: 'red',
        station: 'purple',
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
        setPlayerActionId(action.action_type === 'PLAYER_EVENT' || action.action_type === 'SILO1_OPERATION' || action.action_type === 'STATION_OPERATION' ? action.id : '');
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

                        {gameStarted && world?.station_silo && (
                            <Box w="full" p={4} bg="purple.50" border="1px solid" borderColor="purple.300" borderRadius="md">
                                <HStack justify="space-between" wrap="wrap">
                                    <Text fontWeight="bold" color="purple.800">40 号派出，现驻 {world.station_silo.name}</Text>
                                    <Text fontSize="sm" color="gray.600">可操作：驻地 + 1 号限定通道 · 40 号与 18 号不可操作</Text>
                                </HStack>
                            </Box>
                        )}

                        {gameStarted && silo?.victory_status && (
                            <Box w="full" p={5} borderRadius="md" bg={silo.victory_status.is_won ? 'green.50' : 'red.50'} border="2px solid" borderColor={silo.victory_status.is_won ? 'green.400' : 'red.400'}>
                                <Heading size="sm" color={silo.victory_status.is_won ? 'green.700' : 'red.700'}>
                                    结局：{silo.victory_status.type} · {silo.victory_status.is_won ? '任务成功' : '任务失败'}
                                </Heading>
                                <Text mt={2}>{silo.victory_status.description}</Text>
                                {silo.victory_status.score && <Text mt={2} fontWeight="bold">总分：{silo.victory_status.score.total}（含历史引导 +{silo.victory_status.score.history_influence?.total || 0}）</Text>}
                                {silo.victory_status.decision && (
                                    <Box mt={3} p={3} bg="white" borderRadius="md">
                                        <Text fontSize="sm" fontWeight="bold">多主体权重分流 · 配置 v{silo.victory_status.decision.config_version}</Text>
                                        <Text fontSize="xs" color="gray.600" mt={1}>
                                            {['INFORMATION', 'TIME', 'REBELLION', 'EXCLUSIONIST'].map(outcome => `${outcome} ${(silo.victory_status.decision.scores[outcome] || 0).toFixed(2)}`).join(' · ')}
                                        </Text>
                                    </Box>
                                )}
                                {silo.victory_status.score?.history_influence && (
                                    <Box mt={4} p={4} bg="white" borderRadius="md" border="1px solid" borderColor="blue.200">
                                        <Heading size="sm" color="blue.700">历史引导评分 · {silo.victory_status.score.history_influence.grade}</Heading>
                                        <Text fontSize="sm" color="gray.600" mt={2}>{silo.victory_status.score.history_influence.summary}</Text>
                                        <VStack align="stretch" gap={3} mt={3}>
                                            {silo.victory_status.score.history_influence.silos.map((breakdown: any) => (
                                                <Box key={breakdown.silo_id} p={3} bg="gray.50" borderRadius="md">
                                                    <HStack justify="space-between">
                                                        <Text fontWeight="bold">{breakdown.name}</Text>
                                                        <Badge colorPalette={breakdown.points > 0 ? 'blue' : 'gray'}>+{breakdown.points} · {breakdown.operation_count} 次行动</Badge>
                                                    </HStack>
                                                    <Text fontSize="sm" color="gray.600" mt={2}>{breakdown.summary}</Text>
                                                </Box>
                                            ))}
                                        </VStack>
                                    </Box>
                                )}
                                <Button mt={3} colorPalette="blue" onClick={handleNewGame}>开始新一局</Button>
                            </Box>
                        )}

                        {!gameStarted && !showSetup && (
                            <VStack gap={6} w="full" maxW="400px">
                                <Input
                                    placeholder="Enter operative name"
                                    value={name}
                                    onChange={updateName}
                                    size="md"
                                    bg="gray.100"
                                    border="1px solid"
                                    borderColor="gray.300"
                                    _focus={{ border: "1px solid", borderColor: "blue.500", bg: "white" }}
                                />
                                <Text textAlign="center" color="gray.600">
                                    你是 40 号地堡的特工。通过情报、关系与隐蔽行动引导其他地堡，而不是经营人口。
                                </Text>
                                <Button colorPalette="blue" size="lg" w="full" onClick={() => setShowSetup(true)}>
                                    以 Silo 40 特工身份开局
                                </Button>
                            </VStack>
                        )}

                        {!gameStarted && showSetup && (
                            <SetupPanel onComplete={handleSetupComplete} />
                        )}

                        {gameStarted && (
                            <HStack align="start" gap={6} w="full" wrap="wrap">
                                {/* Left Side: View Content */}
                                <VStack gap={6} flex={{ base: "1 1 100%", lg: 2 }} w="full">
                                    <WorldPanel world={world} agent={agent} />
                                    {activeView === 'map' ? (
                                        silo && <BunkerMap silo={world?.station_silo || silo} agent={agent} />
                                    ) : (
                                        silo && <FactionView silo={world?.station_silo || silo} />
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
                                                                    title={action.disabled_reason || action.description}
                                                                    aria-disabled={!action.enabled}
                                                                    opacity={action.enabled ? 1 : 0.55}
                                                                    bg={isSelected ? `${palette}.500` : "white"}
                                                                    _hover={{ bg: isSelected ? `${palette}.600` : "gray.50" }}
                                                                >
                                                                    <Text fontWeight="bold">{action.label}</Text>
                                                                    <Text fontSize="xs" mt={1}>({action.ap_cost} AP)</Text>
                                                                    {action.disabled_reason && <Text fontSize="xs" color="red.600" mt={1}>{action.disabled_reason}</Text>}
                                                                </Button>
                                                            );
                                                        })}
                                                    </SimpleGrid>
                                                </VStack>
                                            ))}

                                            {selectedActionMeta && (
                                                <Box p={3} bg="white" borderRadius="md" border="1px solid" borderColor="gray.200">
                                                    <Text fontSize="sm" color="gray.600">{selectedActionMeta.description}</Text>
                                                    <Text fontSize="xs" color="gray.500" mt={1}>AP {selectedActionMeta.ap_cost} · 本地耗时 {selectedActionMeta.duration_months} 月</Text>
                                                </Box>
                                            )}

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
                                                            {allFragments.map(f => {
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
                                                <Button colorPalette="blue" w="full" size="lg" onClick={handleExecuteAction} boxShadow="md" disabled={busy || pendingEvents > 0 || !selectedActionMeta || !selectedActionMeta.enabled}>
                                                    {busy ? 'Resolving…' : 'Execute Action'}
                                                </Button>
                                                <Button colorPalette="teal" variant="outline" w="full" size="md" onClick={handlePassTime} bg="white" disabled={busy || pendingEvents > 0}>
                                                    Advance 1 Month
                                                </Button>
                                                <Button colorPalette="gray" variant="outline" w="full" size="sm" onClick={handleSave} disabled={busy || pendingEvents > 0}>
                                                    Save & Continue
                                                </Button>
                                                <Text fontSize="xs" color="gray.600">
                                                    Queue: {pendingEvents} pending{pendingOperation ? ` | ${pendingOperation}` : ''}
                                                </Text>
                                            </VStack>
                                        </VStack>
                                    </Box>

                                    <Box w="full" p={5} bg="gray.50" borderRadius="md" border="1px solid" borderColor="gray.200">
                                        <Heading size="sm" mb={3} color="gray.800">情报与故事记录</Heading>
                                        <VStack align="stretch" gap={3} maxH="360px" overflowY="auto">
                                            {eventHistory.length === 0 && <Text fontSize="sm" color="gray.500">推进月份后，截获的信号与事件会保留在这里。</Text>}
                                            {eventHistory.map(event => (
                                                <Box key={`${event.timestamp}:${event.silo_id}:${event.event_id}`} p={3} bg="white" borderRadius="md" borderLeft="3px solid" borderColor="blue.300">
                                                    <Text fontSize="xs" color="gray.500">Y{event.year} M{event.month} · {world?.silos?.find((ref: any) => ref.id === event.silo_id)?.name || `Silo ${event.silo_id}`} · {event.category}</Text>
                                                    <Text fontWeight="bold" fontSize="sm">{event.title}</Text>
                                                    <Text fontSize="sm" color="gray.600">{event.description}</Text>
                                                </Box>
                                            ))}
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
