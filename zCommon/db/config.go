package db

// DBConfig 数据库配置
type DBConfig struct {
	Host           string `ini:"host"`            // 数据库主机
	Port           int    `ini:"port"`            // 数据库端口
	User           string `ini:"user"`            // 数据库用户名
	Password       string `ini:"password"`        // 数据库密码
	DBName         string `ini:"dbname"`          // 数据库名称
	Charset        string `ini:"charset"`         // 字符集
	MaxIdle        int    `ini:"max_idle"`        // 最大空闲连接数
	MaxOpen        int    `ini:"max_open"`        // 最大打开连接数
	Driver         string `ini:"driver"`          // 数据库驱动类型: mysql, mongo
	URI            string `ini:"uri"`             // 数据库连接URI（用于MongoDB等支持URI的数据库）
	MaxPoolSize    int    `ini:"max_pool_size"`   // 连接池最大连接数（MongoDB）
	MinPoolSize    int    `ini:"min_pool_size"`   // 连接池最小连接数（MongoDB）
	ConnectTimeout int    `ini:"connect_timeout"` // 连接超时时间（秒，MongoDB）
}
