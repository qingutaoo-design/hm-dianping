package handler

import (
	"github.com/gin-gonic/gin"

	"hm-dianping/internal/model"
	"hm-dianping/internal/service"
)

type VoucherHandler struct {
	service *service.VoucherService
}

func NewVoucherHandler(service *service.VoucherService) *VoucherHandler {
	return &VoucherHandler{service: service}
}

func (h *VoucherHandler) Add(c *gin.Context) {
	var voucher model.Voucher
	if err := c.ShouldBindJSON(&voucher); err != nil {
		writeFail(c, err)
		return
	}
	if err := h.service.Add(c.Request.Context(), &voucher); err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, voucher.ID)
}

func (h *VoucherHandler) AddSeckill(c *gin.Context) {
	var voucher model.Voucher
	if err := c.ShouldBindJSON(&voucher); err != nil {
		writeFail(c, err)
		return
	}
	if err := h.service.AddSeckill(c.Request.Context(), &voucher); err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, voucher.ID)
}

func (h *VoucherHandler) List(c *gin.Context) {
	shopID, err := pathUint(c, "shopId")
	if err != nil {
		writeFail(c, err)
		return
	}
	list, err := h.service.QueryOfShop(c.Request.Context(), shopID)
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, list)
}
