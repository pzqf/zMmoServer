package version

// Version 版本号
const Version = "0.0.1"

// BuildTime 构建时间
var BuildTime = ""

// GitCommit Git提交哈希
var GitCommit = ""

// GoVersion Go版本
var GoVersion = ""

// OS 操作系统
var OS = ""

// Arch 架构
var Arch = ""

// GetVersion 获取完整版本信息
func GetVersion() map[string]string {
	return map[string]string{
		"version":    Version,
		"build_time": BuildTime,
		"git_commit": GitCommit,
		"go_version": GoVersion,
		"os":         OS,
		"arch":       Arch,
	}
}
