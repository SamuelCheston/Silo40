package engine

import "sync"

// 碎片标题注册表：service 层装载碎片内容库 (events/fragments.json) 后
// 登记，mechanic 脚本经 prelude 的 fragmentTitle() 助手读取，用于动作
// 结果消息的可读化。纯展示层：只被消息拼接读取，不进入任何判定、不消费
// RNG、不影响同种子复现。未登记时 (旧机制测试路径) fragmentTitle 回退
// 原键，消息行为与改造前逐位一致。

var (
	fragmentTitlesMu sync.RWMutex
	fragmentTitles   = map[string]string{}
)

// RegisterFragmentTitles 整体替换碎片标题注册表 (Bootstrap 期调用)。
func RegisterFragmentTitles(titles map[string]string) {
	fragmentTitlesMu.Lock()
	defer fragmentTitlesMu.Unlock()
	fragmentTitles = make(map[string]string, len(titles))
	for id, title := range titles {
		fragmentTitles[id] = title
	}
}

// FragmentTitle 返回碎片展示标题；未登记或空标题回退原键。
func FragmentTitle(id string) string {
	fragmentTitlesMu.RLock()
	defer fragmentTitlesMu.RUnlock()
	if title, ok := fragmentTitles[id]; ok && title != "" {
		return title
	}
	return id
}

// FragmentTitleSnapshot 返回当前注册表副本 (注入 JS 运行时用)。
// 脚本侧只按键查找、不遍历，map 序不影响任何可观察输出。
func FragmentTitleSnapshot() map[string]string {
	fragmentTitlesMu.RLock()
	defer fragmentTitlesMu.RUnlock()
	out := make(map[string]string, len(fragmentTitles))
	for id, title := range fragmentTitles {
		out[id] = title
	}
	return out
}
