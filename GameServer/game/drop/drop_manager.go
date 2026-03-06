package drop

import (
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// DropManager 掉落管理器
type DropManager struct {
	dropTables map[int]*DropTable
	mutex      sync.RWMutex
}

// NewDropManager 创建掉落管理器
func NewDropManager() *DropManager {
	return &DropManager{
		dropTables: make(map[int]*DropTable),
	}
}

// AddDropTable 添加掉落表
func (dm *DropManager) AddDropTable(table *DropTable) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	dm.dropTables[table.TableID] = table
	zLog.Info("Drop table added",
		zap.Int("table_id", table.TableID),
		zap.String("name", table.Name))
}

// GetDropTable 获取掉落表
func (dm *DropManager) GetDropTable(tableID int) *DropTable {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	return dm.dropTables[tableID]
}

// RemoveDropTable 移除掉落表
func (dm *DropManager) RemoveDropTable(tableID int) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	delete(dm.dropTables, tableID)
}

// CalculateDrop 计算掉落
func (dm *DropManager) CalculateDrop(tableID int) ([]*DropResult, int64, int64) {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	table := dm.dropTables[tableID]
	if table == nil {
		return []*DropResult{}, 0, 0
	}

	return table.CalculateDrops()
}

// CalculateDropWithBonus 计算掉落（带加成）
func (dm *DropManager) CalculateDropWithBonus(tableID int, goldBonus, expBonus float64) ([]*DropResult, int64, int64) {
	items, gold, exp := dm.CalculateDrop(tableID)

	// 应用加成
	if goldBonus > 0 {
		gold = int64(float64(gold) * (1 + goldBonus))
	}
	if expBonus > 0 {
		exp = int64(float64(exp) * (1 + expBonus))
	}

	return items, gold, exp
}

// DropEntity 掉落实体（场景中的掉落物）
type DropEntity struct {
	EntityID    id.ObjectIdType // 实体ID
	DropTableID int             // 掉落表ID
	Items       []*DropResult   // 掉落物品
	Gold        int64           // 金币
	Exp         int64           // 经验
	OwnerID     id.PlayerIdType // 归属玩家ID
	IsProtected bool            // 是否受保护（只有归属者可拾取）
	ProtectTime int             // 保护时间（秒）
	PositionX   float64         // 位置X
	PositionY   float64         // 位置Y
	PositionZ   float64         // 位置Z
}

// NewDropEntity 创建掉落实体
func NewDropEntity(entityID id.ObjectIdType, dropTableID int, ownerID id.PlayerIdType, x, y, z float64) *DropEntity {
	return &DropEntity{
		EntityID:    entityID,
		DropTableID: dropTableID,
		Items:       []*DropResult{},
		Gold:        0,
		Exp:         0,
		OwnerID:     ownerID,
		IsProtected: ownerID != 0,
		ProtectTime: 30, // 默认30秒保护
		PositionX:   x,
		PositionY:   y,
		PositionZ:   z,
	}
}

// GenerateDrops 生成掉落
func (de *DropEntity) GenerateDrops(dm *DropManager) {
	items, gold, exp := dm.CalculateDrop(de.DropTableID)
	de.Items = items
	de.Gold = gold
	de.Exp = exp
}

// CanPickup 检查玩家是否可以拾取
func (de *DropEntity) CanPickup(playerID id.PlayerIdType) bool {
	if !de.IsProtected {
		return true
	}
	return de.OwnerID == playerID
}

// PickupItem 拾取指定物品
func (de *DropEntity) PickupItem(itemIndex int) *DropResult {
	if itemIndex < 0 || itemIndex >= len(de.Items) {
		return nil
	}

	item := de.Items[itemIndex]
	// 从列表中移除
	de.Items = append(de.Items[:itemIndex], de.Items[itemIndex+1:]...)

	return item
}

// PickupGold 拾取金币
func (de *DropEntity) PickupGold() int64 {
	gold := de.Gold
	de.Gold = 0
	return gold
}

// PickupExp 拾取经验
func (de *DropEntity) PickupExp() int64 {
	exp := de.Exp
	de.Exp = 0
	return exp
}

// IsEmpty 检查是否已空
func (de *DropEntity) IsEmpty() bool {
	return len(de.Items) == 0 && de.Gold == 0 && de.Exp == 0
}

// DropEntityManager 掉落实体管理器
type DropEntityManager struct {
	entities map[id.ObjectIdType]*DropEntity
	mutex    sync.RWMutex
}

// NewDropEntityManager 创建掉落实体管理器
func NewDropEntityManager() *DropEntityManager {
	return &DropEntityManager{
		entities: make(map[id.ObjectIdType]*DropEntity),
	}
}

// AddEntity 添加掉落实体
func (dem *DropEntityManager) AddEntity(entity *DropEntity) {
	dem.mutex.Lock()
	defer dem.mutex.Unlock()

	dem.entities[entity.EntityID] = entity
}

// GetEntity 获取掉落实体
func (dem *DropEntityManager) GetEntity(entityID id.ObjectIdType) *DropEntity {
	dem.mutex.RLock()
	defer dem.mutex.RUnlock()

	return dem.entities[entityID]
}

// RemoveEntity 移除掉落实体
func (dem *DropEntityManager) RemoveEntity(entityID id.ObjectIdType) {
	dem.mutex.Lock()
	defer dem.mutex.Unlock()

	delete(dem.entities, entityID)
}

// GetEntitiesInRange 获取范围内的掉落实体
func (dem *DropEntityManager) GetEntitiesInRange(x, y, z, radius float64) []*DropEntity {
	dem.mutex.RLock()
	defer dem.mutex.RUnlock()

	entities := make([]*DropEntity, 0)
	for _, entity := range dem.entities {
		// 简单的距离计算
		dx := entity.PositionX - x
		dy := entity.PositionY - y
		dz := entity.PositionZ - z
		distance := dx*dx + dy*dy + dz*dz

		if distance <= radius*radius {
			entities = append(entities, entity)
		}
	}

	return entities
}

// GetEntitiesByOwner 获取指定归属者的掉落实体
func (dem *DropEntityManager) GetEntitiesByOwner(ownerID id.PlayerIdType) []*DropEntity {
	dem.mutex.RLock()
	defer dem.mutex.RUnlock()

	entities := make([]*DropEntity, 0)
	for _, entity := range dem.entities {
		if entity.OwnerID == ownerID {
			entities = append(entities, entity)
		}
	}

	return entities
}

// CleanupEmptyEntities 清理空掉落实体
func (dem *DropEntityManager) CleanupEmptyEntities() int {
	dem.mutex.Lock()
	defer dem.mutex.Unlock()

	count := 0
	for id, entity := range dem.entities {
		if entity.IsEmpty() {
			delete(dem.entities, id)
			count++
		}
	}

	return count
}
