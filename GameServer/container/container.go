package container

import (
	zcont "github.com/pzqf/zCommon/container"
)

type Container = zcont.Container

var NewContainer = zcont.NewContainer

func GetAs[T any](c *Container, key string) (T, bool) {
	return zcont.GetAs[T](c, key)
}
