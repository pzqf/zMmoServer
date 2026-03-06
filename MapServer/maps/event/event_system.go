package event

import (
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"go.uber.org/zap"
)

// EventType 事件类型
type EventType string

const (
	EventTypeTrigger     EventType = "trigger"     // 触发点事件
	EventTypeArea        EventType = "area"        // 区域事件
	EventTypeTimed       EventType = "timed"       // 定时事件
	EventTypeInteraction EventType = "interaction" // 交互事件
	EventTypeWeather     EventType = "weather"     // 天气事件
	EventTypeStory       EventType = "story"       // 剧情事件
)

// Event 事件接口
type Event interface {
	GetID() int32
	GetName() string
	GetType() EventType
	IsActive() bool
	SetActive(active bool)
	Trigger(player *object.Player) bool
	Update()
}

// TriggerEvent 触发点事件
type TriggerEvent struct {
	id          int32
	name        string
	position    common.Vector3
	radius      float32
	active      bool
	triggered   bool
	cooldown    time.Duration
	lastTrigger time.Time
	action      func(player *object.Player) bool
}

// NewTriggerEvent 创建新的触发点事件
func NewTriggerEvent(id int32, name string, pos common.Vector3, radius float32, cooldown time.Duration, action func(player *object.Player) bool) *TriggerEvent {
	return &TriggerEvent{
		id:          id,
		name:        name,
		position:    pos,
		radius:      radius,
		active:      true,
		triggered:   false,
		cooldown:    cooldown,
		lastTrigger: time.Now().Add(-cooldown), // 初始化为冷却时间之前
		action:      action,
	}
}

// GetID 获取事件ID
func (e *TriggerEvent) GetID() int32 {
	return e.id
}

// GetName 获取事件名称
func (e *TriggerEvent) GetName() string {
	return e.name
}

// GetType 获取事件类型
func (e *TriggerEvent) GetType() EventType {
	return EventTypeTrigger
}

// IsActive 检查事件是否激活
func (e *TriggerEvent) IsActive() bool {
	return e.active
}

// SetActive 设置事件激活状态
func (e *TriggerEvent) SetActive(active bool) {
	e.active = active
}

// Trigger 触发事件
func (e *TriggerEvent) Trigger(player *object.Player) bool {
	if !e.active {
		return false
	}

	// 检查冷却时间
	if time.Since(e.lastTrigger) < e.cooldown {
		return false
	}

	// 检查玩家是否在触发范围内
	distance := player.GetPosition().DistanceTo(e.position)
	if distance > e.radius*e.radius {
		return false
	}

	// 执行事件动作
	success := e.action(player)
	if success {
		e.triggered = true
		e.lastTrigger = time.Now()
		zLog.Info("Trigger event activated",
			zap.Int32("event_id", e.id),
			zap.String("event_name", e.name),
			zap.String("player", player.GetName()))
	}

	return success
}

// Update 更新事件状态
func (e *TriggerEvent) Update() {
	// 重置触发状态
	if e.triggered && time.Since(e.lastTrigger) > e.cooldown {
		e.triggered = false
	}
}

// AreaEvent 区域事件
type AreaEvent struct {
	id            int32
	name          string
	position      common.Vector3
	radius        float32
	active        bool
	playersInArea map[int64]bool
	actionEnter   func(player *object.Player) bool
	actionLeave   func(player *object.Player) bool
}

// NewAreaEvent 创建新的区域事件
func NewAreaEvent(id int32, name string, pos common.Vector3, radius float32, actionEnter, actionLeave func(player *object.Player) bool) *AreaEvent {
	return &AreaEvent{
		id:            id,
		name:          name,
		position:      pos,
		radius:        radius,
		active:        true,
		playersInArea: make(map[int64]bool),
		actionEnter:   actionEnter,
		actionLeave:   actionLeave,
	}
}

// GetID 获取事件ID
func (e *AreaEvent) GetID() int32 {
	return e.id
}

// GetName 获取事件名称
func (e *AreaEvent) GetName() string {
	return e.name
}

// GetType 获取事件类型
func (e *AreaEvent) GetType() EventType {
	return EventTypeArea
}

// IsActive 检查事件是否激活
func (e *AreaEvent) IsActive() bool {
	return e.active
}

// SetActive 设置事件激活状态
func (e *AreaEvent) SetActive(active bool) {
	e.active = active
}

// Trigger 触发事件
func (e *AreaEvent) Trigger(player *object.Player) bool {
	if !e.active {
		return false
	}

	// 检查玩家是否在区域内
	distance := player.GetPosition().DistanceTo(e.position)
	isInArea := distance <= e.radius*e.radius
	playerID := int64(player.GetPlayerID())

	// 检查玩家之前是否在区域内
	wasInArea := e.playersInArea[playerID]

	if isInArea && !wasInArea {
		// 玩家进入区域
		success := e.actionEnter(player)
		if success {
			e.playersInArea[playerID] = true
			zLog.Info("Player entered area event",
				zap.Int32("event_id", e.id),
				zap.String("event_name", e.name),
				zap.String("player", player.GetName()))
		}
		return success
	} else if !isInArea && wasInArea {
		// 玩家离开区域
		success := e.actionLeave(player)
		if success {
			delete(e.playersInArea, playerID)
			zLog.Info("Player left area event",
				zap.Int32("event_id", e.id),
				zap.String("event_name", e.name),
				zap.String("player", player.GetName()))
		}
		return success
	}

	return false
}

// Update 更新事件状态
func (e *AreaEvent) Update() {
	// 区域事件不需要定期更新
}

// TimedEvent 定时事件
type TimedEvent struct {
	id          int32
	name        string
	active      bool
	interval    time.Duration
	lastTrigger time.Time
	action      func() bool
}

// NewTimedEvent 创建新的定时事件
func NewTimedEvent(id int32, name string, interval time.Duration, action func() bool) *TimedEvent {
	return &TimedEvent{
		id:          id,
		name:        name,
		active:      true,
		interval:    interval,
		lastTrigger: time.Now(),
		action:      action,
	}
}

// GetID 获取事件ID
func (e *TimedEvent) GetID() int32 {
	return e.id
}

// GetName 获取事件名称
func (e *TimedEvent) GetName() string {
	return e.name
}

// GetType 获取事件类型
func (e *TimedEvent) GetType() EventType {
	return EventTypeTimed
}

// IsActive 检查事件是否激活
func (e *TimedEvent) IsActive() bool {
	return e.active
}

// SetActive 设置事件激活状态
func (e *TimedEvent) SetActive(active bool) {
	e.active = active
}

// Trigger 触发事件
func (e *TimedEvent) Trigger(player *object.Player) bool {
	// 定时事件不需要玩家触发
	return false
}

// Update 更新事件状态
func (e *TimedEvent) Update() {
	if !e.active {
		return
	}

	if time.Since(e.lastTrigger) >= e.interval {
		success := e.action()
		if success {
			e.lastTrigger = time.Now()
			zLog.Info("Timed event activated",
				zap.Int32("event_id", e.id),
				zap.String("event_name", e.name))
		}
	}
}

// EventManager 事件管理器
type EventManager struct {
	mu     sync.RWMutex
	events map[int32]Event
}

// NewEventManager 创建新的事件管理器
func NewEventManager() *EventManager {
	return &EventManager{
		events: make(map[int32]Event),
	}
}

// AddEvent 添加事件
func (em *EventManager) AddEvent(event Event) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.events[event.GetID()] = event
	zLog.Info("Added event",
		zap.Int32("event_id", event.GetID()),
		zap.String("event_name", event.GetName()),
		zap.String("event_type", string(event.GetType())))
}

// RemoveEvent 移除事件
func (em *EventManager) RemoveEvent(eventID int32) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if event, exists := em.events[eventID]; exists {
		zLog.Info("Removed event",
			zap.Int32("event_id", event.GetID()),
			zap.String("event_name", event.GetName()))
		delete(em.events, eventID)
	}
}

// GetEvent 获取事件
func (em *EventManager) GetEvent(eventID int32) Event {
	em.mu.RLock()
	defer em.mu.RUnlock()

	return em.events[eventID]
}

// GetAllEvents 获取所有事件
func (em *EventManager) GetAllEvents() []Event {
	em.mu.RLock()
	defer em.mu.RUnlock()

	events := make([]Event, 0, len(em.events))
	for _, event := range em.events {
		events = append(events, event)
	}

	return events
}

// TriggerEvents 触发所有事件
func (em *EventManager) TriggerEvents(player *object.Player) {
	em.mu.RLock()
	events := make([]Event, 0, len(em.events))
	for _, event := range em.events {
		events = append(events, event)
	}
	em.mu.RUnlock()

	for _, event := range events {
		event.Trigger(player)
	}
}

// UpdateEvents 更新所有事件
func (em *EventManager) UpdateEvents() {
	em.mu.RLock()
	events := make([]Event, 0, len(em.events))
	for _, event := range em.events {
		events = append(events, event)
	}
	em.mu.RUnlock()

	for _, event := range events {
		event.Update()
	}
}

// CreateDefaultEvents 创建默认事件
func (em *EventManager) CreateDefaultEvents() {
	// 创建触发点事件
	triggerEvent := NewTriggerEvent(
		1,
		"Treasure Chest",
		common.Vector3{X: 100, Y: 100, Z: 0},
		5,
		30*time.Second,
		func(player *object.Player) bool {
			// 给玩家添加物品
			player.AddItem(1001) // 假设1001是宝箱物品
			zLog.Info("Player found treasure", zap.String("player", player.GetName()))
			return true
		},
	)
	em.AddEvent(triggerEvent)

	// 创建区域事件
	areaEvent := NewAreaEvent(
		2,
		"Healing Zone",
		common.Vector3{X: 200, Y: 200, Z: 0},
		20,
		func(player *object.Player) bool {
			// 玩家进入区域，开始恢复生命值
			player.SetHealth(player.GetMaxHealth())
			zLog.Info("Player entered healing zone", zap.String("player", player.GetName()))
			return true
		},
		func(player *object.Player) bool {
			// 玩家离开区域
			zLog.Info("Player left healing zone", zap.String("player", player.GetName()))
			return true
		},
	)
	em.AddEvent(areaEvent)

	// 创建定时事件
	timedEvent := NewTimedEvent(
		3,
		"Weather Change",
		5*time.Minute,
		func() bool {
			// 模拟天气变化
			zLog.Info("Weather changed")
			return true
		},
	)
	em.AddEvent(timedEvent)

	zLog.Info("Created default events", zap.Int("count", 3))
}
