# Go 项目开发规�?

## 技术栈

- **语言版本**: Go 1.25+
- **Web 框架**: gorilla/websocket
- **日志**: go.uber.org/zap + lumberjack
- **配置管理**: gopkg.in/ini.v1, Kubernetes ConfigMap/Secret
- **数据�?*: MySQL (go-sql-driver/mysql), MongoDB (mongo-driver)
- **服务发现**: etcd, Kubernetes Service
- **监控**: Prometheus, Grafana
- **容器编排**: Kubernetes
- **配置中心**: etcd, Kubernetes ConfigMap
- **序列�?*: Protocol Buffers (google.golang.org/protobuf)
- **本地模块**: zEngine, zUtil

## 代码风格

### 命名规范

- **包名**: 小写单词，不使用下划线或驼峰 (�?`znet`, `zlog`)
- **文件�?*: 小写蛇形命名 (�?`tcp_server.go`, `player_dao.go`)
- **导出函数/类型**: 大写驼峰 (�?`TcpServer`, `NewSession`)
- **私有函数/类型**: 小写驼峰 (�?`handleConnection`, `sendPacket`)
- **接口**: �?`er` 结尾�?`I` 前缀 (�?`Handler`, `IPlayer`)
- **常量**: 大写驼峰或全大写下划线分�?

### 代码组织

- 每个包按职责划分，单一职责原则
- 文件按功能模块划�?
  - `xxx.go` - 主要实现
  - `xxx_test.go` - 单元测试
  - `interface.go` - 接口定义 (如有)
  - `config.go` - 配置相关

### 格式规范

- 使用 `gofmt` �?`goimports` 格式化代�?
- 使用 `golangci-lint` 进行静态检�?
- 缩进使用 Tab
- 行长度不超过 120 字符
- 大括号不换行
- 空行使用�?
  - 函数之间空一�?
  - 代码块之间空一�?
  - 逻辑分组之间空一�?

### 注释规范

- 包注释：写在 `package` 语句上方
- 导出函数/类型：必须添加注释，以函数名开�?
- 私有函数/类型：复杂逻辑添加注释
- 行内注释：解释复杂或非直观的代码
- 注释风格：使�?`//` 而非 `/* */`

```go
// NewTcpServer 创建新的 TCP 服务器实�?
// addr: 监听地址，格式为 "host:port"
func NewTcpServer(addr string) *TcpServer {
    // ...
}
```

## 错误处理

- 不忽略错误，必须处理每个错误
- 使用自定义错误类型提供更多上下文
- 错误变量�?`Err` 开�?

```go
var (
    ErrPlayerNotFound = errors.New("player not found")
    ErrInvalidPacket  = errors.New("invalid packet format")
)
```

- 错误传递时添加上下�?
- 避免在循环中使用 `defer`

```go
func processPacket(data []byte) error {
    packet, err := decodePacket(data)
    if err != nil {
        return fmt.Errorf("decode packet: %w", err)
    }
    
    // 处理�?..
    
    return nil
}
```

## 并发规范

- 使用 `sync.Map` 或分�?Map 替代 `map + mutex`
- goroutine 必须有退出机�?
- 使用 `context.Context` 控制生命周期
- channel 优先使用带缓冲区

- goroutine 管理：使�?`sync.WaitGroup` 等待 goroutine 完成
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

- channel 使用：明�?channel 的用途（发�?接收/双向�?

```go
// 带缓冲区�?channel
ch := make(chan int, 10)

// 单向 channel
func sendData(ch chan<- int) {
    // 只发�?
}

func receiveData(ch <-chan int) {
    // 只接�?
}
```

## 项目结构

```
project/
├── GlobalServer/          # 全局�?
├── GatewayServer/         # 网关�?
├── GameServer/            # 游戏�?
├── MapServer/             # 地图�?
├── AdminServer/           # 管理�?
├── zCommon/            # 共享�?(pkg 功能)
├── resources/             # 资源文件
�?  ├── excel_tables/      # 配置�?
�?  ├── maps/              # 地图文件
�?  └── protocol/          # 协议文件
└── docs/                  # 文档
```

## 构建命令

- 格式化代码: `go fmt ./...`
- 静态检查: `golangci-lint run`
- 运行测试: `go test ./... -v -race`
- 测试覆盖率: `go test -cover ./...`
- 构建程序 (Linux/Mac): `go build -o bin/server ./main.go`
- 构建程序 (Windows): `go build -o bin/server.exe ./main.go`
- 构建程序 (所有平台): `go build -o bin/server ./main.go` (Go 会自动添加平台相关的扩展名)

## 测试规范

- 测试文件�?`_test.go` 结尾
- 测试函数�?`Test` 开�?
- 使用 `testify` 断言�?
- 表驱动测试优�?

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

- 基准测试�?`Benchmark` 开�?
- 示例测试�?`Example` 开�?
- 目标覆盖率：核心功能 > 80%

## 依赖管理

- 使用 Go Modules 管理依赖
- 本地模块使用 `replace` 指令
- 定期运行 `go mod tidy` 清理未使用依�?
- 更新依赖: `go get -u ./...`

## 禁止事项

- 禁止使用 `panic` 处理业务错误
- 禁止在循环中使用 `defer`
- 禁止全局变量存储请求状�?
- 禁止忽略 `err` 返回�?
- 禁止�?goroutine 中直接操作共享资�?

## 最佳实�?

- 使用接口解耦，依赖注入
- 优先使用组合而非继承
- 避免过早优化
- 保持函数简短，单一职责
- 使用 `io.Reader` / `io.Writer` 接口处理数据�?

## 性能优化

### 内存管理
- 避免频繁分配内存
- 使用对象池复用对�?
- 合理使用切片容量

```go
// 预分配切片容�?
data := make([]byte, 0, 1024)

// 使用对象�?
var pool = sync.Pool{
    New: func() interface{} {
        return &Buffer{}
    },
}
```

### 网络优化
- 使用连接�?
- 合理设置缓冲区大�?
- 批量处理消息
- 压缩传输数据

### 数据库优�?
- 使用连接�?
- 合理设计索引
- 批量操作
- 缓存热点数据

### 算法优化
- 选择合适的数据结构
- 避免 O(n²) 复杂度的算法
- 优化热点代码路径

## 安全�?

### 输入验证
- 验证所有用户输�?
- 防止 SQL 注入
- 防止 XSS 攻击
- 防止 CSRF 攻击

### 数据加密
- 敏感数据加密存储
- 传输数据加密
- 使用安全的随机数生成

### 权限控制
- 实现基于角色的权限控�?
- 验证用户权限
- 防止越权访问

### 防作�?
- 客户端行为分�?
- 异常操作检�?
- 速率限制
- 数据一致性验�?

## 文档规范

### 项目文档
- `README.md`：项目概述、安装指南、使用说�?
- `docs/` 目录：详细文�?
  - `架构设计方案.md`：系统架构设�?
  - `配置文件管理规范.md`：配置文件管�?
  - `API文档.md`：API 接口说明

### 代码文档
- 包注释：包的功能、使用方�?
- 函数注释：函数功能、参数、返回�?
- 类型注释：类型用途、字段说�?
- 示例代码：使用示�?

### 提交规范
- 提交消息格式：`类型: 描述`
- 类型包括：`feat`（新功能）、`fix`（修复）、`docs`（文档）、`style`（代码风格）、`refactor`（重构）、`test`（测试）、`chore`（构�?依赖�?
- 示例：`feat: 实现防作弊系统`

## 代码审查要点

### 功能正确�?
- 逻辑是否正确
- 边界情况处理
- 错误处理是否完善
- 测试覆盖是否充分

### 代码质量
- 命名是否规范
- 注释是否充分
- 代码风格是否一�?
- 复杂度是否合�?

### 性能
- 是否有性能瓶颈
- 内存使用是否合理
- 并发处理是否正确
- 网络/数据库操作是否优�?

### 安全�?
- 输入验证是否完善
- 权限控制是否正确
- 数据加密是否到位
- 防作弊机制是否有�?

## 工具�?

### 开发工�?
- `gofmt`/`goimports`：代码格式化
- `golangci-lint`：静态代码检�?
- `go test`：单元测�?
- `go vet`：代码分�?

### 构建工具
- `go build`：构建项�?
- `go mod`：依赖管�?
- `make`：构建脚�?

### 监控工具
- Prometheus：指标监�?
- Grafana：可视化面板
- ELK Stack：日志管�?

## 版本控制

### 分支策略
- `main`：稳定版�?
- `develop`：开发分�?
- `feature/*`：功能分�?
- `bugfix/*`�?bug 修复分支

### 版本号规�?
- 语义化版本：`MAJOR.MINOR.PATCH`
- `MAJOR`：不兼容�?API 变更
- `MINOR`：向后兼容的功能添加
- `PATCH`：向后兼容的 bug 修复

## 开发约束（重要�?

### 必须遵循的原�?

1. **基于 zEngine �?zUtil 开�?*
   - 新代码必须优先使�?`zEngine` 提供的核心功�?
   - 工具类优先使�?`zUtil` 中的实现，避免重复造轮�?
   - 网络通信使用 `zEngine/zNet` 模块
   - 日志使用 `zEngine/zLog` 模块
   - 服务管理使用 `zEngine/zService` 模块

2. **参�?zGameServer 实现**
   - 新功能实现前必须查看 `zGameServer` 的对应模�?
   - 遵循 `zGameServer` 的代码组织方式和设计模式
   - 数据库访问层参�?`db/dao` �?`db/models` 的实�?
   - 游戏对象系统参�?`game/object` �?`game/player`

3. **架构设计约束**
   - 必须参�?`zMmoServer/docs/架构设计方案.md`
   - 遵循分布式服务器架构设计（GlobalServer、GatewayServer、GameServer、MapServer�?
   - 服务间通信使用 TCP + Protobuf
   - ID 类型设计遵循 `zGameServer/docs/ID类型使用规范分析.md`
   - **全局统一 ID 类型使用**：必须使�?`id` 包中定义�?ID 类型
     - 会话 ID：使�?`zNet.SessionIdType`
     - 账号 ID：使�?`id.AccountIdType`
     - 玩家 ID：使�?`id.PlayerIdType`
     - 物品 ID：使�?`id.ItemIdType`
     - 技�?ID：使�?`id.SkillIdType`
     - 任务 ID：使�?`id.QuestIdType`
     - 地图 ID：使�?`id.MapIdType`
     - 公会 ID：使�?`id.GuildIdType`
     - 组队 ID：使�?`id.TeamIdType`
     - 宠物 ID：使�?`id.PetIdType`
     - 坐骑 ID：使�?`id.MountIdType`
     - 成就 ID：使�?`id.AchievementIdType`
   - 使用 Snowflake 生成运行�?ID（PlayerIdType、ObjectIdType 等）
   - 配置驱动 ID 使用 int32（MapIdType、SkillIdType 等）

4. **代码质量要求**
   - **去除冗余代码**：每次修改都要检查并删除无用代码
   - **框架合理�?*：确保新代码与现有框架风格一�?
   - **避免重复**：功能已存在�?zEngine/zUtil 的，禁止重复实现

5. **不确定时如何处理**
   - **必须提出疑问**：对实现方案不确定时，必须向用户确认
   - **禁止盲目编码**：不要基于假设编写代�?
   - **提供选项**：给�?2-3 种可行方案供用户选择
   - **引用参�?*：说明参考了哪些现有代码或文�?

6. **zEngine/zUtil 功能扩展原则**
   - **先评估现有能�?*：检�?zEngine �?zUtil 是否已有类似功能或可满足需�?
   - **提取通用功能**：不涉及具体游戏业务、可复用的基础功能块，应提出扩�?zEngine �?zUtil
   - **提出修改需�?*：当现有模块不满足时，向用户说明需求并建议修改方案
   - **用户决策**：由用户决定是否修改 zEngine/zUtil，以及具体修改方�?
   - **避免临时方案**：不在业务代码中写临�?workaround，优先完善基础�?

7. **配置文件管理规范**
   - **配置文件位置**：所有配置表文件必须放在 `resources/excel_tables/` 目录
   - **配置文件格式**：配置表使用 Excel (.xlsx) 格式，地图文件使�?JSON 格式
   - **配置加载方式**：使�?`zCommon/config/tables/` 中的表格管理类加载配�?
   - **禁止事项**：禁止在其他目录创建配置文件，禁止使用本�?JSON 配置文件
   - **配置验证**：启动时必须验证所有必需配置表是否存在且格式正确

## 协议规范

### 协议文件组织

- **协议源文件位�?*：所�?`.proto` 文件必须放在 `resources/protocol/` 目录
- **协议生成文件位置**：所有生成的 `.pb.go` 文件必须放在 `zCommon/protocol/` 目录
- **禁止子目�?*：协议生成文件必须直接放�?`protocol` 目录下，禁止创建子目录（�?`interop`、`net` 等）

### 协议文件分类

```
resources/protocol/
├── common.proto       # 通用定义（错误码、常量等�?
├── auth.proto         # 认证协议（登录、注册、心跳等�?
├── player.proto       # 玩家协议（角色、背包、技能等�?
├── game.proto         # 游戏协议（战斗、地图、任务等�?
└── internal.proto     # 服务间协议（服务注册、路由、心跳等�?

zCommon/protocol/
├── common.pb.go       # �?common.proto 生成
├── auth.pb.go         # �?auth.proto 生成
├── player.pb.go       # �?player.proto 生成
├── game.pb.go         # �?game.proto 生成
└── internal.pb.go     # �?internal.proto 生成
```

### 协议生成规则

- **go_package 选项**：所�?`.proto` 文件必须设置 `go_package` 选项
  ```protobuf
  option go_package = "./;protocol";
  ```
  �?
  ```protobuf
  option go_package = "./;interop";
  ```

- **生成命令**：使�?`protoc` 生成协议文件
  ```bash
  protoc --go_out="..\..\zCommon\protocol" --go_opt=paths=source_relative xxx.proto
  ```

- **构建脚本**：使�?`resources/protocol/build_proto.bat` 批量生成所有协议文�?

### 协议使用规范

- **客户端协�?*：使�?`auth.proto`、`player.proto`、`game.proto` 中定义的消息
- **服务间协�?*：使�?`internal.proto` 中定义的消息
- **心跳协议**：使�?`auth.proto` 中定义的 `ServerHeartbeatRequest` �?`ServerHeartbeatResponse`
- **禁止混用**：客户端协议和服务间协议不能混用

### 协议更新流程

1. 修改 `.proto` 文件
2. 运行 `build_proto.bat` 生成新的 `.pb.go` 文件
3. 更新相关的业务代�?
4. 运行测试确保协议变更正确
5. 提交代码时包�?`.proto` �?`.pb.go` 文件

### 协议命名规范

- **消息类型**：使�?PascalCase 命名，如 `ServerHeartbeatRequest`
- **字段名称**：使�?snake_case 命名，如 `server_id`、`online_count`
- **枚举类型**：使�?PascalCase 命名，如 `ServiceType`
- **枚举�?*：使�?UPPER_SNAKE_CASE 命名，如 `SERVICE_TYPE_GLOBAL`

### 协议版本管理

- **向后兼容**：协议变更必须保持向后兼�?
- **字段编号**：已使用的字段编号不能修改或删除
- **可选字�?*：新增字段必须使�?`optional` 关键�?
- **废弃字段**：废弃的字段保留编号，添�?`deprecated` 注释

### 协议验证

- **编译检�?*：每次修�?`.proto` 文件后必须重新生�?`.pb.go` 文件
- **导入检�?*：确保所有导入的协议包路径正�?
- **类型检�?*：使�?Go 编译器检查协议类型是否匹�?
- **运行时检�?*：在运行时验证协议数据的完整性和正确�?

### 开发前检查清�?

- [ ] 是否查看�?zGameServer 的对应模块实现？
- [ ] 是否查阅了架构设计方案？
- [ ] 是否检查了 zEngine/zUtil 是否已有该功能？
- [ ] 是否需要扩�?zEngine/zUtil 的通用功能�?
- [ ] 是否遵循�?ID 类型使用规范�?
- [ ] 是否考虑了代码复用和去冗余？
- [ ] 对不确定的地方是否已提出疑问�?
- [ ] 是否遵循了配置文件管理规范？
- [ ] 配置文件是否放在了正确的目录�?
- [ ] 配置文件格式是否正确�?
- [ ] 是否支持从环境变量读取配置？
- [ ] 是否为Kubernetes部署准备了配置文件？
- [ ] 是否实现了服务发现和配置中心集成�?
- [ ] 是否配置了健康检查和监控指标�?
- [ ] 是否遵循了代码规范和最佳实践？
- [ ] 是否考虑了性能和安全性？
- [ ] 是否编写了充分的测试用例�?

