import { Silo, Profession, Resource, Floor, Agent } from './models';

export function createInitialSilo(name: string, initialYear: number): Silo {
  const silo: Silo = {
    id: 0,
    name: name,
    total_population: 10000,
    legitimacy: 1.0,
    cohesion: 1.0,
    rebellion: 0.0,
    history_burden: 0.0,
    event_trigger: 0.0,
    current_year: initialYear,
    countdown: 500.0,
    silo1_destroyed: false,
    resources: [],
    professions: [],
    floors: [],
    created_at: '',
    updated_at: '',
  };

  // 1. Initialize 144 Floors
  silo.floors = initFloors();

  // 2. Initialize Core Professions
  silo.professions = initProfessions();

  // 3. Initialize Base Resources
  silo.resources = initResources();

  return silo;
}

function initFloors(): Floor[] {
  const floors: Floor[] = [];
  const configs = [
    { start: 1, end: 10, function: 'Administrative', zone: 'Upper' },
    { start: 11, end: 20, function: 'Public Facilities', zone: 'Upper' },
    { start: 21, end: 30, function: 'Residential A', zone: 'Upper' },
    { start: 31, end: 60, function: 'Residential B', zone: 'Mid' },
    { start: 61, end: 75, function: 'Hydroponics', zone: 'Mid' },
    { start: 76, end: 80, function: 'Medical', zone: 'Mid' },
    { start: 81, end: 90, function: 'Cafeteria & Supply', zone: 'Mid' },
    { start: 91, end: 120, function: 'Residential C', zone: 'Lower' },
    { start: 121, end: 135, function: 'Mechanical', zone: 'Lower' },
    { start: 136, end: 140, function: 'Maintenance', zone: 'Lower' },
    { start: 141, end: 144, function: 'Mines', zone: 'Lower' },
  ];

  configs.forEach((config) => {
    for (let i = config.start; i <= config.end; i++) {
      floors.push({
        id: 0,
        silo_id: 0,
        level: i,
        function: config.function,
        zone: config.zone,
        stability: 1.0,
        population: 0,
        updated_at: '',
      });
    }
  });

  return floors;
}

function initProfessions(): Profession[] {
  const professions: Profession[] = [
    { id: 0, silo_id: 0, name: 'Mayor', power_level: 10, zone: 'Upper', population: 200, ideology_value: 0.5, panic_value: 0.0, productivity: 1.0, known_fragments: [], updated_at: '' },
    { id: 0, silo_id: 0, name: 'Judicial', power_level: 9, zone: 'Upper', population: 400, ideology_value: 0.5, panic_value: 0.0, productivity: 1.0, known_fragments: [], updated_at: '' },
    { id: 0, silo_id: 0, name: 'IT', power_level: 9, zone: 'Upper', population: 600, ideology_value: 0.5, panic_value: 0.0, productivity: 1.0, known_fragments: [], updated_at: '' },
    { id: 0, silo_id: 0, name: 'Sheriff', power_level: 8, zone: 'Upper', population: 300, ideology_value: 0.5, panic_value: 0.0, productivity: 1.0, known_fragments: [], updated_at: '' },
    { id: 0, silo_id: 0, name: 'Medical', power_level: 7, zone: 'Mid', population: 800, ideology_value: 0.5, panic_value: 0.0, productivity: 1.0, known_fragments: [], updated_at: '' },
    { id: 0, silo_id: 0, name: 'Supply', power_level: 6, zone: 'Mid', population: 1200, ideology_value: 0.5, panic_value: 0.0, productivity: 1.0, known_fragments: [], updated_at: '' },
    { id: 0, silo_id: 0, name: 'Mechanical', power_level: 8, zone: 'Lower', population: 1500, ideology_value: 0.5, panic_value: 0.0, productivity: 1.0, known_fragments: [], updated_at: '' },
    { id: 0, silo_id: 0, name: 'Maintenance', power_level: 5, zone: 'Lower', population: 1000, ideology_value: 0.5, panic_value: 0.0, productivity: 1.0, known_fragments: [], updated_at: '' },
    { id: 0, silo_id: 0, name: 'Mines', power_level: 4, zone: 'Lower', population: 2000, ideology_value: 0.5, panic_value: 0.0, productivity: 1.0, known_fragments: [], updated_at: '' },
    { id: 0, silo_id: 0, name: 'Agricultural', power_level: 6, zone: 'Mid', population: 2000, ideology_value: 0.5, panic_value: 0.0, productivity: 1.0, known_fragments: [], updated_at: '' },
  ];
  return professions;
}

export function createInitialAgent(name: string): Agent {
  return {
    id: 1,
    user_id: 1,
    name: name,
    profession: 'Mechanical', // 默认机械部
    traits: ['地堡土著'],
    political_prestige: 10,
    political_points: 0,
    action_points: 50,
    organization_factor: 1.0,
    suspicion_level: 0.0,
    connections: [
      { id: 1, agent_id: 1, profession_id: 6, value: 0.8 }, // 对机械部的初始人脉 (0.0~1.0)
    ],
    known_fragments: ['Mechanical'], // 默认掌握本部门的碎片
    created_at: '',
    updated_at: ''
  };
}

function initResources(): Resource[] {
  const resources: Resource[] = [
    { id: 0, silo_id: 0, type: 'Food', amount: 1000, net_balance: 0, updated_at: '' },
    { id: 0, silo_id: 0, type: 'Energy', amount: 5000, net_balance: 0, updated_at: '' },
    { id: 0, silo_id: 0, type: 'Water', amount: 2000, net_balance: 0, updated_at: '' },
    { id: 0, silo_id: 0, type: 'Materials', amount: 500, net_balance: 0, updated_at: '' },
  ];
  return resources;
}
