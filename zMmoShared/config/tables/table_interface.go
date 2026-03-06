package tables

// TableLoaderInterface 表格加载器接口
type TableLoaderInterface interface {
	// Load 加载表格数据
	Load(dir string) error
	// GetTableName 获取表格名称
	GetTableName() string
}
