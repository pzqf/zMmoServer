package component

import "github.com/pzqf/zMmoServer/GameServer/game/common"

// BaseComponent 基础组件实现
type BaseComponent struct {
	id         string
	gameObject common.IGameObject
	active     bool
}

// NewBaseComponent 创建基础组件
func NewBaseComponent(id string) *BaseComponent {
	return &BaseComponent{
		id:     id,
		active: true,
	}
}

// GetID 获取组件ID
func (c *BaseComponent) GetID() string {
	return c.id
}

// GetGameObject 获取组件所属的游戏对象
func (c *BaseComponent) GetGameObject() common.IGameObject {
	return c.gameObject
}

// SetGameObject 设置组件所属的游戏对象
func (c *BaseComponent) SetGameObject(obj common.IGameObject) {
	c.gameObject = obj
}

// Init 初始化组件
func (c *BaseComponent) Init() error {
	return nil
}

// Update 更新组件
func (c *BaseComponent) Update(deltaTime float64) {
}

// Destroy 销毁组件
func (c *BaseComponent) Destroy() {
}

// IsActive 检查组件是否激活
func (c *BaseComponent) IsActive() bool {
	return c.active
}

// SetActive 设置组件是否激活
func (c *BaseComponent) SetActive(active bool) {
	c.active = active
}

// ComponentManager 组件管理器
type ComponentManager struct {
	components map[string]common.IComponent
	gameObject common.IGameObject
}

// NewComponentManager 创建组件管理器
func NewComponentManager(obj common.IGameObject) *ComponentManager {
	return &ComponentManager{
		components: make(map[string]common.IComponent),
		gameObject: obj,
	}
}

// AddComponent 添加组件
func (cm *ComponentManager) AddComponent(component common.IComponent) {
	component.SetGameObject(cm.gameObject)
	component.Init()
	cm.components[component.GetID()] = component
}

// RemoveComponent 移除组件
func (cm *ComponentManager) RemoveComponent(componentID string) {
	if component, exists := cm.components[componentID]; exists {
		component.Destroy()
		delete(cm.components, componentID)
	}
}

// GetComponent 获取组件
func (cm *ComponentManager) GetComponent(componentID string) common.IComponent {
	return cm.components[componentID]
}

// HasComponent 检查是否存在指定组件
func (cm *ComponentManager) HasComponent(componentID string) bool {
	_, exists := cm.components[componentID]
	return exists
}

// GetAllComponents 获取所有组件
func (cm *ComponentManager) GetAllComponents() []common.IComponent {
	components := make([]common.IComponent, 0, len(cm.components))
	for _, component := range cm.components {
		components = append(components, component)
	}
	return components
}

// Update 更新所有组件
func (cm *ComponentManager) Update(deltaTime float64) {
	for _, component := range cm.components {
		if component.IsActive() {
			component.Update(deltaTime)
		}
	}
}

// Destroy 销毁所有组件
func (cm *ComponentManager) Destroy() {
	for _, component := range cm.components {
		component.Destroy()
	}
	cm.components = make(map[string]common.IComponent)
}
