package handler

import (
	"github.com/gin-gonic/gin"

	"hm-dianping/internal/service"
)

type ShopTypeHandler struct {
	service *service.ShopTypeService
}

func NewShopTypeHandler(service *service.ShopTypeService) *ShopTypeHandler {
	return &ShopTypeHandler{service: service}
}

func (h *ShopTypeHandler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context())
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, list)
}
