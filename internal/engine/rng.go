package engine

import (
	"math/rand"
	"time"
)

// rng 是引擎内唯一的随机源。所有游戏随机性——Go 规则与脚本侧的
// randomFloat()/randomInt()（mechanics.go 通过 SetRandSource 接入同一实例）
// ——都从这一条流上取样，保证"同一种子 + 同一操作序列"可以复现同一局。
//
// GameService 在 CreateGame/Resume 时通过 SetRNG 注入按存档种子构造的源；
// 未注入时保持时间种子，等价于原有的默认随机行为。
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// SetRNG 替换引擎级随机源；传 nil 保持现有源不变。
func SetRNG(r *rand.Rand) {
	if r != nil {
		rng = r
	}
}

// DrawFloat64 / DrawIntn 是引擎单一 RNG 流唯一的导出取样入口。
// 服务层需要随机性的初始化流程（如遗物清单开局实例化）必须经此取样，
// 不得自建随机源，保证"同一种子 + 同一操作序列"可复现同一局。
func DrawFloat64() float64 { return rng.Float64() }

// DrawIntn 返回 [0, n) 的确定性整数；n <= 0 时 panic（与 rand.Intn 一致）。
func DrawIntn(n int) int { return rng.Intn(n) }
