package drop

import (
	"math/rand"
	"time"
)

// DropItem 掉落物品
type DropItem struct {
	ItemID    int64   // 物品ID
	MinCount  int     // 最小数量
	MaxCount  int     // 最大数量
	Weight    int     // 掉落权重
	Chance    float64 // 掉落概率（0-1）
	IsRare    bool    // 是否稀有
	BindType  int     // 绑定类型（0:不绑定, 1:拾取绑定, 2:装备绑定）
}

// DropGroup 掉落组
type DropGroup struct {
	GroupID    int         // 组ID
	Name       string      // 组名称
	DropType   int         // 掉落类型
	MinDrops   int         // 最少掉落数量
	MaxDrops   int         // 最多掉落数量
	Items      []*DropItem // 掉落物品列表
	TotalWeight int        // 总权重
}

// DropType 掉落类型常量
const (
	DropTypeAll      = 0 // 全部掉落
	DropTypeRandom   = 1 // 随机掉落
	DropTypeWeight   = 2 // 权重掉落
	DropTypeSequence = 3 // 顺序掉落
)

// DropResult 掉落结果
type DropResult struct {
	ItemID   int64 // 物品ID
	Count    int   // 数量
	IsRare   bool  // 是否稀有
	BindType int   // 绑定类型
}

// DropTable 掉落表
type DropTable struct {
	TableID     int           // 表ID
	Name        string        // 表名称
	Description string        // 描述
	Groups      []*DropGroup  // 掉落组列表
	GoldMin     int64         // 最小金币
	GoldMax     int64         // 最大金币
	ExpMin      int64         // 最小经验
	ExpMax      int64         // 最大经验
}

// NewDropGroup 创建掉落组
func NewDropGroup(groupID int, name string, dropType int) *DropGroup {
	return &DropGroup{
		GroupID:  groupID,
		Name:     name,
		DropType: dropType,
		MinDrops: 1,
		MaxDrops: 1,
		Items:    []*DropItem{},
	}
}

// AddItem 添加掉落物品
func (dg *DropGroup) AddItem(item *DropItem) {
	dg.Items = append(dg.Items, item)
	dg.TotalWeight += item.Weight
}

// CalculateDrop 计算掉落
func (dg *DropGroup) CalculateDrop() []*DropResult {
	results := make([]*DropResult, 0)

	switch dg.DropType {
	case DropTypeAll:
		// 全部掉落
		for _, item := range dg.Items {
			if rollDrop(item.Chance) {
				count := randomCount(item.MinCount, item.MaxCount)
				results = append(results, &DropResult{
					ItemID:   item.ItemID,
					Count:    count,
					IsRare:   item.IsRare,
					BindType: item.BindType,
				})
			}
		}

	case DropTypeRandom:
		// 随机掉落指定数量
		dropCount := randomCount(dg.MinDrops, dg.MaxDrops)
		for i := 0; i < dropCount && i < len(dg.Items); i++ {
			item := dg.Items[rand.Intn(len(dg.Items))]
			if rollDrop(item.Chance) {
				count := randomCount(item.MinCount, item.MaxCount)
				results = append(results, &DropResult{
					ItemID:   item.ItemID,
					Count:    count,
					IsRare:   item.IsRare,
					BindType: item.BindType,
				})
			}
		}

	case DropTypeWeight:
		// 权重掉落
		dropCount := randomCount(dg.MinDrops, dg.MaxDrops)
		for i := 0; i < dropCount; i++ {
			if item := dg.rollByWeight(); item != nil {
				count := randomCount(item.MinCount, item.MaxCount)
				results = append(results, &DropResult{
					ItemID:   item.ItemID,
					Count:    count,
					IsRare:   item.IsRare,
					BindType: item.BindType,
				})
			}
		}

	case DropTypeSequence:
		// 顺序掉落，按顺序检查直到成功
		for _, item := range dg.Items {
			if rollDrop(item.Chance) {
				count := randomCount(item.MinCount, item.MaxCount)
				results = append(results, &DropResult{
					ItemID:   item.ItemID,
					Count:    count,
					IsRare:   item.IsRare,
					BindType: item.BindType,
				})
				break
			}
		}
	}

	return results
}

// rollByWeight 按权重随机选择
func (dg *DropGroup) rollByWeight() *DropItem {
	if dg.TotalWeight <= 0 {
		return nil
	}

	randomWeight := rand.Intn(dg.TotalWeight)
	currentWeight := 0

	for _, item := range dg.Items {
		currentWeight += item.Weight
		if randomWeight < currentWeight {
			return item
		}
	}

	return nil
}

// rollDrop 掷骰子判断是否掉落
func rollDrop(chance float64) bool {
	if chance >= 1.0 {
		return true
	}
	if chance <= 0 {
		return false
	}
	return rand.Float64() < chance
}

// randomCount 随机数量
func randomCount(min, max int) int {
	if min >= max {
		return min
	}
	return min + rand.Intn(max-min+1)
}

// CalculateDrops 计算掉落表的所有掉落
func (dt *DropTable) CalculateDrops() ([]*DropResult, int64, int64) {
	allResults := make([]*DropResult, 0)

	// 计算各组掉落
	for _, group := range dt.Groups {
		results := group.CalculateDrop()
		allResults = append(allResults, results...)
	}

	// 计算金币
	var gold int64 = 0
	if dt.GoldMax > dt.GoldMin {
		gold = dt.GoldMin + int64(rand.Intn(int(dt.GoldMax-dt.GoldMin)+1))
	} else {
		gold = dt.GoldMin
	}

	// 计算经验
	var exp int64 = 0
	if dt.ExpMax > dt.ExpMin {
		exp = dt.ExpMin + int64(rand.Intn(int(dt.ExpMax-dt.ExpMin)+1))
	} else {
		exp = dt.ExpMin
	}

	return allResults, gold, exp
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
