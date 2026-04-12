package version

import (
	zver "github.com/pzqf/zCommon/version"
)

const Version = zver.Version

var (
	BuildTime  = zver.BuildTime
	GitCommit  = zver.GitCommit
	GoVersion  = zver.GoVersion
	OS         = zver.OS
	Arch       = zver.Arch
)

func init() {
	zver.ServerName = "GatewayServer"
}

var GetVersion = zver.GetVersion
