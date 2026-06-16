package handler

import (
	"github.com/gin-gonic/gin"

	"hm-dianping/internal/ctx"
	"hm-dianping/internal/service"
)

func HandleSeckill(svc *service.VoucherOrderService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := ctx.CurrentUser(c)
		if err != nil { writeFail(c, err); return }
		voucherID, err := pathUint(c, "id")
		if err != nil { writeFail(c, err); return }
		orderID, err := svc.OrderSeckill(c.Request.Context(), voucherID, user.ID)
		if err != nil { writeFail(c, err); return }
		writeOK(c, orderID)
	}
}