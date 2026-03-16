package dao

import (
	"database/sql"
	"fmt"

	"github.com/pzqf/zMmoShared/db/connector"
	"github.com/pzqf/zMmoShared/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

// AccountDAO 账号数据访问对象
type AccountDAO struct {
	connector connector.DBConnector
}

// NewAccountDAO 创建账号DAO实例
func NewAccountDAO(dbConnector connector.DBConnector) *AccountDAO {
	return &AccountDAO{
		connector: dbConnector,
	}
}

// GetAccountByID 根据ID获取账号信息
func (dao *AccountDAO) GetAccountByID(accountID int64, callback func(*models.Account, error)) {
	// 根据数据库驱动类型执行不同的查询操作
	if dao.connector.GetDriver() == "mongo" {
		// MongoDB查询
		collection := dao.connector.GetMongoDB().Collection(models.Account{}.TableName())
		var account models.Account

		// 使用FindOne查询单个文档
		result := collection.FindOne(nil, bson.M{"account_id": accountID})
		err := result.Decode(&account)

		if err != nil {
			// 如果是未找到文档的错误，返回nil
			if err.Error() == "mongo: no documents in result" {
				if callback != nil {
					callback(nil, nil) // 未找到账号
				}
				return
			}

			// 其他错误
			if callback != nil {
				callback(nil, err)
			}
			return
		}

		if callback != nil {
			callback(&account, nil)
		}
	} else {
		// MySQL查询
		query := fmt.Sprintf("SELECT * FROM %s WHERE account_id = ?", models.Account{}.TableName())

		dao.connector.Query(query, []interface{}{accountID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var account models.Account
			if rows.Next() {
				if err := rows.Scan(
					&account.AccountID,
					&account.AccountName,
					&account.Password,
					&account.Status,
					&account.CreatedAt,
					&account.LastLoginAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}

				if callback != nil {
					callback(&account, nil)
				}
			} else {
				if callback != nil {
					callback(nil, nil) // 未找到账号
				}
			}
		})
	}
}

// GetAccountByName 根据名称获取账号信息
func (dao *AccountDAO) GetAccountByName(accountName string, callback func(*models.Account, error)) {
	// 根据数据库驱动类型执行不同的查询操作
	if dao.connector.GetDriver() == "mongo" {
		// MongoDB查询
		collection := dao.connector.GetMongoDB().Collection(models.Account{}.TableName())
		var account models.Account

		// 使用FindOne查询单个文档
		result := collection.FindOne(nil, bson.M{"account_name": accountName})
		err := result.Decode(&account)

		if err != nil {
			// 如果是未找到文档的错误，返回nil
			if err.Error() == "mongo: no documents in result" {
				if callback != nil {
					callback(nil, nil) // 未找到账号
				}
				return
			}

			// 其他错误
			if callback != nil {
				callback(nil, err)
			}
			return
		}

		if callback != nil {
			callback(&account, nil)
		}
	} else {
		// MySQL查询
		query := fmt.Sprintf("SELECT * FROM %s WHERE account_name = ?", models.Account{}.TableName())

		dao.connector.Query(query, []interface{}{accountName}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var account models.Account
			if rows.Next() {
				if err := rows.Scan(
					&account.AccountID,
					&account.AccountName,
					&account.Password,
					&account.Status,
					&account.CreatedAt,
					&account.LastLoginAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}

				if callback != nil {
					callback(&account, nil)
				}
			} else {
				if callback != nil {
					callback(nil, nil) // 未找到账号
				}
			}
		})
	}
}

// CreateAccount 创建账号
func (dao *AccountDAO) CreateAccount(account *models.Account, callback func(int64, error)) {
	// 根据数据库驱动类型执行不同的插入操作
	if dao.connector.GetDriver() == "mongo" {
		// MongoDB插入
		collection := dao.connector.GetMongoDB().Collection(models.Account{}.TableName())

		// 使用InsertOne插入文档
		_, err := collection.InsertOne(nil, account)

		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}

		if callback != nil {
			// MongoDB使用自增ID或ObjectID，但在这个模型中我们使用自定义的account_id
			callback(account.AccountID, nil)
		}
	} else {
		// MySQL插入
		query := fmt.Sprintf("INSERT INTO %s (account_id, account_name, password, status, created_at, last_login_at) VALUES (?, ?, ?, ?, ?, ?)", models.Account{}.TableName())

		args := []interface{}{
			account.AccountID,
			account.AccountName,
			account.Password,
			account.Status,
			account.CreatedAt.Format("2006-01-02 15:04:05"),
			account.LastLoginAt.Format("2006-01-02 15:04:05"),
		}

		dao.connector.Execute(query, args, func(result sql.Result, err error) {
			if err != nil {
				if callback != nil {
					callback(0, err)
				}
				return
			}

			if callback != nil {
				callback(account.AccountID, err)
			}
		})
	}
}

// UpdateAccount 更新账号信息
func (dao *AccountDAO) UpdateAccount(account *models.Account, callback func(bool, error)) {
	// 根据数据库驱动类型执行不同的更新操作
	if dao.connector.GetDriver() == "mongo" {
		// MongoDB更新
		collection := dao.connector.GetMongoDB().Collection(models.Account{}.TableName())

		// 构建更新文档
		update := bson.M{
			"$set": bson.M{
				"account_name":  account.AccountName,
				"password":      account.Password,
				"status":        account.Status,
				"last_login_at": account.LastLoginAt,
			},
		}

		// 使用UpdateOne更新文档
		result, err := collection.UpdateOne(nil, bson.M{"account_id": account.AccountID}, update)

		if err != nil {
			if callback != nil {
				callback(false, err)
			}
			return
		}

		if callback != nil {
			callback(result.ModifiedCount > 0, nil)
		}
	} else {
		// MySQL更新
		query := fmt.Sprintf("UPDATE %s SET account_name = ?, password = ?, status = ?, last_login_at = ? WHERE account_id = ?", models.Account{}.TableName())

		args := []interface{}{
			account.AccountName,
			account.Password,
			account.Status,
			account.LastLoginAt.Format("2006-01-02 15:04:05"),
			account.AccountID,
		}

		dao.connector.Execute(query, args, func(result sql.Result, err error) {
			if err != nil {
				if callback != nil {
					callback(false, err)
				}
				return
			}

			rowsAffected, err := result.RowsAffected()
			if callback != nil {
				callback(rowsAffected > 0, err)
			}
		})
	}
}

// DeleteAccount 删除账号
func (dao *AccountDAO) DeleteAccount(accountID int64, callback func(bool, error)) {
	// 根据数据库驱动类型执行不同的删除操作
	if dao.connector.GetDriver() == "mongo" {
		// MongoDB删除
		collection := dao.connector.GetMongoDB().Collection(models.Account{}.TableName())

		// 使用DeleteOne删除文档
		result, err := collection.DeleteOne(nil, bson.M{"account_id": accountID})

		if err != nil {
			if callback != nil {
				callback(false, err)
			}
			return
		}

		if callback != nil {
			callback(result.DeletedCount > 0, nil)
		}
	} else {
		// MySQL删除
		query := fmt.Sprintf("DELETE FROM %s WHERE account_id = ?", models.Account{}.TableName())

		dao.connector.Execute(query, []interface{}{accountID}, func(result sql.Result, err error) {
			if err != nil {
				if callback != nil {
					callback(false, err)
				}
				return
			}

			rowsAffected, err := result.RowsAffected()
			if callback != nil {
				callback(rowsAffected > 0, err)
			}
		})
	}
}

// UpdateLastLoginAt 更新最后登录时间
func (dao *AccountDAO) UpdateLastLoginAt(accountID int64, lastLoginAt string, callback func(bool, error)) {
	// 根据数据库驱动类型执行不同的更新操作
	if dao.connector.GetDriver() == "mongo" {
		// MongoDB更新
		collection := dao.connector.GetMongoDB().Collection(models.Account{}.TableName())

		// 构建更新文档
		update := bson.M{
			"$set": bson.M{
				"last_login_at": lastLoginAt,
			},
		}

		// 使用UpdateOne更新文档
		result, err := collection.UpdateOne(nil, bson.M{"account_id": accountID}, update)

		if err != nil {
			if callback != nil {
				callback(false, err)
			}
			return
		}

		if callback != nil {
			callback(result.ModifiedCount > 0, nil)
		}
	} else {
		// MySQL更新
		query := fmt.Sprintf("UPDATE %s SET last_login_at = ? WHERE account_id = ?", models.Account{}.TableName())

		dao.connector.Execute(query, []interface{}{lastLoginAt, accountID}, func(result sql.Result, err error) {
			if err != nil {
				if callback != nil {
					callback(false, err)
				}
				return
			}

			rowsAffected, err := result.RowsAffected()
			if callback != nil {
				callback(rowsAffected > 0, err)
			}
		})
	}
}
