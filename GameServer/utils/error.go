package utils

import (
	util "github.com/pzqf/zCommon/util"
)

type ErrorWithStack = util.ErrorWithStack

var (
	WrapError   = util.WrapError
	WrapErrorf  = util.WrapErrorf
	Must        = util.Must
	HandleError = util.HandleError
)
