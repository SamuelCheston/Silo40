// Command sim 无头长跑模拟器：通过 SILO40_DEBUG_HTTP_ADDR 调试端点驱动
// 运行中的 Silo40，逐月推进 N 局游戏，输出结局分布与关键指标，用于调平衡。
//
// 用法:
//
//	# 1. 启动游戏 (GUI; 调试端点随 SILO40_DEBUG_HTTP_ADDR 开启，默认 127.0.0.1:18080)
//	SILO40_DEBUG_HTTP_ADDR=127.0.0.1:18080 ./build/bin/silo40
//	# 2. 跑模拟
//	go run ./cmd/sim -addr http://127.0.0.1:18080 -games 50 -seed 7
//
// 确定性: -seed S 时，第 i 局以种子 S+i 开局，行动策略也由 S 派生的本地
// 随机源驱动，因此整批模拟可逐局复现（服务器同一时刻只有一个活跃会话，
// 模拟期间不要手动操作游戏）。不带 -seed 时种子由服务端决议（若服务端设置了
// SILO40_SEED，则按其主种子派生）。
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"time"
)

type siloView struct {
	Name            string  `json:"name"`
	TotalPopulation int     `json:"total_population"`
	CurrentYear     int     `json:"current_year"`
	CurrentMonth    int     `json:"current_month"`
	Legitimacy      float64 `json:"legitimacy"`
	Cohesion        float64 `json:"cohesion"`
	Rebellion       float64 `json:"rebellion"`
}

type agentView struct {
	ActionPoints float64 `json:"action_points"`
}

type actionMeta struct {
	ID         string `json:"id"`
	ActionType string `json:"action_type"`
	APCost     int    `json:"ap_cost"`
	Enabled    bool   `json:"enabled"`
	TargetType string `json:"target_type"`
}

type victoryView struct {
	IsWon bool   `json:"is_won"`
	Type  string `json:"type"`
}

// stepResult 同时兼容 TickResult (pass-time) 与 ActionOutcome (action) 的
// 模拟所需字段子集。
type stepResult struct {
	Silo             siloView     `json:"silo"`
	Agent            agentView    `json:"agent"`
	AvailableActions []actionMeta `json:"available_actions"`
	GameOver         bool         `json:"game_over"`
}

type gameState struct {
	Silo          siloView     `json:"silo"`
	GameOver      bool         `json:"game_over"`
	VictoryStatus *victoryView `json:"victory_status"`
}

type gameRecord struct {
	seed       int64
	months     int
	ending     string
	won        bool
	population int
	year       int
	rebellion  float64
	fires      map[string]int
}

type eventLogView struct {
	EventID  string `json:"event_id"`
	Category string `json:"category"`
}

type eventHistoryView struct {
	Events []eventLogView `json:"events"`
}

func postJSON(c *http.Client, url string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := c.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("POST %s -> HTTP %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func getJSON(c *http.Client, url string, out any) error {
	resp, err := c.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s -> HTTP %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// playOneGame 跑完一局：逐月推进，间歇按策略挑一个无需目标的可用动作执行。
func playOneGame(c *http.Client, addr string, gameSeed int64, profession string, maxMonths int, policy *rand.Rand) (gameRecord, error) {
	rec := gameRecord{seed: gameSeed}

	var created gameState
	err := postJSON(c, addr+"/debug/game/create", map[string]any{
		"silo_name":   fmt.Sprintf("Sim-%d", gameSeed),
		"agent_name":  "SimAgent",
		"profession":  profession,
		"seed":        gameSeed,
	}, &created)
	if err != nil {
		return rec, fmt.Errorf("create game: %w", err)
	}

	var last stepResult
	for m := 1; m <= maxMonths; m++ {
		var tick stepResult
		if err := postJSON(c, addr+"/debug/game/pass-time", map[string]int{"months": 1}, &tick); err != nil {
			return rec, fmt.Errorf("pass time month %d: %w", m, err)
		}
		last = tick
		rec.months = m
		if tick.GameOver {
			break
		}

		// 策略：约一半的月份尝试执行一个"无需目标且付得起 AP"的可用动作。
		if policy.Float64() < 0.5 {
			var options []actionMeta
			for _, a := range tick.AvailableActions {
				if a.Enabled && a.TargetType == "NONE" && float64(a.APCost) <= tick.Agent.ActionPoints {
					options = append(options, a)
				}
			}
			if len(options) > 0 {
				pick := options[policy.Intn(len(options))]
				var outcome stepResult
				err := postJSON(c, addr+"/debug/game/action", map[string]any{
					"type":      pick.ActionType,
					"action_id": pick.ID,
					"cost":      pick.APCost,
				}, &outcome)
				if err != nil {
					return rec, fmt.Errorf("action %s month %d: %w", pick.ID, m, err)
				}
				last = outcome
				if outcome.GameOver {
					break
				}
			}
		}
	}

	rec.population = last.Silo.TotalPopulation
	rec.year = last.Silo.CurrentYear

	var final gameState
	if err := getJSON(c, addr+"/debug/game/state", &final); err != nil {
		return rec, fmt.Errorf("final state: %w", err)
	}
	if final.VictoryStatus != nil {
		rec.ending = final.VictoryStatus.Type
		rec.won = final.VictoryStatus.IsWon
	} else {
		rec.ending = "NO_ENDING(max_months)"
	}
	rec.rebellion = final.Silo.Rebellion

	// 事件触发次数：读事件历史（limit=0 返回全部），按事件名计数。
	var history eventHistoryView
	if err := getJSON(c, addr+"/debug/game/events", &history); err != nil {
		return rec, fmt.Errorf("event history: %w", err)
	}
	rec.fires = map[string]int{}
	for _, entry := range history.Events {
		key := entry.EventID
		if entry.Category != "" {
			key = entry.Category + ":" + entry.EventID
		}
		rec.fires[key]++
	}
	return rec, nil
}

func main() {
	addr := flag.String("addr", "http://127.0.0.1:18080", "调试端点基地址 (游戏进程 SILO40_DEBUG_HTTP_ADDR)")
	games := flag.Int("games", 10, "模拟局数")
	seed := flag.Int64("seed", 0, "模拟种子；0 = 由服务端决议种子")
	maxMonths := flag.Int("months", 1200, "单局最长月数")
	profession := flag.String("profession", "IT", "特工职业")
	flag.Parse()

	c := &http.Client{Timeout: 30 * time.Second}
	policy := rand.New(rand.NewSource(*seed))

	fmt.Printf("Silo40 headless sim: addr=%s games=%d seed=%d maxMonths=%d profession=%s\n",
		*addr, *games, *seed, *maxMonths, *profession)

	var records []gameRecord
	for i := 0; i < *games; i++ {
		gameSeed := int64(0)
		if *seed != 0 {
			gameSeed = *seed + int64(i)
		}
		rec, err := playOneGame(c, *addr, gameSeed, *profession, *maxMonths, policy)
		if err != nil {
			fmt.Fprintf(os.Stderr, "game %d failed: %v\n", i+1, err)
			os.Exit(1)
		}
		records = append(records, rec)
		fmt.Printf("game %-3d seed=%-12d months=%-5d ending=%-22s won=%-5v pop=%-6d reb=%.3f year=%d\n",
			i+1, rec.seed, rec.months, rec.ending, rec.won, rec.population, rec.rebellion, rec.year)
	}

	// 汇总：结局分布 + 平均时长/人口/叛乱终值 + 各事件平均触发次数。
	dist := map[string]int{}
	wins := 0
	totalMonths := 0
	totalPop := 0
	totalReb := 0.0
	fireTotals := map[string]int{}
	for _, r := range records {
		dist[r.ending]++
		if r.won {
			wins++
		}
		totalMonths += r.months
		totalPop += r.population
		totalReb += r.rebellion
		for key, count := range r.fires {
			fireTotals[key] += count
		}
	}
	keys := make([]string, 0, len(dist))
	for k := range dist {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Println("\n=== ending distribution ===")
	for _, k := range keys {
		fmt.Printf("%-24s %4d  (%.1f%%)\n", k, dist[k], 100*float64(dist[k])/float64(len(records)))
	}
	n := len(records)
	fmt.Printf("\ngames=%d wins=%d (%.1f%%) avg_months=%.1f avg_final_population=%.0f avg_final_rebellion=%.3f\n",
		n, wins, 100*float64(wins)/float64(n), float64(totalMonths)/float64(n), float64(totalPop)/float64(n), totalReb/float64(n))

	fireKeys := make([]string, 0, len(fireTotals))
	for k := range fireTotals {
		fireKeys = append(fireKeys, k)
	}
	sort.Strings(fireKeys)
	fmt.Println("\n=== event fires (avg per game) ===")
	for _, k := range fireKeys {
		fmt.Printf("%-42s %6.2f\n", k, float64(fireTotals[k])/float64(n))
	}
}
