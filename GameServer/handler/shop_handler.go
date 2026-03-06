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
func (sh *ShopHandler) HandleShopItemList(sessionID string, categoryID int) (*protocol.Response, error) {
	zLog.Info("Handling shop item list request", zap.String("session_id", sessionID), zap.Int("category_id", categoryID))

	_, exists := sh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
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
			ItemId:         item.ItemID,
			Name:           item.Name,
			Description:    item.Description,
			Icon:           item.Icon,
			PriceType:      int32(item.PriceType),
			Price:          item.Price,
			OriginalPrice:  item.OriginalPrice,
			Category:       int32(item.Category),
			ItemType:       int32(item.ItemType),
			ItemCount:      int32(item.ItemCount),
			LevelReq:       int32(item.LevelReq),
			VipReq:         int32(item.VipReq),
			LimitType:      int32(item.LimitType),
			LimitCount:     int32(item.LimitCount),
			IsHot:          item.IsHot,
			IsNew:          item.IsNew,
			StartTime:      item.StartTime.Unix(),
			EndTime:        item.EndTime.Unix(),
		})
	}

	response := &protocol.ShopItemListResponse{
		Success: true,
		Items:   itemList,
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}

// HandleShopCategoryList 获取分类列表
func (sh *ShopHandler) HandleShopCategoryList(sessionID string) (*protocol.Response, error) {
	zLog.Info("Handling shop category list request", zap.String("session_id", sessionID))

	_, exists := sh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	categories := sh.shopManager.GetAllCategories()

	// 构建分类列表响应
	categoryList := make([]*protocol.ShopCategoryInfo, 0, len(categories))
	for _, category := range categories {
		categoryList = append(categoryList, &protocol.ShopCategoryInfo{
			CategoryId: int32(category.CategoryID),
			Name:       category.Name,
			Icon:       category.Icon,
			SortOrder:  int32(category.SortOrder),
			IsVisible:  category.IsVisible,
		})
	}

	response := &protocol.ShopCategoryListResponse{
		Success:    true,
		Categories: categoryList,
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}

// HandleShopBuy 购买商品
func (sh *ShopHandler) HandleShopBuy(sessionID string, itemID int64, count int) (*protocol.Response, error) {
	zLog.Info("Handling shop buy request", zap.String("session_id", sessionID), zap.Int64("item_id", itemID), zap.Int("count", count))

	session, exists := sh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 购买商品
	item, success := sh.shopManager.BuyItem(session.PlayerID, itemID, count)

	var response *protocol.ShopBuyResponse
	if success && item != nil {
		response = &protocol.ShopBuyResponse{
			Success:  true,
			ItemId:   item.ItemID,
			Count:    int32(count),
			TotalPrice: item.Price * int64(count),
		}
	} else {
		response = &protocol.ShopBuyResponse{
			Success: false,
		}
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}

// HandleShopHotItems 获取热销商品
func (sh *ShopHandler) HandleShopHotItems(sessionID string) (*protocol.Response, error) {
	zLog.Info("Handling shop hot items request", zap.String("session_id", sessionID))

	_, exists := sh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	hotItems := sh.shopManager.GetHotItems()

	// 构建热销商品列表
	hotItemList := make([]*protocol.ShopItemInfo, 0, len(hotItems))
	for _, item := range hotItems {
		hotItemList = append(hotItemList, &protocol.ShopItemInfo{
			ItemId:         item.ItemID,
			Name:           item.Name,
			Description:    item.Description,
			Icon:           item.Icon,
			PriceType:      int32(item.PriceType),
			Price:          item.Price,
			OriginalPrice:  item.OriginalPrice,
			Category:       int32(item.Category),
			ItemType:       int32(item.ItemType),
			ItemCount:      int32(item.ItemCount),
			LevelReq:       int32(item.LevelReq),
			VipReq:         int32(item.VipReq),
			LimitType:      int32(item.LimitType),
			LimitCount:     int32(item.LimitCount),
			IsHot:          item.IsHot,
			IsNew:          item.IsNew,
			StartTime:      item.StartTime.Unix(),
			EndTime:        item.EndTime.Unix(),
		})
	}

	response := &protocol.ShopHotItemsResponse{
		Success: true,
		Items:   hotItemList,
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}
