package db

// DBConfig 数据库配置
type DBConfig struct {
	Host           string // 数据库主机
	Port           int    // 数据库端口
	User           string // 数据库用户名
	Password       string // 数据库密码
	DBName         string // 数据库名称
	Charset        string // 字符集
	MaxIdle        int    // 最大空闲连接数
	MaxOpen        int    // 最大打开连接数
	Driver         string // 数据库驱动类型: mysql, mongo
	URI            string // 数据库连接URI（用于MongoDB等支持URI的数据库）
	MaxPoolSize    int    // 连接池最大连接数（MongoDB）
	MinPoolSize    int    // 连接池最小连接数（MongoDB）
	ConnectTimeout int    // 连接超时时间（秒，MongoDB）
}
