# Silo40 内容作者手册 · 机制脚本变异封闭集

> 拆分自 [Silo40 内容作者手册](../07-content-authoring-guide.md)；目录、章节映射与出处见该索引。

## 5. 机制脚本（`events/mechanics/`）变异封闭集

`defineMechanics` 的 `apply` 返回 `{ mutations, emit, logs, action_result, stats, score }`；
变异由 `internal/engine/mechanics.go` 的 `applyMechanicMutations` 解释：

`actor_metric_delta|set`（字段：`action_points` `suspicion_level` `political_prestige`
`political_points` `propaganda_level`）、`actor_connection_delta`、`actor_fragment_add`、
`profession_metric_delta|set`（`panic_value` `productivity`）、`profession_ideology_delta`、
`profession_fragment_add|remove`、`silo_metric_delta|set`（比内容效果多 `safeguard_risk`）、
`resource_delta`、`resource_net_balance_set`、`cohort_ideology_delta(_all)`（**唯一能持久改
思潮的入口**，可按 `class_type`/`profession` 过滤）、`sync_profession_ideologies_from_cohorts`、
`apply_population_deaths`、`refresh_profession_population_from_cohorts`、
`faction_metric_delta_actor`、`faction_is_public_actor`、`silo_flag_set`、
`victory_status_set`、`score_set`、`log`。

机制脚本里可用 `randomFloat()/randomInt()`（引擎单一随机源，种子可复现）。
核心系统事件：`AGENT_UPDATE` `RESOURCE_UPDATE` `IDEOLOGY_UPDATE` `METRICS_UPDATE`
`VICTORY_CHECK`（`events/mechanics/core_systems.js`）——改结算公式改这里，Go 侧有降级副本
（`victory_fallback_test.go` 锁定语义一致）。

