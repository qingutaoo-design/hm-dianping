package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"hm-dianping/internal/model"
	"hm-dianping/internal/service"
)

func HandleShopGet(svc *service.ShopService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := pathUint(c, "id")
		if err != nil { writeFail(c, err); return }
		shop, err := svc.GetByID(c.Request.Context(), id)
		if err != nil { writeFail(c, err); return }
		if shop == nil { writeFail(c, errors.New("商户信息不存在")); return }
		writeOK(c, shop)
	}
}

func HandleShopCreate(svc *service.ShopService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var shop model.Shop
		if err := c.ShouldBindJSON(&shop); err != nil { writeFail(c, err); return }
		if err := svc.Create(c.Request.Context(), &shop); err != nil { writeFail(c, err); return }
		writeOK(c, shop.ID)
	}
}

func HandleShopUpdate(svc *service.ShopService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var shop model.Shop
		if err := c.ShouldBindJSON(&shop); err != nil { writeFail(c, err); return }
		if err := svc.Update(c.Request.Context(), &shop); err != nil { writeFail(c, err); return }
		writeOK(c, nil)
	}
}

func HandleShopOfType(svc *service.ShopService) gin.HandlerFunc {
	return func(c *gin.Context) {
		typeID, err := queryUint(c, "typeId")
		if err != nil { writeFail(c, err); return }
		shops, err := svc.PageByType(c.Request.Context(), typeID, queryInt(c, "current", 1))
		if err != nil { writeFail(c, err); return }
		writeOK(c, shops)
	}
}

func HandleShopOfName(svc *service.ShopService) gin.HandlerFunc {
	return func(c *gin.Context) {
		shops, err := svc.PageByName(c.Request.Context(), c.Query("name"), queryInt(c, "current", 1))
		if err != nil { writeFail(c, err); return }
		writeOK(c, shops)
	}
}