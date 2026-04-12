package utils

import (
	util "github.com/pzqf/zCommon/util"
)

type RetryFunc = util.RetryFunc
type RetryConfig = util.RetryConfig

var (
	DefaultRetryConfig = util.DefaultRetryConfig
	Retry              = util.Retry
	RetryWithDefault   = util.RetryWithDefault
	SimpleRetry        = util.SimpleRetry
)
