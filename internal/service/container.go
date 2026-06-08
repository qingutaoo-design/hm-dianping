package service

import (
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"

	"hm-dianping/internal/config"
)

type Container struct {
	Users         *UserService
	Shops         *ShopService
	ShopTypes     *ShopTypeService
	Blogs         *BlogService
	Follows       *FollowService
	Vouchers      *VoucherService
	VoucherOrders *VoucherOrderService
	Uploads       *UploadService
}

func NewContainer(cfg *config.Config, db *gorm.DB, rdb *redis.Client, writer *kafka.Writer) *Container {
	users := NewUserService(db, rdb)
	blogs := NewBlogService(db, rdb)
	return &Container{
		Users:         users,
		Shops:         NewShopService(db, rdb),
		ShopTypes:     NewShopTypeService(db, rdb),
		Blogs:         blogs,
		Follows:       NewFollowService(db, rdb),
		Vouchers:      NewVoucherService(db, rdb),
		VoucherOrders: NewVoucherOrderService(db, rdb, writer),
		Uploads:       NewUploadService(cfg),
	}
}
