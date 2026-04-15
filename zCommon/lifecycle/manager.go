package lifecycle

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

type ObjectState uint8

const (
	ObjectStateNone ObjectState = iota
	ObjectStateCreating
	ObjectStateActive
	ObjectStateSuspending
	ObjectStateSuspended
	ObjectStateDestroying
	ObjectStateDestroyed
)

func (s ObjectState) String() string {
	switch s {
	case ObjectStateNone:
		return "None"
	case ObjectStateCreating:
		return "Creating"
	case ObjectStateActive:
		return "Active"
	case ObjectStateSuspending:
		return "Suspending"
	case ObjectStateSuspended:
		return "Suspended"
	case ObjectStateDestroying:
		return "Destroying"
	case ObjectStateDestroyed:
		return "Destroyed"
	default:
		return "Unknown"
	}
}

type Object interface {
	GetObjectID() int64
	SetObjectID(id int64)
	GetObjectType() string
	GetState() ObjectState
	SetState(state ObjectState)
}

type BaseLifecycleObject struct {
	ObjectID   int64       `json:"object_id"`
	ObjectType string      `json:"object_type"`
	State      ObjectState `json:"state"`
	CreatedAt  int64       `json:"created_at"`
	UpdatedAt  int64       `json:"updated_at"`
}

func (o *BaseLifecycleObject) GetObjectID() int64 {
	return o.ObjectID
}

func (o *BaseLifecycleObject) SetObjectID(id int64) {
	o.ObjectID = id
}

func (o *BaseLifecycleObject) GetObjectType() string {
	return o.ObjectType
}

func (o *BaseLifecycleObject) GetState() ObjectState {
	return o.State
}

func (o *BaseLifecycleObject) SetState(state ObjectState) {
	o.State = state
	o.UpdatedAt = time.Now().UnixMilli()
}

type LifecycleHook func(obj Object) error

type LifecycleHooks struct {
	OnCreate   LifecycleHook
	OnActivate LifecycleHook
	OnSuspend  LifecycleHook
	OnResume   LifecycleHook
	OnDestroy  LifecycleHook
}

type Serializer interface {
	Serialize(obj Object) ([]byte, error)
	Deserialize(data []byte) (Object, error)
}

type JSONSerializer struct {
	factory func() Object
}

func NewJSONSerializer(factory func() Object) *JSONSerializer {
	return &JSONSerializer{factory: factory}
}

func (s *JSONSerializer) Serialize(obj Object) ([]byte, error) {
	return json.Marshal(obj)
}

func (s *JSONSerializer) Deserialize(data []byte) (Object, error) {
	obj := s.factory()
	if err := json.Unmarshal(data, obj); err != nil {
		return nil, fmt.Errorf("unmarshal object: %w", err)
	}
	return obj, nil
}

type ObjectFactory func(id int64) Object

type Manager struct {
	objects    *zMap.TypedMap[int64, Object]
	hooks      *zMap.TypedMap[string, *LifecycleHooks]
	serializers *zMap.TypedMap[string, Serializer]
	factories  *zMap.TypedMap[string, ObjectFactory]
	mu         sync.Mutex
}

func NewManager() *Manager {
	return &Manager{
		objects:     zMap.NewTypedMap[int64, Object](),
		hooks:       zMap.NewTypedMap[string, *LifecycleHooks](),
		serializers: zMap.NewTypedMap[string, Serializer](),
		factories:   zMap.NewTypedMap[string, ObjectFactory](),
	}
}

func (m *Manager) RegisterType(objectType string, factory ObjectFactory, hooks *LifecycleHooks) {
	m.factories.Store(objectType, factory)
	if hooks != nil {
		m.hooks.Store(objectType, hooks)
	}
	zLog.Info("Object type registered",
		zap.String("type", objectType))
}

func (m *Manager) RegisterSerializer(objectType string, serializer Serializer) {
	m.serializers.Store(objectType, serializer)
}

func (m *Manager) Create(objectType string, id int64) (Object, error) {
	factory, exists := m.factories.Load(objectType)
	if !exists {
		return nil, fmt.Errorf("unknown object type: %s", objectType)
	}

	obj := factory(id)
	obj.SetState(ObjectStateCreating)

	if hooks, ok := m.hooks.Load(objectType); ok && hooks.OnCreate != nil {
		if err := hooks.OnCreate(obj); err != nil {
			obj.SetState(ObjectStateDestroyed)
			return nil, fmt.Errorf("onCreate hook failed: %w", err)
		}
	}

	obj.SetState(ObjectStateActive)
	m.objects.Store(id, obj)

	zLog.Debug("Object created",
		zap.String("type", objectType),
		zap.Int64("id", id))

	return obj, nil
}

func (m *Manager) Get(id int64) (Object, bool) {
	return m.objects.Load(id)
}

func (m *Manager) Destroy(id int64) error {
	obj, exists := m.objects.Load(id)
	if !exists {
		return fmt.Errorf("object not found: %d", id)
	}

	obj.SetState(ObjectStateDestroying)

	if hooks, ok := m.hooks.Load(obj.GetObjectType()); ok && hooks.OnDestroy != nil {
		if err := hooks.OnDestroy(obj); err != nil {
			zLog.Error("OnDestroy hook failed",
				zap.Int64("id", id),
				zap.Error(err))
		}
	}

	obj.SetState(ObjectStateDestroyed)
	m.objects.Delete(id)

	zLog.Debug("Object destroyed",
		zap.String("type", obj.GetObjectType()),
		zap.Int64("id", id))

	return nil
}

func (m *Manager) Suspend(id int64) error {
	obj, exists := m.objects.Load(id)
	if !exists {
		return fmt.Errorf("object not found: %d", id)
	}

	obj.SetState(ObjectStateSuspending)

	if hooks, ok := m.hooks.Load(obj.GetObjectType()); ok && hooks.OnSuspend != nil {
		if err := hooks.OnSuspend(obj); err != nil {
			return fmt.Errorf("onSuspend hook failed: %w", err)
		}
	}

	obj.SetState(ObjectStateSuspended)
	return nil
}

func (m *Manager) Resume(id int64) error {
	obj, exists := m.objects.Load(id)
	if !exists {
		return fmt.Errorf("object not found: %d", id)
	}

	if hooks, ok := m.hooks.Load(obj.GetObjectType()); ok && hooks.OnResume != nil {
		if err := hooks.OnResume(obj); err != nil {
			return fmt.Errorf("onResume hook failed: %w", err)
		}
	}

	obj.SetState(ObjectStateActive)
	return nil
}

func (m *Manager) Serialize(id int64) ([]byte, error) {
	obj, exists := m.objects.Load(id)
	if !exists {
		return nil, fmt.Errorf("object not found: %d", id)
	}

	serializer, exists := m.serializers.Load(obj.GetObjectType())
	if !exists {
		return nil, fmt.Errorf("no serializer for type: %s", obj.GetObjectType())
	}

	return serializer.Serialize(obj)
}

func (m *Manager) Deserialize(objectType string, data []byte) (Object, error) {
	serializer, exists := m.serializers.Load(objectType)
	if !exists {
		return nil, fmt.Errorf("no serializer for type: %s", objectType)
	}

	obj, err := serializer.Deserialize(data)
	if err != nil {
		return nil, fmt.Errorf("deserialize: %w", err)
	}

	m.objects.Store(obj.GetObjectID(), obj)
	return obj, nil
}

func (m *Manager) Count() int {
	return int(m.objects.Len())
}

func (m *Manager) CountByType(objectType string) int {
	count := 0
	m.objects.Range(func(id int64, obj Object) bool {
		if obj.GetObjectType() == objectType {
			count++
		}
		return true
	})
	return count
}

func (m *Manager) CountByState(state ObjectState) int {
	count := 0
	m.objects.Range(func(id int64, obj Object) bool {
		if obj.GetState() == state {
			count++
		}
		return true
	})
	return count
}

func (m *Manager) Range(fn func(id int64, obj Object) bool) {
	m.objects.Range(fn)
}

func (m *Manager) DestroyAll() {
	m.objects.Range(func(id int64, obj Object) bool {
		if err := m.Destroy(id); err != nil {
			zLog.Error("Failed to destroy object",
				zap.Int64("id", id),
				zap.Error(err))
		}
		return true
	})
}
