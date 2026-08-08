export type VictoryType = 'NONE' | 'INFORMATION' | 'TIME' | 'REBELLION' | 'EXCLUSIONIST' | 'DEATH' | 'AGENT_COMPROMISED';

export interface VictoryStatus {
  is_won: boolean;
  type: VictoryType;
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
  organization_factor: number; // 组织度系数
  propaganda_level: number; // 宣传力度
  suspicion_level: number; // 怀疑度指数
  connections: Connection[];
  known_fragments: string[]; // 特工个人掌握的信息碎片
  created_at: string;
  updated_at: string;
}

export type AgentActionType = 'GATHER_INFO' | 'SHARE_INFO' | 'BUILD_CONNECTION' | 'INCITE_REBELLION' | 'CONDUCT_PROPAGANDA';

export const ACTION_COSTS: Record<AgentActionType, number> = {
  GATHER_INFO: 10,
  SHARE_INFO: 20,
  BUILD_CONNECTION: 15,
  INCITE_REBELLION: 30,
  CONDUCT_PROPAGANDA: 20,
};

export const ACTION_DURATIONS: Record<AgentActionType, number> = {
  GATHER_INFO: 0, // 即时操作
  SHARE_INFO: 0,  // 即时操作
  BUILD_CONNECTION: 3, // 耗时3个月
  INCITE_REBELLION: 2, // 耗时2个月
  CONDUCT_PROPAGANDA: 1, // 耗时1个月
};

export interface AgentAction {
  type: AgentActionType;
  source_dept?: string;
  target_dept?: string;
  fragment_ids?: string[]; // 传播/造谣时使用的碎片数组
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
  class_type: 'ELITE' | 'COMMONER';
  population: number;
  ideology_value: number;
  panic_value: number;
  productivity: number;
  power_level: number;
  zone: string;
  known_fragments: string[]; // 掌握的其他部门信息碎片来源 (部门名称)
  relations?: Record<string, number>; // NPC部门之间的人脉/关系网
  updated_at: string;
}

export const ALL_FRAGMENTS: string[] = [
  'Mayor_1', 'Mayor_2', 'Mayor_3', 'Mayor_4', 'Mayor_5',
  'Judicial_1', 'Judicial_2', 'Judicial_3', 'Judicial_4', 'Judicial_5',
  'IT_1', 'IT_2', 'IT_3', 'IT_4', 'IT_5',
  'Police_1', 'Police_2',
  'Medical_1', 'Medical_2',
  'Mechanical_1', 'Mechanical_2',
  'Supply_1', 'Supply_2',
  'Mines_1', 'Mines_2',
  'Agricultural_1'
];

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

export interface GameEvent {
  id: string;
  title: string;
  description: string;
  type: 'SOCIAL' | 'TECHNICAL' | 'EXTERNAL';
  effects: (silo: Silo) => void;
}
