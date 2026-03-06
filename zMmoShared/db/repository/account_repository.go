package repository

import (
	"fmt"
	"time"

	"github.com/pzqf/zMmoShared/db/dao"
	"github.com/pzqf/zMmoShared/db/models"
	"github.com/pzqf/zUtil/zCache"
)

// AccountRepositoryImpl 账号数据仓库实现
type AccountRepositoryImpl struct {
	accountDAO *dao.AccountDAO
	cache      zCache.Cache
}

// NewAccountRepository 创建账号数据仓库实例
func NewAccountRepository(accountDAO *dao.AccountDAO) *AccountRepositoryImpl {
	return &AccountRepositoryImpl{
		accountDAO: accountDAO,
		cache:      zCache.NewLRUCache(1000, 5*time.Minute), // 1000容量，5分钟过期
	}
}

// GetByIDAsync 根据ID异步获取账号
func (r *AccountRepositoryImpl) GetByIDAsync(accountID int64, callback func(*models.Account, error)) {
	// 尝试从缓存获取
	cacheKey := fmt.Sprintf("account:%d", accountID)
	if cached, err := r.cache.Get(cacheKey); err == nil {
		if account, ok := cached.(*models.Account); ok {
			if callback != nil {
				callback(account, nil)
			}
			return
		}
	}

	// 缓存未命中，从数据库获取
	r.accountDAO.GetAccountByID(accountID, func(a *models.Account, err error) {
		if err == nil && a != nil {
			_ = r.cache.Set(cacheKey, a, 5*time.Minute)
		}

		if callback != nil {
			callback(a, err)
		}
	})
}

// GetByNameAsync 根据名称异步获取账号
func (r *AccountRepositoryImpl) GetByNameAsync(accountName string, callback func(*models.Account, error)) {
	// 尝试从缓存获取
	cacheKey := fmt.Sprintf("account:name:%s", accountName)
	if cached, err := r.cache.Get(cacheKey); err == nil {
		if account, ok := cached.(*models.Account); ok {
			if callback != nil {
				callback(account, nil)
			}
			return
		}
	}

	// 缓存未命中，从数据库获取
	r.accountDAO.GetAccountByName(accountName, func(a *models.Account, err error) {
		if err == nil && a != nil {
			_ = r.cache.Set(cacheKey, a, 5*time.Minute)
			idCacheKey := fmt.Sprintf("account:%d", a.AccountID)
			_ = r.cache.Set(idCacheKey, a, 5*time.Minute)
		}

		if callback != nil {
			callback(a, err)
		}
	})
}

// CreateAsync 异步创建账号
func (r *AccountRepositoryImpl) CreateAsync(account *models.Account, callback func(int64, error)) {
	// 异步执行创建
	r.accountDAO.CreateAccount(account, func(id int64, err error) {
		if err == nil && id > 0 {
			cacheKey := fmt.Sprintf("account:%d", id)
			_ = r.cache.Set(cacheKey, account, 5*time.Minute)
			nameCacheKey := fmt.Sprintf("account:name:%s", account.AccountName)
			_ = r.cache.Set(nameCacheKey, account, 5*time.Minute)
		}

		if callback != nil {
			callback(id, err)
		}
	})
}

// UpdateAsync 异步更新账号
func (r *AccountRepositoryImpl) UpdateAsync(account *models.Account, callback func(bool, error)) {
	// 异步执行更新
	r.accountDAO.UpdateAccount(account, func(updated bool, err error) {
		if err == nil && updated {
			cacheKey := fmt.Sprintf("account:%d", account.AccountID)
			_ = r.cache.Set(cacheKey, account, 5*time.Minute)
			nameCacheKey := fmt.Sprintf("account:name:%s", account.AccountName)
			_ = r.cache.Set(nameCacheKey, account, 5*time.Minute)
		}

		if callback != nil {
			callback(updated, err)
		}
	})
}

// DeleteAsync 异步删除账号
func (r *AccountRepositoryImpl) DeleteAsync(accountID int64, callback func(bool, error)) {
	// 异步执行删除
	r.accountDAO.DeleteAccount(accountID, func(deleted bool, err error) {
		if err == nil && deleted {
			cacheKey := fmt.Sprintf("account:%d", accountID)
			_ = r.cache.Delete(cacheKey)
		}

		if callback != nil {
			callback(deleted, err)
		}
	})
}

// UpdateLastLoginAtAsync 异步更新最后登录时间
func (r *AccountRepositoryImpl) UpdateLastLoginAtAsync(accountID int64, lastLoginAt string, callback func(bool, error)) {
	// 异步执行更新
	r.accountDAO.UpdateLastLoginAt(accountID, lastLoginAt, func(updated bool, err error) {
		if err == nil && updated {
			cacheKey := fmt.Sprintf("account:%d", accountID)
			_ = r.cache.Delete(cacheKey)
		}

		if callback != nil {
			callback(updated, err)
		}
	})
}

// GetByID 根据ID获取账号（同步兼容方法）
func (r *AccountRepositoryImpl) GetByID(accountID int64) (*models.Account, error) {
	var result *models.Account
	var resultErr error
	ch := make(chan struct{})
	r.GetByIDAsync(accountID, func(a *models.Account, err error) {
		result = a
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// GetByName 根据名称获取账号（同步兼容方法）
func (r *AccountRepositoryImpl) GetByName(accountName string) (*models.Account, error) {
	var result *models.Account
	var resultErr error
	ch := make(chan struct{})
	r.GetByNameAsync(accountName, func(a *models.Account, err error) {
		result = a
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// Create 创建账号（同步兼容方法）
func (r *AccountRepositoryImpl) Create(account *models.Account) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.CreateAsync(account, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// Update 更新账号（同步兼容方法）
func (r *AccountRepositoryImpl) Update(account *models.Account) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.UpdateAsync(account, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// Delete 删除账号（同步兼容方法）
func (r *AccountRepositoryImpl) Delete(accountID int64) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.DeleteAsync(accountID, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// UpdateLastLoginAt 更新最后登录时间（同步兼容方法）
func (r *AccountRepositoryImpl) UpdateLastLoginAt(accountID int64, lastLoginAt string) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.UpdateLastLoginAtAsync(accountID, lastLoginAt, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}
