// newrealm —— 开新 realm（区服）的开服工具（realm 建议⑤：扩展=加服）。
//
// realm 模型（魔兽/传奇）里，承载翻倍就开一个新 realm——一个自成一体、与现有服**完全隔离**的世界：
// {独立 GameDB + 独立 GameServer/Gateway/MapServer 进程 + 服务器列表条目}。本工具负责其中**需要 DB 客户端**
// 的一步：创建隔离的 GameDB_<区服ID>（新世界的空库，表由服务进程首启时自动建）。其余（生成配置、登记服务器
// 列表、起进程）是纯操作步骤，本工具在末尾打印完整 runbook。
//
// 用法：
//   go run ./cmd/newrealm -id 000102                      # 开"1区2服"
//   go run ./cmd/newrealm -id 000102 -host 192.168.251.134:3306 -user root -pass 123456
//   go run ./cmd/newrealm -id 000102 -dry                 # 只打印，不建库
//
// 隔离保证：各服独立库互不可见；一个服 DB 故障不影响别服。ServerID 不入业务主键（用 Snowflake），
// 为将来"互联服合并低人口服"不撞 ID 留余地。
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	id := flag.String("id", "", "区服ID（6位：4位GroupID+2位ServerIndex，如 000102=1区2服）")
	host := flag.String("host", "192.168.251.134:3306", "MySQL 地址 host:port")
	user := flag.String("user", "root", "MySQL 用户名")
	pass := flag.String("pass", "123456", "MySQL 密码")
	dry := flag.Bool("dry", false, "只打印 runbook，不实际建库")
	flag.Parse()

	if *id == "" {
		fmt.Println("必须指定 -id（6位区服ID，如 000102）")
		os.Exit(2)
	}
	group, index, err := parseRealmID(*id)
	if err != nil {
		fmt.Println("无效 -id：", err)
		os.Exit(2)
	}
	dbName := "GameDB_" + *id

	fmt.Printf("== 开新 realm：区服ID=%s（GroupID=%d，ServerIndex=%d），独立库=%s ==\n", *id, group, index, dbName)

	if !*dry {
		if err := createRealmDB(*user, *pass, *host, dbName); err != nil {
			fmt.Println("建库失败：", err)
			os.Exit(1)
		}
		fmt.Printf("[✓] 已创建隔离库 %s（空库；表由服务进程首启时自动建）\n", dbName)
		if others, err := listRealmDBs(*user, *pass, *host); err == nil {
			fmt.Printf("[i] 当前 realm 库：%v（各自独立、互不可见）\n", others)
		}
	}

	printRunbook(*id, group, index, dbName)
}

// parseRealmID 校验 6 位区服ID 并拆出 GroupID(前4) + ServerIndex(后2)。
func parseRealmID(id string) (group, index int, err error) {
	if len(id) != 6 {
		return 0, 0, fmt.Errorf("区服ID 须为 6 位数字（4位组+2位序号），得到 %q", id)
	}
	g, err := strconv.Atoi(id[:4])
	if err != nil {
		return 0, 0, fmt.Errorf("GroupID 段非数字：%q", id[:4])
	}
	i, err := strconv.Atoi(id[4:])
	if err != nil {
		return 0, 0, fmt.Errorf("ServerIndex 段非数字：%q", id[4:])
	}
	if g < 1 || i < 1 {
		return 0, 0, fmt.Errorf("GroupID/ServerIndex 应 >=1")
	}
	return g, i, nil
}

func createRealmDB(user, pass, host, dbName string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/?timeout=8s", user, pass, host)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	// CREATE DATABASE IF NOT EXISTS —— 幂等；utf8mb4 与各服一致。
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS `" + dbName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci")
	return err
}

func listRealmDBs(user, pass, host string) ([]string, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/?timeout=8s", user, pass, host)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query("SHOW DATABASES LIKE 'GameDB_%'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err == nil {
			out = append(out, s)
		}
	}
	return out, rows.Err()
}

func printRunbook(id string, group, index int, dbName string) {
	// 端口按 ServerIndex 偏移，避免同机多 realm 撞端口（已有的端口占用探测会兜底拒绝启动）。
	off := (index - 1) * 100
	fmt.Printf(`
== 后续手动步骤（完整 runbook；本工具只负责建库） ==
1. 配置：复制 GameServer/Gateway/MapServer 的 config.ini，改：
   - GameServer: ServerID=%d, GroupID=%d, DBName=%s, ListenAddr 0.0.0.0:%d
   - Gateway:    ServerID=%d, GroupID=%d, 客户端监听端口按需偏移（如 %d）
   - MapServer:  ServerID/GroupID 同上, ListenAddr 端口偏移（如 %d）
   （同机多 realm 各端口 +%d 偏移；跨机则各用本机端口即可）
2. 登记服务器列表：向 GlobalDB.game_servers 插一行（ServerID=%s, Name, Region, Status=新服/推荐），
   GlobalServer 会并进 /api/v1/server/list，客户端即可见此服。
3. 起进程：Global（共用）→ 本服 Gateway → 本服 MapServer → 本服 GameServer；
   各自 etcd 注册，GameServer 发现本组 MapServer；表首启自动建。
4. 上线：客户端拉服务器列表 → 见新服 → 选入 → 进入全新空世界（与旧服数据互不可见）。

隔离：数据(独立库)/进程(独立)/发现(GroupID 分组) 三重隔离。承载估算=Σ各图层数×softCap，超了再加服或加层。
`,
		index, group, dbName, 20001+off,
		index, group, 10001+off,
		30001+off,
		off,
		id)
}
