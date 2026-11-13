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

## Political Faction Gate

- `loyalty` does not participate in faction grouping signatures.
- `ambition` participates only via the potential-leader bypass: a cohort is
  political when `pro_foreign >= PoliticalEntryThreshold` **or** it has a
  resident with `Ambition >= AmbitionHighThreshold` and
  `Influence >= LeaderInfluenceThreshold`. (Note: with initial resident
  ambition capped below the threshold, the leader path has never fired in
  simulation; the ideology path is the effective gate.)
- Political faction grouping uses only `pro_foreign` for the gate, then
  applies the birth-bucket profile tag plus `drive`.
- A political faction forms when a cohort's `pro_foreign` reaches
  `PoliticalEntryThreshold` (0.7). The birth-bucket profile tag
  (`Medium`/`High`) in the signature is assigned at cohort creation and is
  not refreshed afterwards.
- **Hysteresis**: once political, a cohort depoliticizes only when
  `pro_foreign` falls below `PoliticalExitThreshold` (0.6). The band keeps
  cohorts hovering near the gate from flipping every tick.
- Cohorts without a formed political profile are folded into `Unaffiliated`
  and contribute `0` political influence.
- All gate/grouping tunables live in `engine.FactionConfig`
  (`internal/engine/faction_config.go`); no save-format fields are involved.

## Resident Ambition

- `ambition` now belongs only to explicit `Resident` actors.
- Representative residents generate a native `Ambition` value when they are created.
- Resident ambition is fixed after creation and does not drift with cohort, faction, legitimacy, or panic updates.
- `Ambition >= 0.62` still marks a resident as `drive:ambition` / `status:ambitious`, but that affects personal tagging only.

Source:
- [residents.go](../internal/engine/residents.go#L13-L23)
- [residents.go](../internal/engine/residents.go#L458-L669)

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
  explicit pipeline step after `RebuildImplicitFactions` (which is now
  grouping-only).

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
