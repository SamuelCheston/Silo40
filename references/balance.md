# Balance Notes

## Time Cost (ACTION_DURATIONS)

- `GATHER_INFO`: 0 months
- `SHARE_INFO`: 0 months
- `BUILD_CONNECTION`: 1 month
- `INCITE_REBELLION`: 2 months
- `CONDUCT_PROPAGANDA`: 1 month
- `PROFESSION_ACTION`: 1 month

Sources:
- Backend: [models.go](../internal/model/models.go#L216-L337)
- Frontend mirror: [models.ts](../frontend/src/logic/models.ts#L54-L90)

## Header Metrics Source Of Truth

- Header tooltip values such as AP recovery, prestige breakdown, and propaganda multiplier now come from backend `agent_stats`.
- Frontend no longer recomputes these formulas locally.

Sources:
- Backend DTO: [models.go](../internal/model/models.go)
- Backend derivation: [agent_stats.go](../internal/engine/agent_stats.go)

## 0-Month Action Execution Rule

For 0-month actions (`GATHER_INFO` / `SHARE_INFO`):

- NPC turn runs (no time passes).
- Random story events do not run.
- Victory conditions are evaluated immediately after the NPC turn.

Source:
- [game_service.go](../internal/service/game_service.go#L356-L415)

## Suspicion Profession Multipliers

Suspicion gains are single-tracked in the JS mechanic prelude
(`professionSuspicionFactor` + `suspicionDelta`) and apply to **both**
common actions and profession actions:

- `Mayor`: `x2.0`
- `IT`: `x0.3`
- `Mines`: `x0.3`
- `Police`: `x0.7` (deterministic; the legacy Go fallback's random
  `x(0.5~0.9)` discount is dead and slated for removal)
- Trait `隐秘行事`: additional `x0.8`

IT passive: every successful IT action worsens `safeguard_risk` by
`action cost x 0.002` (applied by the ACTOR_ACTION pipeline, so it
covers common and profession actions alike).

Sources:
- Prelude helpers: [mechanics.go](../internal/engine/mechanics.go) (`mechanicPrelude`)
- Profession action usage: [profession_actions.js](../events/mechanics/profession_actions.js)
- Common action usage: [common_actions.js](../events/mechanics/common_actions.js)
- IT safeguard coupling: [engine.go](../internal/engine/engine.go) (ACTOR_ACTION subscriber)

## Political Faction Lifecycle (event-driven)

Factions are no longer produced by per-tick distance clustering. The
authoritative design is [24-faction-event-algorithm-plan.md](../docs/research/24-faction-event-algorithm-plan.md);
all tunables live in `engine.FactionConfig`
(`internal/engine/faction_config.go`), no save-format fields involved.

- **Founding is the only faction source.** `FOUND_FACTION` requires
  derived prestige ≥ 50, ideology intensity ≥ 0.25 and AP ≥ 20 (cost 20,
  no cooldown). Zero-member factions are legal (preparation period).
  Player founds by direct choice; NPCs roll a weighted decision
  (ambition/prestige/intensity are weights, never hard gates).
- **Derived prestige**: `prestige = 50 × (Ambition + Influence)` at every
  layer (Agent/Profession/Resident/Cohort via representative). There is no
  independent PoliticalPrestige field. Ambition is fixed at creation and
  only events may change it (`actor_metric_delta/set: ambition`).
- **Awareness**: founding seeds a `faction:<id>:founded` belief fact for
  every observer (departments, cohort-bound resident representatives,
  player), reliability = three-channel awareness (relation 0.40 /
  ideology 0.35 / tier 0.25, same-class bonus 0.15). These facts never
  mature and never count toward the co-evidence gate (same exemption as
  `leak:`). Monthly gossip spreads them along relation networks
  (×0.6 per hop); info actions piggyback rumors (chance 0.15, player
  deterministic, zero draws while no political faction exists).
- **Join/exit**: perceived cohorts (belief ≥ AwarenessJoinFloor 0.30)
  roll monthly, ideology closeness^2 dominates; relation < 0.10 vetoes,
  cross-tier (PowerLevel gap ≥ 3) needs relation ≥ 0.40, cross-class
  needs ≥ 0.60, mutual department hostility < 0.05 hard-vetoes. Exit is
  a symmetric inertia-protected roll after drifting outside the affinity
  radius (0.30). Membership is persistent; exit phase runs before join
  phase so no cohort round-trips in one month. `JOIN_FACTION` (AP 10) is
  player direct choice, exempt from awareness/relation gates (ADR-0010).
- **Lifecycle**: a faction younger than 18 months with MemberCount < 200
  dissolves (members return to the Unaffiliated bucket). Leader death →
  automatic succession by highest `0.5×(prestige/100) + 0.5×ambition`
  among member representatives (tie → smaller ID); no successor →
  dissolve. Player/NPC leaders never die.
- **Merge**: `MERGE_FACTION` (AP 15) requires both leaders' personal
  relation ≥ 0.60 (LeaderRelations sub-matrix) and centroid distance ≤
  affinity radius. Survivor keeps its ID; absorbed members transfer,
  Memory records the merge. The relation gate applies to players too —
  merge is mutual consent, not intervention.
- The old feature-distance machinery (`faction_cluster.go`) survives only
  as the **affinity field**: join probability and exit pressure inputs,
  never a faction source. Cohorts without political membership sit in the
  `special:unaffiliated` bucket and contribute 0 political influence.

## Ambition & Influence

- `Ambition` / `Influence` exist on Agent, Profession and Resident
  (ADR-0018 isomorphism); Cohorts derive prestige via their representative.
- Ambition is fixed after creation and does not drift with cohort,
  faction, legitimacy, or panic updates; only event mutations change it.
- Influence evolves: residents/player grow `(0.005 + Ambition×0.05)/year`;
  departments converge toward PowerLevel/10. Annual settlement uses
  start-of-period prestige (Go/JS agent_update match agent_stats display).
- `Ambition >= 0.62` still marks a resident as `drive:ambition` /
  `status:ambitious`, but that affects personal tagging only.

Source:
- [residents.go](../internal/engine/residents.go)
- [faction_found.go](../internal/engine/faction_found.go)
- [faction_awareness.go](../internal/engine/faction_awareness.go)
- [faction_join.go](../internal/engine/faction_join.go)
- [faction_merge.go](../internal/engine/faction_merge.go)

## Post-Grouping Loyalty Dynamics

- `loyalty` is updated twice:
  - first by profession panic / silo legitimacy in the normal cohort update;
  - then again after faction grouping (`ApplyFactionLoyaltyDynamics`), where
    formed political factions amplify loyalty drift based on ideology
    intensity.
- Low-legitimacy states punish stronger dissent factions more strongly; status-quo factions recover loyalty slightly faster under high legitimacy.
- **Time scaling**: both the resident→cohort sync and the faction-adjusted
  drift are per-month steps converted by `1-(1-base)^(12*dt)`
  (`monthlyTimeScale`). A monthly tick keeps the historical rate exactly;
  zero-duration steps (instant/0-month actions) apply nothing, so instant
  actions cannot accelerate loyalty evolution. The dynamics run as an
  explicit pipeline step after `MaintainFactions` (grouping-only
  maintenance: stats recompute, grace dissolution, death succession,
  delegate top-up, flag repair — no evolution draws).

Source:
- [residents.go](../internal/engine/residents.go#L779-L836)
- [residents.go](../internal/engine/residents.go#L926-L985)

## Removed: Ideology → Productivity Penalty

The `checkOperationalConditions` mechanic (which previously adjusted `HistoryBurden` and globally increased/decreased `Productivity` based on `pro_foreign` thresholds) has been removed.

Rationale: ideology alignment should not directly punish production efficiency.

## Resource System (Per-Capita & Conversion)

The resource system is now entirely population-driven:

### Consumption
- Each person in the Silo consumes **1.0 units** of Energy, Materials, and Supplies per year.
- Total consumption = `1.0 * TotalPopulation`.

### Production
- Producers (Mines, Mechanical, Agricultural, Supply) have a base rate of **4.5 units per worker** per year.
- Production = `4.5 * WorkerCount * Efficiency`.
- **Default Efficiency**: 1.0 (unless in special states like Rebellion; `Rebellion > 0.7` drops it to 0.3).

### Conversion Chain
- **Agricultural (农业部)**: Consumes `Energy` to produce `Supplies` (1:1 ratio). Output is capped by labor capacity OR available Energy.
- **Supply (供给部)**: Consumes `Materials` to produce `Supplies` (1:1 ratio). Output is capped by labor capacity OR available Materials.
- **Mines (矿工)**: Produces `Materials`.
- **Mechanical (机械部)**: Produces `Energy`.

Sources:
- Logic: [engine.go](../internal/engine/engine.go#L1207-L1272)
- Constants: [engine.go](../internal/engine/engine.go#L59-L64)
- Initial Reserves: [initializer.go](../internal/engine/initializer.go#L187-L192)
