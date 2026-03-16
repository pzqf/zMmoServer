package handler

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/shop"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
)

type ShopHandler struct {
	sessionManager *session.SessionManager
	shopManager    *shop.ShopManager
}

func NewShopHandler(sessionManager *session.SessionManager, shopManager *shop.ShopManager) *ShopHandler {
	return &ShopHandler{
		sessionManager: sessionManager,
		shopManager:    shopManager,
	}
}

// HandleShopItemList 获取商品列表
func (sh *ShopHandler) HandleShopItemList(sessionID string, categoryID int) (*protocol.CommonResponse, error) {
	zLog.Info("Handling shop item list request", zap.String("session_id", sessionID), zap.Int("category_id", categoryID))

	_, exists := sh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	var items []*shop.ShopItem
	if categoryID > 0 {
		items = sh.shopManager.GetItemsByCategory(categoryID)
	} else {
		items = sh.shopManager.GetAllItems()
	}

	// 构建商品列表响应
	itemList := make([]*protocol.ShopItemInfo, 0, len(items))
	for _, item := range items {
		itemList = append(itemList, &protocol.ShopItemInfo{
			ItemId:       int32(item.ItemID),
			ShopId:       1, // 暂时设为1
			CategoryId:   int32(item.Category),
			ItemName:     item.Name,
			ItemDesc:     item.Description,
			Price:        int32(item.Price),
			CurrencyType: int32(item.PriceType),
			Stock:        999, // 暂时设为999
			MaxStock:     999, // 暂时设为999
			IsLimited:    item.LimitType > 0,
			RefreshTime:  item.StartTime.Unix(),
		})
	}

	// 这里应该直接返回ShopItemListResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}

// HandleShopCategoryList 获取分类列表
func (sh *ShopHandler) HandleShopCategoryList(sessionID string) (*protocol.CommonResponse, error) {
	zLog.Info("Handling shop category list request", zap.String("session_id", sessionID))

	_, exists := sh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	categories := sh.shopManager.GetAllCategories()

	// 构建分类列表响应
	categoryList := make([]*protocol.ShopCategoryInfo, 0, len(categories))
	for _, category := range categories {
		categoryList = append(categoryList, &protocol.ShopCategoryInfo{
			CategoryId:   int32(category.CategoryID),
			ShopId:       1, // 暂时设为1
			CategoryName: category.Name,
			CategoryDesc: "", // 暂时设为空
			SortOrder:    int32(category.SortOrder),
		})
	}

	// 这里应该直接返回ShopCategoryListResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}

// HandleShopBuy 购买商品
func (sh *ShopHandler) HandleShopBuy(sessionID string, itemID int64, count int) (*protocol.CommonResponse, error) {
	zLog.Info("Handling shop buy request", zap.String("session_id", sessionID), zap.Int64("item_id", itemID), zap.Int("count", count))

	session, exists := sh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 购买商品
	item, success := sh.shopManager.BuyItem(session.PlayerID, itemID, count)

	// 这里应该直接返回ShopBuyResponse，但由于函数签名限制，暂时返回CommonResponse
	if success && item != nil {
		return &protocol.CommonResponse{
			Result: 0,
		}, nil
	} else {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Failed to buy item",
		}, nil
	}
}

// HandleShopHotItems 获取热销商品
func (sh *ShopHandler) HandleShopHotItems(sessionID string) (*protocol.CommonResponse, error) {
	zLog.Info("Handling shop hot items request", zap.String("session_id", sessionID))

	_, exists := sh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	hotItems := sh.shopManager.GetHotItems()

	// 构建热销商品列表
	hotItemList := make([]*protocol.ShopItemInfo, 0, len(hotItems))
	for _, item := range hotItems {
		hotItemList = append(hotItemList, &protocol.ShopItemInfo{
			ItemId:       int32(item.ItemID),
			ShopId:       1, // 暂时设为1
			CategoryId:   int32(item.Category),
			ItemName:     item.Name,
			ItemDesc:     item.Description,
			Price:        int32(item.Price),
			CurrencyType: int32(item.PriceType),
			Stock:        999, // 暂时设为999
			MaxStock:     999, // 暂时设为999
			IsLimited:    item.LimitType > 0,
			RefreshTime:  item.StartTime.Unix(),
		})
	}

	// 这里应该直接返回ShopHotItemsResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}
