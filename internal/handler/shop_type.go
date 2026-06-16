package handler

import (
	"github.com/gin-gonic/gin"

	"hm-dianping/internal/service"
)

func HandleShopTypeList(svc *service.ShopTypeService) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := svc.List(c.Request.Context())
		if err != nil { writeFail(c, err); return }
		writeOK(c, list)
	}
}