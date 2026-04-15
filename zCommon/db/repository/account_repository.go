package repository

import (
	"fmt"
	"time"

	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/models"
	"github.com/pzqf/zUtil/zCache"
)

type AccountRepositoryImpl struct {
	accountDAO *dao.AccountDAO
	cache      zCache.Cache
}

func NewAccountRepository(accountDAO *dao.AccountDAO) *AccountRepositoryImpl {
	return &AccountRepositoryImpl{
		accountDAO: accountDAO,
		cache:      zCache.NewLRUCache(1000, 5*time.Minute),
	}
}

func (r *AccountRepositoryImpl) GetByID(accountID int64) (*models.Account, error) {
	cacheKey := fmt.Sprintf("account:%d", accountID)
	if cached, err := r.cache.Get(cacheKey); err == nil {
		if account, ok := cached.(*models.Account); ok {
			return account, nil
		}
	}

	account, err := r.accountDAO.GetAccountByID(accountID)
	if err == nil && account != nil {
		_ = r.cache.Set(cacheKey, account, 5*time.Minute)
	}
	return account, err
}

func (r *AccountRepositoryImpl) GetByName(accountName string) (*models.Account, error) {
	cacheKey := fmt.Sprintf("account:name:%s", accountName)
	if cached, err := r.cache.Get(cacheKey); err == nil {
		if account, ok := cached.(*models.Account); ok {
			return account, nil
		}
	}

	account, err := r.accountDAO.GetAccountByName(accountName)
	if err == nil && account != nil {
		_ = r.cache.Set(cacheKey, account, 5*time.Minute)
		idCacheKey := fmt.Sprintf("account:%d", account.AccountID)
		_ = r.cache.Set(idCacheKey, account, 5*time.Minute)
	}
	return account, err
}

func (r *AccountRepositoryImpl) Create(account *models.Account) (int64, error) {
	id, err := r.accountDAO.CreateAccount(account)
	if err == nil && id > 0 {
		cacheKey := fmt.Sprintf("account:%d", id)
		_ = r.cache.Set(cacheKey, account, 5*time.Minute)
		nameCacheKey := fmt.Sprintf("account:name:%s", account.AccountName)
		_ = r.cache.Set(nameCacheKey, account, 5*time.Minute)
	}
	return id, err
}

func (r *AccountRepositoryImpl) Update(account *models.Account) (bool, error) {
	updated, err := r.accountDAO.UpdateAccount(account)
	if err == nil && updated {
		cacheKey := fmt.Sprintf("account:%d", account.AccountID)
		_ = r.cache.Set(cacheKey, account, 5*time.Minute)
		nameCacheKey := fmt.Sprintf("account:name:%s", account.AccountName)
		_ = r.cache.Set(nameCacheKey, account, 5*time.Minute)
	}
	return updated, err
}

func (r *AccountRepositoryImpl) Delete(accountID int64) (bool, error) {
	deleted, err := r.accountDAO.DeleteAccount(accountID)
	if err == nil && deleted {
		cacheKey := fmt.Sprintf("account:%d", accountID)
		_ = r.cache.Delete(cacheKey)
	}
	return deleted, err
}

func (r *AccountRepositoryImpl) UpdateLastLoginAt(accountID int64, lastLoginAt string) (bool, error) {
	updated, err := r.accountDAO.UpdateLastLoginAt(accountID, lastLoginAt)
	if err == nil && updated {
		cacheKey := fmt.Sprintf("account:%d", accountID)
		_ = r.cache.Delete(cacheKey)
	}
	return updated, err
}
