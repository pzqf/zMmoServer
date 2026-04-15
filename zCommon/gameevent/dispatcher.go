package gameevent

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

type EventID uint32

type Priority int

const (
	PriorityLowest Priority = iota
	PriorityLow
	PriorityNormal
	PriorityHigh
	PriorityHighest
	PriorityMonitor
)

type Event struct {
	ID        EventID
	Name      string
	Timestamp time.Time
	Source    interface{}
	Data      interface{}
	Canceled  bool
}

func NewEvent(id EventID, name string, source interface{}, data interface{}) *Event {
	return &Event{
		ID:        id,
		Name:      name,
		Timestamp: time.Now(),
		Source:    source,
		Data:      data,
		Canceled:  false,
	}
}

func (e *Event) Cancel() {
	e.Canceled = true
}

func (e *Event) IsCanceled() bool {
	return e.Canceled
}

type EventHandler func(event *Event)

type handlerEntry struct {
	handler  EventHandler
	priority Priority
	name     string
}

type EventDispatcher struct {
	handlers  *zMap.TypedMap[EventID, []*handlerEntry]
	running   atomic.Bool
	eventPool chan *Event
	poolSize  int
}

func NewEventDispatcher(poolSize int) *EventDispatcher {
	if poolSize <= 0 {
		poolSize = 1024
	}

	d := &EventDispatcher{
		handlers:  zMap.NewTypedMap[EventID, []*handlerEntry](),
		eventPool: make(chan *Event, poolSize),
		poolSize:  poolSize,
	}
	d.running.Store(true)
	return d
}

func (d *EventDispatcher) Subscribe(eventID EventID, handler EventHandler, opts ...func(*handlerEntry)) {
	entry := &handlerEntry{
		handler:  handler,
		priority: PriorityNormal,
		name:     "anonymous",
	}

	for _, opt := range opts {
		opt(entry)
	}

	entries, exists := d.handlers.Load(eventID)
	if !exists {
		entries = make([]*handlerEntry, 0, 4)
	}

	inserted := false
	for i, e := range entries {
		if entry.priority > e.priority {
			entries = append(entries[:i+1], entries[i:]...)
			entries[i] = entry
			inserted = true
			break
		}
	}
	if !inserted {
		entries = append(entries, entry)
	}

	d.handlers.Store(eventID, entries)

	zLog.Debug("Event handler subscribed",
		zap.Uint32("event_id", uint32(eventID)),
		zap.String("handler_name", entry.name),
		zap.Int("priority", int(entry.priority)))
}

func WithPriority(p Priority) func(*handlerEntry) {
	return func(e *handlerEntry) {
		e.priority = p
	}
}

func WithName(name string) func(*handlerEntry) {
	return func(e *handlerEntry) {
		e.name = name
	}
}

func (d *EventDispatcher) Unsubscribe(eventID EventID, handlerName string) {
	entries, exists := d.handlers.Load(eventID)
	if !exists {
		return
	}

	filtered := make([]*handlerEntry, 0, len(entries))
	for _, e := range entries {
		if e.name != handlerName {
			filtered = append(filtered, e)
		}
	}

	if len(filtered) == 0 {
		d.handlers.Delete(eventID)
	} else {
		d.handlers.Store(eventID, filtered)
	}
}

func (d *EventDispatcher) Dispatch(event *Event) {
	if !d.running.Load() {
		return
	}

	entries, exists := d.handlers.Load(event.ID)
	if !exists || len(entries) == 0 {
		return
	}

	for _, entry := range entries {
		if event.Canceled && entry.priority < PriorityMonitor {
			break
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					zLog.Error("Panic in event handler",
						zap.String("event", event.Name),
						zap.String("handler", entry.name),
						zap.Any("recover", r))
				}
			}()
			entry.handler(event)
		}()
	}
}

func (d *EventDispatcher) DispatchAsync(event *Event) {
	if !d.running.Load() {
		return
	}

	select {
	case d.eventPool <- event:
	default:
		zLog.Warn("Event pool full, dropping event",
			zap.String("event", event.Name),
			zap.Int("pool_size", d.poolSize))
	}
}

func (d *EventDispatcher) ProcessEvents() {
	for {
		select {
		case event := <-d.eventPool:
			d.Dispatch(event)
		default:
			return
		}
	}
}

func (d *EventDispatcher) StartProcessor() {
	go func() {
		for d.running.Load() {
			d.ProcessEvents()
			time.Sleep(time.Millisecond)
		}
	}()
}

func (d *EventDispatcher) Close() {
	if !d.running.CompareAndSwap(true, false) {
		return
	}

	d.handlers.Range(func(eventID EventID, entries []*handlerEntry) bool {
		d.handlers.Delete(eventID)
		return true
	})

	for len(d.eventPool) > 0 {
		<-d.eventPool
	}

	zLog.Info("EventDispatcher closed")
}

func (d *EventDispatcher) HandlerCount(eventID EventID) int {
	entries, exists := d.handlers.Load(eventID)
	if !exists {
		return 0
	}
	return len(entries)
}

func (d *EventDispatcher) PendingCount() int {
	return len(d.eventPool)
}

type GameEvents struct {
	dispatcher *EventDispatcher
	mu         sync.Mutex
}

var globalGameEvents *GameEvents
var once sync.Once

func GetGlobalGameEvents() *GameEvents {
	once.Do(func() {
		globalGameEvents = NewGameEvents(4096)
		globalGameEvents.dispatcher.StartProcessor()
	})
	return globalGameEvents
}

func NewGameEvents(poolSize int) *GameEvents {
	return &GameEvents{
		dispatcher: NewEventDispatcher(poolSize),
	}
}

func (ge *GameEvents) On(eventID EventID, handler EventHandler, opts ...func(*handlerEntry)) {
	ge.dispatcher.Subscribe(eventID, handler, opts...)
}

func (ge *GameEvents) Off(eventID EventID, handlerName string) {
	ge.dispatcher.Unsubscribe(eventID, handlerName)
}

func (ge *GameEvents) Fire(event *Event) {
	ge.dispatcher.Dispatch(event)
}

func (ge *GameEvents) FireAsync(event *Event) {
	ge.dispatcher.DispatchAsync(event)
}

func (ge *GameEvents) FireEvent(id EventID, name string, source interface{}, data interface{}) {
	ge.dispatcher.Dispatch(NewEvent(id, name, source, data))
}

func (ge *GameEvents) Close() {
	ge.dispatcher.Close()
}

func RegisterEventName(id EventID, name string) string {
	return fmt.Sprintf("%d:%s", id, name)
}
