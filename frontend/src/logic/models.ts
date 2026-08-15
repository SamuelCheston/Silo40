export type VictoryType =
  | "NONE"
  | "INFORMATION"
  | "TIME"
  | "REBELLION"
  | "EXCLUSIONIST"
  | "DEATH"
  | "AGENT_COMPROMISED";

export interface VictoryStatus {
  is_won: boolean;
  // 后端返回字符串类型 (与 wailsjs 生成类型兼容)
  type: string;
  description: string;
  score?: GameScore;
}

export interface GameScore {
  total: number;
  survival_points: number;
  diversity_points: number;
  heritage_points: number;
  ideology_points: number;
  multiplier: number;
}

export interface User {
  id: number;
  username: string;
  silo_id: number;
  agent?: Agent;
  created_at: string;
  updated_at: string;
}

export interface Agent {
  id: number;
  user_id: number;
  name: string;
  profession: string;
  traits: string[];
  political_prestige: number;
  political_points: number;
  action_points: number; // 行动点数 (用于执行信息传播等操作)
  propaganda_level: number; // 宣传力度
  suspicion_level: number; // 怀疑度指数
  connections: Connection[];
  known_fragments: string[]; // 特工个人掌握的信息碎片
  relics?: Relic[]; // 特工私吞的遗物
  created_at: string;
  updated_at: string;
}

export type AgentActionType =
  | "GATHER_INFO"
  | "SHARE_INFO"
  | "BUILD_CONNECTION"
  | "INCITE_REBELLION"
  | "CONDUCT_PROPAGANDA"
  | "PROFESSION_ACTION";

export const ACTION_COSTS: Record<AgentActionType, number> = {
  GATHER_INFO: 10,
  SHARE_INFO: 20,
  BUILD_CONNECTION: 15,
  INCITE_REBELLION: 30,
  CONDUCT_PROPAGANDA: 20,
  PROFESSION_ACTION: 0, // 实际成本由职业行动注册表 (professionActions.ts) 决定
};

export const ACTION_DURATIONS: Record<AgentActionType, number> = {
  GATHER_INFO: 0, // 即时操作
  SHARE_INFO: 0, // 即时操作
  BUILD_CONNECTION: 1, // 耗时1个月
  INCITE_REBELLION: 2, // 耗时2个月
  CONDUCT_PROPAGANDA: 1, // 耗时1个月
  PROFESSION_ACTION: 1, // 职业专属行动默认为耗时操作
};

export interface AgentAction {
  type: AgentActionType;
  source_dept?: string;
  target_dept?: string;
  fragment_ids?: string[]; // 传播/造谣时使用的碎片数组
  /** 职业专属行动 id (type === 'PROFESSION_ACTION' 时必填) */
  profession_action?: string;
  /** 职业专属行动的资源目标 (如 Supply 的 Ration Allocation) */
  resource_target?: string;
  cost: number;
}

export interface ActionResult {
  executed: boolean;
  message: string;
}

export interface Connection {
  id: number;
  agent_id: number;
  profession_id: number;
  value: number;
}

export interface Silo {
  id: number;
  name: string;
  traits?: string[];
  safeguard_risk?: number; // IT 专属风险系数
  total_population: number;
  legitimacy: number;
  cohesion: number;
  rebellion: number;
  history_burden: number;
  event_trigger: number;
  current_year: number;
  current_month: number;
  countdown: number;
  silo1_destroyed: boolean; // 1号地堡是否已覆灭
  victory_status?: VictoryStatus;
  resources: Resource[];
  professions: Profession[];
  floors: Floor[];
  cohorts?: PopulationCohort[];
  factions?: Faction[];
  relics?: Relic[]; // 地堡中存在的所有遗物
  created_at: string;
  updated_at: string;
}

export interface Resource {
  id: number;
  silo_id: number;
  type: string;
  amount: number;
  net_balance: number;
  updated_at: string;
}

export interface Profession {
  id: number;
  silo_id: number;
  name: string;
  class_type: string; // 'ELITE' | 'COMMONER' (后端返回字符串，与 wailsjs 生成类型兼容)
  population: number;
  ideologies: Record<string, number>;
  panic_value: number;
  productivity: number;
  power_level: number;
  zone: string;
  known_fragments: string[]; // 掌握的其他部门信息碎片来源 (部门名称)
  relations?: Record<string, number>; // NPC部门之间的人脉/关系网
  // ---- 统一 Actor 经济体系 (与 Agent 同构，使 NPC 与玩家共用同一执行管线) ----
  action_points?: number; // 行动点数 (AP)
  suspicion_level?: number; // 怀疑度
  political_prestige?: number; // 政治威望
  propaganda_level?: number; // 宣传力度
  traits?: string[]; // 特质
  relics?: Relic[]; // 部门保管的遗物 (如司法部)
  updated_at: string;
}

export const ALL_FRAGMENTS: string[] = [
  "Mayor_1",
  "Mayor_2",
  "Mayor_3",
  "Mayor_4",
  "Mayor_5",
  "Judicial_1",
  "Judicial_2",
  "Judicial_3",
  "Judicial_4",
  "Judicial_5",
  "IT_1",
  "IT_2",
  "IT_3",
  "IT_4",
  "IT_5",
  "Police_1",
  "Police_2",
  "Medical_1",
  "Medical_2",
  "Mechanical_1",
  "Mechanical_2",
  "Supply_1",
  "Supply_2",
  "Mines_1",
  "Mines_2",
  "Agricultural_1",
];

export interface Relic {
  id: number;
  silo_id: number;
  name: string;
  description: string;
  source_dept: string;
  discovery_year: number;
  effects: Record<string, number>;

  // 归属关系
  agent_id?: number;
  profession_id?: number;

  created_at: string;
  updated_at: string;
}

export interface Floor {
  id: number;
  silo_id: number;
  level: number;
  function: string;
  zone: string;
  stability: number;
  population: number;
  updated_at: string;
}

export interface Faction {
  id: number;
  silo_id: number;
  name: string;
  signature: string;
  tags: string[];
  member_count: number;
  representative_resident_id: number;
  representative_cohort_id?: number;
  representative_name: string;
  influence: number;
  cohesion: number;
  updated_at: string;
}

export interface PopulationCohort {
  id: number;
  silo_id: number;
  profession_id: number;
  faction_id?: number;
  name: string;
  count: number;
  home_zone: string;
  influence: number;
  action_points: number;
  political_prestige: number;
  panic_sensitivity: number;
  ideologies: Record<string, number>;
  known_fragments?: string[];
  tags?: string[];
  updated_at: string;
}

/**
 * 剧情随机事件 (由 Go 后端引擎生成，仅含展示信息；执行逻辑在 Go)
 */
export interface StoryEvent {
  id: string;
  title: string;
  description: string;
  type: string; // 'SOCIAL' | 'TECHNICAL' | 'EXTERNAL'
}

export interface StoryEventLog {
  id: number;
  silo_id: number;
  timestamp: number; // Y100-M1-D1 = 0, mapped with 30 days per month
  year: number;
  month: number;
  source: string; // 'PASS_TIME' | `ACTION:${string}`
  event_id: string;
  title: string;
  description: string;
  type: string; // 'SOCIAL' | 'TECHNICAL' | 'EXTERNAL'
  created_at: string;
  updated_at: string;
}

// ============ 后端通信 DTO (镜像 Go model 包 JSON) ============

export interface CreateGameRequest {
  silo_name: string;
  start_year: number;
  trait_ids: string[];
  agent_name: string;
  profession: string;
}

export interface ProfessionActionMeta {
  id: string;
  profession: string;
  label: string;
  description: string;
  ap_cost: number;
  target_type: string; // 'NONE' | 'DEPT' | 'RESOURCE'
  suspicion_penalty: number;
}

export interface GameState {
  silo: Silo;
  agent: Agent;
  game_over: boolean;
  ending_narrative?: string;
  victory_status?: VictoryStatus;
  profession_actions?: ProfessionActionMeta[];
}

export interface TickResult {
  silo: Silo;
  agent: Agent;
  logs: string[];
  stories: StoryEvent[];
  game_over: boolean;
  ending_narrative?: string;
}

export interface ActionOutcome {
  silo: Silo;
  agent: Agent;
  result: ActionResult;
  logs: string[];
  stories: StoryEvent[];
  game_over: boolean;
  ending_narrative?: string;
}

export interface EventHistoryResult {
  events: StoryEventLog[];
}
