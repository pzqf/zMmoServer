package game

// Warehouse 仓库：与背包分离的存储容器（业务层建设，2026-07-25）。
// 内部复用 Inventory 的存取/堆叠逻辑，但类型独立、容量通常更大，语义上与背包区分——
// 存入/取出是跨容器搬运（背包↔仓库），由 Player 层的 handler 编排（先从一端取出、再存入另一端）。
type Warehouse struct {
	inv *Inventory
}

func NewWarehouse(size int32) *Warehouse {
	return &Warehouse{inv: NewInventory(size)}
}

func (w *Warehouse) GetSize() int32                   { return w.inv.GetSize() }
func (w *Warehouse) GetItem(slot int32) (*Item, bool) { return w.inv.GetItem(slot) }
func (w *Warehouse) GetAllItems() map[int32]*Item     { return w.inv.GetAllItems() }

// Store 存入一件物品，返回落入的格子（复用背包的堆叠/找空位）。
func (w *Warehouse) Store(item *Item) (int32, error) { return w.inv.AddItem(item) }

// Retrieve 从指定格取出 count 个（复用背包移除语义）。
func (w *Warehouse) Retrieve(slot, count int32) (*Item, error) { return w.inv.RemoveItem(slot, count) }
