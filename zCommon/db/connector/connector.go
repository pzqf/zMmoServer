package connector

import (
	"database/sql"
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

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

// DBQuery 数据库查询请求
type DBQuery struct {
	Query    string
	Args     []interface{}
	Callback func(*sql.Rows, error)
}

// DBConnector 数据库连接器接口
type DBConnector interface {
	Init(dbConfig DBConfig) error
	Start() error
	Query(sql string, args []interface{}, callback func(*sql.Rows, error))
	QuerySync(sql string, args ...interface{}) (*sql.Rows, error)
	Execute(sql string, args []interface{}, callback func(sql.Result, error))
	ExecSync(sql string, args ...interface{}) (sql.Result, error)
	Close() error
	GetDriver() string
	GetMongoClient() *mongo.Client
	GetMongoDB() *mongo.Database
}

// BaseConnector 基础数据库连接器实现
type BaseConnector struct {
	name        string          // 数据库名称
	dbConfig    DBConfig        // 数据库配置
	driver      string          // 数据库驱动类型
	mu          sync.Mutex      // 互斥锁，用于并发控制
	mongoClient *mongo.Client   // MongoDB客户端
	mongoDB     *mongo.Database // MongoDB数据库
}

// NewDBConnector 创建数据库连接器实例
func NewDBConnector(name string, driver string, capacity int) DBConnector {
	if capacity <= 0 {
		capacity = 1000
	}

	// 根据驱动类型创建不同的数据库连接器
	switch driver {
	case "mongo":
		return NewMongoConnector(name)
	case "mysql":
		return NewMySQLConnector(name, capacity)
	default:
		// 默认使用MySQL驱动
		zLog.GetLogger().Warn("Unknown database driver, using MySQL as default", zap.String("driver", driver))
		return NewMySQLConnector(name, capacity)
	}
}
