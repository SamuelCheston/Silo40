import { Silo, Agent, Profession, Resource, VictoryStatus, VictoryType, GameEvent } from './models';
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
    Sheriff: 0.3,
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
  public updateAgentState(agent: Agent, deltaYears: number): void {
    if (deltaYears <= 0) return;

    // 1. 计算平均人脉值
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
    agent.political_prestige = totalConnection * 100 * (1 + profFactor) * (1 + traitFactor);
    if (agent.political_prestige < 0) agent.political_prestige = 0;

    // 5. 给予政治点数
    const pointGainRate = 0.1;
    agent.political_points += agent.political_prestige * pointGainRate * deltaYears;
  }

  // 核心逻辑更新引擎
  public updateSiloState(silo: Silo, deltaYears: number, agent?: Agent): GameEvent[] {
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
      // 计算组织人数：各部门人脉值 * 组织度的和
      let organizedPopulation = 0;
      if (agent.connections && agent.connections.length > 0) {
        agent.connections.forEach(conn => {
          organizedPopulation += conn.value * (agent.organization_factor || 1.0);
        });
      }

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
      if (p.panic_value > 0.3 && stability < 0.5) {
        const drift = p.panic_value * (1.0 - stability) * deltaYears * 0.01;
        p.ideology_value += drift;
      }

      if (p.ideology_value > 1.0) p.ideology_value = 1.0;
      else if (p.ideology_value < 0) p.ideology_value = 0;
    });
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
