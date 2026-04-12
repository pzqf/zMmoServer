package container

import (
	zcont "github.com/pzqf/zCommon/container"
)

type Container = zcont.Container

var NewContainer = zcont.NewContainer

func RegisterSingleton(c *Container, name string, component interface{}) {
	c.Register(name, component)
}

func GetSingletonAs[T any](c *Container, key string) (T, bool) {
	return zcont.GetAs[T](c, key)
}
