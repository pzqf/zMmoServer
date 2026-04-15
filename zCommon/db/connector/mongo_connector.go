package connector

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// MongoConnector MongoDB数据库连接器实现
type MongoConnector struct {
	BaseConnector
	isRunning bool // 运行状态
}

// NewMongoConnector 创建MongoDB数据库连接器
func NewMongoConnector(name string) *MongoConnector {
	return &MongoConnector{
		BaseConnector: BaseConnector{
			name:   name,
			driver: "mongo",
		},
	}
}

// Init 初始化MongoDB数据库连接
func (c *MongoConnector) Init(dbConfig DBConfig) error {
	c.dbConfig = dbConfig
	c.driver = dbConfig.Driver

	// 构建MongoDB连接字符串
	var uri string
	if dbConfig.User != "" && dbConfig.Password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=admin",
			dbConfig.User,
			dbConfig.Password,
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.DBName,
		)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%d/%s",
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.DBName,
		)
	}

	// 创建MongoDB客户端选项
	clientOptions := options.Client().ApplyURI(uri)

	// 配置连接池
	clientOptions.SetMaxPoolSize(uint64(dbConfig.MaxOpen))
	clientOptions.SetMinPoolSize(uint64(dbConfig.MaxIdle))

	// 创建MongoDB客户端
	var err error
	c.mongoClient, err = mongo.Connect(nil, clientOptions)
	if err != nil {
		zLog.Error("Failed to create MongoDB client", zap.Error(err))
		return err
	}

	// 测试连接
	if err := c.mongoClient.Ping(nil, nil); err != nil {
		zLog.Error("Failed to ping MongoDB database", zap.Error(err))
		return err
	}

	// 获取MongoDB数据库实例
	c.mongoDB = c.mongoClient.Database(dbConfig.DBName)

	zLog.Info("MongoDB connection established",
		zap.String("host", dbConfig.Host),
		zap.Int("port", dbConfig.Port),
		zap.String("dbname", dbConfig.DBName),
	)

	if err := c.ensureIndexes(); err != nil {
		zLog.Warn("Failed to ensure MongoDB indexes", zap.Error(err))
	}

	return nil
}

func (c *MongoConnector) ensureIndexes() error {
	ctx := context.Background()
	dbName := c.mongoDB.Name()

	if dbName == "account" || dbName == "accounts" {
		accountsCollection := c.mongoDB.Collection("accounts")
		_, err := accountsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.D{{Key: "account_name", Value: 1}},
			Options: options.Index().SetUnique(true),
		})
		if err != nil {
			return fmt.Errorf("failed to create unique index on accounts: %w", err)
		}
		zLog.Info("MongoDB account indexes ensured")
	} else if dbName == "game" || dbName == "game_1" {
		playersCollection := c.mongoDB.Collection("players")
		_, err := playersCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.D{{Key: "player_name", Value: 1}},
			Options: options.Index().SetUnique(true),
		})
		if err != nil {
			return fmt.Errorf("failed to create unique index on players: %w", err)
		}
		zLog.Info("MongoDB game indexes ensured")
	}

	return nil
}

// Start 启动MongoDB数据库连接
func (c *MongoConnector) Start() error {
	if c.isRunning {
		return nil
	}

	c.isRunning = true

	// MongoDB不需要额外的启动逻辑，连接已经在Init时建立
	zLog.Info("MongoDB connector started")
	return nil
}

// Query 执行MongoDB查询操作
func (c *MongoConnector) Query(sql string, args []interface{}, callback func(*sql.Rows, error)) {
	if !c.isRunning {
		zLog.Error("MongoConnector is not running")
		if callback != nil {
			callback(nil, fmt.Errorf("mongo connector is not running"))
		}
		return
	}

	// MongoDB不直接支持SQL查询，这里需要将SQL查询转换为MongoDB查询
	// 这是一个简化实现，实际应用中需要更复杂的SQL到MongoDB查询转换
	zLog.Warn("SQL to MongoDB query conversion not fully implemented", zap.String("sql", sql))

	if callback != nil {
		callback(nil, fmt.Errorf("SQL query not supported in MongoDB"))
	}
}

// Execute 执行MongoDB执行操作
func (c *MongoConnector) Execute(sql string, args []interface{}, callback func(sql.Result, error)) {
	if !c.isRunning {
		zLog.Error("MongoConnector is not running")
		if callback != nil {
			callback(nil, fmt.Errorf("mongo connector is not running"))
		}
		return
	}

	// MongoDB不直接支持SQL执行，这里需要将SQL执行转换为MongoDB操作
	// 这是一个简化实现，实际应用中需要更复杂的SQL到MongoDB操作转换
	zLog.Warn("SQL to MongoDB execute conversion not fully implemented", zap.String("sql", sql))

	if callback != nil {
		callback(nil, fmt.Errorf("SQL execute not supported in MongoDB"))
	}
}

// Close 关闭MongoDB数据库连接
func (c *MongoConnector) Close() error {
	if !c.isRunning {
		return nil
	}

	c.isRunning = false

	// 关闭MongoDB连接
	if c.mongoClient != nil {
		if err := c.mongoClient.Disconnect(nil); err != nil {
			return fmt.Errorf("failed to close MongoDB connection: %v", err)
		}
	}

	zLog.Info("MongoDB connection closed")
	return nil
}

// GetDriver 获取当前数据库驱动类型
func (c *MongoConnector) GetDriver() string {
	return c.driver
}

// GetMongoClient 获取MongoDB客户端实例
func (c *MongoConnector) GetMongoClient() *mongo.Client {
	return c.mongoClient
}

// GetMongoDB 获取MongoDB数据库实例
func (c *MongoConnector) GetMongoDB() *mongo.Database {
	return c.mongoDB
}

func (c *MongoConnector) QuerySync(query string, args ...interface{}) (*sql.Rows, error) {
	return nil, fmt.Errorf("SQL query not supported in MongoDB, use GetMongoDB() directly")
}

func (c *MongoConnector) ExecSync(query string, args ...interface{}) (sql.Result, error) {
	return nil, fmt.Errorf("SQL execute not supported in MongoDB, use GetMongoDB() directly")
}
