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
  suspicion_level: number; // 怀疑度指数
  connections: Connection[];
  known_fragments: string[]; // 特工个人掌握的信息碎片
  created_at: string;
  updated_at: string;
}

export type AgentActionType = 'GATHER_INFO' | 'SHARE_INFO' | 'BUILD_CONNECTION' | 'INCITE_REBELLION';

export const ACTION_COSTS: Record<AgentActionType, number> = {
  GATHER_INFO: 10,
  SHARE_INFO: 20,
  BUILD_CONNECTION: 15,
  INCITE_REBELLION: 30,
};

export interface AgentAction {
  type: AgentActionType;
  source_dept?: string;
  target_dept?: string;
  fragment_id?: string;
  adulteration_level?: number; // 掺杂信息的程度 (0.0 ~ 1.0)
  cost: number;
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
  total_population: number;
  legitimacy: number;
  cohesion: number;
  rebellion: number;
  history_burden: number;
  event_trigger: number;
  current_year: number;
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
  population: number;
  ideology_value: number;
  panic_value: number;
  productivity: number;
  power_level: number;
  zone: string;
  known_fragments: string[]; // 掌握的其他部门信息碎片来源 (部门名称)
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

export interface GameEvent {
  id: string;
  title: string;
  description: string;
  type: 'SOCIAL' | 'TECHNICAL' | 'EXTERNAL';
  effects: (silo: Silo) => void;
}
