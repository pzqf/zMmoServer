package dao

import (
	"fmt"

	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type AccountDAO struct {
	connector connector.DBConnector
}

func NewAccountDAO(dbConnector connector.DBConnector) *AccountDAO {
	return &AccountDAO{
		connector: dbConnector,
	}
}

func (dao *AccountDAO) GetAccountByID(accountID int64) (*models.Account, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Account{}.TableName())
		var account models.Account
		err := collection.FindOne(nil, bson.M{"account_id": accountID}).Decode(&account)
		if err != nil {
			if err.Error() == "mongo: no documents in result" {
				return nil, nil
			}
			return nil, err
		}
		return &account, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE account_id = ?", models.Account{}.TableName())
	rows, err := dao.connector.QuerySync(query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var account models.Account
	if rows.Next() {
		if err := rows.Scan(&account.AccountID, &account.AccountName, &account.Password, &account.Status, &account.CreatedAt, &account.LastLoginAt); err != nil {
			return nil, err
		}
		return &account, nil
	}
	return nil, nil
}

func (dao *AccountDAO) GetAccountByName(accountName string) (*models.Account, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Account{}.TableName())
		var account models.Account
		err := collection.FindOne(nil, bson.M{"account_name": accountName}).Decode(&account)
		if err != nil {
			if err.Error() == "mongo: no documents in result" {
				return nil, nil
			}
			return nil, err
		}
		return &account, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE account_name = ?", models.Account{}.TableName())
	rows, err := dao.connector.QuerySync(query, accountName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var account models.Account
	if rows.Next() {
		if err := rows.Scan(&account.AccountID, &account.AccountName, &account.Password, &account.Status, &account.CreatedAt, &account.LastLoginAt); err != nil {
			return nil, err
		}
		return &account, nil
	}
	return nil, nil
}

func (dao *AccountDAO) CreateAccount(account *models.Account) (int64, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Account{}.TableName())
		_, err := collection.InsertOne(nil, account)
		if err != nil {
			return 0, err
		}
		return account.AccountID, nil
	}

	query := fmt.Sprintf("INSERT INTO %s (account_id, account_name, password, status, created_at, last_login_at) VALUES (?, ?, ?, ?, ?, ?)", models.Account{}.TableName())
	result, err := dao.connector.ExecSync(query,
		account.AccountID, account.AccountName, account.Password, account.Status,
		account.CreatedAt.Format("2006-01-02 15:04:05"), account.LastLoginAt.Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (dao *AccountDAO) UpdateAccount(account *models.Account) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Account{}.TableName())
		result, err := collection.UpdateOne(nil, bson.M{"account_id": account.AccountID}, bson.M{
			"$set": bson.M{
				"account_name":  account.AccountName,
				"password":      account.Password,
				"status":        account.Status,
				"last_login_at": account.LastLoginAt,
			},
		})
		if err != nil {
			return false, err
		}
		return result.ModifiedCount > 0, nil
	}

	query := fmt.Sprintf("UPDATE %s SET account_name = ?, password = ?, status = ?, last_login_at = ? WHERE account_id = ?", models.Account{}.TableName())
	result, err := dao.connector.ExecSync(query,
		account.AccountName, account.Password, account.Status,
		account.LastLoginAt.Format("2006-01-02 15:04:05"), account.AccountID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected > 0, err
}

func (dao *AccountDAO) DeleteAccount(accountID int64) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Account{}.TableName())
		result, err := collection.DeleteOne(nil, bson.M{"account_id": accountID})
		if err != nil {
			return false, err
		}
		return result.DeletedCount > 0, nil
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE account_id = ?", models.Account{}.TableName())
	result, err := dao.connector.ExecSync(query, accountID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected > 0, err
}

func (dao *AccountDAO) UpdateLastLoginAt(accountID int64, lastLoginAt string) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Account{}.TableName())
		result, err := collection.UpdateOne(nil, bson.M{"account_id": accountID}, bson.M{
			"$set": bson.M{"last_login_at": lastLoginAt},
		})
		if err != nil {
			return false, err
		}
		return result.ModifiedCount > 0, nil
	}

	query := fmt.Sprintf("UPDATE %s SET last_login_at = ? WHERE account_id = ?", models.Account{}.TableName())
	result, err := dao.connector.ExecSync(query, lastLoginAt, accountID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected > 0, err
}
