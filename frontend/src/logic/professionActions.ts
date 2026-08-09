import { Silo, ActionResult, ALL_FRAGMENTS } from './models';
import { ActorView } from './actor';

// ============ 职业专属行动注册表 (数据驱动) ============
/**
 * 每个职业拥有 2 个专属行动。行动定义独立于执行管线，
 * 玩家与 NPC 通过统一 Actor 管线 (engine.executeActionInternal) 提交执行，
 * AP 成本 / 怀疑度惩罚由 engine 统一结算，effect 只负责世界状态变更。
 */

export type ProfessionActionTargetType = 'NONE' | 'DEPT' | 'RESOURCE';

export interface ProfessionActionDefinition {
  /** 唯一 id (AgentAction.profession_action) */
  id: string;
  /** 所属职业 (须与 Profession.name / Agent.profession 一致) */
  profession: string;
  /** UI 按钮标题 (英文) */
  label: string;
  /** UI tooltip 描述 (英文) */
  description: string;
  apCost: number;
  targetType: ProfessionActionTargetType;
  /** 行动的基础怀疑度惩罚 (执行管线还会叠加职业修正) */
  suspicionPenalty: number;
  effect: (silo: Silo, view: ActorView, target?: string) => ActionResult;
}

// ---- 通用数值辅助 ----
const clamp = (v: number, min = 0, max = 1) => Math.max(min, Math.min(max, v));

function findDept(silo: Silo, name: string) {
  return silo.professions?.find(p => p.name === name);
}

function addResource(silo: Silo, type: string, amount: number) {
  const res = silo.resources?.find(r => r.type === type);
  if (res) res.amount += amount;
}

/** 从指定部门的信息碎片池中随机获取 count 个未知碎片给 Actor */
function gainFragment(view: ActorView, deptName: string, count = 1): string[] {
  const gained: string[] = [];
  const pool = ALL_FRAGMENTS.filter(f => f.startsWith(deptName + '_'));
  const unknown = pool.filter(f => !view.knownFragments.includes(f));
  for (let i = 0; i < count && unknown.length > 0; i++) {
    const idx = Math.floor(Math.random() * unknown.length);
    const frag = unknown.splice(idx, 1)[0];
    view.knownFragments.push(frag);
    gained.push(frag);
  }
  return gained;
}

/** 随机移除目标部门 count 个已掌握的碎片 (阻碍信息胜利) */
function removeFragments(silo: Silo, deptName: string, count = 1): string[] {
  const dept = findDept(silo, deptName);
  if (!dept) return [];
  const pool = [...(dept.known_fragments || [])];
  const removed: string[] = [];
  for (let i = 0; i < count && pool.length > 0; i++) {
    const idx = Math.floor(Math.random() * pool.length);
    const frag = pool.splice(idx, 1)[0];
    dept.known_fragments = (dept.known_fragments || []).filter(f => f !== frag);
    removed.push(frag);
  }
  return removed;
}

export const PROFESSION_ACTIONS: ProfessionActionDefinition[] = [
  // ================= Mayor (政治领袖) =================
  {
    id: 'MAYOR_PUBLIC_ADDRESS',
    profession: 'Mayor',
    label: 'Public Address',
    description: 'Deliver a rousing speech to the silo. Legitimacy +6%, all departments panic -4%, cohesion +3%.',
    apCost: 15,
    targetType: 'NONE',
    suspicionPenalty: 0.01,
    effect: (silo) => {
      silo.legitimacy = clamp(silo.legitimacy + 0.06);
      silo.cohesion = clamp(silo.cohesion + 0.03);
      silo.professions?.forEach(p => { p.panic_value = clamp(p.panic_value - 0.04); });
      return { executed: true, message: 'Public address delivered. Legitimacy rose and panic eased across the silo.' };
    },
  },
  {
    id: 'MAYOR_DIRECT_ORDER',
    profession: 'Mayor',
    label: 'Direct Order',
    description: 'Issue an executive order to a department. Target productivity +10%, panic -5%.',
    apCost: 20,
    targetType: 'DEPT',
    suspicionPenalty: 0.02,
    effect: (silo, _view, target) => {
      const dept = findDept(silo, target || '');
      if (!dept) return { executed: false, message: 'Target department not found.' };
      dept.productivity = clamp(dept.productivity + 0.10);
      dept.panic_value = clamp(dept.panic_value - 0.05);
      return { executed: true, message: `Executive order issued. ${dept.name} productivity improved.` };
    },
  },

  // ================= Judicial (司法部) =================
  {
    id: 'JUDICIAL_SEARCH_WARRANT',
    profession: 'Judicial',
    label: 'Search Warrant',
    description: 'Serve a search warrant on a department to seize intel. Gain 1 fragment; target panic +3%.',
    apCost: 15,
    targetType: 'DEPT',
    suspicionPenalty: 0.02,
    effect: (silo, view, target) => {
      const dept = findDept(silo, target || '');
      if (!dept) return { executed: false, message: 'Target department not found.' };
      const gained = gainFragment(view, dept.name, 1);
      dept.panic_value = clamp(dept.panic_value + 0.03);
      if (gained.length === 0) {
        return { executed: false, message: `Nothing new was found while searching ${dept.name}.` };
      }
      return { executed: true, message: `Search warrant executed. Seized intel on ${gained[0]} from ${dept.name}.` };
    },
  },
  {
    id: 'JUDICIAL_ARREST',
    profession: 'Judicial',
    label: 'Arrest',
    description: 'Arrest key figures in a department. Target action points -20, panic +10%, legitimacy +4%.',
    apCost: 25,
    targetType: 'DEPT',
    suspicionPenalty: 0.03,
    effect: (silo, _view, target) => {
      const dept = findDept(silo, target || '');
      if (!dept) return { executed: false, message: 'Target department not found.' };
      dept.action_points = Math.max(0, (dept.action_points || 0) - 20);
      dept.panic_value = clamp(dept.panic_value + 0.10);
      silo.legitimacy = clamp(silo.legitimacy + 0.04);
      return { executed: true, message: `Arrests carried out in ${dept.name}. The law is reaffirmed.` };
    },
  },

  // ================= IT (IT部门) =================
  {
    id: 'IT_SURVEILLANCE',
    profession: 'IT',
    label: 'Surveillance',
    description: 'Place a department under full surveillance. Connection +15% with the target (leverage of fear).',
    apCost: 15,
    targetType: 'DEPT',
    suspicionPenalty: 0,
    effect: (silo, view, target) => {
      const dept = findDept(silo, target || '');
      if (!dept) return { executed: false, message: 'Target department not found.' };
      view.addConnection(dept.id, 0.15);
      return { executed: true, message: `${dept.name} is now under surveillance. You hold leverage over them.` };
    },
  },
  {
    id: 'IT_COVER_UP',
    profession: 'IT',
    label: 'Cover-Up',
    description: 'Erase sensitive records from a department. Remove 1-2 fragments (blocks the information victory); safeguard risk +3%.',
    apCost: 25,
    targetType: 'DEPT',
    suspicionPenalty: 0,
    effect: (silo, _view, target) => {
      const dept = findDept(silo, target || '');
      if (!dept) return { executed: false, message: 'Target department not found.' };
      const count = Math.random() < 0.5 ? 1 : 2;
      const removed = removeFragments(silo, dept.name, count);
      dept.panic_value = clamp(dept.panic_value + 0.05);
      silo.safeguard_risk = (silo.safeguard_risk || 0) + 0.03;
      if (removed.length === 0) {
        return { executed: false, message: `${dept.name} holds no records worth erasing.` };
      }
      return { executed: true, message: `Records erased from ${dept.name}: ${removed.join(', ')}. Safeguard risk grows...` };
    },
  },

  // ================= Police (警察) =================
  {
    id: 'POLICE_INTERROGATE',
    profession: 'Police',
    label: 'Interrogate',
    description: 'Interrogate detainees from a department. Gain 1 fragment; target panic +5%.',
    apCost: 15,
    targetType: 'DEPT',
    suspicionPenalty: 0.02,
    effect: (silo, view, target) => {
      const dept = findDept(silo, target || '');
      if (!dept) return { executed: false, message: 'Target department not found.' };
      const gained = gainFragment(view, dept.name, 1);
      dept.panic_value = clamp(dept.panic_value + 0.05);
      if (gained.length === 0) {
        return { executed: false, message: `Interrogation yielded nothing new from ${dept.name}.` };
      }
      return { executed: true, message: `Interrogation extracted intel on ${gained[0]} from ${dept.name}.` };
    },
  },
  {
    id: 'POLICE_CRACKDOWN',
    profession: 'Police',
    label: 'Crackdown',
    description: 'Suppress a department by force. Target panic -15%, ideology -5%, productivity -3%.',
    apCost: 25,
    targetType: 'DEPT',
    suspicionPenalty: 0.03,
    effect: (silo, _view, target) => {
      const dept = findDept(silo, target || '');
      if (!dept) return { executed: false, message: 'Target department not found.' };
      dept.panic_value = clamp(dept.panic_value - 0.15);
      dept.ideology_value = clamp(dept.ideology_value - 0.05);
      dept.productivity = clamp(dept.productivity - 0.03);
      return { executed: true, message: `Crackdown executed in ${dept.name}. Order restored, at a cost.` };
    },
  },

  // ================= Medical (医疗部) =================
  {
    id: 'MEDICAL_TREAT',
    profession: 'Medical',
    label: 'Community Treatment',
    description: 'Deploy medics to a department. Target panic -12%, productivity +5%.',
    apCost: 15,
    targetType: 'DEPT',
    suspicionPenalty: 0.01,
    effect: (silo, _view, target) => {
      const dept = findDept(silo, target || '');
      if (!dept) return { executed: false, message: 'Target department not found.' };
      dept.panic_value = clamp(dept.panic_value - 0.12);
      dept.productivity = clamp(dept.productivity + 0.05);
      return { executed: true, message: `Medics deployed to ${dept.name}. Panic eased and health improved.` };
    },
  },
  {
    id: 'MEDICAL_QUARANTINE',
    profession: 'Medical',
    label: 'Quarantine',
    description: 'Quarantine a department "for its own safety". Target panic -20%, but productivity -12% and ideology -4%.',
    apCost: 20,
    targetType: 'DEPT',
    suspicionPenalty: 0.02,
    effect: (silo, _view, target) => {
      const dept = findDept(silo, target || '');
      if (!dept) return { executed: false, message: 'Target department not found.' };
      dept.panic_value = clamp(dept.panic_value - 0.20);
      dept.productivity = clamp(dept.productivity - 0.12);
      dept.ideology_value = clamp(dept.ideology_value - 0.04);
      return { executed: true, message: `${dept.name} placed under quarantine. The silence is oppressive.` };
    },
  },

  // ================= Supply (供给部) =================
  {
    id: 'SUPPLY_RATION',
    profession: 'Supply',
    label: 'Ration Allocation',
    description: 'Reallocate stockpiles. Add +150 to a chosen resource (Energy / Materials / Supplies).',
    apCost: 15,
    targetType: 'RESOURCE',
    suspicionPenalty: 0.01,
    effect: (silo, _view, target) => {
      if (!target) return { executed: false, message: 'Invalid resource target.' };
      addResource(silo, target, 150);
      return { executed: true, message: `Reallocated stockpiles. +150 ${target}.` };
    },
  },
  {
    id: 'SUPPLY_SHELTER',
    profession: 'Supply',
    label: 'Shelter',
    description: 'Smuggle a department under your protection. Target panic -10%, connection +15%, productivity +5%.',
    apCost: 20,
    targetType: 'DEPT',
    suspicionPenalty: 0.01,
    effect: (silo, view, target) => {
      const dept = findDept(silo, target || '');
      if (!dept) return { executed: false, message: 'Target department not found.' };
      dept.panic_value = clamp(dept.panic_value - 0.10);
      dept.productivity = clamp(dept.productivity + 0.05);
      view.addConnection(dept.id, 0.15);
      return { executed: true, message: `${dept.name} is now sheltered by the Supply network.` };
    },
  },

  // ================= Mechanical (机械部) =================
  {
    id: 'MECHANICAL_OVERHAUL',
    profession: 'Mechanical',
    label: 'Overhaul',
    description: 'Overhaul the silo machinery. Energy +60, Materials +30, own productivity +5%.',
    apCost: 15,
    targetType: 'NONE',
    suspicionPenalty: 0.01,
    effect: (silo, view) => {
      addResource(silo, 'Energy', 60);
      addResource(silo, 'Materials', 30);
      view.productivity = clamp((view.productivity || 1) + 0.05);
      return { executed: true, message: 'Machinery overhauled. Energy and materials production improved.' };
    },
  },
  {
    id: 'MECHANICAL_PIPE_TAP',
    profession: 'Mechanical',
    label: 'Pipe Tap',
    description: 'Eavesdrop through the pipes that carry every whisper. Gain 1 fragment from a department.',
    apCost: 15,
    targetType: 'DEPT',
    suspicionPenalty: 0.02,
    effect: (silo, view, target) => {
      const dept = findDept(silo, target || '');
      if (!dept) return { executed: false, message: 'Target department not found.' };
      const gained = gainFragment(view, dept.name, 1);
      if (gained.length === 0) {
        return { executed: false, message: `The pipes carried nothing new about ${dept.name}.` };
      }
      return { executed: true, message: `Eavesdropped through the pipes. Learned about ${gained[0]} from ${dept.name}.` };
    },
  },

  // ================= Mines (矿工) =================
  {
    id: 'MINES_DEEP_EXCAVATION',
    profession: 'Mines',
    label: 'Deep Excavation',
    description: 'Push the mines deeper. Materials +120, own productivity +5%.',
    apCost: 15,
    targetType: 'NONE',
    suspicionPenalty: 0.005,
    effect: (silo, view) => {
      addResource(silo, 'Materials', 120);
      view.productivity = clamp((view.productivity || 1) + 0.05);
      return { executed: true, message: 'Deep excavation completed. Materials reserves increased.' };
    },
  },
  {
    id: 'MINES_TUNNEL_NETWORK',
    profession: 'Mines',
    label: 'Tunnel Network',
    description: 'Spin a network through the lower tunnels. All commoner departments connection +10%, ideology +3%.',
    apCost: 20,
    targetType: 'NONE',
    suspicionPenalty: 0.005,
    effect: (silo, view) => {
      silo.professions?.forEach(p => {
        if (p.class_type === 'COMMONER') {
          view.addConnection(p.id, 0.10);
          p.ideology_value = clamp(p.ideology_value + 0.03);
        }
      });
      return { executed: true, message: 'The tunnel network hums with new alliances and whispered hopes.' };
    },
  },

  // ================= Agricultural (农业) =================
  {
    id: 'AGRICULTURAL_INTENSIVE_HARVEST',
    profession: 'Agricultural',
    label: 'Intensive Harvest',
    description: 'Work the hydroponics around the clock. Supplies +200, own productivity +8%.',
    apCost: 15,
    targetType: 'NONE',
    suspicionPenalty: 0.01,
    effect: (silo, view) => {
      addResource(silo, 'Supplies', 200);
      view.productivity = clamp((view.productivity || 1) + 0.08);
      return { executed: true, message: 'Intensive harvest completed. Supplies increased.' };
    },
  },
  {
    id: 'AGRICULTURAL_FIELD_GOSSIP',
    profession: 'Agricultural',
    label: 'Field Gossip',
    description: 'Let rumors ripen in the fields. Gain 1 fragment from a department; target ideology +3%.',
    apCost: 10,
    targetType: 'DEPT',
    suspicionPenalty: 0.015,
    effect: (silo, view, target) => {
      const dept = findDept(silo, target || '');
      if (!dept) return { executed: false, message: 'Target department not found.' };
      const gained = gainFragment(view, dept.name, 1);
      dept.ideology_value = clamp(dept.ideology_value + 0.03);
      if (gained.length === 0) {
        return { executed: false, message: `The fields whispered nothing new about ${dept.name}.` };
      }
      return { executed: true, message: `Rumors spread from the fields. Heard about ${gained[0]} from ${dept.name}.` };
    },
  },
];

export function getProfessionActions(profession: string): ProfessionActionDefinition[] {
  return PROFESSION_ACTIONS.filter(a => a.profession === profession);
}

export function getProfessionAction(id: string): ProfessionActionDefinition | undefined {
  return PROFESSION_ACTIONS.find(a => a.id === id);
}
