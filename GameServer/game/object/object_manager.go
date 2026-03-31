package object

import (
	"errors"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/pzqf/zUtil/zMap"
)

// ObjectManager 游戏对象管理器
type ObjectManager struct {
	objects       *zMap.Map
	objectsByType *zMap.TypedMap[common.GameObjectType, *zMap.TypedMap[id.ObjectIdType, common.IGameObject]]
}

// NewObjectManager 创建对象管理器
func NewObjectManager() *ObjectManager {
	return &ObjectManager{
		objects:       zMap.NewMap(),
		objectsByType: zMap.NewTypedMap[common.GameObjectType, *zMap.TypedMap[id.ObjectIdType, common.IGameObject]](),
	}
}

// AddObject 添加对象
func (om *ObjectManager) AddObject(obj common.IGameObject) error {
	if obj == nil {
		return errors.New("object can't be nil")
	}

	objectID := obj.GetID()
	if objectID == 0 {
		return errors.New("object id can't be 0")
	}

	_, exists := om.objects.Load(objectID)
	if exists {
		return errors.New("object already exists")
	}

	om.objects.Store(objectID, obj)

	objectType := obj.GetType()
	typeMap, ok := om.objectsByType.Load(objectType)
	if !ok {
		typeMap = zMap.NewTypedMap[id.ObjectIdType, common.IGameObject]()
		om.objectsByType.Store(objectType, typeMap)
	}
	typeMap.Store(objectID, obj)

	return nil
}

// GetObject 获取对象
func (om *ObjectManager) GetObject(objectID id.ObjectIdType) (common.IGameObject, error) {
	v, ok := om.objects.Load(objectID)
	if !ok {
		return nil, errors.New("object not found")
	}
	return v.(common.IGameObject), nil
}

// RemoveObject 移除对象
func (om *ObjectManager) RemoveObject(objectID id.ObjectIdType) error {
	v, ok := om.objects.Load(objectID)
	if !ok {
		return errors.New("object not found")
	}

	obj := v.(common.IGameObject)
	objectType := obj.GetType()

	om.objects.Delete(objectID)

	if typeMap, ok := om.objectsByType.Load(objectType); ok {
		typeMap.Delete(objectID)
	}

	return nil
}

// GetObjectsByType 获取指定类型的所有对象
func (om *ObjectManager) GetObjectsByType(objectType common.GameObjectType) []common.IGameObject {
	typeMap, ok := om.objectsByType.Load(objectType)
	if !ok {
		return nil
	}

	result := make([]common.IGameObject, 0)
	typeMap.Range(func(key id.ObjectIdType, value common.IGameObject) bool {
		result = append(result, value)
		return true
	})
	return result
}

// GetAllObjects 获取所有对象
func (om *ObjectManager) GetAllObjects() []common.IGameObject {
	objs := make([]common.IGameObject, 0)
	om.objects.Range(func(key, value interface{}) bool {
		objs = append(objs, value.(common.IGameObject))
		return true
	})
	return objs
}

// GetObjectCount 获取对象数量
func (om *ObjectManager) GetObjectCount() int64 {
	return om.objects.Len()
}

// GetObjectCountByType 获取指定类型的对象数量
func (om *ObjectManager) GetObjectCountByType(objectType common.GameObjectType) int {
	typeMap, ok := om.objectsByType.Load(objectType)
	if !ok {
		return 0
	}

	count := 0
	typeMap.Range(func(key id.ObjectIdType, value common.IGameObject) bool {
		count++
		return true
	})
	return count
}

// UpdateAll 更新所有对象
func (om *ObjectManager) UpdateAll(deltaTime float64) {
	om.objects.Range(func(key, value interface{}) bool {
		obj := value.(common.IGameObject)
		if obj.IsActive() {
			obj.Update(deltaTime)
		}
		return true
	})
}

// ClearAll 清除所有对象
func (om *ObjectManager) ClearAll() {
	om.objects.Range(func(key, value interface{}) bool {
		obj := value.(common.IGameObject)
		obj.Destroy()
		return true
	})

	om.objects.Clear()
	om.objectsByType.Clear()
}

// Range 遍历所有对象
func (om *ObjectManager) Range(f func(objectID id.ObjectIdType, obj common.IGameObject) bool) {
	om.objects.Range(func(key, value interface{}) bool {
		return f(key.(id.ObjectIdType), value.(common.IGameObject))
	})
}
