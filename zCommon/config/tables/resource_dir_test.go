package tables

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveResourceDir_EnvOverride(t *testing.T) {
	t.Setenv(ResourceDirEnv, "/custom/res/path")
	if got := ResolveResourceDir(); got != "/custom/res/path" {
		t.Fatalf("env override not honored: got %q", got)
	}
}

func TestResolveResourceDir_UpwardSearch(t *testing.T) {
	// 确保环境变量不干扰本用例
	t.Setenv(ResourceDirEnv, "")

	// 构造 tmp/resources/excel_tables，并从其深层子目录出发
	root := t.TempDir()
	excel := filepath.Join(root, "resources", "excel_tables")
	if err := os.MkdirAll(excel, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("mkdir deep: %v", err)
	}

	oldwd, _ := os.Getwd()
	defer os.Chdir(oldwd)
	if err := os.Chdir(deep); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	got := ResolveResourceDir()
	// 需能从深层子目录向上找到 root/resources。用 EvalSymlinks 规避 macOS/tmp 软链差异。
	wantResolved, _ := filepath.EvalSymlinks(filepath.Join(root, "resources"))
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != wantResolved {
		t.Fatalf("upward search failed: got %q want %q", gotResolved, wantResolved)
	}

	if filepath.Base(ExcelTablesDir()) != "excel_tables" {
		t.Fatalf("ExcelTablesDir should end with excel_tables, got %q", ExcelTablesDir())
	}
}
