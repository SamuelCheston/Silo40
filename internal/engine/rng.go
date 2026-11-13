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
