# Realm 架构 — 落地详细设计（建议 ①②③④⑤）

> 本文是 [realm架构与跨服设计.md](realm架构与跨服设计.md) §4 realm 建议的**可执行深化**：把方向落成机制、数据结构、协议、
> 流程与分步任务，并锚定到现有代码。原为 ②③④⑤ 的设计，现已随各项落地补入 ① 无缝交接章节与**实现状态总览**（见下）；
> 各章末的"分步任务/验证"标注了真实落地情况。细节仍以代码 + 测试/E2E 为准。
>
> 覆盖：① 无缝地图交接｜② 热点图动态分线（Layering）｜③ 实例生命周期｜④ 权威分层 + 事件回流｜⑤ 扩展=加服（开新服流程）
>
> 关键前提（现状核准）：
> - `MapManager.maps` 是 `TypedMap[MapIdType, *Map]`，**一个 mapID 一个 `*Map` 实例**；`CreateMap/GetMap/Cleanup` 均按 mapID。
> - 每个 `*Map` 是**独立单写者 goroutine**（`map_actor.go` 的 `Map.Do` 同步投递）+ 独立 AOI + 独立 100ms tick。
> - **实例地图基础设施已存在**：`CreateDungeonMap(dungeonID, players)→(*Map,*DungeonInstance)`、`dungeonLifecycleMgr`、
>   `DestroyDungeonMap(instanceID)`、派生 `dungeonMapID`、`instance.InstanceID`。③④ 的很多东西是"泛化它"。
> - 回流通道已成形：crossserver `500-505` AOI/状态广播、`506 ItemGrant`、`507 ExpGrant`。

---

## 实现状态总览

realm 各建议的落地情况（代码位置 + 测试/E2E 背书）。教学骨架以"能跑通的最小闭环 + 真实背书"为完成标准。

| 建议 | 机制 | 主要代码 | 测试/背书 | 状态 |
|------|------|----------|-----------|------|
| ① | 无缝地图交接（仅注册的无缝邻居图对才走，否则回落普通 enter/leave） | `maps/seamless.go` | `seamless_test`（含并发交接守护） | ✅ 服务端机制已落地 |
| ② | 热点图动态分线（softCap/hardCap 分摊 + resolveMap 透明路由 + 在途预留额度） | `maps/layer_manager.go`、`maps/map_manager.go` | `layer_manager_test`/`layer_enter_test`（含并发不击穿 cap） | ✅ 已落地 |
| ②-b | 客户端视野单一权威（删 GameServer 冗余自建 AOI，改由 MapServer 分层 AOI 经 HandleAOINotify 推） | `GameServer/game/maps/*`、`.../player_aoi_handler.go` | 真机双客户端（同图见 / 分层不见） | ✅ 已修并真机验收 |
| ③ | 实例生命周期统一（InstanceKind + CreateInstance/DestroyInstance/ReapEmpty，副本/分线/战场/跨服同构） | `maps/instance_manager.go` | `crossserver_instance_test`、`map_cleanup_test` | ✅ 已落地 |
| ④ | 权威分层 + 事件回流（AttrGrant 508 幂等回流，MapServer 不落库） | `zCommon/crossserver`、GameServer grant 处理 | 真机战斗击杀 → 家服落库 | ✅ 已落地 |
| ⑤ | 扩展=加服（newrealm 建隔离库 + runbook） | `GameServer/cmd/newrealm` | 真机建 GameDB_000102 两库隔离 | ✅ 已落地 |
| §5.1 | 跨服临时实例（Kind=CrossServer，依赖 ③④） | `maps/map_manager.go` CreateCrossServerMap | `crossserver_instance_test` | ✅ 实例层已备（物理跨服路由待匹配系统） |

> 并发正确性：① 无缝交接的字段读写经 `Map.Do` 收敛到地图 actor（MAP-2 边界）；② 分线用"在途预留额度"消除
> AllocateLayer 的 TOCTOU 与空层回收竞态。`-race` 守护由 CI/t1 race runner 承担（本机 Windows 无 gcc 跑不了 -race）。

---

## ① 无缝地图交接（realm 建议①）

> 设计方向与完整机制说明见 [realm架构与跨服设计.md](realm架构与跨服设计.md) §4 建议①；此处记落地要点与代码锚点。

### 目标
同一块连续大陆切成相邻分区；玩家走过区界时把他**透明交接**给拥有下一分区的 `*Map`，实时态（位置/血量/等级）随身
带过去、客户端不断线、无 loading——"单服却世界很大"的关键。**纯单服内机制**（无缝邻居通常同在一台 MapServer）。

### ★关键约束（用户定）
**只有设计成无缝的图对才走无缝流程。** 若目标不是源的无缝邻居（跨大陆/进副本/进城），就**不走无缝交接、回落普通
enter/leave**（有 loading）。无缝性是地图（对）的属性，靠 `RegisterSeamlessLink` 登记；未登记链路的跨图一律走普通流程。

### 落地（`maps/seamless.go`）
- `RegisterSeamlessLink(a,b)`（双向）登记相邻无缝分区；`IsSeamlessNeighbor(from,to)` 判定。
- `HandleSeamlessHandoff(playerID, from, to, x,y,z) (handed, err)`：
  - to 非 from 的无缝邻居 → 返回 `(false, nil)`：调用方改走普通 enter/leave（"非无缝不走无缝流程"落点）。
  - 是无缝邻居 → MapServer 内部交接：**在源图 actor 上读实时态快照 → 先加入目标图（成功后再摘源图=回滚安全）
    → 在目标图 actor 上恢复实时态 → 更新 `playerMap`（②-b 实际所在图记录）**。不重登、位置连续。
- **MAP-2 边界**：对 `*object.Player` 字段的读/写都经 `Map.Do` 收敛到各自地图的单写者 goroutine，绝不在网络
  goroutine 上裸读写（否则与目标图 tick 并发改同一对象=数据竞争）。

### 验证
`seamless_test`：邻居交接带状态迁移（HP=37/Lv5/新坐标）/ 非邻居 `handed=false` 回落 / 链路双向 /
`TestSeamless_ConcurrentHandoff_Race`（并发交接 + tick 并发的竞争守护，-race 由 CI/t1 跑）。

### 剩余（非本机制）
- 移动到区界的**边界检测**触发（现由直接 API 驱动，接 move 边检即自动）。
- 客户端**平滑过场**（无 loading 表现）——客户端侧。
- 跨机无缝分区需跨服迁移（走 §5 的 MigrationManager 雏形）。

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

  > ⚠ **实现补正（②-b，已修）**：上面"AOI 本就 per-Map 无需改"只对 **MapServer** 成立。实际 **GameServer 曾另藏一套自建客户端 AOI**（`game/maps/map.go`），按 **`logicalMapID`（如 1001）** 键值、**不感知分线**——它才是驱动客户端 `MSG_MAP_ENTER_VIEW` 的源头，导致**不同层玩家互相串视野**（战斗走 MapServer 按层隔离故不漏，呈"能看见打不到"的割裂）。
  > **已按单一权威修复**：删除 GameServer 自建客户端 AOI，客户端视野统一由 **MapServer 分层 AOI** 经 `crossserver 500-505 → map_service.HandleAOINotify` 推送（该路径本就覆盖 enter/leave/move/attr/death/buff 全 6 类）。GameServer 侧仅留网格记账、无 listener 故不产生客户端事件。真机双客户端验：同图 A 见 B / 分层 A 不见 B（0 视野行）/ 且消除了原本每事件重复投递。提交 `aa97ebf`。

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
5. 验证：确定性单测（模拟 N 人进图→断言层数按 cap 增长、每层 ≤ hardCap、空层回收）；真机多客户端进同一热图→分到不同层、AOI 互不可见、组队同层。✅ 已真机验收：双客户端同图 A 见 B、分层 A 不见 B（客户端 `[AOI]` 视野观测，见 ②-b 实现补正）。

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

### 与跨服的接口（§5.1 跨服临时实例）
跨服临时实例 = `Kind=CrossServer` 的实例：玩家来自多个服、结果走事件回流。**已落地**：
- `MapManager.CreateCrossServerMap` 改为经 ③ `CreateInstance(InstanceKindCrossServer)` 建图 → 于是跨服图是
  **可追踪、空置自动回收**的实例（打完人走即由 `ReapEmpty` 销毁）。测试 `crossserver_instance_test`
  （跨服实例=CrossServer 种类/派生 mapID/有人不回收/空置回收）绿。
- **结果回流复用 ④**：跨服玩家的战斗/结算持久变更由 `AttrGrant`(508) 按玩家归属
  （net 层 `PlayerGameServerManager`，进图时记 origin）**回流到各自家服**落库——跨服实例只产事件、不碰持久资产。
  ④ 的真机 E2E 已证单玩家 reward→其家服；多归属服只是对每个玩家各走一次同一路径。

**剩余缺口（非本层）**：把不同服的客户端**物理路由**进同一台 MapServer 的这个实例（server B 的玩家进 server A 的
实例），属**跨服匹配 + 网关跨服路由**的职责，不是实例层的事。实例层（生命周期 + 结果回流）已备好，等匹配把人送进来即可。

进度（2026-07-28）：**跨 realm 物理路由已全线打通并过多进程 E2E**。

- ✅ **传输层**：`CrossTransport`（`crossserver/transport.go`）补上 msgID 上线 + 响应/错误回投 +
  RequestID 关联；`MigrationManager.ExecuteMigration` 的 2PC 控制面首次跑通（双端集成测试 + t1 `-race`）。
- ✅ **① 跨 realm 发现**：`MapServerManager.doCrossRealmDiscovery` 用 `Discover("map", "")` 拿全量，
  结果存**单独一张表** `crossRealmServers`——绝不并入 `mapToServer`，普通进图路由仍只看本 realm（铁律5）。
- ✅ **② 分配服务**：`GlobalServer/crossmatch` —— 全区唯一决策方，按 `activity_id` **粘性**选一台
  MapServer（人少优先、平票按 serverID 升序、承载服消失才改选），HTTP `/api/v1/cross/allocate`。
- ✅ **③ 入站建实例**：MapServer 620/621，`MapManager.EnsureCrossServerInstance` 按 `activity_id` 幂等——
  多个 realm 的 GameServer 都来问，收敛到同一张实例地图。
- ✅ **④ 定点路由**：GameServer 按 `serverID` 建跨域连接 + `crossBindings[playerID]`，该玩家的地图消息
  （含 outbox 重投）全部定点发往承载服。**不能按 mapID 路由**：实例 mapID 是各 MapServer 实例池的派生号，
  跨服务器会撞号（守护测试 `TestCrossEnter_DoesNotLeakIntoLocalRealmRouting`）。
- ✅ **⑤ 多进程 E2E**：`scripts/e2e-cross-realm.ps1` —— 两 realm 全栈 + 两客户端，实测两个 realm 的玩家
  落在**同一台 MapServer 的同一张实例图**并在 AOI 里互见。
- 🐞 E2E 揪出的真缺陷：`requestID` 只在进程内唯一，而收侧按裸 requestID 去重 → 两个 realm 的同号请求
  撞车，第二个 realm 玩家的进图被当成重放丢弃。已改为 `ComposeRequestID`（高位放 serverID）全集群唯一。
- 🗑 `cross_server_entry.go`（`CrossServerMapEntry`）已删：0 生产接线，且其中的迁移调用把玩家"迁到本服自己"、
  fire-and-forget 忽略失败——留着会让人误以为跨服进图已接通。真入口已按上面重做。
- ⏸ `MigrationManager`（玩家存档整体迁移）仍零生产接线——跨服活动是"人过去、家还在原服、结果回流"，
  用不到它；只有"永久换服 / 跨机无缝分区"才需要，届时按 `协议契约.md` §三.2 接线即可。

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
1. 把开新服流程脚本化（建库+建表+配置模板+起进程+登记列表）。**已落地**：`GameServer/cmd/newrealm`
   —— `go run ./cmd/newrealm -id 000102` 校验 6 位区服ID、创建隔离库 `GameDB_000102`（表由服务首启自动建）、
   列出所有 realm 库证隔离、并打印完整 runbook（端口按 ServerIndex 偏移、登记 game_servers、起进程顺序、三重隔离）。
   已验证：跑 000102 → `[GameDB_000101 GameDB_000102]` 两库并存互不可见。
2. 服务器列表管理：新服/推荐/爆满/维护 状态字段 + 客户端展示（runbook 步骤2，待接）。
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
- **① 无缝交接依赖 ②**（交接会更新 ② 的 `playerMap` 实际所在图记录）+ 须守 MAP-2 actor 边界；性价比路线上排在 ② 之后（见方向文档 §7）。

> 每项都以"能跑通的最小闭环 + 测试/真机 E2E"为完成标准，不追求生产级完整度——教学骨架的价值是把 realm 架构讲清楚、且每步都有真实背书。
