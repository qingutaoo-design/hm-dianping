package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"hm-dianping/internal/model"
	"hm-dianping/internal/service"
)

type ShopHandler struct {
	service *service.ShopService
}

func NewShopHandler(service *service.ShopService) *ShopHandler {
	return &ShopHandler{service: service}
}

func (h *ShopHandler) Get(c *gin.Context) {
	id, err := pathUint(c, "id")
	if err != nil {
		writeFail(c, err)
		return
	}
	shop, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		writeFail(c, err)
		return
	}
	if shop == nil {
		writeFail(c, errors.New("商铺信息不存在"))
		return
	}
	writeOK(c, shop)
}

func (h *ShopHandler) Create(c *gin.Context) {
	var shop model.Shop
	if err := c.ShouldBindJSON(&shop); err != nil {
		writeFail(c, err)
		return
	}
	if err := h.service.Create(c.Request.Context(), &shop); err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, shop.ID)
}

func (h *ShopHandler) Update(c *gin.Context) {
	var shop model.Shop
	if err := c.ShouldBindJSON(&shop); err != nil {
		writeFail(c, err)
		return
	}
	if err := h.service.Update(c.Request.Context(), &shop); err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, nil)
}

func (h *ShopHandler) OfType(c *gin.Context) {
	typeID, err := queryUint(c, "typeId")
	if err != nil {
		writeFail(c, err)
		return
	}
	shops, err := h.service.PageByType(c.Request.Context(), typeID, queryInt(c, "current", 1))
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, shops)
}

func (h *ShopHandler) OfName(c *gin.Context) {
	shops, err := h.service.PageByName(c.Request.Context(), c.Query("name"), queryInt(c, "current", 1))
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, shops)
}
