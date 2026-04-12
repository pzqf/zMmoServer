package util

import (
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

type ErrorWithStack struct {
	Err   error
	Stack string
}

func (e *ErrorWithStack) Error() string {
	return fmt.Sprintf("%v\n%s", e.Err, e.Stack)
}

func (e *ErrorWithStack) Unwrap() error {
	return e.Err
}

func WrapError(err error, message string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	wrappedErr := fmt.Errorf(message+": %w", append(args, err)...)
	zLog.Error(message, zap.Error(err))
	return wrappedErr
}

func WrapErrorf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func HandleError(err error, message string, fields ...zap.Field) {
	if err != nil {
		zLog.Error(message, append([]zap.Field{zap.Error(err)}, fields...)...)
	}
}
