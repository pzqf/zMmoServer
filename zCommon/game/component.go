package game

type IComponent interface {
	GetID() string
	Init() error
	Update(deltaTime float64)
	Destroy()
	IsActive() bool
	SetActive(active bool)
}

type BaseComponent struct {
	id     string
	active bool
}

func NewBaseComponent(id string) BaseComponent {
	return BaseComponent{id: id, active: true}
}

func (c *BaseComponent) GetID() string {
	return c.id
}

func (c *BaseComponent) Init() error {
	return nil
}

func (c *BaseComponent) Update(deltaTime float64) {}

func (c *BaseComponent) Destroy() {}

func (c *BaseComponent) IsActive() bool {
	return c.active
}

func (c *BaseComponent) SetActive(active bool) {
	c.active = active
}
