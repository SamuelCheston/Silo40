// @ts-nocheck
import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
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
import { LinesPanel } from './components/LinesPanel';
import { Silo, Agent, AgentAction, AgentActionType, GameState, AgentStats, PlayerActionMeta, StoryEventLog } from './logic/models';
import { LayoutGrid, Users } from 'lucide-react';

// 月度反馈里的故事带上来源堡前缀，多堡同月触发相同内容时玩家才分得清是谁家的事。
function formatStory(story: any): string {
    if (story && story.silo_name) {
        return `[${story.silo_name}] ${story.title}`;
    }
    return story ? story.title : '';
}

// 首次关键反馈：阶段跃迁与第一次吸收是"这套东西是活的"的时刻，
// 界面要给够分量——不是弹窗，是常驻一屏的醒目横幅，下一次行动时收起。
const stageMoments: Record<string, { title: string; body: string }> = {
    first_signal: {
        title: '第一次本地观察完成',
        body: '驻地的门开了一条缝——「Share Info」与职务动作已解锁。当你手握确信 B ≥ 0.30 的材料时，把它交给能独立查账的人：事实要到你的叙述之外去成立。',
    },
    archive_awakened: {
        title: '档案苏醒：第一次有本地人用自己的职务语言重新提出问题',
        body: '材料离手了。对方不是在复述你的句子——他们在查自己的账。感染成立；接下来是复制：让接收者培养出下一位接收者。',
    },
    chain_living: {
        title: '活链成立',
        body: '感染、复制、隐匿三者齐备：链不需要你逐日维持也能复制。剩下的是公共事实，以及守住宿主活到结算。',
    },
    chain_public_fact: {
        title: '核心真相已成为公共事实',
        body: '它有了自己的来源说明、自己的核验路径、自己的保管人。你的沉默、离开或失忆，都不再能把它收回。守住宿主，直到结算。',
    },
};
const absorptionMoment = {
    title: '第一次有部门吸收了你的事实',
    body: '事实进入了他们的信念层——开始不依赖你而存在，也开始随时间淡忘。这就是"这套东西是活的"：持续流通，才跑赢淡忘。',
};

// 结局类型 → 可读结局名（对应结局体系 12 的线名与硬失败名）。
// 界面上不允许出现裸枚举。
const endingTitle: Record<string, string> = {
    INFORMATION: '《共证》',
    REBELLION: '《越岭》',
    RULE_OF_LAW: '《公议》',
    TIME: '《无人裁决》',
    PERSONAL_RULE: '《人治》',
    EXCLUSIONIST: '《深闭》',
    RESET: '《遗忘》',
    FRACTURE: '《断层》',
    DEATH: '《寂堡》',
    AGENT_COMPROMISED: '《除名》',
};

function App() {
    const [resultText, setResultText] = useState("在下方输入特工代号 👇");
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
    const [infoProgress, setInfoProgress] = useState<any>(null);
    const [linesProgress, setLinesProgress] = useState<any>(null);
    const [moments, setMoments] = useState<{ title: string; body: string }[]>([]);
    const prevStageRef = useRef<string | null>(null);
    const prevAbsorbedRef = useRef<number | null>(null);
    const prevMainLineRef = useRef<string | null>(null);
    const prevRegressedRef = useRef<Record<string, boolean>>({});

    // 从状态快照里检测"关键时刻"：阶段跃迁、第一次吸收、主线确立与改道、
    // 线回摆。只在已有前值（即非首次加载）时比较，开局与读档不会误报。
    const detectMoments = useCallback((state: any) => {
        // 结局已定：不再弹任何过程性横幅——结局框就是唯一的焦点。
        if (state?.silo?.victory_status) {
            setMoments([]);
            return;
        }
        const fresh: { title: string; body: string }[] = [];
        const stage = state?.world?.progress_stage || null;
        if (prevStageRef.current !== null && stage && stage !== prevStageRef.current) {
            const moment = stageMoments[stage];
            if (moment) fresh.push(moment);
        }
        prevStageRef.current = stage;
        const absorbed = (state?.information_progress?.departments || [])
            .reduce((sum: number, dept: any) => sum + (dept.above_floor || 0), 0);
        if (prevAbsorbedRef.current === 0 && absorbed > 0) {
            fresh.push(absorptionMoment);
        }
        prevAbsorbedRef.current = absorbed;

        // 主线确立与改道：路变了要看得见，不能是"某个数字悄悄不涨"。
        const lp = state?.lines_progress;
        if (lp) {
            const mainLine = lp.main_line || "";
            const titleOf = (id: string) => (lp.lines || []).find((l: any) => l.line === id)?.title || id;
            if (prevMainLineRef.current !== null && mainLine !== prevMainLineRef.current && mainLine) {
                if (prevMainLineRef.current) {
                    fresh.push({
                        title: `路改了：${titleOf(prevMainLineRef.current)} → ${titleOf(mainLine)}`,
                        body: "你脚下的路变了——新主线的里程碑是你最近留下的脚印。旧路没有关闭，它还开着，只是不再是你正在走的那条。",
                    });
                } else {
                    fresh.push({
                        title: `主线确立：${titleOf(mainLine)}`,
                        body: "第一个里程碑点亮：这条路上有了你的脚印。其余的路仍然开着——主线由脚印决定，不由宣言决定。",
                    });
                }
            }
            prevMainLineRef.current = mainLine;

            // 回摆：里程碑仍亮而成立条件不再成立（如《人治》的 W 回升）。
            const regressedNow: Record<string, boolean> = {};
            for (const line of lp.lines || []) {
                regressedNow[line.line] = !!line.regressed;
                if (line.regressed && prevRegressedRef.current[line.line] === false) {
                    fresh.push({
                        title: `${line.title}回摆`,
                        body: `${line.qual_note ? line.qual_note + "。" : ""}里程碑仍然亮着——路没有消失，是秩序摆回去了，要重新推。`,
                    });
                }
            }
            prevRegressedRef.current = regressedNow;
        }

        if (fresh.length > 0) setMoments(fresh);
    }, []);

    // Action Form State
    const [actionType, setActionType] = useState<AgentActionType>('GATHER_INFO');
    const [targetDept, setTargetDept] = useState<string>('');
    const [selectedFragments, setSelectedFragments] = useState<string[]>([]);
    const [professionActionId, setProfessionActionId] = useState<string>('');
    const [playerActionId, setPlayerActionId] = useState<string>('');
    const [resourceTarget, setResourceTarget] = useState<string>('');

    const toggleFragment = useCallback((f: string) => {
        setSelectedFragments(prev => prev.includes(f) ? prev.filter(x => x !== f) : [...prev, f]);
    }, []);

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

    const groupedActions = useMemo(() => availableActions.reduce((acc, action) => {
        if (!acc[action.group]) {
            acc[action.group] = [];
        }
        acc[action.group].push(action);
        return acc;
    }, {} as Record<string, PlayerActionMeta[]>), [availableActions]);

    // 确信 (B) 查找表：碎片够不够格离手要能在徽章上一眼看出来，
    // 而不是点了执行被拒后才知道。
    const beliefOf = useMemo(() => {
        const m: Record<string, number> = {};
        (agent?.beliefs || []).forEach((b: any) => { m[b.fact_id] = b.b; });
        return m;
    }, [agent?.beliefs]);
    // 与 events/mechanics 的 INFO_CFG.qual_rumor 保持一致的分享门槛。
    const SHARE_FLOOR = 0.30; // TODO(balance)

    const selectedActionMeta = useMemo(() => availableActions.find((action) => {
        if (action.action_type !== actionType) return false;
        if (action.action_type === 'PROFESSION_ACTION') return action.id === professionActionId;
        if (action.action_type === 'PLAYER_EVENT' || action.action_type === 'SILO1_OPERATION' || action.action_type === 'STATION_OPERATION') return action.id === playerActionId;
        // 内置通用动作的 id 是 "action_gather_info" 这类定义 ID，与 action_type
        // 不同名；类型本身唯一，按类型匹配即可 (按 id===actionType 永远匹配不上，
        // 会导致选中通用动作后 Execute 恒禁用)。
        return true;
    }), [availableActions, actionType, professionActionId, playerActionId]);
    const showDeptSelector = selectedActionMeta?.target_type === 'DEPT';
    const showResourceSelector = selectedActionMeta?.target_type === 'RESOURCE';

    // 应用后端返回的游戏状态快照
    const applyGameState = useCallback((state: any) => {
        setSilo(state.silo);
        setAgent(state.agent);
        setAgentStats(state.agent_stats);
        setAvailableActions(state.available_actions || []);
        setWorld(state.world || null);
        setInfoProgress(state.information_progress || null);
        setLinesProgress(state.lines_progress || null);
        setPendingEvents(0);
        setPendingOperation('');
        setGameStarted(true);
        setShowSetup(false);
        detectMoments(state);
    }, [detectMoments]);

    const applyEventState = useCallback((state: any) => {
        setSilo(state.silo);
        setAgent(state.agent);
        setAgentStats(state.agent_stats);
        setAvailableActions(state.available_actions || []);
        setWorld(state.world || null);
        setInfoProgress(state.information_progress || null);
        setLinesProgress(state.lines_progress || null);
        setPendingEvents(state.pending_events || 0);
        setPendingOperation(state.pending_operation || '');
        setGameStarted(true);
        setShowSetup(false);
        detectMoments(state);
    }, [detectMoments]);

    useEffect(() => {
        if (!availableActions.some((action) => {
            if (action.action_type !== actionType) return false;
            if (actionType === 'PROFESSION_ACTION') return action.id === professionActionId;
            if (actionType === 'PLAYER_EVENT' || actionType === 'SILO1_OPERATION' || actionType === 'STATION_OPERATION') return action.id === playerActionId;
            return true; // 内置通用动作按类型匹配（类型唯一），见 selectedActionMeta 注释
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
            .catch(err => updateResultText(`碎片清单加载失败：${err}`));
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
                refreshEventHistory().catch(err => updateResultText(`事件历史加载失败：${err}`));
                setResultText(`已恢复 ${state.silo.name} 的存档。当前 ${state.silo.current_year} 年 ${state.silo.current_month} 月。`);
            })
            .catch(err => updateResultText(`存档加载失败：${err}`));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const updateName = useCallback((e: any) => setName(e.target.value), []);
    const updateResultText = useCallback((result: string) => setResultText(result), []);

    const refreshEventHistory = useCallback(async () => {
        const history = await GetEventHistory(40);
        setEventHistory(history.events || []);
    }, []);

    const drainOperation = useCallback(async () => {
        let last: any = null;
        for (let i = 0; i < 256; i++) {
            const step = await ProcessNextEvent();
            last = step;
            if (step.operation_complete) break;
        }
        if (!last || !last.operation_complete) {
            throw new Error('Operation exceeded 256 event steps; save was not attempted.');
        }
        // 只在最终步应用状态：中间事件步不触发整树重渲染，避免一次操作多次重渲染。
        applyEventState(last);
        await refreshEventHistory();
        return last;
    }, [applyEventState, refreshEventHistory]);

    const handleNewGame = useCallback(() => {
        setGameStarted(false);
        setShowSetup(false);
        setSilo(null);
        setAgent(null);
        setWorld(null);
        setEventHistory([]);
        setMoments([]);
        setLinesProgress(null);
        prevStageRef.current = null;
        prevAbsorbedRef.current = null;
        prevMainLineRef.current = null;
        prevRegressedRef.current = {};
        setResultText('开始新一局 Silo 40 行动。创建新局之前，旧存档仍然保留。');
    }, []);

    const handleSave = useCallback(async () => {
        try {
            await SaveGame();
            updateResultText(`已保存 ${silo?.name || '驻地'}（${silo?.current_year} 年 ${silo?.current_month} 月）。`);
        } catch (err) {
            updateResultText(`保存失败：${err}`);
        }
    }, [silo, updateResultText]);

    // 新建游戏：初始化/落库/缓存全部由 Go 后端完成
    const handleSetupComplete = useCallback(async (selectedTraitIds: string[], profession: string) => {
        const agentName = name || "无名特工";
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
            updateResultText(`40 号派出，现驻 ${state.silo?.name || '未知地堡'}。你的行动只作用于驻地；1 号仅开放有限情报与欺骗通道。`);
        } catch (err) {
            updateResultText(`地堡初始化失败：${err}`);
        }
    }, [name, startYear, applyGameState, updateResultText]);

    const handlePassTime = useCallback(async () => {
        if (!silo || !agent || busy) return;

        setBusy(true);
        setMoments([]);
        try {
            const state = await BeginPassTime(1);
            applyEventState(state);
            const step = await drainOperation();
            const stories = (step.stories || []).map(formatStory).join(' | ');
            updateResultText(`已推进至 ${step.silo.current_year} 年 ${step.silo.current_month} 月。${stories ? ` ${stories}` : ' 本月没有新的故事事件。'}`);
        } catch (err) {
            updateResultText(`时间推进失败：${err}`);
        } finally {
            setBusy(false);
        }
    }, [silo, agent, busy, applyEventState, drainOperation, updateResultText]);

    // 表单提交：执行动作 (执行/结算/NPC 回合均在 Go)
    const handleExecuteAction = useCallback(async () => {
        if (!silo || !agent) return;

        const selected = selectedActionMeta;
        const isGlobalAction = selected?.target_type === 'NONE';

        // 目标校验统一以后端下发的 action meta 为准
        let actionTarget: string | undefined = targetDept;
        if (selected?.target_type === 'DEPT') {
            if (!targetDept) {
                updateResultText("请先选择目标部门。");
                return;
            }
        } else if (selected?.target_type === 'RESOURCE') {
            if (!resourceTarget) {
                updateResultText("请先选择目标资源。");
                return;
            }
            actionTarget = resourceTarget;
        } else if (!isGlobalAction && !targetDept) {
            updateResultText("请先选择目标部门。");
            return;
        }
        if (actionType === 'SHARE_INFO' && selectedFragments.length === 0) {
            updateResultText("请至少选择一份要分享的碎片。");
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
        setMoments([]);
        try {
            const state = await BeginAction(action);
            applyEventState(state);
            const step = await drainOperation();
            const result = step.action_result?.message || '动作已结算。';
            const stories = (step.stories || []).map(formatStory).join(' | ');
            updateResultText(`${result}${stories ? ` | ${stories}` : ''}`);
        } catch (err) {
            updateResultText(`动作执行失败：${err}`);
        } finally {
            setBusy(false);
            // 分享执行后清空碎片选中：残留选中会让下一次执行带上玩家没选的
            // 碎片（报错指向意料之外的碎片，"反应在别的地方"）。
            if (actionType === 'SHARE_INFO') setSelectedFragments([]);
        }
    }, [silo, agent, selectedActionMeta, targetDept, resourceTarget, actionType, selectedFragments, professionActionId, playerActionId, busy, applyEventState, drainOperation, updateResultText]);

    const groupOrder = ['station', 'silo1', 'common', 'profession', 'profession_group', 'faction_member', 'faction_leader'];
    const groupTitle: Record<string, string> = {
        silo1: '跨堡行动 · 1 号',
        station: `驻地行动 · ${silo?.name || '加载中…'}`,
        common: '通用动作',
        profession: `职务动作 · ${agent?.profession || ''}`,
        profession_group: '职务圈层动作',
        faction_member: '派系成员动作',
        faction_leader: '派系领袖动作',
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
    // 每个动作分组在目标里的意义：玩家要能回答"我为什么要做这些"。
    const groupNote: Record<string, string> = {
        // 与阶段门控一致：arrival 阶段只有 Gather 可见，Share/职务动作在
        // 第一次本地观察后解锁——注释不能承诺界面上还看不到的按钮。
        common: world?.progress_stage === 'arrival'
            ? '流通的主干动词是取材料（Gather）、交材料（Share）、建信任（Build）。完成第一次「Gather Info」后，Share 与职务动作解锁。'
            : '流通的主干动词：取材料（Gather）、交材料（Share）、建信任（Build）。《共证》线靠它们推进。',
        profession: '职务动作是你的合法外壳与就近渠道：维持岗位身份，部分动作能直接取得或深化材料。',
        profession_group: '职务圈层的协同动作：以岗位身份为掩护，做单人做不了的事。',
        station: '针对驻地的行动：把本地关系转化为可核验的结果。',
        silo1: '面向 1 号的有限通道：只有情报与欺骗——驻地的事要经驻地自己的手。',
        faction_member: '派系成员动作：以公开身份参与派系议程。',
        faction_leader: '派系领袖动作：公开位置换来的杠杆，用之前想好暴露的代价。',
    };
    const orderedGroups = useMemo(() => groupOrder.filter((group) => (groupedActions[group] || []).length > 0), [groupedActions]);
    const selectAction = useCallback((action: PlayerActionMeta) => {
        setActionType(action.action_type as AgentActionType);
        setProfessionActionId(action.action_type === 'PROFESSION_ACTION' ? action.id : '');
        setPlayerActionId(action.action_type === 'PLAYER_EVENT' || action.action_type === 'SILO1_OPERATION' || action.action_type === 'STATION_OPERATION' ? action.id : '');
    }, []);

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

                        {gameStarted && silo && (
                            <Box w="full" p={4} bg="purple.50" border="1px solid" borderColor="purple.300" borderRadius="md">
                                <HStack justify="space-between" wrap="wrap">
                                    <Text fontWeight="bold" color="purple.800">40 号派出，现驻 {silo.name}</Text>
                                    <Text fontSize="sm" color="gray.600">可操作：驻地 + 1 号限定通道 · 40 号与 18 号不可操作</Text>
                                </HStack>
                            </Box>
                        )}

                        {gameStarted && silo?.victory_status && (
                            <Box w="full" p={5} borderRadius="md" bg={silo.victory_status.is_won ? 'green.50' : 'red.50'} border="2px solid" borderColor={silo.victory_status.is_won ? 'green.400' : 'red.400'}>
                                <Heading size="sm" color={silo.victory_status.is_won ? 'green.700' : 'red.700'}>
                                    结局：{endingTitle[silo.victory_status.type] || silo.victory_status.type} · {silo.victory_status.is_won ? '任务成功' : '任务失败'}
                                </Heading>
                                <Text mt={2}>{silo.victory_status.description}</Text>
                                {silo.victory_status.score && (
                                    <Box mt={2}>
                                        <Text fontWeight="bold">总分：{silo.victory_status.score.total}（含历史引导 +{silo.victory_status.score.history_influence?.total || 0}）</Text>
                                        <Text fontSize="xs" color="gray.600" mt={1}>
                                            构成：存续（人口）{silo.victory_status.score.survival_points} ·
                                            生产多样 {silo.victory_status.score.diversity_points} ·
                                            历史遗产 {silo.victory_status.score.heritage_points} ·
                                            思潮开放 {silo.victory_status.score.ideology_points}，
                                            × 结局倍率 {Number(silo.victory_status.score.multiplier).toFixed(1)}
                                        </Text>
                                    </Box>
                                )}
                                {silo.victory_status.decision && (
                                    <Box mt={3} p={3} bg="white" borderRadius="md">
                                        <Text fontSize="sm" fontWeight="bold">多主体权重分流 · 配置 v{silo.victory_status.decision.config_version}</Text>
                                        <Text fontSize="xs" color="gray.600" mt={1}>
                                            {['INFORMATION', 'REBELLION', 'RULE_OF_LAW', 'TIME', 'PERSONAL_RULE', 'EXCLUSIONIST'].map(outcome => `${endingTitle[outcome] || outcome} ${(silo.victory_status.decision.scores[outcome] || 0).toFixed(2)}`).join(' · ')}
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
                                    placeholder="输入特工代号"
                                    value={name}
                                    onChange={updateName}
                                    size="md"
                                    bg="gray.100"
                                    border="1px solid"
                                    borderColor="gray.300"
                                    _focus={{ border: "1px solid", borderColor: "blue.500", bg: "white" }}
                                />
                                <Text textAlign="center" color="gray.600">
                                    你是 40 号地堡的特工，被派往一座仍在运转的地堡。
                                    这一局的目标：让 40 号的覆灭与 1 号的控制方式不再只能由控制者单方叙述——
                                    使真相能被独立核验、被驻地的人保有。你通过情报、关系与隐蔽行动引导驻地，而不是经营人口。
                                </Text>
                                <Button colorPalette="blue" size="lg" w="full" onClick={() => { setShowSetup(true); updateResultText('定制你的特工：选择所属部门与特质，然后确认开局。'); }}>
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
                                    {moments.map(moment => (
                                        <Box key={moment.title} w="full" p={4} bg="green.50" border="2px solid" borderColor="green.400" borderRadius="md">
                                            <Heading size="sm" color="green.800">{moment.title}</Heading>
                                            <Text fontSize="sm" color="gray.700" mt={1}>{moment.body}</Text>
                                        </Box>
                                    ))}
                                    <LinesPanel lines={linesProgress} progress={infoProgress} silo={silo} world={world} agent={agent} />
                                    <WorldPanel world={world} agent={agent} />
                                    {activeView === 'map' ? (
                                        silo && <BunkerMap silo={silo} agent={agent} />
                                    ) : (
                                        silo && <FactionView silo={silo} />
                                    )}
                                </VStack>

                                {/* Right Side: Operations */}
                                <VStack gap={6} flex={{ base: "1 1 100%", lg: 1 }} w="full" position="sticky" top="0px">
                                    {/* Actions Panel — 结局已定后收起操作区，服务端也会拒绝动作 */}
                                    {!silo?.victory_status && (
                                    <Box w="full" p={5} bg="gray.50" borderRadius="md" border="1px solid" borderColor="gray.200" boxShadow="sm">
                                        <Heading size="sm" mb={4} color="gray.800" borderBottom="1px solid" borderColor="gray.200" pb={2}>特工行动台</Heading>
                                        <VStack gap={5} align="stretch">
                                            {orderedGroups.map((group) => (
                                                <VStack key={group} align="start" gap={2}>
                                                    <Text fontSize="sm" fontWeight="bold" color={`${groupColor[group] || 'gray'}.700`}>
                                                        {groupTitle[group] || group}
                                                    </Text>
                                                    {groupNote[group] && (
                                                        <Text fontSize="xs" color="gray.500">{groupNote[group]}</Text>
                                                    )}
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
                                                    <Text fontSize="sm" fontWeight="bold" color="gray.700">目标部门：</Text>
                                                    <NativeSelect.Root size="md" w="full" bg="white">
                                                        <NativeSelect.Field value={targetDept} onChange={(e) => setTargetDept(e.target.value)}>
                                                            <option value="" disabled>选择部门…</option>
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
                                                    <Text fontSize="sm" fontWeight="bold" color="gray.700">目标资源：</Text>
                                                    <NativeSelect.Root size="md" w="full" bg="white">
                                                        <NativeSelect.Field value={resourceTarget} onChange={(e) => setResourceTarget(e.target.value)}>
                                                            <option value="" disabled>选择资源…</option>
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
                                                        <Text fontSize="sm" fontWeight="bold" color="gray.700">要分享的碎片（徽章带确信 B，B ≥ {SHARE_FLOOR.toFixed(2)} 才够格分享）:</Text>
                                                        {(agent?.known_fragments?.length || 0) === 0 && (
                                                            <Text fontSize="xs" color="orange.700">
                                                                你还没有任何可作为事实分享的材料——先用「Gather Info」把确信提到 B ≥ {SHARE_FLOOR.toFixed(2)}，材料才够格离手。
                                                            </Text>
                                                        )}
                                                        <HStack wrap="wrap" gap={2} maxH="200px" overflowY="auto" p={2} border="1px solid" borderColor="gray.200" borderRadius="md" w="full">
                                                            {allFragments.map(f => {
                                                                const isSelected = selectedFragments.includes(f);
                                                                const isKnown = agent?.known_fragments?.includes(f);
                                                                const b = beliefOf[f];
                                                                const shareable = isKnown && typeof b === 'number' && b >= SHARE_FLOOR;
                                                                let palette: string;
                                                                let title: string;
                                                                let label = f;
                                                                if (isKnown) {
                                                                    label = `${f} · B ${(typeof b === 'number' ? b : 0).toFixed(2)}`;
                                                                    if (shareable) {
                                                                        palette = isSelected ? "blue" : "teal";
                                                                        title = `确信 B ${b.toFixed(2)} ≥ ${SHARE_FLOOR.toFixed(2)}：可作为事实分享`;
                                                                    } else {
                                                                        palette = "orange";
                                                                        title = `确信 B ${(typeof b === 'number' ? b : 0).toFixed(2)} < ${SHARE_FLOOR.toFixed(2)}：仍是私人猜测，不能分享——继续对该线「Gather Info」加深确信`;
                                                                    }
                                                                } else {
                                                                    label = `${f}（假）`;
                                                                    palette = isSelected ? "red" : "orange";
                                                                    title = "假碎片：不在你手中。分享会大幅抬升怀疑并降低接收率";
                                                                }
                                                                return (
                                                                    <Badge
                                                                        key={f}
                                                                        colorPalette={palette}
                                                                        variant={isSelected ? "solid" : "subtle"}
                                                                        cursor="pointer"
                                                                        onClick={() => toggleFragment(f)}
                                                                        title={title}
                                                                    >
                                                                        {label}
                                                                    </Badge>
                                                                );
                                                            })}
                                                        </HStack>
                                                    </VStack>
                                                    <Text fontSize="xs" color="gray.600" mt={1}>
                                                        分享假碎片会让思潮传得更快，但会大幅抬升怀疑度、拉低接收率。
                                                    </Text>
                                                </>
                                            )}

                                            <VStack gap={3} mt={2}>
                                                <Button colorPalette="blue" w="full" size="lg" onClick={handleExecuteAction} boxShadow="md" disabled={busy || pendingEvents > 0 || !selectedActionMeta || !selectedActionMeta.enabled}>
                                                    {busy ? '结算中…' : '执行动作'}
                                                </Button>
                                                <Button colorPalette="teal" variant="outline" w="full" size="md" onClick={handlePassTime} bg="white" disabled={busy || pendingEvents > 0}>
                                                    推进 1 个月
                                                </Button>
                                                <Button colorPalette="gray" variant="outline" w="full" size="sm" onClick={handleSave} disabled={busy || pendingEvents > 0}>
                                                    保存并继续
                                                </Button>
                                                <Text fontSize="xs" color="gray.600">
                                                    队列：{pendingEvents} 项待处理{pendingOperation ? ` | ${pendingOperation}` : ''}
                                                </Text>
                                            </VStack>
                                        </VStack>
                                    </Box>
                                    )}

                                    <Box w="full" p={5} bg="gray.50" borderRadius="md" border="1px solid" borderColor="gray.200">
                                        <Heading size="sm" mb={3} color="gray.800">情报与故事记录</Heading>
                                        <VStack align="stretch" gap={3} maxH="360px" overflowY="auto">
                                            {eventHistory.length === 0 && <Text fontSize="sm" color="gray.500">还没有任何记录。你的第一次「Gather Info」与第一次分享会写下最早的条目；此后截获的信号与事件都保留在这里。</Text>}
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
                                Silo40 控制台 · 特工行动
                            </Text>
                        </Box>
                    </VStack>
                </Box>
            </HStack>
        </Box>
    )
}

export default App;
