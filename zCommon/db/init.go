package db

import (
	"fmt"
	"strings"

	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// InitTables 初始化数据库表结构
func InitTables(conn connector.DBConnector, repoType RepoType) error {
	if conn.GetDriver() != "mysql" {
		zLog.Info("Skipping table initialization for non-MySQL database", zap.String("driver", conn.GetDriver()))
		return nil
	}

	zLog.Info("Initializing database tables...", zap.String("repoType", string(repoType)))

	// 定义所有表的创建语句
	var createTablesSQL []string

	// 根据仓库类型选择需要创建的表
	switch repoType {
	case RepoTypeGlobal:
		// GlobalServer 只需要账号表和游戏服务器表
		createTablesSQL = []string{
			// 账号表
			`CREATE TABLE IF NOT EXISTS accounts (
				account_id BIGINT PRIMARY KEY,
				account_name VARCHAR(64) NOT NULL,
				password VARCHAR(128) NOT NULL,
				status INT NOT NULL DEFAULT 0,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				last_login_at DATETIME,
				UNIQUE KEY idx_account_name (account_name)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 游戏服务器表
			`CREATE TABLE IF NOT EXISTS game_servers (
				server_id INT PRIMARY KEY,
				server_name VARCHAR(255) NOT NULL,
				server_type VARCHAR(50) NOT NULL,
				group_id INT NOT NULL DEFAULT 0,
				address VARCHAR(255) NOT NULL DEFAULT '',
				port INT NOT NULL DEFAULT 0,
				status INT NOT NULL DEFAULT 0,
				online_count INT NOT NULL DEFAULT 0,
				max_online_count INT NOT NULL DEFAULT 5000,
				region VARCHAR(100) NOT NULL DEFAULT '',
				version VARCHAR(50) NOT NULL DEFAULT '',
				last_heartbeat DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				UNIQUE KEY idx_server_name (server_name),
				INDEX idx_group_id (group_id),
				INDEX idx_status (status),
				INDEX idx_region (region)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		}
	case RepoTypeGameServer:
		// GameServer 需要所有业务表
		createTablesSQL = []string{
			// 玩家表
			`CREATE TABLE IF NOT EXISTS players (
				id BIGINT PRIMARY KEY,
				account_id BIGINT NOT NULL,
				name VARCHAR(64) NOT NULL,
				gender INT NOT NULL DEFAULT 0,
				age INT NOT NULL DEFAULT 0,
				level INT NOT NULL DEFAULT 1,
				exp BIGINT NOT NULL DEFAULT 0,
				gold BIGINT NOT NULL DEFAULT 0,
				diamond BIGINT NOT NULL DEFAULT 0,
				vip_level INT NOT NULL DEFAULT 0,
				last_login_at DATETIME,
				last_logout_at DATETIME,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				UNIQUE KEY idx_name (name),
				INDEX idx_account_id (account_id),
				INDEX idx_level (level),
				INDEX idx_vip_level (vip_level)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 账号表
			`CREATE TABLE IF NOT EXISTS accounts (
				account_id BIGINT PRIMARY KEY,
				account_name VARCHAR(64) NOT NULL,
				password VARCHAR(128) NOT NULL,
				status INT NOT NULL DEFAULT 0,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				last_login_at DATETIME,
				UNIQUE KEY idx_account_name (account_name)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 玩家物品表
			`CREATE TABLE IF NOT EXISTS player_items (
				id BIGINT PRIMARY KEY AUTO_INCREMENT,
				player_id BIGINT NOT NULL,
				item_id INT NOT NULL,
				count INT NOT NULL DEFAULT 1,
				position INT NOT NULL DEFAULT 0,
				is_equipped BOOLEAN NOT NULL DEFAULT FALSE,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				INDEX idx_player_id (player_id),
				INDEX idx_item_id (item_id),
				INDEX idx_position (position)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 玩家技能表
			`CREATE TABLE IF NOT EXISTS player_skills (
				id BIGINT PRIMARY KEY AUTO_INCREMENT,
				player_id BIGINT NOT NULL,
				skill_id INT NOT NULL,
				level INT NOT NULL DEFAULT 1,
				exp INT NOT NULL DEFAULT 0,
				is_active BOOLEAN NOT NULL DEFAULT TRUE,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				INDEX idx_player_id (player_id),
				INDEX idx_skill_id (skill_id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 玩家任务表
			`CREATE TABLE IF NOT EXISTS player_quests (
				id BIGINT PRIMARY KEY AUTO_INCREMENT,
				player_id BIGINT NOT NULL,
				quest_id INT NOT NULL,
				status INT NOT NULL DEFAULT 0,
				progress JSON NOT NULL,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				INDEX idx_player_id (player_id),
				INDEX idx_quest_id (quest_id),
				INDEX idx_status (status)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 玩家宠物表
			`CREATE TABLE IF NOT EXISTS player_pets (
				id BIGINT PRIMARY KEY AUTO_INCREMENT,
				player_id BIGINT NOT NULL,
				pet_id INT NOT NULL,
				name VARCHAR(64) NOT NULL,
				level INT NOT NULL DEFAULT 1,
				exp INT NOT NULL DEFAULT 0,
				is_active BOOLEAN NOT NULL DEFAULT FALSE,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				INDEX idx_player_id (player_id),
				INDEX idx_pet_id (pet_id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 玩家邮件表
			`CREATE TABLE IF NOT EXISTS player_mails (
				id BIGINT PRIMARY KEY AUTO_INCREMENT,
				player_id BIGINT NOT NULL,
				sender VARCHAR(64) NOT NULL,
				title VARCHAR(128) NOT NULL,
				content TEXT NOT NULL,
				is_read BOOLEAN NOT NULL DEFAULT FALSE,
				is_claimed BOOLEAN NOT NULL DEFAULT FALSE,
				attachments JSON,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				INDEX idx_player_id (player_id),
				INDEX idx_is_read (is_read),
				INDEX idx_is_claimed (is_claimed)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 玩家 Buff 表
			`CREATE TABLE IF NOT EXISTS player_buffs (
				id BIGINT PRIMARY KEY AUTO_INCREMENT,
				player_id BIGINT NOT NULL,
				buff_id INT NOT NULL,
				level INT NOT NULL DEFAULT 1,
				duration INT NOT NULL DEFAULT 0,
				remaining_time INT NOT NULL DEFAULT 0,
				is_active BOOLEAN NOT NULL DEFAULT TRUE,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				INDEX idx_player_id (player_id),
				INDEX idx_buff_id (buff_id),
				INDEX idx_is_active (is_active)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 公会表
			`CREATE TABLE IF NOT EXISTS guilds (
				id BIGINT PRIMARY KEY,
				name VARCHAR(64) NOT NULL,
				level INT NOT NULL DEFAULT 1,
				exp INT NOT NULL DEFAULT 0,
				member_count INT NOT NULL DEFAULT 1,
				max_members INT NOT NULL DEFAULT 20,
				leader_id BIGINT NOT NULL,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				UNIQUE KEY idx_name (name),
				INDEX idx_level (level),
				INDEX idx_leader_id (leader_id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 公会成员表
			`CREATE TABLE IF NOT EXISTS guild_members (
				id BIGINT PRIMARY KEY AUTO_INCREMENT,
				guild_id BIGINT NOT NULL,
				player_id BIGINT NOT NULL,
				guild_rank INT NOT NULL DEFAULT 0,
				contribution INT NOT NULL DEFAULT 0,
				join_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				last_active_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				UNIQUE KEY idx_guild_player (guild_id, player_id),
				INDEX idx_guild_id (guild_id),
				INDEX idx_player_id (player_id),
				INDEX idx_guild_rank (guild_rank)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 游戏服务器表
			`CREATE TABLE IF NOT EXISTS game_servers (
				server_id INT PRIMARY KEY,
				server_name VARCHAR(255) NOT NULL,
				server_type VARCHAR(50) NOT NULL,
				group_id INT NOT NULL DEFAULT 0,
				address VARCHAR(255) NOT NULL DEFAULT '',
				port INT NOT NULL DEFAULT 0,
				status INT NOT NULL DEFAULT 0,
				online_count INT NOT NULL DEFAULT 0,
				max_online_count INT NOT NULL DEFAULT 5000,
				region VARCHAR(100) NOT NULL DEFAULT '',
				version VARCHAR(50) NOT NULL DEFAULT '',
				last_heartbeat DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				UNIQUE KEY idx_server_name (server_name),
				INDEX idx_group_id (group_id),
				INDEX idx_status (status),
				INDEX idx_region (region)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 登录日志表
			`CREATE TABLE IF NOT EXISTS login_logs (
				id BIGINT PRIMARY KEY AUTO_INCREMENT,
				account_id BIGINT NOT NULL,
				player_id BIGINT,
				login_ip VARCHAR(32) NOT NULL,
				login_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				device_id VARCHAR(128),
				device_info VARCHAR(256),
				INDEX idx_account_id (account_id),
				INDEX idx_player_id (player_id),
				INDEX idx_login_time (login_time)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 邮件日志表
			`CREATE TABLE IF NOT EXISTS mail_logs (
				id BIGINT PRIMARY KEY AUTO_INCREMENT,
				sender_id BIGINT,
				receiver_id BIGINT NOT NULL,
				title VARCHAR(128) NOT NULL,
				content TEXT NOT NULL,
				send_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				is_system BOOLEAN NOT NULL DEFAULT FALSE,
				INDEX idx_sender_id (sender_id),
				INDEX idx_receiver_id (receiver_id),
				INDEX idx_send_time (send_time)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 任务日志表
			`CREATE TABLE IF NOT EXISTS quest_logs (
				id BIGINT PRIMARY KEY AUTO_INCREMENT,
				player_id BIGINT NOT NULL,
				quest_id INT NOT NULL,
				status INT NOT NULL,
				complete_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				reward_claimed BOOLEAN NOT NULL DEFAULT FALSE,
				INDEX idx_player_id (player_id),
				INDEX idx_quest_id (quest_id),
				INDEX idx_status (status)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 拍卖行表
			`CREATE TABLE IF NOT EXISTS auctions (
				id BIGINT PRIMARY KEY AUTO_INCREMENT,
				item_id INT NOT NULL,
				item_count INT NOT NULL DEFAULT 1,
				seller_id BIGINT NOT NULL,
				starting_price BIGINT NOT NULL,
				current_price BIGINT NOT NULL,
				buyer_id BIGINT,
				end_time DATETIME NOT NULL,
				status INT NOT NULL DEFAULT 0,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				INDEX idx_seller_id (seller_id),
				INDEX idx_buyer_id (buyer_id),
				INDEX idx_status (status),
				INDEX idx_end_time (end_time)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 拍卖行日志表
			`CREATE TABLE IF NOT EXISTS auction_logs (
				id BIGINT PRIMARY KEY AUTO_INCREMENT,
				auction_id BIGINT NOT NULL,
				bidder_id BIGINT NOT NULL,
				bid_price BIGINT NOT NULL,
				bid_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				INDEX idx_auction_id (auction_id),
				INDEX idx_bidder_id (bidder_id),
				INDEX idx_bid_time (bid_time)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		}
	default:
		zLog.Warn("Unknown repo type, creating minimal tables", zap.String("repoType", string(repoType)))
		// 默认只创建账号表和游戏服务器表
		createTablesSQL = []string{
			// 账号表
			`CREATE TABLE IF NOT EXISTS accounts (
				account_id BIGINT PRIMARY KEY,
				account_name VARCHAR(64) NOT NULL,
				password VARCHAR(128) NOT NULL,
				status INT NOT NULL DEFAULT 0,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				last_login_at DATETIME,
				UNIQUE KEY idx_account_name (account_name)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

			// 游戏服务器表
			`CREATE TABLE IF NOT EXISTS game_servers (
				server_id INT PRIMARY KEY,
				server_name VARCHAR(255) NOT NULL,
				server_type VARCHAR(50) NOT NULL,
				group_id INT NOT NULL DEFAULT 0,
				address VARCHAR(255) NOT NULL DEFAULT '',
				port INT NOT NULL DEFAULT 0,
				status INT NOT NULL DEFAULT 0,
				online_count INT NOT NULL DEFAULT 0,
				max_online_count INT NOT NULL DEFAULT 5000,
				region VARCHAR(100) NOT NULL DEFAULT '',
				version VARCHAR(50) NOT NULL DEFAULT '',
				last_heartbeat DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				UNIQUE KEY idx_server_name (server_name),
				INDEX idx_group_id (group_id),
				INDEX idx_status (status),
				INDEX idx_region (region)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		}
	}

	// 执行创建表语句
	for i, sqlStmt := range createTablesSQL {
		sqlStmt = strings.TrimSpace(sqlStmt)
		if sqlStmt == "" {
			continue
		}

		zLog.Info("Creating table...", zap.Int("index", i), zap.String("sql", sqlStmt[:50]+"..."))

		_, err := conn.ExecSync(sqlStmt)
		if err != nil {
			zLog.Error("Failed to create table", zap.Error(err), zap.String("sql", sqlStmt))
		} else {
			zLog.Info("Table created successfully")
		}
	}

	zLog.Info("Database tables initialized successfully", zap.String("repoType", string(repoType)))
	return nil
}

// InitDefaultData 初始化默认数据
func InitDefaultData(conn connector.DBConnector) error {
	if conn.GetDriver() != "mysql" {
		zLog.Info("Skipping default data initialization for non-MySQL database", zap.String("driver", conn.GetDriver()))
		return nil
	}

	zLog.Info("Initializing default data...")

	rows, err := conn.QuerySync("SELECT COUNT(*) FROM game_servers")
	if err != nil {
		zLog.Error("Failed to check game servers", zap.Error(err))
		return nil
	}

	var count int
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			zLog.Error("Failed to scan count", zap.Error(err))
			rows.Close()
			return nil
		}
	}
	rows.Close()

	if count == 0 {
		defaultServers := []struct {
			ID      int
			Name    string
			Type    string
			GroupID int
			Address string
			Port    int
		}{
			{1, "GameServer-000101", "game", 1, "127.0.0.1", 20001},
			{2, "GameServer-000102", "game", 1, "127.0.0.1", 20003},
			{101, "Gateway-000101", "gateway", 1, "127.0.0.1", 10001},
			{102, "Gateway-000102", "gateway", 1, "127.0.0.1", 10002},
			{201, "MapServer-1", "map", 1, "127.0.0.1", 30001},
		}

		for _, server := range defaultServers {
			insertSQL := fmt.Sprintf(
				"INSERT INTO game_servers (server_id, server_name, server_type, group_id, address, port) VALUES (%d, '%s', '%s', %d, '%s', %d)",
				server.ID, server.Name, server.Type, server.GroupID, server.Address, server.Port,
			)
			_, err := conn.ExecSync(insertSQL)
			if err != nil {
				zLog.Error("Failed to insert default server", zap.Error(err), zap.String("name", server.Name))
			} else {
				zLog.Info("Default server inserted", zap.String("name", server.Name))
			}
		}
	}

	zLog.Info("Default data initialized successfully")
	return nil
}
