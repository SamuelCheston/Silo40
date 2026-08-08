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
  connections: Connection[];
  created_at: string;
  updated_at: string;
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
  info_fragments: number;
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
