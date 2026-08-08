import { Silo, Agent, Profession, Resource } from './models';

export class GameEngine {
  // 基础消耗基数 (万人/小时)
  private readonly baseConsumption: Record<string, number> = {
    Food: 100.0,
    Energy: 500.0,
    Water: 200.0,
    Materials: 50.0,
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
  public updateSiloState(silo: Silo, deltaYears: number): void {
    if (deltaYears <= 0) return;

    // 1. 更新部门生产力与资源结余
    this.updateResources(silo, deltaYears);

    // 2. 更新地堡状态
    this.updateSiloMetrics(silo, deltaYears);

    // 3. 更新思潮演化
    this.updateIdeology(silo, deltaYears);
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
    let panicDeptCount = 0;
    silo.professions?.forEach((p) => {
      if (p.panic_value > 0.5) panicDeptCount++;
    });

    const isRebelling = silo.rebellion > 0.7;

    silo.resources?.forEach((r) => {
      const baseCons = this.baseConsumption[r.type] || 0;
      let balanceMultiplier = 0.2 - panicDeptCount * 0.05;

      if (isRebelling) {
        balanceMultiplier = -0.5;
      }

      r.net_balance = balanceMultiplier * baseCons;
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
  }
}
