# Go 项目开发规则

## 技术栈

- **语言版本**: Go 1.25+
- **Web 框架**: gorilla/websocket
- **日志**: go.uber.org/zap + lumberjack
- **配置管理**: gopkg.in/ini.v1, Kubernetes ConfigMap/Secret
- **数据库**: MySQL (go-sql-driver/mysql), MongoDB (mongo-driver)
- **服务发现**: etcd, Kubernetes Service
- **监控**: Prometheus, Grafana
- **容器编排**: Kubernetes
- **配置中心**: etcd, Kubernetes ConfigMap
- **序列化**: Protocol Buffers (google.golang.org/protobuf)
- **本地模块**: zEngine, zUtil

## 代码风格

### 命名规范

- **包名**: 小写单词，不使用下划线或驼峰 (如 `znet`, `zlog`)
- **文件名**: 小写蛇形命名 (如 `tcp_server.go`, `player_dao.go`)
- **导出函数/类型**: 大写驼峰 (如 `TcpServer`, `NewSession`)
- **私有函数/类型**: 小写驼峰 (如 `handleConnection`, `sendPacket`)
- **接口**: 以 `er` 结尾或 `I` 前缀 (如 `Handler`, `IPlayer`)
- **常量**: 大写驼峰或全大写下划线分隔

### 代码组织

- 每个包按职责划分，单一职责原则
- 文件按功能模块划分:
  - `xxx.go` - 主要实现
  - `xxx_test.go` - 单元测试
  - `interface.go` - 接口定义 (如有)
  - `config.go` - 配置相关

### 格式规范

- 使用 `gofmt` 或 `goimports` 格式化代码
- 使用 `golangci-lint` 进行静态检查
- 缩进使用 Tab
- 行长度不超过 120 字符
- 大括号不换行
- 空行使用：
  - 函数之间空一行
  - 代码块之间空一行
  - 逻辑分组之间空一行

### 注释规范

- 包注释：写在 `package` 语句上方
- 导出函数/类型：必须添加注释，以函数名开头
- 私有函数/类型：复杂逻辑添加注释
- 行内注释：解释复杂或非直观的代码
- 注释风格：使用 `//` 而非 `/* */`

```go
// NewTcpServer 创建新的 TCP 服务器实例
// addr: 监听地址，格式为 "host:port"
func NewTcpServer(addr string) *TcpServer {
    // ...
}
```

## 错误处理

- 不忽略错误，必须处理每个错误
- 使用自定义错误类型提供更多上下文
- 错误变量以 `Err` 开头

```go
var (
    ErrPlayerNotFound = errors.New("player not found")
    ErrInvalidPacket  = errors.New("invalid packet format")
)
```

- 错误传递时添加上下文
- 避免在循环中使用 `defer`

```go
func processPacket(data []byte) error {
    packet, err := decodePacket(data)
    if err != nil {
        return fmt.Errorf("decode packet: %w", err)
    }
    
    // 处理包...
    
    return nil
}
```

## 并发规范

- 使用 `sync.Map` 或分片 Map 替代 `map + mutex`
- goroutine 必须有退出机制
- 使用 `context.Context` 控制生命周期
- channel 优先使用带缓冲区

- goroutine 管理：使用 `sync.WaitGroup` 等待 goroutine 完成
- 避免创建过多 goroutine

```go
func processTasks(tasks []Task) {
    var wg sync.WaitGroup
    for _, task := range tasks {
        wg.Add(1)
        go func(t Task) {
            defer wg.Done()
            // 处理任务...
        }(task)
    }
    wg.Wait()
}
```

- channel 使用：明确 channel 的用途（发送/接收/双向）

```go
// 带缓冲区的 channel
ch := make(chan int, 10)

// 单向 channel
func sendData(ch chan<- int) {
    // 只发送
}

func receiveData(ch <-chan int) {
    // 只接收
}
```

## 项目结构

```
project/
├── GlobalServer/          # 全局服
├── GatewayServer/         # 网关服
├── GameServer/            # 游戏服
├── MapServer/             # 地图服
├── AdminServer/           # 管理服
├── zMmoShared/            # 共享包 (pkg 功能)
├── resources/             # 资源文件
│   ├── excel_tables/      # 配置表
│   ├── maps/              # 地图文件
│   └── protocol/          # 协议文件
└── docs/                  # 文档
```

## 构建命令

- 格式化代码: `go fmt ./...`
- 静态检查: `golangci-lint run`
- 运行测试: `go test ./... -v -race`
- 测试覆盖率: `go test -cover ./...`
- 构建程序: `go build -o bin/server ./main.go`

## 测试规范

- 测试文件以 `_test.go` 结尾
- 测试函数以 `Test` 开头
- 使用 `testify` 断言库
- 表驱动测试优先

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive", 1, 2, 3},
        {"negative", -1, -2, -3},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.Equal(t, tt.expected, Add(tt.a, tt.b))
        })
    }
}
```

- 基准测试以 `Benchmark` 开头
- 示例测试以 `Example` 开头
- 目标覆盖率：核心功能 > 80%

## 依赖管理

- 使用 Go Modules 管理依赖
- 本地模块使用 `replace` 指令
- 定期运行 `go mod tidy` 清理未使用依赖
- 更新依赖: `go get -u ./...`

## 禁止事项

- 禁止使用 `panic` 处理业务错误
- 禁止在循环中使用 `defer`
- 禁止全局变量存储请求状态
- 禁止忽略 `err` 返回值
- 禁止在 goroutine 中直接操作共享资源

## 最佳实践

- 使用接口解耦，依赖注入
- 优先使用组合而非继承
- 避免过早优化
- 保持函数简短，单一职责
- 使用 `io.Reader` / `io.Writer` 接口处理数据流

## 性能优化

### 内存管理
- 避免频繁分配内存
- 使用对象池复用对象
- 合理使用切片容量

```go
// 预分配切片容量
data := make([]byte, 0, 1024)

// 使用对象池
var pool = sync.Pool{
    New: func() interface{} {
        return &Buffer{}
    },
}
```

### 网络优化
- 使用连接池
- 合理设置缓冲区大小
- 批量处理消息
- 压缩传输数据

### 数据库优化
- 使用连接池
- 合理设计索引
- 批量操作
- 缓存热点数据

### 算法优化
- 选择合适的数据结构
- 避免 O(n²) 复杂度的算法
- 优化热点代码路径

## 安全性

### 输入验证
- 验证所有用户输入
- 防止 SQL 注入
- 防止 XSS 攻击
- 防止 CSRF 攻击

### 数据加密
- 敏感数据加密存储
- 传输数据加密
- 使用安全的随机数生成

### 权限控制
- 实现基于角色的权限控制
- 验证用户权限
- 防止越权访问

### 防作弊
- 客户端行为分析
- 异常操作检测
- 速率限制
- 数据一致性验证

## 文档规范

### 项目文档
- `README.md`：项目概述、安装指南、使用说明
- `docs/` 目录：详细文档
  - `架构设计方案.md`：系统架构设计
  - `配置文件管理规范.md`：配置文件管理
  - `API文档.md`：API 接口说明

### 代码文档
- 包注释：包的功能、使用方法
- 函数注释：函数功能、参数、返回值
- 类型注释：类型用途、字段说明
- 示例代码：使用示例

### 提交规范
- 提交消息格式：`类型: 描述`
- 类型包括：`feat`（新功能）、`fix`（修复）、`docs`（文档）、`style`（代码风格）、`refactor`（重构）、`test`（测试）、`chore`（构建/依赖）
- 示例：`feat: 实现防作弊系统`

## 代码审查要点

### 功能正确性
- 逻辑是否正确
- 边界情况处理
- 错误处理是否完善
- 测试覆盖是否充分

### 代码质量
- 命名是否规范
- 注释是否充分
- 代码风格是否一致
- 复杂度是否合理

### 性能
- 是否有性能瓶颈
- 内存使用是否合理
- 并发处理是否正确
- 网络/数据库操作是否优化

### 安全性
- 输入验证是否完善
- 权限控制是否正确
- 数据加密是否到位
- 防作弊机制是否有效

## 工具链

### 开发工具
- `gofmt`/`goimports`：代码格式化
- `golangci-lint`：静态代码检查
- `go test`：单元测试
- `go vet`：代码分析

### 构建工具
- `go build`：构建项目
- `go mod`：依赖管理
- `make`：构建脚本

### 监控工具
- Prometheus：指标监控
- Grafana：可视化面板
- ELK Stack：日志管理

## 版本控制

### 分支策略
- `main`：稳定版本
- `develop`：开发分支
- `feature/*`：功能分支
- `bugfix/*`： bug 修复分支

### 版本号规范
- 语义化版本：`MAJOR.MINOR.PATCH`
- `MAJOR`：不兼容的 API 变更
- `MINOR`：向后兼容的功能添加
- `PATCH`：向后兼容的 bug 修复

## 开发约束（重要）

### 必须遵循的原则

1. **基于 zEngine 和 zUtil 开发**
   - 新代码必须优先使用 `zEngine` 提供的核心功能
   - 工具类优先使用 `zUtil` 中的实现，避免重复造轮子
   - 网络通信使用 `zEngine/zNet` 模块
   - 日志使用 `zEngine/zLog` 模块
   - 服务管理使用 `zEngine/zService` 模块

2. **参考 zGameServer 实现**
   - 新功能实现前必须查看 `zGameServer` 的对应模块
   - 遵循 `zGameServer` 的代码组织方式和设计模式
   - 数据库访问层参考 `db/dao` 和 `db/models` 的实现
   - 游戏对象系统参考 `game/object` 和 `game/player`

3. **架构设计约束**
   - 必须参考 `zMmoServer/docs/架构设计方案.md`
   - 遵循分布式服务器架构设计（GlobalServer、GatewayServer、GameServer、MapServer）
   - 服务间通信使用 TCP + Protobuf
   - ID 类型设计遵循 `zGameServer/docs/ID类型使用规范分析.md`
   - **全局统一 ID 类型使用**：必须使用 `id` 包中定义的 ID 类型
     - 会话 ID：使用 `zNet.SessionIdType`
     - 账号 ID：使用 `id.AccountIdType`
     - 玩家 ID：使用 `id.PlayerIdType`
     - 物品 ID：使用 `id.ItemIdType`
     - 技能 ID：使用 `id.SkillIdType`
     - 任务 ID：使用 `id.QuestIdType`
     - 地图 ID：使用 `id.MapIdType`
     - 公会 ID：使用 `id.GuildIdType`
     - 组队 ID：使用 `id.TeamIdType`
     - 宠物 ID：使用 `id.PetIdType`
     - 坐骑 ID：使用 `id.MountIdType`
     - 成就 ID：使用 `id.AchievementIdType`
   - 使用 Snowflake 生成运行时 ID（PlayerIdType、ObjectIdType 等）
   - 配置驱动 ID 使用 int32（MapIdType、SkillIdType 等）

4. **代码质量要求**
   - **去除冗余代码**：每次修改都要检查并删除无用代码
   - **框架合理性**：确保新代码与现有框架风格一致
   - **避免重复**：功能已存在于 zEngine/zUtil 的，禁止重复实现

5. **不确定时如何处理**
   - **必须提出疑问**：对实现方案不确定时，必须向用户确认
   - **禁止盲目编码**：不要基于假设编写代码
   - **提供选项**：给出 2-3 种可行方案供用户选择
   - **引用参考**：说明参考了哪些现有代码或文档

6. **zEngine/zUtil 功能扩展原则**
   - **先评估现有能力**：检查 zEngine 和 zUtil 是否已有类似功能或可满足需求
   - **提取通用功能**：不涉及具体游戏业务、可复用的基础功能块，应提出扩展 zEngine 或 zUtil
   - **提出修改需求**：当现有模块不满足时，向用户说明需求并建议修改方案
   - **用户决策**：由用户决定是否修改 zEngine/zUtil，以及具体修改方式
   - **避免临时方案**：不在业务代码中写临时 workaround，优先完善基础库

7. **配置文件管理规范**
   - **配置文件位置**：所有配置表文件必须放在 `resources/excel_tables/` 目录
   - **配置文件格式**：配置表使用 Excel (.xlsx) 格式，地图文件使用 JSON 格式
   - **配置加载方式**：使用 `zMmoShared/config/tables/` 中的表格管理类加载配置
   - **禁止事项**：禁止在其他目录创建配置文件，禁止使用本地 JSON 配置文件
   - **配置验证**：启动时必须验证所有必需配置表是否存在且格式正确

### 开发前检查清单

- [ ] 是否查看了 zGameServer 的对应模块实现？
- [ ] 是否查阅了架构设计方案？
- [ ] 是否检查了 zEngine/zUtil 是否已有该功能？
- [ ] 是否需要扩展 zEngine/zUtil 的通用功能？
- [ ] 是否遵循了 ID 类型使用规范？
- [ ] 是否考虑了代码复用和去冗余？
- [ ] 对不确定的地方是否已提出疑问？
- [ ] 是否遵循了配置文件管理规范？
- [ ] 配置文件是否放在了正确的目录？
- [ ] 配置文件格式是否正确？
- [ ] 是否支持从环境变量读取配置？
- [ ] 是否为Kubernetes部署准备了配置文件？
- [ ] 是否实现了服务发现和配置中心集成？
- [ ] 是否配置了健康检查和监控指标？
- [ ] 是否遵循了代码规范和最佳实践？
- [ ] 是否考虑了性能和安全性？
- [ ] 是否编写了充分的测试用例？
