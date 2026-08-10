package engine

import (
	"reflect"
	"sync"
)

// EventHandler 事件处理函数
type EventHandler func(event *GameEvent, ctx *EventContext)

// EventBus 事件总线
// 两种发布方式：
// - Publish: 新因果链 (创建全新 EventContext)
// - Emit:    在既有因果链内继续发布 (防事件风暴生效)
type EventBus struct {
	mu       sync.Mutex
	handlers map[string][]EventHandler
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

// Publish 发布事件 (创建新因果链)
func (b *EventBus) Publish(event *GameEvent) {
	b.dispatch(event, NewEventContext())
}

// Emit 在既有因果链内发布事件
func (b *EventBus) Emit(event *GameEvent, ctx *EventContext) {
	b.dispatch(event, ctx)
}

func (b *EventBus) dispatch(event *GameEvent, ctx *EventContext) {
	if !ctx.CanFire(event.ID) {
		return
	}
	ctx.MarkFired(event.ID)
	ctx.depth++
	defer func() { ctx.depth-- }()

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
