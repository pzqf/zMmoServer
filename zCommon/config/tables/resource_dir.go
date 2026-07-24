package tables

import (
	"os"
	"path/filepath"
)

// ResourceDirEnv 是覆盖资源根目录的环境变量名。
const ResourceDirEnv = "ZMMO_RESOURCE_DIR"

// ResolveResourceDir 健壮地解析资源根目录（其下应含 excel_tables/、maps/ 等）。
//
// 背景（成熟化改造 Phase 2.5 去硬编码）：此前各处用 cwd 相对硬编码路径
// （LoadAllTables 用 "resources"，GameServer 用 "../resources"），随运行目录不同
// 而失效（GameServer 从 repo 根跑时 "../resources" 指向仓库外→退默认地图）。
// 本函数按优先级解析，消除 cwd 依赖与不一致：
//  1. 环境变量 ZMMO_RESOURCE_DIR（部署/容器覆盖）
//  2. 从 cwd 向上逐级查找含 resources/excel_tables 的目录
//  3. 从可执行文件所在目录向上逐级查找
//  4. 兜底：cwd/resources
func ResolveResourceDir() string {
	if d := os.Getenv(ResourceDirEnv); d != "" {
		return d
	}
	if cwd, err := os.Getwd(); err == nil {
		if d := findResourcesUpward(cwd); d != "" {
			return d
		}
	}
	if exe, err := os.Executable(); err == nil {
		if d := findResourcesUpward(filepath.Dir(exe)); d != "" {
			return d
		}
	}
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "resources")
}

// ExcelTablesDir 返回配置表（excel_tables）目录的绝对/相对路径，基于 ResolveResourceDir。
func ExcelTablesDir() string {
	return filepath.Join(ResolveResourceDir(), "excel_tables")
}

// findResourcesUpward 从 start 起向上最多 6 层，查找存在 resources/excel_tables 的目录，
// 命中则返回该目录下的 resources 路径；未命中返回空串。
func findResourcesUpward(start string) string {
	dir := start
	for i := 0; i < 6; i++ {
		if st, err := os.Stat(filepath.Join(dir, "resources", "excel_tables")); err == nil && st.IsDir() {
			return filepath.Join(dir, "resources")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
