#!/bin/bash

# 构建脚本

# 设置版本信息
VERSION=$(git describe --tags --always || echo "0.0.1")
BUILD_TIME=$(date +"%Y-%m-%d %H:%M:%S")
GIT_COMMIT=$(git rev-parse HEAD || echo "unknown")
GO_VERSION=$(go version | awk '{print $3}')
OS=$(uname -s)
ARCH=$(uname -m)

# 生成版本文件
echo "package version

// Version 版本号
const Version = \"$VERSION\"

// BuildTime 构建时间
var BuildTime = \"$BUILD_TIME\"

// GitCommit Git提交哈希
var GitCommit = \"$GIT_COMMIT\"

// GoVersion Go版本
var GoVersion = \"$GO_VERSION\"

// OS 操作系统
var OS = \"$OS\"

// Arch 架构
var Arch = \"$ARCH\"

// GetVersion 获取完整版本信息
func GetVersion() map[string]string {
	return map[string]string{
		\"version\":    Version,
		\"build_time\": BuildTime,
		\"git_commit\": GitCommit,
		\"go_version\": GoVersion,
		\"os\":         OS,
		\"arch\":       Arch,
	}
}
" > version/version.go

# 构建
echo "Building with version: $VERSION"
go build -o bin/global-server main.go

# 显示版本信息
echo "Build completed. Version info:"
echo "Version: $VERSION"
echo "Build Time: $BUILD_TIME"
echo "Git Commit: $GIT_COMMIT"
echo "Go Version: $GO_VERSION"
echo "OS: $OS"
echo "Arch: $ARCH"
