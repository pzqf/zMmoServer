package version

const Version = "0.0.1"

var (
	BuildTime = ""
	GitCommit = ""
	GoVersion = ""
	OS        = ""
	Arch      = ""
	ServerName = ""
)

func GetVersion() map[string]string {
	return map[string]string{
		"version":     Version,
		"build_time":  BuildTime,
		"git_commit":  GitCommit,
		"go_version":  GoVersion,
		"os":          OS,
		"arch":        Arch,
		"server_name": ServerName,
	}
}
