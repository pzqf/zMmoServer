package lifecycle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	m := NewManager()

	assert.NotNil(t, m)
	assert.Equal(t, 0, m.Count())
}

func TestManagerRegisterType(t *testing.T) {
	m := NewManager()

	m.RegisterType("player", func(id int64) Object {
		return &BaseLifecycleObject{ObjectID: id, ObjectType: "player"}
	}, nil)

	obj, err := m.Create("player", 1)

	assert.NoError(t, err)
	assert.NotNil(t, obj)
	assert.Equal(t, int64(1), obj.GetObjectID())
	assert.Equal(t, "player", obj.GetObjectType())
	assert.Equal(t, ObjectStateActive, obj.GetState())
}

func TestManagerCreateUnknownType(t *testing.T) {
	m := NewManager()

	_, err := m.Create("unknown", 1)
	assert.Error(t, err)
}

func TestManagerDestroy(t *testing.T) {
	m := NewManager()
	m.RegisterType("player", func(id int64) Object {
		return &BaseLifecycleObject{ObjectID: id, ObjectType: "player"}
	}, nil)

	_, _ = m.Create("player", 1)
	assert.Equal(t, 1, m.Count())

	err := m.Destroy(1)
	assert.NoError(t, err)
	assert.Equal(t, 0, m.Count())
}

func TestManagerDestroyNotFound(t *testing.T) {
	m := NewManager()

	err := m.Destroy(999)
	assert.Error(t, err)
}

func TestManagerSuspendResume(t *testing.T) {
	m := NewManager()
	m.RegisterType("player", func(id int64) Object {
		return &BaseLifecycleObject{ObjectID: id, ObjectType: "player"}
	}, nil)

	_, _ = m.Create("player", 1)

	err := m.Suspend(1)
	assert.NoError(t, err)

	obj, ok := m.Get(1)
	assert.True(t, ok)
	assert.Equal(t, ObjectStateSuspended, obj.GetState())

	err = m.Resume(1)
	assert.NoError(t, err)

	obj, _ = m.Get(1)
	assert.Equal(t, ObjectStateActive, obj.GetState())
}

func TestManagerSuspendNotFound(t *testing.T) {
	m := NewManager()

	err := m.Suspend(999)
	assert.Error(t, err)
}

func TestManagerResumeNotFound(t *testing.T) {
	m := NewManager()

	err := m.Resume(999)
	assert.Error(t, err)
}

func TestManagerLifecycleHooks(t *testing.T) {
	m := NewManager()

	var createCalled, suspendCalled, resumeCalled, destroyCalled bool

	m.RegisterType("player", func(id int64) Object {
		return &BaseLifecycleObject{ObjectID: id, ObjectType: "player"}
	}, &LifecycleHooks{
		OnCreate: func(obj Object) error {
			createCalled = true
			return nil
		},
		OnSuspend: func(obj Object) error {
			suspendCalled = true
			return nil
		},
		OnResume: func(obj Object) error {
			resumeCalled = true
			return nil
		},
		OnDestroy: func(obj Object) error {
			destroyCalled = true
			return nil
		},
	})

	_, _ = m.Create("player", 1)
	assert.True(t, createCalled)

	_ = m.Suspend(1)
	assert.True(t, suspendCalled)

	_ = m.Resume(1)
	assert.True(t, resumeCalled)

	_ = m.Destroy(1)
	assert.True(t, destroyCalled)
}

func TestManagerCountByType(t *testing.T) {
	m := NewManager()
	m.RegisterType("player", func(id int64) Object {
		return &BaseLifecycleObject{ObjectID: id, ObjectType: "player"}
	}, nil)
	m.RegisterType("npc", func(id int64) Object {
		return &BaseLifecycleObject{ObjectID: id, ObjectType: "npc"}
	}, nil)

	_, _ = m.Create("player", 1)
	_, _ = m.Create("player", 2)
	_, _ = m.Create("npc", 3)

	assert.Equal(t, 2, m.CountByType("player"))
	assert.Equal(t, 1, m.CountByType("npc"))
	assert.Equal(t, 3, m.Count())
}

func TestManagerCountByState(t *testing.T) {
	m := NewManager()
	m.RegisterType("player", func(id int64) Object {
		return &BaseLifecycleObject{ObjectID: id, ObjectType: "player"}
	}, nil)

	_, _ = m.Create("player", 1)
	_, _ = m.Create("player", 2)
	_ = m.Suspend(2)

	assert.Equal(t, 1, m.CountByState(ObjectStateActive))
	assert.Equal(t, 1, m.CountByState(ObjectStateSuspended))
}

func TestManagerGet(t *testing.T) {
	m := NewManager()
	m.RegisterType("player", func(id int64) Object {
		return &BaseLifecycleObject{ObjectID: id, ObjectType: "player"}
	}, nil)

	_, _ = m.Create("player", 1)

	obj, ok := m.Get(1)
	assert.True(t, ok)
	assert.Equal(t, int64(1), obj.GetObjectID())

	_, ok = m.Get(999)
	assert.False(t, ok)
}

func TestManagerSerializeDeserialize(t *testing.T) {
	m := NewManager()
	m.RegisterType("player", func(id int64) Object {
		return &BaseLifecycleObject{ObjectID: id, ObjectType: "player"}
	}, nil)
	m.RegisterSerializer("player", NewJSONSerializer(func() Object {
		return &BaseLifecycleObject{}
	}))

	_, _ = m.Create("player", 1)

	data, err := m.Serialize(1)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	obj, err := m.Deserialize("player", data)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), obj.GetObjectID())
}

func TestObjectStateString(t *testing.T) {
	assert.Equal(t, "None", ObjectStateNone.String())
	assert.Equal(t, "Creating", ObjectStateCreating.String())
	assert.Equal(t, "Active", ObjectStateActive.String())
	assert.Equal(t, "Suspended", ObjectStateSuspended.String())
	assert.Equal(t, "Destroyed", ObjectStateDestroyed.String())
}

func TestBaseLifecycleObject(t *testing.T) {
	obj := &BaseLifecycleObject{ObjectID: 1, ObjectType: "player"}

	assert.Equal(t, int64(1), obj.GetObjectID())
	assert.Equal(t, "player", obj.GetObjectType())
	assert.Equal(t, ObjectStateNone, obj.GetState())

	obj.SetObjectID(2)
	assert.Equal(t, int64(2), obj.GetObjectID())

	obj.SetState(ObjectStateActive)
	assert.Equal(t, ObjectStateActive, obj.GetState())
}
