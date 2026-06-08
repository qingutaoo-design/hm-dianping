package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hm-dianping/internal/config"
	"hm-dianping/internal/dto"
	"hm-dianping/internal/service"
)

type Container struct {
	Users         *UserHandler
	Shops         *ShopHandler
	ShopTypes     *ShopTypeHandler
	Blogs         *BlogHandler
	Follows       *FollowHandler
	Vouchers      *VoucherHandler
	VoucherOrders *VoucherOrderHandler
	Uploads       *UploadHandler
}

func NewContainer(cfg *config.Config, services *service.Container) *Container {
	return &Container{
		Users:         NewUserHandler(services.Users),
		Shops:         NewShopHandler(services.Shops),
		ShopTypes:     NewShopTypeHandler(services.ShopTypes),
		Blogs:         NewBlogHandler(services.Blogs),
		Follows:       NewFollowHandler(services.Follows),
		Vouchers:      NewVoucherHandler(services.Vouchers),
		VoucherOrders: NewVoucherOrderHandler(services.VoucherOrders),
		Uploads:       NewUploadHandler(services.Uploads),
	}
}

func writeOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, dto.OK(data))
}

func writeFail(c *gin.Context, err error) {
	if err == nil {
		c.JSON(http.StatusOK, dto.Fail("服务器异常"))
		return
	}
	c.JSON(http.StatusOK, dto.Fail(err.Error()))
}
