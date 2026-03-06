package util

import (
	"runtime"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// RecoverFunc 恢复函数类型
type RecoverFunc func(recover interface{}, stack string)

// DefaultRecoverFunc 默认恢复函数
var DefaultRecoverFunc RecoverFunc = func(recover interface{}, stack string) {
	zLog.Fatal("Server crashed with panic",
		zap.Any("panic", recover),
		zap.String("stack", stack),
	)
}

// Recover 通用的错误恢复函数
func Recover(recoverFunc ...RecoverFunc) {
	if r := recover(); r != nil {
		// 获取堆栈信息
		stack := make([]byte, 4096)
		stack = stack[:runtime.Stack(stack, false)]

		// 调用恢复函数
		if len(recoverFunc) > 0 && recoverFunc[0] != nil {
			recoverFunc[0](r, string(stack))
		} else {
			DefaultRecoverFunc(r, string(stack))
		}
	}
}

// Safe 安全执行函数
func Safe(f func(), recoverFunc ...RecoverFunc) {
	defer Recover(recoverFunc...)
	f()
}

// SafeGoroutine 安全启动协程
func SafeGoroutine(f func(), recoverFunc ...RecoverFunc) {
	go Safe(f, recoverFunc...)
}
