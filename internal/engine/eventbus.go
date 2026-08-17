package engine

import (
	"reflect"
	"sync"
)

// EventHandler 事件处理函数
type EventHandler func(event *GameEvent, ctx *EventContext)

type queuedEvent struct {
	event *GameEvent
	ctx   *EventContext
}

// EventBus 事件总线
// 两种发布方式：
// - Publish: 普通追加到队尾
// - Emit:    插入到当前事件之后（若当前正在 dispatch）
type EventBus struct {
	mu            sync.Mutex
	handlers      map[string][]EventHandler
	queue         []*queuedEvent
	dispatching   bool
	emitInsertPos int
}

// ANY 通配类型：订阅后响应所有事件 (供 RuleEngine 等使用)
const ANY = "*"

func NewEventBus() *EventBus {
	return &EventBus{handlers: map[string][]EventHandler{}}
}

// Subscribe 订阅事件，返回取消订阅函数
func (b *EventBus) Subscribe(typ string, handler EventHandler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[typ] = append(b.handlers[typ], handler)
	h := handler
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		list := b.handlers[typ]
		for i, f := range list {
			if reflect.ValueOf(f).Pointer() == reflect.ValueOf(h).Pointer() {
				b.handlers[typ] = append(list[:i], list[i+1:]...)
				break
			}
		}
	}
}

// SubscribeAny 订阅所有事件
func (b *EventBus) SubscribeAny(handler EventHandler) func() {
	return b.Subscribe(ANY, handler)
}

// Publish 发布事件：总是追加到队尾。
func (b *EventBus) Publish(event *GameEvent) {
	b.enqueue(event, NewEventContext())
}

// Emit 发布后续事件：
// - 若当前正在处理事件，则插入到当前事件之后、未来事件之前；
// - 否则退化为普通追加。
func (b *EventBus) Emit(event *GameEvent, ctx *EventContext) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.dispatching {
		idx := b.emitInsertPos
		if idx < 0 {
			idx = 0
		}
		if idx > len(b.queue) {
			idx = len(b.queue)
		}
		b.queue = append(b.queue, nil)
		copy(b.queue[idx+1:], b.queue[idx:])
		b.queue[idx] = &queuedEvent{event: event, ctx: ctx}
		b.emitInsertPos++
		return
	}
	b.queue = append(b.queue, &queuedEvent{event: event, ctx: ctx})
}

func (b *EventBus) enqueue(event *GameEvent, ctx *EventContext) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.queue = append(b.queue, &queuedEvent{event: event, ctx: ctx})
}

// Snapshot 返回当前队列事件快照。
func (b *EventBus) Snapshot() []*GameEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	snapshot := make([]*GameEvent, 0, len(b.queue))
	for _, item := range b.queue {
		if item == nil {
			continue
		}
		snapshot = append(snapshot, item.event)
	}
	return snapshot
}

// InsertAt 在指定位置插入一个新事件。
func (b *EventBus) InsertAt(index int, event *GameEvent) {
	b.InsertAtWithContext(index, event, NewEventContext())
}

// InsertAtWithContext 在指定位置插入一个带元数据的事件。
func (b *EventBus) InsertAtWithContext(index int, event *GameEvent, ctx *EventContext) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if index < 0 {
		index = 0
	}
	if index > len(b.queue) {
		index = len(b.queue)
	}
	b.queue = append(b.queue, nil)
	copy(b.queue[index+1:], b.queue[index:])
	b.queue[index] = &queuedEvent{event: event, ctx: ctx}
	if b.dispatching && index <= b.emitInsertPos {
		b.emitInsertPos++
	}
}

// Pending 返回当前待处理事件数。
func (b *EventBus) Pending() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.queue)
}

// ClearQueue 清空待处理队列。
func (b *EventBus) ClearQueue() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.queue = nil
}

// Step 处理队列中的下一个事件，若无事件则返回 nil。
func (b *EventBus) Step() *GameEvent {
	b.mu.Lock()
	if len(b.queue) == 0 {
		b.mu.Unlock()
		return nil
	}
	next := b.queue[0]
	b.queue = b.queue[1:]
	b.dispatching = true
	b.emitInsertPos = 0
	b.mu.Unlock()
	b.dispatch(next.event, next.ctx)
	b.mu.Lock()
	b.dispatching = false
	b.emitInsertPos = 0
	b.mu.Unlock()
	return next.event
}

// Drain 处理当前队列中的所有事件。
func (b *EventBus) Drain() []*GameEvent {
	var processed []*GameEvent
	for {
		event := b.Step()
		if event == nil {
			return processed
		}
		processed = append(processed, event)
	}
}

func (b *EventBus) dispatch(event *GameEvent, ctx *EventContext) {
	b.mu.Lock()
	typeHandlers := b.handlers[event.Type]
	anyHandlers := b.handlers[ANY]
	b.mu.Unlock()

	for _, h := range typeHandlers {
		h(event, ctx)
	}
	for _, h := range anyHandlers {
		h(event, ctx)
	}
}
