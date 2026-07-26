# Realm 架构 — 落地详细设计（建议 ②③④⑤）

> 本文是 [realm架构与跨服设计.md](realm架构与跨服设计.md) §4 建议 ②③④⑤ 的**可执行深化**：把方向落成机制、数据结构、协议、
> 流程与分步任务，并锚定到现有代码。**本文只做设计，不含实现**；每项动手时应再出实现方案并配测试/E2E 背书。
>
> 覆盖：② 热点图动态分线（Layering）｜③ 实例生命周期｜④ 权威分层 + 事件回流｜⑤ 扩展=加服（开新服流程）
>
> 关键前提（现状核准）：
> - `MapManager.maps` 是 `TypedMap[MapIdType, *Map]`，**一个 mapID 一个 `*Map` 实例**；`CreateMap/GetMap/Cleanup` 均按 mapID。
> - 每个 `*Map` 是**独立单写者 goroutine**（`map_actor.go` 的 `Map.Do` 同步投递）+ 独立 AOI + 独立 100ms tick。
> - **实例地图基础设施已存在**：`CreateDungeonMap(dungeonID, players)→(*Map,*DungeonInstance)`、`dungeonLifecycleMgr`、
>   `DestroyDungeonMap(instanceID)`、派生 `dungeonMapID`、`instance.InstanceID`。③④ 的很多东西是"泛化它"。
> - 回流通道已成形：crossserver `500-505` AOI/状态广播、`506 ItemGrant`、`507 ExpGrant`。

---

## ② 热点图动态分线（Layering）

### 目标
一张开放世界图爆满时，**开多份平行副本（层/line）**，玩家分摊进去、跨层互不可见；治「每张 map 单 goroutine、一核封顶（~千级 CCU）」的天花板。**纯单服内伸缩，不涉及跨服。**

### 核心洞见
**"层"就是"开放世界版的实例"**——和 dungeon 实例同构：一个派生的 `*Map` 实例 + 独立 AOI + 独立 goroutine。差别只在：
- dungeon 实例是**队伍绑定、进即创、空即毁**；
- 层是**同一逻辑地图的常驻并行实例**，按人数动态增减，进图时**按负载分配**。

所以 layering ≈ 复用实例基础设施 + 加一个「层分配器」。

### 数据模型
- 逻辑地图 ID `logicalMapID`（玩家/策划眼里的"新手村"）与物理层实例 ID `layerMapID`（派生，如 `logicalMapID*1000 + layerIndex`）分离——沿用 dungeon 的"派生 mapID"手法。
- 新增 `LayerSet`（每个可分线的 `logicalMapID` 一个）：
  ```
  LayerSet {
    logicalMapID   MapIdType
    layers         []*Map          // 当前存活的层
    softCap        int             // 单层软上限（如 200），到达触发新层
    hardCap        int             // 单层硬上限（拒绝进入）
    minLayers      int             // 常驻最少层数（通常 1）
    allocPolicy    最少人数优先 / 亲和(队伍/好友同层)
  }
  ```
- 哪些图可分线由**地图配置**决定（`mapconfig` 加 `layerable bool` + `softCap/hardCap`）；副本/战场不分线（它们本就是隔离实例）。

### 分配流程（进图时）
1. 玩家请求进 `logicalMapID`。
2. 层分配器在该 `LayerSet` 里选层：**亲和优先**（同队/组队目标所在层，且未满 hardCap）→ 否则**最少人数层**（<softCap）→ 都满则**开新层**（`CreateMap(layerMapID=派生)`，复用实例创建路径）。
3. 玩家进入选中的 `*Map`（该层的 AOI/tick 独立，天然隔离）。
4. 空层回收：某层人数为 0 且层数 > `minLayers`，延迟 N 秒无人再进则 `Cleanup()` 销毁（复用实例清理）。

### 与 GameServer 路由的衔接
- GameServer 侧 `MapServerManager` 现在按 `mapID→serverID` 路由。层实例的 `layerMapID` 与 `logicalMapID` 属**同一 MapServer**（同一台机负责该图及其所有层），所以：
  - **推荐**：层分配器放 **MapServer 侧**（它掌握各层实时人数）。GameServer 按 `logicalMapID` 路由到该 MapServer，MapServer 内部分配层并把最终 `layerMapID` 回传给 GameServer（玩家的当前 map 归属记 `layerMapID`）。
  - AOI/广播本就 per-`*Map`，无需改——分到不同层的玩家自然互不可见。

### 边界与坑
- **同队/好友要能同层**：分配策略必须支持"亲和"，否则组队进热图被拆散。至少支持"跟随队长层"。
- **跨层可见性**：世界聊天（本服扇出）**跨层可见**是对的（世界频道是"本服世界"）；但 AOI/喊话/附近只在本层。要想清楚每种广播的作用域（全服 / 本层 / AOI）。
- **层间迁移**：一般不做（换层=重新进图）；若要"手动切线"，走一次 leave+enter（等价建议①的地图交接，先不做）。
- **单写者仍是每层一核**：分线是把"一张热图"摊成"多层各占一核"，**没有打破单 map 单核**，而是绕过它——这正是 WoW layering 的本质。

### 分步任务
1. `mapconfig` 加 `layerable/softCap/hardCap`；派生 `layerMapID` 规则。
2. `LayerSet` + 层分配器（MapServer 侧），进图路由改为"选层"。
3. 空层延迟回收（复用 `Cleanup`）。
4. 亲和分配（跟随队长层）。
5. 验证：确定性单测（模拟 N 人进图→断言层数按 cap 增长、每层 ≤ hardCap、空层回收）；真机多客户端进同一热图→分到不同层、AOI 互不可见、组队同层。

---

## ③ 实例生命周期（做干净、泛化）

### 目标
把副本/战场/（层）统一成一套**干净的实例生命周期**：按需创建、归属绑定、空了销毁、无 goroutine/内存泄漏。作为 ②（层）与 §5 跨服临时实例的公共基础设施。

### 现状
已有 dungeon 专用：`CreateDungeonMap`/`dungeonLifecycleMgr`/`DestroyDungeonMap`/`DungeonInstance{InstanceID}`；本项目已修过"spawnLoop 用 stopCh 干净收尾、Cleanup 先 StopActor 再停 spawn"。**基础对，但是 dungeon 专用**，需泛化。

### 设计：统一 `MapInstance` 抽象
把"一次性/并行的地图实例"抽象成统一类型，dungeon/battleground/layer 都是它的 kind：
```
MapInstance {
  InstanceID   InstanceIdType   // 全实例唯一（现有 dungeon 已有）
  LogicalMapID MapIdType        // 逻辑地图
  Kind         Dungeon | Battleground | Layer | CrossServer
  Binding      队伍 / 无(开放层) / 跨服会话
  Map          *Map             // 承载的地图实例
  State        Creating→Running→Draining→Destroyed
  CreatedAt / EmptySince        // 空置计时，驱动回收
}
```
- 统一生命周期管理器 `InstanceManager`（把 `dungeonLifecycleMgr` 提升为通用）：`Create(kind, logicalMapID, binding)→InstanceID`、`Destroy(instanceID)`、`ReapEmpty()`（周期扫 `EmptySince` 超阈值且非常驻的实例，`Cleanup`）。
- **回收统一走一条路**：`Destroy` = StopActor（停单写者 goroutine）→ 停 spawnLoop（stopCh）→ 从 `maps`/`InstanceManager` 摘除 → 释放。dungeon 已验证的顺序作为模板。

### 关键不变式（防泄漏）
- **销毁前必须先 StopActor**：否则回收时还有命令在 `Map.Do` 队列里跑，UAF/泄漏（本项目已有此教训）。
- **spawnLoop/tick 用 stopCh 干净退出**，不留孤儿 goroutine。
- **回收幂等**：`Destroy` 对已销毁实例是 no-op。
- **归属清理**：实例销毁时，其内玩家要被踢回主城/安全点（不能悬空指向已销毁 map）。

### 与跨服的接口
§5.1 跨服临时实例 = `Kind=CrossServer` 的 `MapInstance`，`Binding=跨服会话`；创建/销毁复用同一 `InstanceManager`，只是玩家来自多个服、结果走事件回流（见 §5 与 ④）。

### 分步任务
1. 抽 `MapInstance` + 把 `dungeonLifecycleMgr` 泛化为 `InstanceManager`（dungeon 作为一个 Kind，保持现有行为）。
2. `ReapEmpty()` 周期回收（层/跨服实例复用）。
3. 玩家在实例销毁时的安全撤离。
4. 验证：建 N 实例→跑→清空→`ReapEmpty`→断言 goroutine/内存回基线（本项目已有 `map_cleanup_test` 模板）；崩溃/异常路径不泄漏。

---

## ④ 权威分层 + 事件回流（把"两份表示"讲成原则）

### 目标
不合并两份玩家表示（大重构），而是**定成原则让分工正当化**，并**补齐战斗→持久的回流通道**，消除"战斗改了持久属性却丢失"这类结构性问题（F-2 是第一块）。

### 权威边界（定死）
| 权威 | 持有者 | 内容 | 生命周期 |
|------|--------|------|----------|
| **持久业务权威** | GameServer `player.Player` actor | 背包 / 技能 / 货币 / 钻石 / 经验 / 等级 / 任务 / 仓库 / 邮件 | 跨登录持久，落库 |
| **瞬时模拟权威** | MapServer `object.Player` | 血量 / 位置 / 朝向 / 战斗内 buff / 仇恨 | 随进出图/战斗存续，不持久 |

**红线**：MapServer **绝不直接写库**；凡战斗产生的**持久变更**，一律经**单向 grant 事件**回流到 GameServer actor 落库。

### 统一回流通道（泛化 506/507）
现有 `506 ItemGrant`（拾取→背包）、`507 ExpGrant`（战斗经验→actor）已是这个形态。把它**收敛成一族**，避免每种属性各开一个 msgId：
- 设计 `AttrGrantNotify { playerID; changes[] }`，`changes` 是 `{kind, id, delta}` 列表：
  - `kind=Exp`（delta=经验）、`kind=Currency`（id=币种,delta）、`kind=Item`（id=itemID,delta=数量,可为负=消耗）、`kind=SkillExp` 等。
- 保留 506/507 已验证不动（兼容），**新增战斗奖励统一走 `AttrGrant`**（把 dormant 的 `HandleBattleReward{Exp,Gold,Items}` 接到这条通道上，一次结算多种）。
- **幂等**：每条 grant 带唯一 `requestID`，GameServer 侧 `inbox.TryAccept(requestID)` 去重（对齐 506/507），防重复投递把加成算两次——**加性变更（经验/货币/物品数量）必须幂等**。

### 回流的可靠性分级
- **在线即时**：玩家在线 → grant 路由到其 actor 即时 apply（现状）。
- **崩溃安全（可选升级）**：把 grant 投进已有**崩溃安全 Outbox/Inbox**（`zCommon/consistency`），确保 MapServer→GameServer 在途结算**进程重启不丢**（战斗刚结算完 GameServer 崩溃的窗口）。这是把 F-5/F-6 那类"崩溃丢增益"在跨服回流路径上也堵上。

### 反向（GameServer→MapServer）：进场快照
玩家进图时，GameServer 把**模拟所需的持久属性快照**（等级/攻防/技能等派生战斗数值）下发给 MapServer 初始化 `object.Player`。**只读快照、不是第二权威**——MapServer 战斗中改的是瞬时态，持久变更走回流。这条现在是隐式的（进图初始化），设计上要**明确成"快照下发"**，避免又滋生"第三份权威"。

### 边界与坑
- **只回流"持久"变更**：血量/位置这类瞬时态**不回流**（回流了反而制造双写）。清单要一条条定死哪些持久、哪些瞬时。
- **顺序/一致性**：同一玩家的 grant 在 actor 单写者内串行 apply，天然有序；跨玩家无序无妨（各自独立）。
- **负 delta 的下限**：消耗类（物品-N/货币-N）要在 actor 侧校验余量（对齐现有 `ReduceGold` CAS），不足则拒绝并回执，别扣成负数。

### 分步任务
1. 写死"持久/瞬时属性清单"（文档 + 代码注释锚点）。
2. `AttrGrantNotify` 统一通道 + 幂等；把 `HandleBattleReward` 接上（战斗掉落/经验/货币一次结算）。
3. 明确"进场快照下发"为唯一反向路径。
4.（可选）grant 走 Outbox/Inbox 做崩溃安全。
5. 验证：真机战斗击杀→掉落+经验+货币一次回流、DB 三者一致；重复投递幂等（数量不翻倍）；消耗不足被拒。

---

## ⑤ 扩展 = 加服（开新 realm 的完整流程）

### 目标
把"承载翻倍就加一个 realm"落成**可操作的开服流程**：一个新 realm = 一套 {GameServer + 它的 MapServer(s) + 独立 GameDB + 服务器列表条目}，与现有服**完全隔离**。

### 现状
- GlobalServer `gameserverlist`：静态服（MySQL 表）+ 动态状态（etcd）合并成服务器列表；`ServerID` 经 `id.ParseServerIDInt/ServerIDString` 解析（`GroupID(4)+ServerIndex(2)`）。
- 每服独立库 `GameDB_区服ID`（如 `GameDB_000101`）；GlobalDB 独立存账号/服务器列表。
- 服务发现按 GroupID 分组（一个 GameServer 只见本组 MapServer）。

### 开新服流程（Runbook 化）
以开 "1 区 2 服"（ServerID=000102）为例：
1. **分配 ServerID**：`GroupID=0001` + `ServerIndex=02` → `000102`（同组共享 etcd 发现域；跨组则另起 GroupID）。
2. **建库**：`CREATE DATABASE GameDB_000102` + 建表（复用现有建表脚本/`ensureAuxSchema` 那套自动建表）。**空库=全新世界**，无需迁移。
3. **配置**：GameServer `config.ini` 设 `ServerID=102`/`GroupID=1`/`DBName=GameDB_000102`；MapServer/Gateway 对应 `ServerID` 与监听端口（走已加的**端口占用探测**避免撞端口）。
4. **登记服务器列表**：GlobalDB 静态服表插一行（ServerID=102, Name, Region, 状态=新服/推荐）；GlobalServer 会把它并进 `/api/v1/server/list`。
5. **起进程**：Global（共用）→ 该服 Gateway → 该服 MapServer(s) → 该服 GameServer；各自 etcd 注册，GameServer 发现本组 MapServer。
6. **上线**：客户端拉服务器列表即见"1 区 2 服"，选入即进全新世界。

### 隔离性保证（realm 的核心价值）
- **数据隔离**：各服独立库，互不可见；一个服的 DB 故障不影响别服。
- **进程隔离**：各服 GameServer/MapServer 独立进程；一个服崩不影响别服（Global 共用，需保证 Global 高可用——它只管账号/列表，无游戏态，轻）。
- **发现隔离**：GroupID 分组，避免跨服串图/串路由。

### 容量与运营
- **单服容量估算**：单 GameServer(协调,轻) + N×MapServer(战斗,重)；瓶颈在热图单核 → 由 ② layering 缓解。给出"单服建议承载 = Σ 各图层数×softCap"的估算，超了就**开新服**或**加层**。
- **自动化方向（按需）**：把上述 Runbook 脚本化（建库+配置+起进程+登记一条命令）；再进一步可做"自动开服"（监控在线到阈值→拉起下一个 ServerID）。**教学骨架先脚本化即可，自动开服是运营级、按需。**
- **ServerID 空间**：`GroupID(4)+ServerIndex(2)` = 9999 组 × 99 服，足够；注意 `ServerID` **不入业务主键**（用 Snowflake），为将来"互联服合并"不撞 ID 留余地（见方向文档 §5.2）。

### 边界与坑
- **Global 是单点共用**：所有服共用 Global 做账号/列表。Global 要么无状态多副本、要么保证快速重启（它无游戏态，重启代价低）。**别把游戏逻辑塞进 Global**。
- **跨组不互通**：跨组玩家默认完全隔离；跨服玩法（§5.1）需显式设计跨组通道，不是"加服"自动带来的。
- **合服 ≠ 加服**：加服是开新世界；合并低人口服是**互联服**（方向文档 §5.2），走不同机制（逻辑并服、不迁库），本文不展开。

### 分步任务
1. 把开新服流程脚本化（建库+建表+配置模板+起进程+登记列表）。
2. 服务器列表管理：新服/推荐/爆满/维护 状态字段 + 客户端展示。
3. 单服容量估算文档（结合 ② 的 softCap/层数）。
4.（可选/远期）自动开服触发器。
5. 验证：脚本一键开出"1 区 2 服"→客户端列表可见→进入是全新空世界→与 1 服数据互不可见（直查两库隔离）。

---

## 依赖与推荐顺序

```
③ 实例生命周期(泛化 InstanceManager)  ──┐
                                        ├─→ ② Layering(层=开放世界实例，依赖③)
④ 权威分层+事件回流(泛化 grant)         │     └─→ §5.1 跨服临时实例(Kind=CrossServer，依赖③④)
                                        │
⑤ 开新服流程(独立，随时可做)  ──────────┘
```

- **③ 是 ② 和跨服实例的公共地基**，建议先做（泛化实例生命周期）。
- **② 依赖 ③**（层复用实例设施）。
- **④ 独立**，且 F-2 已起头，可并行推进（泛化 grant + 幂等 + 崩溃安全可选）。
- **⑤ 独立**，纯运营流程，随时可脚本化。

> 每项都以"能跑通的最小闭环 + 测试/真机 E2E"为完成标准，不追求生产级完整度——教学骨架的价值是把 realm 架构讲清楚、且每步都有真实背书。
