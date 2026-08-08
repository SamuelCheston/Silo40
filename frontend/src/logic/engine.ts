import { Silo, Agent, Profession, Resource, VictoryStatus, VictoryType, GameEvent, AgentAction, ALL_FRAGMENTS, ActionResult } from './models';
import { EventEngine } from './events';

export class GameEngine {
  private eventEngine: EventEngine = new EventEngine();

  // 每万人每年消耗基数
  private readonly perCapitaConsumption: Record<string, number> = {
    Food: 50.0,
    Energy: 100.0,
    Water: 80.0,
    Materials: 20.0,
  };

  // 职业对资源的产出贡献
  private readonly resourceProducers: Record<string, string[]> = {
    Food: ['Agricultural'],
    Energy: ['Mechanical', 'IT'],
    Water: ['Mechanical', 'Supply'],
    Materials: ['Mines', 'Mechanical'],
  };

  // 职业威望加成系数
  private readonly professionFactors: Record<string, number> = {
    Mayor: 0.5,
    Judicial: 0.4,
    IT: 0.3,
    Police: 0.3,
    Mechanical: 0.2,
    Medical: 0.2,
  };

  // 特质威望加成系数
  private readonly traitFactors: Record<string, number> = {
    地堡土著: 0.1,
    一号地堡密使: 0.5,
    煽动者: 0.2,
    守旧派: -0.1,
  };

  // 更新特工数值逻辑
  public updateAgentState(agent: Agent, deltaYears: number, silo?: Silo, addLog?: (msg: string) => void): void {
    if (deltaYears <= 0) return;

    // Medical profession passive trait: randomly gain information fragments over time
    if (agent.profession === 'Medical' && silo && addLog) {
      if (Math.random() < 0.2) { // 20% chance per year
        const availableFragments = ALL_FRAGMENTS.filter(f => !agent.known_fragments.includes(f));
        if (availableFragments.length > 0) {
          const randomFragment = availableFragments[Math.floor(Math.random() * availableFragments.length)];
          agent.known_fragments.push(randomFragment);
          addLog(`[Medical Passive] Your medical duties allowed you to overhear rumors, gaining information about ${randomFragment}.`);
        }
      }
    }

    // 1. 计算平均人脉值 (0.0 - 1.0)
    let totalConnection = 0;
    const count = agent.connections?.length || 0;
    if (count > 0) {
      agent.connections.forEach((conn) => {
        totalConnection += conn.value;
      });
      totalConnection /= count;
    }

    // 2. 计算职业修正系数
    const profFactor = this.professionFactors[agent.profession] || 0;

    // 3. 计算特质修正系数
    let traitFactor = 0;
    agent.traits?.forEach((trait) => {
      traitFactor += this.traitFactors[trait] || 0;
    });

    // 4. 计算政治威望
    // 因为 totalConnection 现在是 0.0~1.0，乘以 100 使得基础威望保持在 0~100 范围
    agent.political_prestige = totalConnection * 100 * (1 + profFactor) * (1 + traitFactor);
    if (agent.political_prestige < 0) agent.political_prestige = 0;

    // 5. 给予政治点数和行动点数 (AP)
    const pointGainRate = 0.1;
    agent.political_points += agent.political_prestige * pointGainRate * deltaYears;
    
    // 行动点数恢复：基础恢复 10 点/年，受威望和组织度加成
    const apGainRate = 10 + (agent.political_prestige * 0.05) + (agent.organization_factor * 2);
    agent.action_points += apGainRate * deltaYears;
    // 设置 AP 上限
    const maxAp = 100 + (agent.organization_factor * 10);
    if (agent.action_points > maxAp) {
      agent.action_points = maxAp;
    }

    // 6. 怀疑度随时间衰减
    // 如果特工保持低调（不执行激进行动），怀疑度会自然下降
    const suspicionDecayRate = 0.05; // 每年降低5%
    if (agent.suspicion_level > 0) {
      agent.suspicion_level -= suspicionDecayRate * deltaYears;
      if (agent.suspicion_level < 0) agent.suspicion_level = 0;
    }
  }

  // 特工执行动作 (信息获取与传播)
  public executeAgentAction(silo: Silo, agent: Agent, action: AgentAction): ActionResult {
    if (agent.action_points < action.cost) {
      return { executed: false, message: "Not enough Action Points (AP)." };
    }

    const preSuspicion = agent.suspicion_level || 0;
    let result: ActionResult = { executed: false, message: "" };

    switch (action.type) {
      case 'GATHER_INFO':
        result = this.gatherInformation(silo, agent, action);
        break;
      case 'SHARE_INFO':
        result = this.shareInformation(silo, agent, action);
        break;
      case 'BUILD_CONNECTION':
        result = this.buildConnection(silo, agent, action);
        break;
      case 'INCITE_REBELLION':
        result = this.inciteRebellion(silo, agent, action);
        break;
      case 'CONDUCT_PROPAGANDA':
        result = this.conductPropaganda(silo, agent, action);
        break;
    }

    if (result.executed) {
      let gained = (agent.suspicion_level || 0) - preSuspicion;
      
      // 基础行为怀疑度惩罚 (兜底产生)
      if (action.type === 'INCITE_REBELLION') gained += 0.05;
      else if (action.type === 'SHARE_INFO') gained += 0.01;
      else if (action.type === 'BUILD_CONNECTION') gained += 0.01;
      else if (action.type === 'GATHER_INFO') gained += 0.005;
      else if (action.type === 'CONDUCT_PROPAGANDA') gained += 0.02;

      // 职业修正
      if (agent.profession === 'Mayor') {
        gained *= 3.0;
      } else if (agent.profession === 'IT') {
        gained = 0; // IT部门行动不增加怀疑度
      } else if (agent.profession === 'Police') {
        // Police 随机获得 5-9 折减免 (即 0.5 到 0.9 之间的乘数)
        const discount = 0.5 + Math.random() * 0.4;
        gained *= discount;
      } else if (agent.profession === 'Mines') {
        // Mines 行动怀疑度打 0.05 折
        gained *= 0.05;
      }

      // 特质修正
      if (agent.traits?.includes('隐秘行事')) {
        gained *= 0.8;
      }

      agent.suspicion_level = preSuspicion + gained;
      
      // IT 专属机制：恶化 safeguard 风险系数
      if (agent.profession === 'IT') {
        silo.safeguard_risk = (silo.safeguard_risk || 0) + (action.cost * 0.002);
      }
    }

    return result;
  }

  // 特工建立或强化与目标部门的人脉
  private buildConnection(silo: Silo, agent: Agent, action: AgentAction): ActionResult {
    if (!action.target_dept) return { executed: false, message: "Invalid target department." };

    const targetProf = silo.professions?.find(p => p.name === action.target_dept);
    if (!targetProf) return { executed: false, message: "Target department not found." };

    if (!agent.connections) agent.connections = [];
    
    let connection = agent.connections.find(c => c.profession_id === targetProf.id);
    if (!connection) {
      connection = { id: Date.now(), agent_id: agent.id, profession_id: targetProf.id, value: 0 };
      agent.connections.push(connection);
    }

    // 建立人脉的效率受特工政治威望影响
    let increaseValue = 0.05 + (agent.political_prestige * 0.005);
    if (agent.traits?.includes('魅力非凡')) {
      increaseValue *= 1.5; // 额外提升建立速度
    }
    connection.value += increaseValue;
    if (connection.value > 1.0) connection.value = 1.0; // 人脉上限 1.0

    agent.action_points -= action.cost;
    return { executed: true, message: `Successfully built connections with ${targetProf.name}.` };
  }

  // 特工煽动底层叛乱（全局增加所有平民阶层的恐慌和排外/亲外极端化）
  private inciteRebellion(silo: Silo, agent: Agent, action: AgentAction): ActionResult {
    const commoners = silo.professions?.filter(p => p.class_type === 'COMMONER') || [];
    if (commoners.length === 0) return { executed: false, message: "No commoner departments found to incite." };

    commoners.forEach(prof => {
        const connection = agent.connections?.find(c => c.profession_id === prof.id);
        const connectionValue = connection ? connection.value : 0;

        // 煽动效果受人脉值、政治威望和宣传力度影响
        const baseEffect = 0.05 + (agent.political_prestige * 0.002);
        const propagandaMultiplier = 1 + (agent.propaganda_level || 0) * 0.2; // 宣传力度额外加成
        const multiplier = (1 + connectionValue) * propagandaMultiplier;
        const finalEffect = baseEffect * multiplier;

        prof.panic_value += finalEffect;
        prof.ideology_value += finalEffect * 0.5;
    });

    agent.action_points -= action.cost;
    return { executed: true, message: `Incited unrest among all commoner departments.` };
  }

  // 特工主动进行宣传，提升宣传力度
  private conductPropaganda(silo: Silo, agent: Agent, action: AgentAction): ActionResult {
    // 每次宣传提升 1.0 的宣传力度
    agent.propaganda_level = (agent.propaganda_level || 0) + 1.0;
    agent.action_points -= action.cost;
    return { executed: true, message: `Conducted propaganda. Propaganda Level increased by 1.0.` };
  }

  // 特工搜集其他部门的信息碎片
  private gatherInformation(silo: Silo, agent: Agent, action: AgentAction): ActionResult {
    if (!action.target_dept) return { executed: false, message: "Invalid target department." };

    if (!agent.known_fragments) agent.known_fragments = [];
    
    // 随机获取目标部门的一个碎片
    const targetFragments = ALL_FRAGMENTS.filter(f => f.startsWith(action.target_dept! + '_'));
    const unknownTargetFragments = targetFragments.filter(f => !agent.known_fragments?.includes(f));

    if (unknownTargetFragments.length > 0) {
      const fragmentToGather = unknownTargetFragments[Math.floor(Math.random() * unknownTargetFragments.length)];
      agent.known_fragments.push(fragmentToGather);
      agent.action_points -= action.cost;
      return { executed: true, message: `Gathered intel on ${fragmentToGather}.` };
    }

    return { executed: false, message: `Your department already knows everything about ${action.target_dept}.` };
  }

  // 特工将自己掌握的信息碎片分享给目标部门
  private shareInformation(silo: Silo, agent: Agent, action: AgentAction): ActionResult {
    if (!action.target_dept || !action.fragment_ids || action.fragment_ids.length === 0) {
      return { executed: false, message: "Invalid target or no fragments selected." };
    }

    const targetProf = silo.professions?.find(p => p.name === action.target_dept);
    if (!targetProf) return { executed: false, message: "Target department not found." };

    const connection = agent.connections?.find(c => c.profession_id === targetProf.id);
    const connectionValue = connection ? connection.value : 0;

    // AP 即使被拒绝也会消耗
    agent.action_points -= action.cost;

    // 计算实际拥有和凭空掌握的碎片
    const unexplainedFragments = action.fragment_ids.filter(id => !agent.known_fragments.includes(id));
    const unexplainedCount = unexplainedFragments.length;

    // 根据凭空掌握的碎片数量计算直接增加的怀疑度 (指数级增长)
    if (unexplainedCount > 0) {
      const suspicionPenalty = (unexplainedCount * 0.1) + (Math.pow(unexplainedCount, 1.5) * 0.05);
      agent.suspicion_level = (agent.suspicion_level || 0) + suspicionPenalty;
    }

    // 计算对方的接受度 (接受度受亲外度、双方人脉影响)
    // 如果提供的凭空碎片太多，显得过于可疑，也会降低接受度
    let acceptanceRate = 0.1 + targetProf.ideology_value + connectionValue;
    acceptanceRate -= (unexplainedCount * 0.1); 
    if (acceptanceRate < 0.05) acceptanceRate = 0.05;
    if (acceptanceRate > 1.0) acceptanceRate = 1.0;

    const roll = Math.random();
    if (roll > acceptanceRate) {
        return { 
            executed: true, // AP 和 suspicion 已经扣除
            message: `Attempted to share info with ${targetProf.name}, but they rejected it! (Acceptance rate was ${(acceptanceRate*100).toFixed(0)}%)` 
        };
    }

    // 接受成功，目标部门获得所有提供的碎片 (包括真实的和伪造的)
    if (!targetProf.known_fragments) targetProf.known_fragments = [];
    for (const f of action.fragment_ids) {
        if (!targetProf.known_fragments.includes(f)) {
            targetProf.known_fragments.push(f);
        }
    }

    // 目标部门受到信息冲击，恐慌和亲外度上升
    // 凭空掌握的碎片越多，造成的冲击越大 (煽动性越强)
    const panicImpact = 0.05 + unexplainedCount * 0.05;
    targetProf.panic_value = Math.min(1.0, targetProf.panic_value + panicImpact);
    
    if (connectionValue >= 0.3) {
      const ideologyImpact = 0.02 + unexplainedCount * 0.02;
      targetProf.ideology_value = Math.min(1.0, targetProf.ideology_value + ideologyImpact);
    }

    return { 
      executed: true, 
      message: `Successfully shared ${action.fragment_ids.length} fragments with ${targetProf.name}. (Included ${unexplainedCount} pieces of unexplained knowledge)` 
    };
  }

  // 核心逻辑更新引擎
  public updateSiloState(silo: Silo, deltaYears: number, agent?: Agent, addLog?: (msg: string) => void): GameEvent[] {
    const events: GameEvent[] = [];
    if (deltaYears <= 0) return events;

    // 1. 运作条件校验 (技术传承)
    this.checkOperationalConditions(silo, deltaYears);

    // 2. 更新部门生产力与资源结余
    this.updateResources(silo, deltaYears);

    // 3. 更新地堡状态
    this.updateSiloMetrics(silo, deltaYears);

    // 4. 更新思潮演化
    this.updateIdeology(silo, deltaYears);

    // 4.5 模拟其他部门的自主行为 (NPC 意志)
    if (agent) {
      this.triggerNPCActions(silo, agent, deltaYears, addLog);
    }

    // 5. 判定胜利路径
    this.checkVictoryConditions(silo, agent);

    // 6. 随机事件触发
    const event = this.eventEngine.triggerRandomEvent(silo);
    if (event) {
      events.push(event);
    }

    // 7. 如果游戏结束，计算最终评分
    if (silo.victory_status?.is_won !== undefined && !silo.victory_status.score) {
      silo.victory_status.score = this.calculateScore(silo);
    }

    return events;
  }

  public getOrganizedPopulation(silo: Silo, agent: Agent): number {
    let organizedPopulation = 0;
    if (agent.connections && agent.connections.length > 0) {
      agent.connections.forEach(conn => {
        let orgFactor = agent.organization_factor || 1.0;
        if (agent.traits?.includes('魅力非凡')) {
          orgFactor *= 1.2; // 组织化力量额外增益 20%
        }

        const targetProf = silo.professions?.find(p => p.id === conn.profession_id);
        if (targetProf) {
          const isAgentCommoner = ['Supply', 'Mechanical', 'Mines', 'Agricultural'].includes(agent.profession);
          
          if (isAgentCommoner && targetProf.class_type === 'COMMONER') {
             if (agent.profession === 'Mechanical') {
                 orgFactor *= 2.0; // Mechanical is highest tech commoner, stronger org bonus
             } else {
                 orgFactor *= 1.5; // Default commoner organizing commoner bonus
             }
          }

          // 计算对该部门的基础号召力 (Appeal)
          let appeal = 0.1; // 基础号召力
          
          // 根据设定，只有机械部出身的特工对机械部有额外的号召力加成
          if (agent.profession === 'Mechanical' && targetProf.name === 'Mechanical') {
              appeal += 0.4;
          }

          // 特质号召力加成
          if (agent.traits?.includes('魅力非凡')) {
              appeal += 0.2;
          }

          // 引入宣传力度作为乘数
          // 宣传力度默认为 0，所以如果不进行宣传，号召力加成将为 0
          // 这里使用 (propaganda_level) 作为乘数
          const propagandaMultiplier = agent.propaganda_level || 0;

          // 最终部门转化率：号召力与人脉的综合体现 (上限控制在合理范围)
          // 取号召力(带宣传乘数)和人脉的加权，再乘以组织度
          const appealEffect = appeal * propagandaMultiplier;
          const conversionRate = (appealEffect * 0.4 + conn.value * 0.6) * orgFactor * targetProf.ideology_value;
          
          // 计算该部门加入组织的实际人数，且加入上限限制（最多转化该部门 20% 的人口）
          const maxConvertible = targetProf.population * 0.20;
          let deptOrganized = targetProf.population * conversionRate;
          
          if (deptOrganized > maxConvertible) {
              deptOrganized = maxConvertible;
          }
          
          organizedPopulation += deptOrganized;
        }
      });
    }
    return Math.floor(organizedPopulation);
  }

  // 模拟 NPC 部门的自主行为
  public triggerNPCActions(silo: Silo, agent: Agent, deltaYears: number, addLog?: (msg: string) => void): void {
    if (!silo.professions) return;

    silo.professions.forEach(prof => {
      // 不模拟玩家控制的部门
      if (prof.name === agent.profession) return;

      // Medical 专属被动：随机获取碎片
      if (prof.name === 'Medical') {
          if (Math.random() < 0.2) {
              const unknownFragments = ALL_FRAGMENTS.filter(f => !prof.known_fragments?.includes(f));
              if (unknownFragments.length > 0) {
                  const newFrag = unknownFragments[Math.floor(Math.random() * unknownFragments.length)];
                  if (!prof.known_fragments) prof.known_fragments = [];
                  prof.known_fragments.push(newFrag);
                  if (Math.random() < 0.5 && addLog) {
                      addLog(`Medical (NPC) overheard rumors and gained intel on ${newFrag}`);
                  }
              }
          }
      }

      // 只有亲外度(Pro-Foreign)较高的部门，才有意愿去收集或泄露情报
      // 内部维稳(压制恐慌)不受此限制
      
      const ideology = prof.ideology_value;
      const willToAct = ideology > 0.4 ? ideology : 0.1; // 亲外度小于0.4时行动意愿极低

      // 基础行动概率受亲外度、权力等级和时间流逝影响
      const actionChance = willToAct * (0.1 + prof.power_level * 0.05) * deltaYears;
      
      if (Math.random() < actionChance) {
        const actionType = Math.random();
        
        if (actionType < 0.4 && ideology > 0.4) {
          // 40% 概率：收集情报 (Gather Info)
          const unknownFragments = ALL_FRAGMENTS.filter(f => !prof.known_fragments?.includes(f));
          if (unknownFragments.length > 0) {
            const newFrag = unknownFragments[Math.floor(Math.random() * unknownFragments.length)];
            if (!prof.known_fragments) prof.known_fragments = [];
            prof.known_fragments.push(newFrag);
            if (Math.random() < 0.4 && addLog) {
                addLog(`${prof.name} is secretly gathering intel on ${newFrag}`);
            }
          }
        } else if (actionType < 0.8 && ideology > 0.4) {
          // 40% 概率：分享情报 (Share Info)
          if (prof.known_fragments && prof.known_fragments.length > 0) {
            const fragmentToShare = prof.known_fragments[Math.floor(Math.random() * prof.known_fragments.length)];
            
            // 挑选目标：NPC 只有与另一个部门的人脉极高 (>= 0.8) 时才会互通有无
            const targetCandidates = silo.professions!.filter(p => 
              p.id !== prof.id && 
              prof.relations && prof.relations[p.name] >= 0.8
            );

            if (targetCandidates.length > 0) {
              const targetProf = targetCandidates[Math.floor(Math.random() * targetCandidates.length)];
              
              if (!targetProf.known_fragments) targetProf.known_fragments = [];
              if (!targetProf.known_fragments.includes(fragmentToShare)) {
                targetProf.known_fragments.push(fragmentToShare);
                
                // NPC 分享信息也会引起目标部门的恐慌值和亲外度变化
                targetProf.panic_value = Math.min(1.0, targetProf.panic_value + 0.05);
                targetProf.ideology_value = Math.min(1.0, targetProf.ideology_value + 0.02);
                
                if (Math.random() < 0.8 && addLog) {
                    addLog(`${prof.name} confidentially shared ${fragmentToShare} secrets with ${targetProf.name}`);
                }
              }
            }
          }
        } else {
          // 20% 概率 或 亲外度不够时的保底行为：内部管理 (Internal Management)
          if (prof.panic_value > 0.1) {
              prof.panic_value = Math.max(0, prof.panic_value - 0.15);
              if (Math.random() < 0.4 && addLog) {
                  addLog(`${prof.name} suppressed their internal panic`);
              }
          } else {
              prof.productivity = Math.min(1.0, prof.productivity + 0.05);
          }
        }
      }
    });
  }

  private calculateScore(silo: Silo): any {
    const survival_points = silo.total_population * 1;
    
    const diversity_points = (silo.professions?.filter(p => p.productivity > 0.5).length || 0) * 100;
    
    const heritage_points = Math.floor((1.0 - silo.history_burden) * 500);
    
    let avgIdeology = 0;
    silo.professions?.forEach(p => avgIdeology += p.ideology_value);
    avgIdeology /= (silo.professions?.length || 1);
    const ideology_points = Math.floor(avgIdeology * 200);

    let multiplier = 1.0;
    switch (silo.victory_status?.type) {
      case 'INFORMATION': multiplier = 2.0; break;
      case 'TIME': multiplier = 1.5; break;
      case 'REBELLION': multiplier = 1.2; break;
      case 'EXCLUSIONIST': multiplier = 0.5; break;
      case 'DEATH': multiplier = 0; break;
      case 'AGENT_COMPROMISED': multiplier = 0; break;
    }

    const total = Math.floor((survival_points + diversity_points + heritage_points + ideology_points) * multiplier);

    return {
      total,
      survival_points,
      diversity_points,
      heritage_points,
      ideology_points,
      multiplier
    };
  }

  public getEndingNarrative(silo: Silo): string {
    if (!silo.victory_status) return "地堡的故事仍在继续...";

    let narrative = silo.victory_status.description + "\n\n";

    // 补充社会状态描述
    const proForeignRatio = (silo.professions?.filter(p => p.ideology_value > 0.5).length || 0) / (silo.professions?.length || 1);
    
    if (proForeignRatio > 0.7) {
      narrative += "地堡社会展现出了前所未有的开放性，人们渴望与外界建立联系。";
    } else if (proForeignRatio < 0.2) {
      narrative += "地堡社会深陷排外情绪，人们对任何来自外部的事物都充满敌意。";
    } else {
      narrative += "地堡社会在保守与开放之间艰难地维持着平衡。";
    }

    if (silo.history_burden > 0.5) {
      narrative += " 沉重的历史包袱如阴影般笼罩着每一个人，文明的进步举步维艰。";
    } else {
      narrative += " 过去的一页已被翻开，新的一代正以轻松的姿态面对未来。";
    }

    return narrative;
  }

  private checkVictoryConditions(silo: Silo, agent?: Agent): void {
    if (silo.victory_status?.is_won) return;

    // 0. Safeguard 危机 (IT专属)
    if (silo.safeguard_risk !== undefined && silo.safeguard_risk >= 1.0) {
      silo.victory_status = {
        is_won: false,
        type: 'DEATH',
        description: 'Safeguard 协议被激活。IT部门的过度干预触发了底层核心逻辑，清理程序启动，40号地堡被彻底清洗。',
      };
      return;
    }

    // 1. 信息胜利：每个部门至少获得5个其他部门的信息碎片
    let allDeptsHaveFragments = true;
    if (silo.professions && silo.professions.length > 0) {
      for (const prof of silo.professions) {
        // 去重后检查碎片数量
        const uniqueFragments = new Set(prof.known_fragments || []);
        if (uniqueFragments.size < 5) {
          allDeptsHaveFragments = false;
          break;
        }
      }
    } else {
      allDeptsHaveFragments = false;
    }

    if (allDeptsHaveFragments) {
      silo.victory_status = {
        is_won: true,
        type: 'INFORMATION',
        description: '你成功让真相在所有部门间流传。全知视角的拼图终于拼凑完整，地堡的居民迎来了觉醒的黎明。',
      };
      return;
    }

    // 2. 时间胜利判定：由“1号地堡覆灭”事件触发后结算 (后续详细实现)
    if (silo.silo1_destroyed) {
       silo.victory_status = {
         is_won: true,
         type: 'TIME',
         description: '一号地堡已经覆灭，控制网络断开。40号地堡迎来了属于自己的时间。',
       };
       return;
    }

    // 3. 叛乱胜利
    if (agent && silo.total_population > 0) {
      // 检查特工怀疑度是否超过阈值 (判定为个人失败)
      const SUSPICION_THRESHOLD = 1.0;
      if (agent.suspicion_level >= SUSPICION_THRESHOLD) {
        silo.victory_status = {
          is_won: false,
          type: 'AGENT_COMPROMISED',
          description: '由于传播过多掺杂了个人意图的虚假信息，你的特工身份彻底暴露。司法部已经下达了逮捕令。',
        };
        return;
      }

      // 计算组织人数
      let organizedPopulation = this.getOrganizedPopulation(silo, agent);

      // 叛乱基础条件：组织人数达到总人口 3%
      if (organizedPopulation >= silo.total_population * 0.03) {
        
        // 胜利条件 A：最终有至少 3% 任意人口幸存
        const hasEnoughSurvivors = silo.total_population >= 10000 * 0.03; // 假设初始10000人，3%即300人

        // 胜利条件 B：多部门劳动力逃离 (至少3个部门逃离倾向人数 > 10人)
        let escapingDeptsCount = 0;
        silo.professions?.forEach(p => {
          const escapingPeople = p.population * p.ideology_value; // 亲外度 * 部门总人数
          if (escapingPeople > 10) {
            escapingDeptsCount++;
          }
        });
        const hasLaborEscape = escapingDeptsCount >= 3;

        if (hasEnoughSurvivors || hasLaborEscape) {
          silo.victory_status = {
            is_won: true,
            type: 'REBELLION',
            description: '你成功组织了反抗力量并发动了叛乱。旧的统治被推翻，幸存者们冲破了封闭的牢笼。',
          };
          return;
        }
      }
    }

    // 4. 失败判定 (人口灭绝)
    if (silo.total_population <= 0) {
      silo.victory_status = {
        is_won: false,
        type: 'DEATH',
        description: '地堡内已无生命迹象。人类最后的堡垒沦为了一座寂静的坟墓。',
      };
      return;
    }
  }

  private checkOperationalConditions(silo: Silo, deltaYears: number): void {
    const proForeignDepts = silo.professions?.filter(p => p.ideology_value >= 0.1).length || 0;
    
    // 技术传承判定：至少三个部门保持开放思潮
    if (proForeignDepts < 3) {
      // 历史包袱增加
      silo.history_burden += 0.05 * deltaYears;
      
      // 生产力缓慢下降
      silo.professions?.forEach(p => {
        p.productivity -= 0.02 * deltaYears;
        if (p.productivity < 0.1) p.productivity = 0.1;
      });
    } else {
      // 条件满足时，历史包袱缓慢消退，生产力缓慢恢复
      silo.history_burden -= 0.01 * deltaYears;
      if (silo.history_burden < 0) silo.history_burden = 0;
      
      silo.professions?.forEach(p => {
        p.productivity += 0.01 * deltaYears;
        if (p.productivity > 1.0) p.productivity = 1.0;
      });
    }
  }

  private updateIdeology(silo: Silo, deltaYears: number): void {
    silo.professions?.forEach((p) => {
      const stability = silo.cohesion;
      
      // 1. 原有的根据恐慌值和不稳定度产生的思潮偏移
      if (p.panic_value > 0.3 && stability < 0.5) {
        const drift = p.panic_value * (1.0 - stability) * deltaYears * 0.01;
        p.ideology_value += drift;
      }

      // 2. 恐慌值转化为亲外度 (Panic -> Pro-Foreign)
      // 设定转化率：每年将现有恐慌值的 10% 转化为亲外度
      if (p.panic_value > 0) {
        const conversionRate = 0.10;
        const convertedAmount = p.panic_value * conversionRate * deltaYears;
        
        p.panic_value -= convertedAmount;
        if (p.panic_value < 0) p.panic_value = 0;

        p.ideology_value += convertedAmount;
      }

      // 限制边界
      if (p.ideology_value > 1.0) p.ideology_value = 1.0;
      else if (p.ideology_value < 0) p.ideology_value = 0;
    });

    // 3. 精神药物常态化投放：缓慢向 IT 部门思潮靠拢
    if (silo.traits?.includes('psychoactive_meds')) {
      const itDept = silo.professions?.find(p => p.name === 'IT');
      if (itDept) {
        const targetIdeology = itDept.ideology_value;
        silo.professions?.forEach(p => {
          if (p.name !== 'IT') {
            const diff = targetIdeology - p.ideology_value;
            p.ideology_value += diff * 0.05 * deltaYears; // 每年拉近 5%
          }
        });
      }
    }
  }

  private updateResources(silo: Silo, deltaYears: number): void {
    const populationFactor = silo.total_population / 10000.0;
    const isRebelling = silo.rebellion > 0.7;

    silo.resources?.forEach((r) => {
      // 1. 计算总消耗
      const consumption = (this.perCapitaConsumption[r.type] || 0) * populationFactor;

      // 2. 计算总产出
      let production = 0;
      const producers = this.resourceProducers[r.type] || [];
      
      producers.forEach(profName => {
        const prof = silo.professions?.find(p => p.name === profName);
        if (prof) {
          // 产出受效率影响：(1 - 恐慌值) * 生产力系数
          const efficiency = (1.0 - prof.panic_value) * prof.productivity;
          // 基础产出设定为消耗的 1.2 倍（理想状态下盈余）
          const baseProd = (this.perCapitaConsumption[r.type] || 0) * 1.2 / producers.length;
          production += baseProd * efficiency;
        }
      });

      // 3. 叛乱惩罚
      if (isRebelling) {
        production *= 0.3; // 生产停滞
      }

      // 4. 结算
      r.net_balance = production - consumption;
      r.amount += r.net_balance * deltaYears;

      if (r.amount < 0) r.amount = 0;
    });
  }

  private updateSiloMetrics(silo: Silo, deltaYears: number): void {
    silo.countdown -= deltaYears;
    if (silo.countdown < 0) silo.countdown = 0;

    silo.event_trigger += (1.0 - silo.cohesion) * deltaYears * 0.1;

    let avgPanic = 0;
    const profCount = silo.professions?.length || 0;
    if (profCount > 0) {
      silo.professions.forEach((p) => {
        avgPanic += p.panic_value;
      });
      avgPanic /= profCount;
    }

    const threshold = 0.1;
    const stressFactor = (1.0 - silo.legitimacy) * avgPanic;
    if (stressFactor > threshold) {
      silo.rebellion += (stressFactor - threshold) * deltaYears * 0.05;
    } else {
      silo.rebellion -= 0.01 * deltaYears;
    }

    if (silo.rebellion > 1.0) silo.rebellion = 1.0;
    else if (silo.rebellion < 0) silo.rebellion = 0;

    // 4. 更新人口 (受资源不足与叛乱影响)
    this.updatePopulation(silo, deltaYears);
  }

  private updatePopulation(silo: Silo, deltaYears: number): void {
    let deathRate = 0.001; // 基础死亡率

    // 资源不足增加死亡率
    silo.resources?.forEach(r => {
      if (r.amount <= 0) {
        deathRate += 0.05;
      }
    });

    // 高叛乱增加死亡率
    if (silo.rebellion > 0.8) {
      deathRate += (silo.rebellion - 0.8) * 0.2;
    }

    const deaths = silo.total_population * deathRate * deltaYears;
    silo.total_population -= Math.floor(deaths);
    if (silo.total_population < 0) silo.total_population = 0;
  }
}
