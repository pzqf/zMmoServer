package container

import (
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

type Container struct {
	components map[string]interface{}
	mutex      sync.RWMutex
}

func NewContainer() *Container {
	return &Container{
		components: make(map[string]interface{}),
	}
}

func (c *Container) Register(name string, component interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.components[name] = component
	zLog.Debug("Component registered", zap.String("name", name))
}

func (c *Container) Get(name string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	component, exists := c.components[name]
	return component, exists
}

func (c *Container) GetOrPanic(name string) interface{} {
	component, exists := c.Get(name)
	if !exists {
		zLog.Panic("Component not found", zap.String("name", name))
	}
	return component
}

func (c *Container) Remove(name string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.components, name)
	zLog.Debug("Component removed", zap.String("name", name))
}

func (c *Container) GetAll() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	components := make(map[string]interface{})
	for name, component := range c.components {
		components[name] = component
	}
	return components
}

func (c *Container) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.components)
}

func (c *Container) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.components = make(map[string]interface{})
	zLog.Debug("Container cleared")
}

func GetAs[T any](c *Container, key string) (T, bool) {
	var zero T
	component, exists := c.Get(key)
	if !exists || component == nil {
		return zero, false
	}
	t, ok := component.(T)
	return t, ok
}
