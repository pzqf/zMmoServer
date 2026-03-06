package tables

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

// ExcelConfig Excel表格配置
type ExcelConfig struct {
	FileName   string // 文件名
	SheetName  string // 工作表名
	MinColumns int    // 最小列数
	TableName  string // 表格名称（用于日志）
}

// ReadExcelFile 通用Excel文件读取函数
func ReadExcelFile(config ExcelConfig, dir string, rowProcessor func([]string) error) error {
	// 构建文件路径
	filePath := filepath.Join(dir, config.FileName)

	// 打开Excel文件
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", config.FileName, err)
	}
	defer f.Close()

	// 获取指定工作表的所有行
	rows, err := f.GetRows(config.SheetName)
	if err != nil {
		return fmt.Errorf("failed to get rows from %s: %w", config.FileName, err)
	}

	// 跳过表头行，处理数据行
	successCount := 0
	for i := 1; i < len(rows); i++ {
		row := rows[i]

		// 检查是否有足够的列
		if len(row) < config.MinColumns {
			zap.L().Warn(fmt.Sprintf("%s row %d has insufficient columns (expected: %d, actual: %d)",
				config.FileName, i+1, config.MinColumns, len(row)))
			continue
		}

		// 调用行处理器
		if err := rowProcessor(row); err != nil {
			zap.L().Error(fmt.Sprintf("%s row %d processing error: %v", config.FileName, i+1, err))
			continue
		}

		successCount++
	}

	zap.L().Info(fmt.Sprintf("Loaded %d %s from %s", successCount, config.TableName, config.FileName))
	return nil
}

// StrToInt32 将字符串转换为int32（无错误处理，向后兼容）
func StrToInt32(s string) int32 {
	v, _ := StrToInt32WithError(s)
	return v
}

// StrToInt32WithError 将字符串转换为int32（带错误处理）
func StrToInt32WithError(s string) (int32, error) {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to convert string '%s' to int32: %w", s, err)
	}
	return int32(v), nil
}

// StrToInt64 将字符串转换为int64（无错误处理，向后兼容）
func StrToInt64(s string) int64 {
	v, _ := StrToInt64WithError(s)
	return v
}

// StrToInt64WithError 将字符串转换为int64（带错误处理）
func StrToInt64WithError(s string) (int64, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to convert string '%s' to int64: %w", s, err)
	}
	return v, nil
}

// StrToFloat32 将字符串转换为float32（无错误处理，向后兼容）
func StrToFloat32(s string) float32 {
	v, _ := StrToFloat32WithError(s)
	return v
}

// StrToFloat32WithError 将字符串转换为float32（带错误处理）
func StrToFloat32WithError(s string) (float32, error) {
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to convert string '%s' to float32: %w", s, err)
	}
	return float32(v), nil
}

// StrToFloat64 将字符串转换为float64（无错误处理，向后兼容）
func StrToFloat64(s string) float64 {
	v, _ := StrToFloat64WithError(s)
	return v
}

// StrToFloat64WithError 将字符串转换为float64（带错误处理）
func StrToFloat64WithError(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to convert string '%s' to float64: %w", s, err)
	}
	return v, nil
}
