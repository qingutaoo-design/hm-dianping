package handler

import (
	"github.com/gin-gonic/gin"

	"hm-dianping/internal/service"
)

type VoucherOrderHandler struct {
	service *service.VoucherOrderService
}

func NewVoucherOrderHandler(service *service.VoucherOrderService) *VoucherOrderHandler {
	return &VoucherOrderHandler{service: service}
}

func (h *VoucherOrderHandler) Seckill(c *gin.Context) {
	user, err := currentUser(c)
	if err != nil {
		writeFail(c, err)
		return
	}
	voucherID, err := pathUint(c, "id")
	if err != nil {
		writeFail(c, err)
		return
	}
	orderID, err := h.service.OrderSeckill(c.Request.Context(), voucherID, user.ID)
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, orderID)
}
