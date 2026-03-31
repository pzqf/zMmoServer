package tables

import (
	"github.com/pzqf/zCommon/config/models"
)

// ItemTableLoader 物品表加载器
// 从Excel表加载物品配置数据
type ItemTableLoader struct {
	items map[int32]*models.ItemBase // 物品配置映射（物品ID -> 配置）
}

// NewItemTableLoader 创建物品表加载器
// 功能: 初始化物品表加载器实例
func NewItemTableLoader() *ItemTableLoader {
	return &ItemTableLoader{
		items: make(map[int32]*models.ItemBase),
	}
}

// Load ������Ʒ������
// ��item.xlsx�ļ���ȡ��Ʒ����
// ����:
//   - dir: Excel�ļ�����Ŀ¼
//
// ����: ���ش���
func (itl *ItemTableLoader) Load(dir string) error {
	config := ExcelConfig{
		FileName:   "item.xlsx",
		SheetName:  "Sheet1",
		MinColumns: 11,
		TableName:  "items",
	}

	// ʹ����ʱmap������������
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

	// ������ɺ�һ���Ը�ֵ
	if err == nil {
		itl.items = tempItems
	}

	return err
}

// GetTableName ��ȡ��������
// ����: ��������"items"
func (itl *ItemTableLoader) GetTableName() string {
	return "items"
}

// GetItem ����ID��ȡ��Ʒ
// ����:
//   - itemID: ��ƷID
//
// ����: ��Ʒ���ú��Ƿ����
func (itl *ItemTableLoader) GetItem(itemID int32) (*models.ItemBase, bool) {
	item, ok := itl.items[itemID]
	return item, ok
}

// GetAllItems ��ȡ������Ʒ
// �������õĸ���map�������ⲿ�޸��ڲ�����
// ����: ��Ʒ����ӳ�丱��
func (itl *ItemTableLoader) GetAllItems() map[int32]*models.ItemBase {
	// ����һ�������������ⲿ�޸��ڲ�����
	itemsCopy := make(map[int32]*models.ItemBase, len(itl.items))
	for id, item := range itl.items {
		itemsCopy[id] = item
	}
	return itemsCopy
}

