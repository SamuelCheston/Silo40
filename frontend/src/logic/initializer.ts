import { Silo, Profession, Resource, Floor, Agent, ALL_FRAGMENTS } from './models';

export function createInitialSilo(name: string, initialYear: number, traitIds: string[] = []): Silo {
  const silo: Silo = {
    id: 0,
    name: name,
    traits: traitIds,
    safeguard_risk: 0.0,
    total_population: 10000,
    legitimacy: 1.0,
    cohesion: 1.0,
    rebellion: 0.0,
    history_burden: 0.0,
    event_trigger: 0.0,
    current_year: initialYear,
    current_month: 1,
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
  
  // Initialize NPC-NPC relations and traits based on specific logic
  silo.professions.forEach(prof => {
      if (!prof.relations) prof.relations = {};
      
      // IT: Knows all fragments initially
      if (prof.name === 'IT') {
          prof.known_fragments = [...ALL_FRAGMENTS];
      }

      silo.professions.forEach(otherProf => {
          if (prof.id !== otherProf.id) {
              let baseRelation = 0;
              
              if (prof.name === 'Mayor' || prof.name === 'Judicial') {
                  baseRelation = otherProf.class_type === 'ELITE' ? 0.20 : 0.15;
              } else if (prof.name === 'Police' || prof.name === 'Medical') {
                  baseRelation = otherProf.class_type === 'ELITE' ? 0.15 : 0.10;
              } else if (prof.class_type === 'COMMONER') {
                  if (otherProf.class_type === 'COMMONER') {
                      baseRelation = 0.15;
                  }
              }

              // Specific overrides
              if (prof.name === 'Mayor' && otherProf.name === 'Police') baseRelation = 1.0;
              if (prof.name === 'Police' && otherProf.name === 'Mayor') baseRelation = 1.0;
              
              if (prof.name === 'IT' && otherProf.name === 'Judicial') baseRelation = 1.0;
              if (prof.name === 'Judicial' && otherProf.name === 'IT') baseRelation = 1.0;

              if (prof.relations) {
                  prof.relations[otherProf.name] = baseRelation;
              }
          }
      });
  });

  // 3. Initialize Base Resources
  silo.resources = initResources();

  // Apply Silo Traits
  if (traitIds.includes('abundant')) {
    const food = silo.resources.find(r => r.type === 'Food');
    if (food) food.amount *= 2;
    const energy = silo.resources.find(r => r.type === 'Energy');
    if (energy) energy.amount *= 2;
  }
  
  if (traitIds.includes('leak')) {
    silo.professions.forEach(p => {
      if (p.class_type === 'COMMONER') {
        p.ideology_value = Math.min(1.0, p.ideology_value + 0.3);
      }
    });
  }

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
    { id: 1, silo_id: 0, name: 'Mayor', class_type: 'ELITE', power_level: 10, zone: 'Upper', population: 1, ideology_value: 0.05, panic_value: 0.0, productivity: 1.0, known_fragments: ['Mayor_1', 'Mayor_2', 'Mayor_3', 'Mayor_4', 'Mayor_5'], relations: {}, updated_at: '' },
    { id: 2, silo_id: 0, name: 'Judicial', class_type: 'ELITE', power_level: 9, zone: 'Upper', population: 200, ideology_value: 0.05, panic_value: 0.0, productivity: 1.0, known_fragments: ['Judicial_1', 'Judicial_2', 'Judicial_3', 'Judicial_4', 'Judicial_5'], relations: {}, updated_at: '' },
    { id: 3, silo_id: 0, name: 'IT', class_type: 'ELITE', power_level: 9, zone: 'Upper', population: 150, ideology_value: 0.05, panic_value: 0.0, productivity: 1.0, known_fragments: ['IT_1', 'IT_2', 'IT_3', 'IT_4', 'IT_5'], relations: {}, updated_at: '' },
    { id: 4, silo_id: 0, name: 'Police', class_type: 'ELITE', power_level: 8, zone: 'Upper', population: 250, ideology_value: 0.05, panic_value: 0.0, productivity: 1.0, known_fragments: ['Police_1', 'Police_2'], relations: {}, updated_at: '' },
    { id: 5, silo_id: 0, name: 'Medical', class_type: 'ELITE', power_level: 7, zone: 'Mid', population: 400, ideology_value: 0.05, panic_value: 0.0, productivity: 1.0, known_fragments: ['Medical_1', 'Medical_2'], relations: {}, updated_at: '' },
    { id: 6, silo_id: 0, name: 'Supply', class_type: 'COMMONER', power_level: 6, zone: 'Mid', population: 2500, ideology_value: 0.05, panic_value: 0.0, productivity: 1.0, known_fragments: ['Supply_1', 'Supply_2'], relations: {}, updated_at: '' },
    { id: 7, silo_id: 0, name: 'Mechanical', class_type: 'COMMONER', power_level: 8, zone: 'Lower', population: 3000, ideology_value: 0.05, panic_value: 0.0, productivity: 1.0, known_fragments: ['Mechanical_1', 'Mechanical_2'], relations: {}, updated_at: '' },
    { id: 8, silo_id: 0, name: 'Mines', class_type: 'COMMONER', power_level: 4, zone: 'Lower', population: 1000, ideology_value: 0.05, panic_value: 0.0, productivity: 1.0, known_fragments: ['Mines_1', 'Mines_2'], relations: {}, updated_at: '' },
    { id: 9, silo_id: 0, name: 'Agricultural', class_type: 'COMMONER', power_level: 6, zone: 'Mid', population: 2499, ideology_value: 0.05, panic_value: 0.0, productivity: 1.0, known_fragments: ['Agricultural_1'], relations: {}, updated_at: '' },
  ];
  return professions;
}

export function createInitialAgent(name: string, profession: string, traitIds: string[] = [], silo?: Silo): Agent {
  const agent: Agent = {
    id: 1,
    user_id: 1,
    name: name,
    profession: profession, // Use the selected profession
    traits: [], // start empty
    political_prestige: 10,
    political_points: 0,
    action_points: 50,
    organization_factor: 1.0,
    suspicion_level: 0.0,
    connections: [], // Will populate based on profession below
    known_fragments: ALL_FRAGMENTS.filter(f => f.startsWith(profession + '_')), // 默认掌握本部门的碎片
    created_at: '',
    updated_at: ''
  };

  // Find profession ID from silo if available, otherwise fallback to finding by name (if we hardcoded IDs)
  if (silo && silo.professions) {
      if (profession === 'Mayor' || profession === 'Judicial') {
          agent.political_prestige = profession === 'Mayor' ? 80 : 5; // Mayor high prestige, Judicial almost none
          silo.professions.forEach(p => {
              let randomVal = 0;
              if (p.class_type === 'ELITE') {
                  randomVal = 0.2 + Math.random() * 0.05; // 0.2 - 0.25 (was 0.4 - 0.5)
              } else if (p.class_type === 'COMMONER') {
                  randomVal = 0.15 + Math.random() * 0.05; // 0.15 - 0.2 (was 0.3 - 0.4)
              }
              agent.connections.push({
                  id: Date.now() + p.id,
                  agent_id: agent.id,
                  profession_id: p.id,
                  value: randomVal
              });
          });
      } else if (profession === 'IT') {
          agent.political_prestige = 5; // Almost no political prestige
          agent.known_fragments = [...ALL_FRAGMENTS]; // Knows all fragments
          
          const judicialProf = silo.professions.find(p => p.name === 'Judicial');
          if (judicialProf) {
              agent.connections.push({
                  id: Date.now() + judicialProf.id,
                  agent_id: agent.id,
                  profession_id: judicialProf.id,
                  value: 1.0 // 100% connection with Judicial (restored to 100%)
              });
          }
          const itProf = silo.professions.find(p => p.name === 'IT');
          if (itProf) {
              agent.connections.push({
                  id: Date.now() + itProf.id,
                  agent_id: agent.id,
                  profession_id: itProf.id,
                  value: 0.4 // 40% (was 0.8)
              });
          }
      } else if (profession === 'Police') {
          silo.professions.forEach(p => {
              let connValue = p.class_type === 'ELITE' ? 0.15 : 0.10;
              if (p.name === 'Mayor' || p.name === 'Police') {
                  connValue = 1.0; // 100% with Mayor and own department (restored to 100%)
              }
              agent.connections.push({
                  id: Date.now() + p.id,
                  agent_id: agent.id,
                  profession_id: p.id,
                  value: connValue
              });
          });
      } else if (profession === 'Medical') {
          agent.political_prestige = 5; // Extremely low political prestige
          silo.professions.forEach(p => {
              agent.connections.push({
                  id: Date.now() + p.id,
                  agent_id: agent.id,
                  profession_id: p.id,
                  value: p.class_type === 'ELITE' ? 0.15 : 0.10
              });
          });
      } else {
          // Default connection setup for other professions
          const isCommoner = ['Supply', 'Mechanical', 'Mines', 'Agricultural'].includes(profession);
          
          if (profession === 'Mines') {
              agent.political_prestige = 5;
          }

          silo.professions.forEach(p => {
              if (p.name === profession) {
                  agent.connections.push({
                      id: Date.now() + p.id,
                      agent_id: agent.id,
                      profession_id: p.id,
                      value: 0.4
                  });
              } else if (isCommoner && p.class_type === 'COMMONER') {
                  agent.connections.push({
                      id: Date.now() + p.id,
                      agent_id: agent.id,
                      profession_id: p.id,
                      value: 0.15
                  });
              }
          });
      }
  }

  // Apply Agent Traits
  if (traitIds.includes('native')) {
    agent.traits.push('地堡土著');
    if (silo && silo.professions) {
      const agentProf = silo.professions.find(p => p.name === agent.profession);
      if (agentProf) {
        const classType = agentProf.class_type;
        silo.professions.filter(p => p.class_type === classType).forEach((p, idx) => {
          if (p.id !== agentProf.id) { // not own profession
            const randomVal = 0.10 + Math.random() * 0.05; // 0.10 to 0.15
            agent.connections.push({
              id: Date.now() + idx,
              agent_id: agent.id,
              profession_id: p.id,
              value: randomVal
            });
          }
        });
      }
    }
  }
  if (traitIds.includes('charismatic')) {
    agent.traits.push('魅力非凡');
    agent.political_prestige += 20;
  }
  if (traitIds.includes('shadowy')) {
    agent.traits.push('隐秘行事');
  }

  return agent;
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
