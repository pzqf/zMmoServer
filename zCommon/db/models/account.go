package models

import (
	"time"
)

// Account 账号模型，映射account表
type Account struct {
	AccountID   int64     `db:"account_id" bson:"account_id"`
	AccountName string    `db:"account_name" bson:"account_name"`
	Password    string    `db:"password" bson:"password"`
	Status      int       `db:"status" bson:"status"`
	CreatedAt   time.Time `db:"created_at" bson:"created_at"`
	LastLoginAt time.Time `db:"last_login_at" bson:"last_login_at"`
}

// TableName 返回表名
func (Account) TableName() string {
	return "accounts"
}
