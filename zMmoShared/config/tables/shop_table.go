package tables

import (
	"github.com/pzqf/zMmoShared/config/models"
)

// ShopTableLoader 商店表加载器
type ShopTableLoader struct {
	shops map[int32]*models.Shop // 商店配置映射
}

// NewShopTableLoader 创建商店表加载器
func NewShopTableLoader() *ShopTableLoader {
	return &ShopTableLoader{
		shops: make(map[int32]*models.Shop),
	}
}

// Load 加载商店表数据
func (stl *ShopTableLoader) Load(dir string) error {
	config := ExcelConfig{
		FileName:   "shop.xlsx",
		SheetName:  "Sheet1",
		MinColumns: 9,
		TableName:  "shops",
	}

	tempShops := make(map[int32]*models.Shop)

	err := ReadExcelFile(config, dir, func(row []string) error {
		shop := &models.Shop{
			ShopID:       StrToInt32(row[0]),
			ItemID:       StrToInt32(row[1]),
			Price:        StrToInt32(row[2]),
			CurrencyType: StrToInt32(row[3]),
			Stock:        StrToInt32(row[4]),
			LimitPerDay:  StrToInt32(row[5]),
			MinLevel:     StrToInt32(row[6]),
			ShopType:     StrToInt32(row[7]),
		}

		tempShops[shop.ShopID] = shop
		return nil
	})

	if err == nil {
		stl.shops = tempShops
	}

	return err
}

// GetTableName 获取表格名称
func (stl *ShopTableLoader) GetTableName() string {
	return "shops"
}

// GetShop 根据ID获取商店配置
func (stl *ShopTableLoader) GetShop(shopID int32) (*models.Shop, bool) {
	shop, ok := stl.shops[shopID]
	return shop, ok
}

// GetAllShops 获取所有商店配置
func (stl *ShopTableLoader) GetAllShops() map[int32]*models.Shop {
	shopsCopy := make(map[int32]*models.Shop, len(stl.shops))
	for id, shop := range stl.shops {
		shopsCopy[id] = shop
	}
	return shopsCopy
}
