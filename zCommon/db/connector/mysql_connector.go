package connector

import (
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zCommon/metrics"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// CacheItem 缓存项
type CacheItem struct {
	Value      interface{}
	ExpireTime time.Time
}

// CacheManager 缓存管理器
type CacheManager struct {
	cache map[string]*CacheItem
	mu    sync.RWMutex
}

// NewCacheManager 创建缓存管理器实例
func NewCacheManager() *CacheManager {
	return &CacheManager{
		cache: make(map[string]*CacheItem),
	}
}

// Get 获取缓存项
func (cm *CacheManager) Get(key string) (interface{}, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	item, ok := cm.cache[key]
	if !ok {
		return nil, false
	}

	// 检查缓存是否过期
	if time.Now().After(item.ExpireTime) {
		delete(cm.cache, key)
		return nil, false
	}

	return item.Value, true
}

// Set 设置缓存项
func (cm *CacheManager) Set(key string, value interface{}, duration time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.cache[key] = &CacheItem{
		Value:      value,
		ExpireTime: time.Now().Add(duration),
	}
}

// Delete 删除缓存项
func (cm *CacheManager) Delete(key string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.cache, key)
}

// Clear 清理过期缓存项
func (cm *CacheManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	for key, item := range cm.cache {
		if now.After(item.ExpireTime) {
			delete(cm.cache, key)
		}
	}
}

// MySQLConnector MySQL数据库连接器实现
type MySQLConnector struct {
	BaseConnector
	db           *sql.DB                  // MySQL数据库连接
	wg           sync.WaitGroup           // 等待组，用于优雅关闭
	isRunning    atomic.Bool              // 运行状态（原子：多 goroutine 读写，此前裸 bool 竞争）
	done         chan struct{}            // 关闭信号：替代 close(queryCh)，杜绝 send-on-closed panic
	closeOnce    sync.Once                // 保证 Close 只执行一次
	queryCh      chan *DBQuery            // 查询通道
	capacity     int                      // 通道容量
	workerCount  int                      // 工作协程数量
	metrics      *metrics.BusinessMetrics // 数据库指标监控
	cacheManager *CacheManager            // 缓存管理器
}

// NewMySQLConnector 创建MySQL数据库连接器
func NewMySQLConnector(name string, capacity int) *MySQLConnector {
	if capacity <= 0 {
		capacity = 1000
	}
	return &MySQLConnector{
		BaseConnector: BaseConnector{
			name:   name,
			driver: "mysql",
		},
		queryCh:      make(chan *DBQuery, capacity),
		done:         make(chan struct{}),
		capacity:     capacity,
		workerCount:  10, // 默认10个工作协程
		metrics:      metrics.GetBusinessMetrics("mysql_" + name),
		cacheManager: NewCacheManager(),
	}
}

// Init 初始化MySQL数据库连接
func (c *MySQLConnector) Init(dbConfig DBConfig) error {
	c.dbConfig = dbConfig
	c.driver = dbConfig.Driver

	// 设置默认字符集
	charset := dbConfig.Charset
	if charset == "" {
		charset = "utf8mb4"
	}

	// 设置默认超时
	timeout := dbConfig.ConnectTimeout
	if timeout <= 0 {
		timeout = 30
	}

	// 构建DSN字符串（添加超时参数）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local&timeout=%ds&readTimeout=%ds&writeTimeout=%ds",
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.DBName,
		charset,
		timeout,
		timeout,
		timeout,
	)

	// 连接信息脱敏：不打印含密码的 DSN（OPT-3）。
	zLog.Info("Connecting to MySQL database",
		zap.String("host", dbConfig.Host),
		zap.Int("port", dbConfig.Port),
		zap.String("dbname", dbConfig.DBName),
		zap.String("user", dbConfig.User))

	// 打开数据库连接
	var err error
	c.db, err = sql.Open("mysql", dsn)
	if err != nil {
		zLog.Error("Failed to open MySQL connection", zap.Error(err))
		return err
	}

	// 配置连接池
	c.db.SetMaxIdleConns(dbConfig.MaxIdle)
	c.db.SetMaxOpenConns(dbConfig.MaxOpen)
	c.db.SetConnMaxLifetime(time.Hour)

	// 测试连接
	if err := c.db.Ping(); err != nil {
		zLog.Error("Failed to ping MySQL database", zap.Error(err))
		return err
	}

	zLog.Info("MySQL connection established",
		zap.String("host", dbConfig.Host),
		zap.Int("port", dbConfig.Port),
		zap.String("dbname", dbConfig.DBName),
	)

	return nil
}

// Start 启动MySQL数据库连接和查询处理协程
func (c *MySQLConnector) Start() error {
	if c.isRunning.Load() {
		return nil
	}

	zLog.Info("Starting MySQL connector...", zap.String("name", c.name))
	c.isRunning.Store(true)

	// 启动多个查询处理协程
	zLog.Info("Starting query worker goroutines...", zap.Int("count", c.workerCount))
	for i := 0; i < c.workerCount; i++ {
		c.wg.Add(1)
		go c.queryWorker()
	}
	zLog.Info("Query worker goroutines started")

	// 启动缓存清理协程
	zLog.Info("Starting cache cleaner goroutine...")
	c.wg.Add(1)
	go c.cacheCleaner()
	zLog.Info("Cache cleaner goroutine started")

	// 启动指标监控协程
	zLog.Info("Starting metrics printer goroutine...")
	c.wg.Add(1)
	go c.metricsPrinter()
	zLog.Info("Metrics printer goroutine started")

	zLog.Info("MySQL connector started successfully", zap.String("name", c.name))
	return nil
}

// cacheCleaner 缓存清理协程
func (c *MySQLConnector) cacheCleaner() {
	defer c.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// select on done：既修 isRunning 裸 bool 竞争，也去掉此前每 100ms 的忙轮询。
	for {
		select {
		case <-ticker.C:
			c.cacheManager.Clear()
		case <-c.done:
			return
		}
	}
}

// metricsPrinter 指标监控协程
func (c *MySQLConnector) metricsPrinter() {
	defer c.wg.Done()

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			counters := c.metrics.GetAllCounters()
			timers := c.metrics.GetAllTimers()
			stats := make(map[string]interface{})
			for k, v := range counters {
				stats[k] = v
			}
			for k, v := range timers {
				stats[k] = v.Milliseconds()
			}
			zLog.Debug("Database metrics",
				zap.String("database", c.name),
				zap.Any("stats", stats),
			)
		case <-c.done:
			return
		}
	}
}

// queryWorker 处理MySQL数据库查询请求的工作协程
func (c *MySQLConnector) queryWorker() {
	defer c.wg.Done()

	for {
		select {
		case query := <-c.queryCh:
			startTime := time.Now()
			c.metrics.IncCounter("total_queries")

			rows, err := c.db.Query(query.Query, query.Args...)
			c.metrics.RecordTimer("query_latency", time.Since(startTime))

			if err != nil {
				c.metrics.IncCounter("total_errors")
				zLog.Error("Failed to execute MySQL query", zap.Error(err), zap.String("sql", query.Query))
			}

			if query.Callback != nil {
				query.Callback(rows, err)
			} else if rows != nil {
				_ = rows.Close() // 无回调时也须关闭 rows，否则连接泄漏（INF-8）
			}
		case <-c.done:
			return
		}
	}
}

// Query 异步执行MySQL数据库查询
func (c *MySQLConnector) Query(sql string, args []interface{}, callback func(*sql.Rows, error)) {
	if !c.isRunning.Load() {
		zLog.Error("MySQLConnector is not running")
		if callback != nil {
			callback(nil, fmt.Errorf("mysql connector is not running"))
		}
		return
	}

	// 发送查询请求到通道。select 含 done：Close 后不再向 queryCh 发送——配合 Close 不
	// close(queryCh) 一起杜绝 send-on-closed panic（此前 Close 关 channel 与本处发送有竞态）。
	select {
	case c.queryCh <- &DBQuery{Query: sql, Args: args, Callback: callback}:
	case <-c.done:
		if callback != nil {
			callback(nil, fmt.Errorf("mysql connector is closing"))
		}
	default:
		zLog.Error("MySQL query channel is full")
		if callback != nil {
			callback(nil, fmt.Errorf("mysql query channel is full"))
		}
	}
}

// Execute 异步执行MySQL数据库执行操作（插入、更新、删除等）
func (c *MySQLConnector) Execute(sql string, args []interface{}, callback func(sql.Result, error)) {
	if !c.isRunning.Load() {
		zLog.Error("MySQLConnector is not running")
		if callback != nil {
			callback(nil, fmt.Errorf("mysql connector is not running"))
		}
		return
	}

	// 异步执行查询
	go func() {
		startTime := time.Now()
		c.metrics.IncCounter("total_executes")

		result, err := c.db.Exec(sql, args...)
		latency := time.Since(startTime)
		c.metrics.RecordTimer("execute_latency", latency)

		if err != nil {
			c.metrics.IncCounter("total_errors")
			zLog.Error("Failed to execute MySQL statement", zap.Error(err), zap.String("sql", sql))
		}

		if callback != nil {
			callback(result, err)
		}
	}()
}

// Close 关闭MySQL数据库连接
func (c *MySQLConnector) Close() error {
	var closeErr error
	// closeOnce：防止重复 Close 二次 close(done) panic；用 done 信号停 worker，
	// 不 close(queryCh)——配合 Query 侧 select done，杜绝 send-on-closed。
	c.closeOnce.Do(func() {
		if !c.isRunning.Load() {
			return
		}
		c.isRunning.Store(false)
		close(c.done)
		c.wg.Wait()
		if c.db != nil {
			if err := c.db.Close(); err != nil {
				closeErr = fmt.Errorf("failed to close MySQL connection: %v", err)
				return
			}
		}
		zLog.Info("MySQL connection closed")
	})
	return closeErr
}

// GetDriver 获取当前数据库驱动类型
func (c *MySQLConnector) GetDriver() string {
	return c.driver
}

// GetSQLDB 返回底层 *sql.DB，供需要直接用 database/sql 的组件（如持久化 Outbox/Inbox）复用同一连接池。
func (c *MySQLConnector) GetSQLDB() *sql.DB {
	return c.db
}

// GetMongoClient 获取MongoDB客户端（MySQL实现中不支持）
func (c *MySQLConnector) GetMongoClient() *mongo.Client {
	zLog.Warn("GetMongoClient called on MySQLConnector")
	return nil
}

// GetMongoDB 获取MongoDB数据库（MySQL实现中不支持）
func (c *MySQLConnector) GetMongoDB() *mongo.Database {
	zLog.Warn("GetMongoDB called on MySQLConnector")
	return nil
}

func (c *MySQLConnector) QuerySync(query string, args ...interface{}) (*sql.Rows, error) {
	if !c.isRunning.Load() {
		return nil, fmt.Errorf("mysql connector is not running")
	}
	startTime := time.Now()
	c.metrics.IncCounter("total_queries")
	rows, err := c.db.Query(query, args...)
	c.metrics.RecordTimer("query_latency", time.Since(startTime))
	if err != nil {
		c.metrics.IncCounter("total_errors")
	}
	return rows, err
}

func (c *MySQLConnector) ExecSync(query string, args ...interface{}) (sql.Result, error) {
	if !c.isRunning.Load() {
		return nil, fmt.Errorf("mysql connector is not running")
	}
	startTime := time.Now()
	c.metrics.IncCounter("total_executes")
	result, err := c.db.Exec(query, args...)
	c.metrics.RecordTimer("execute_latency", time.Since(startTime))
	if err != nil {
		c.metrics.IncCounter("total_errors")
	}
	return result, err
}

