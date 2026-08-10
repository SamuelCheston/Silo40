package engine

// ScheduledEntry 调度器条目
type ScheduledEntry struct {
	key   string
	time  GameTime
	event *GameEvent
}

// Scheduler 时间调度器
// 区分即时与延时事件：延时事件进入 MinHeap，每个 Tick 将到期事件发布到总线
type Scheduler struct {
	heap    []*ScheduledEntry
	counter int
}

func NewScheduler() *Scheduler {
	return &Scheduler{}
}

// Schedule 注册延时事件：at 时刻到期后自动发布到总线
func (s *Scheduler) Schedule(event *GameEvent, at GameTime) {
	s.counter++
	entry := &ScheduledEntry{
		key:  event.ID + "#" + itoa(s.counter),
		time: at,
		event: &GameEvent{
			ID:          event.ID,
			Type:        event.Type,
			Source:      event.Source,
			Target:      event.Target,
			TriggerTime: &at,
			Data:        event.Data,
		},
	}
	s.heap = append(s.heap, entry)
	s.bubbleUp(len(s.heap) - 1)
}

// Tick 每个游戏 Tick 调用：将所有到期事件发布到总线
func (s *Scheduler) Tick(now GameTime, bus *EventBus) []*GameEvent {
	var due []*GameEvent
	for len(s.heap) > 0 && CompareTime(s.heap[0].time, now) <= 0 {
		entry := s.popMin()
		bus.Publish(entry.event)
		due = append(due, entry.event)
	}
	return due
}

// Peek 下一次到期时间 (无则返回零值 + false)
func (s *Scheduler) Peek() (GameTime, bool) {
	if len(s.heap) == 0 {
		return GameTime{}, false
	}
	return s.heap[0].time, true
}

// Clear 清除所有延时事件 (新游戏初始化)
func (s *Scheduler) Clear() {
	s.heap = nil
	s.counter = 0
}

// Size 延时事件数量
func (s *Scheduler) Size() int {
	return len(s.heap)
}

// ============ MinHeap 实现 ============

func (s *Scheduler) popMin() *ScheduledEntry {
	if len(s.heap) == 0 {
		return nil
	}
	top := s.heap[0]
	last := s.heap[len(s.heap)-1]
	s.heap = s.heap[:len(s.heap)-1]
	if len(s.heap) > 0 {
		s.heap[0] = last
		s.siftDown(0)
	}
	return top
}

func (s *Scheduler) bubbleUp(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if s.less(i, parent) {
			s.swap(i, parent)
			i = parent
		} else {
			break
		}
	}
}

func (s *Scheduler) siftDown(i int) {
	n := len(s.heap)
	for {
		left := 2*i + 1
		right := 2*i + 2
		smallest := i
		if left < n && s.less(left, smallest) {
			smallest = left
		}
		if right < n && s.less(right, smallest) {
			smallest = right
		}
		if smallest == i {
			break
		}
		s.swap(i, smallest)
		i = smallest
	}
}

func (s *Scheduler) less(a, b int) bool {
	return CompareTime(s.heap[a].time, s.heap[b].time) < 0
}

func (s *Scheduler) swap(a, b int) {
	s.heap[a], s.heap[b] = s.heap[b], s.heap[a]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
