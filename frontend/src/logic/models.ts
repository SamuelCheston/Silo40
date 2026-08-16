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
  | "PUBLICIZE_FACTION";

export type AgentAction = Omit<model.AgentAction, "type"> & {
  type: AgentActionType;
};

export const ACTION_COSTS: Record<AgentActionType, number> = {
  GATHER_INFO: 10,
  SHARE_INFO: 20,
  BUILD_CONNECTION: 15,
  INCITE_REBELLION: 30,
  CONDUCT_PROPAGANDA: 20,
  PROFESSION_ACTION: 0,
  PLAYER_EVENT: 0,
  PUBLICIZE_FACTION: 25,
};

export const ACTION_DURATIONS: Record<AgentActionType, number> = {
  GATHER_INFO: 0,
  SHARE_INFO: 0,
  BUILD_CONNECTION: 1,
  INCITE_REBELLION: 2,
  CONDUCT_PROPAGANDA: 1,
  PROFESSION_ACTION: 1,
  PLAYER_EVENT: 0,
  PUBLICIZE_FACTION: 0,
};

export const ALL_FRAGMENTS: string[] = [
  "Mayor_1",
  "Mayor_2",
  "Mayor_3",
  "Mayor_4",
  "Mayor_5",
  "Judicial_1",
  "Judicial_2",
  "Judicial_3",
  "Judicial_4",
  "Judicial_5",
  "IT_1",
  "IT_2",
  "IT_3",
  "IT_4",
  "IT_5",
  "Police_1",
  "Police_2",
  "Medical_1",
  "Medical_2",
  "Mechanical_1",
  "Mechanical_2",
  "Supply_1",
  "Supply_2",
  "Mines_1",
  "Mines_2",
  "Agricultural_1",
];
