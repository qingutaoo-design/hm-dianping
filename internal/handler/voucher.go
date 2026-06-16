package handler

import (
	"github.com/gin-gonic/gin"

	"hm-dianping/internal/model"
	"hm-dianping/internal/service"
)

func HandleVoucherAdd(svc *service.VoucherService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var voucher model.Voucher
		if err := c.ShouldBindJSON(&voucher); err != nil { writeFail(c, err); return }
		if err := svc.Add(c.Request.Context(), &voucher); err != nil { writeFail(c, err); return }
		writeOK(c, voucher.ID)
	}
}

func HandleVoucherAddSeckill(svc *service.VoucherService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var voucher model.Voucher
		if err := c.ShouldBindJSON(&voucher); err != nil { writeFail(c, err); return }
		if err := svc.AddSeckill(c.Request.Context(), &voucher); err != nil { writeFail(c, err); return }
		writeOK(c, voucher.ID)
	}
}

func HandleVoucherList(svc *service.VoucherService) gin.HandlerFunc {
	return func(c *gin.Context) {
		shopID, err := pathUint(c, "shopId")
		if err != nil { writeFail(c, err); return }
		list, err := svc.QueryOfShop(c.Request.Context(), shopID)
		if err != nil { writeFail(c, err); return }
		writeOK(c, list)
	}
}