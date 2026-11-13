import { model } from "../../wailsjs/go/models";

export type VictoryType =
  | "NONE"
  | "INFORMATION"
  | "TIME"
  | "REBELLION"
  | "EXCLUSIONIST"
  | "DEATH"
  | "AGENT_COMPROMISED";

export type VictoryStatus = model.VictoryStatus;
export type GameScore = model.GameScore;

export type Agent = model.Agent;
export type Connection = model.Connection;
export type Silo = model.Silo;
export type Resource = model.Resource;
export type Profession = model.Profession;
export type Relic = model.Relic;
export type Floor = model.Floor;
export type Faction = model.Faction;
export type PopulationCohort = model.PopulationCohort;
export type StoryEvent = model.StoryEvent;
export type StoryEventLog = model.StoryEventLog;
export type ActionResult = model.ActionResult;
export type AgentStats = model.AgentStats;
export type ProfessionActionMeta = model.ProfessionActionMeta;
export type PlayerActionMeta = model.PlayerActionMeta;
export type GameState = model.GameState;
export type TickResult = model.TickResult;
export type ActionOutcome = model.ActionOutcome;
export type EventHistoryResult = model.EventHistoryResult;
export type CreateGameRequest = model.CreateGameRequest;

export type AgentActionType =
  | "GATHER_INFO"
  | "SHARE_INFO"
  | "BUILD_CONNECTION"
  | "INCITE_REBELLION"
  | "CONDUCT_PROPAGANDA"
  | "PROFESSION_ACTION"
  | "PLAYER_EVENT"
  | "PUBLICIZE_FACTION"
  | "SILO1_OPERATION"
  | "STATION_OPERATION";

export type AgentAction = Omit<model.AgentAction, "type"> & {
  type: AgentActionType;
};

// 行动成本/耗时/碎片清单不再维护前端副本：
// 成本与耗时由后端 PlayerActionMeta (ap_cost / duration_months) 下发，
// 碎片清单经 GetAllFragments() 绑定从后端 model 权威加载。
