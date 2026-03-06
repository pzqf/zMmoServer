package tables

import (
	"github.com/pzqf/zMmoShared/config/models"
)

// ItemTableLoader 物品表格加载器
// 负责从Excel加载物品配置数据
type ItemTableLoader struct {
	items map[int32]*models.ItemBase // 物品配置映射（物品ID -> 配置）
}

// NewItemTableLoader 创建物品表格加载器
// 返回: 初始化后的物品表加载器实例
func NewItemTableLoader() *ItemTableLoader {
	return &ItemTableLoader{
		items: make(map[int32]*models.ItemBase),
	}
}

// Load 加载物品表数据
// 从item.xlsx文件读取物品配置
// 参数:
//   - dir: Excel文件所在目录
//
// 返回: 加载错误
func (itl *ItemTableLoader) Load(dir string) error {
	config := ExcelConfig{
		FileName:   "item.xlsx",
		SheetName:  "Sheet1",
		MinColumns: 11,
		TableName:  "items",
	}

	// 使用临时map批量加载数据
	tempItems := make(map[int32]*models.ItemBase)

	err := ReadExcelFile(config, dir, func(row []string) error {
		item := &models.ItemBase{
			ItemID:      StrToInt32(row[0]),
			Name:        row[1],
			Type:        StrToInt32(row[2]),
			SubType:     StrToInt32(row[3]),
			Level:       StrToInt32(row[4]),
			Quality:     StrToInt32(row[5]),
			Price:       StrToInt32(row[6]),
			SellPrice:   StrToInt32(row[7]),
			StackLimit:  StrToInt32(row[8]),
			Description: row[9],
			Effects:     row[10],
		}

		tempItems[item.ItemID] = item
		return nil
	})

	// 加载完成后一次性赋值
	if err == nil {
		itl.items = tempItems
	}

	return err
}

// GetTableName 获取表格名称
// 返回: 表格名称"items"
func (itl *ItemTableLoader) GetTableName() string {
	return "items"
}

// GetItem 根据ID获取物品
// 参数:
//   - itemID: 物品ID
//
// 返回: 物品配置和是否存在
func (itl *ItemTableLoader) GetItem(itemID int32) (*models.ItemBase, bool) {
	item, ok := itl.items[itemID]
	return item, ok
}

// GetAllItems 获取所有物品
// 返回配置的副本map，避免外部修改内部数据
// 返回: 物品配置映射副本
func (itl *ItemTableLoader) GetAllItems() map[int32]*models.ItemBase {
	// 创建一个副本，避免外部修改内部数据
	itemsCopy := make(map[int32]*models.ItemBase, len(itl.items))
	for id, item := range itl.items {
		itemsCopy[id] = item
	}
	return itemsCopy
}
