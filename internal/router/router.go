package router

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"hm-dianping/internal/config"
	"hm-dianping/internal/handler"
	"hm-dianping/internal/middleware"
	"hm-dianping/internal/service"
)

func New(
	cfg *config.Config,
	rdb *redis.Client,
	userSvc *service.UserService,
	shopSvc *service.ShopService,
	shopTypeSvc *service.ShopTypeService,
	blogSvc *service.BlogService,
	followSvc *service.FollowService,
	voucherSvc *service.VoucherService,
	voucherOrderSvc *service.VoucherOrderService,
	uploadSvc *service.UploadService,
) *gin.Engine {
	r := gin.New()

	// === 全局中间件 ===
	r.Use(gin.Logger(), middleware.Recovery())
	r.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "authorization"},
		AllowCredentials: true,
		AllowOriginFunc:  func(string) bool { return true },
		MaxAge:           12 * time.Hour,
	}))
	r.Use(middleware.Auth(cfg, rdb))

	// === 用户模块 ===
	user := r.Group("/user")
	user.POST("/code", handler.HandleUserSendCode(userSvc))
	user.POST("/login", handler.HandleUserLogin(userSvc))
	user.POST("/logout", handler.HandleUserLogout(userSvc))
	user.GET("/me", handler.HandleUserMe())
	user.GET("/info/:id", handler.HandleUserInfo(userSvc))

	// === 商户模块 ===
	shop := r.Group("/shop")
	shop.POST("", handler.HandleShopCreate(shopSvc))
	shop.PUT("", handler.HandleShopUpdate(shopSvc))
	shop.GET("/of/type", handler.HandleShopOfType(shopSvc))
	shop.GET("/of/name", handler.HandleShopOfName(shopSvc))

	r.GET("/shop-type/list", handler.HandleShopTypeList(shopTypeSvc))

	// === 博客模块 ===
	blog := r.Group("/blog")
	blog.POST("", handler.HandleBlogSave(blogSvc))
	blog.PUT("/like/:id", handler.HandleBlogLike(blogSvc))
	blog.GET("/of/me", handler.HandleBlogOfMe(blogSvc))
	blog.GET("/hot", handler.HandleBlogHot(blogSvc))
	blog.GET("/likes/:id", handler.HandleBlogLikes(blogSvc))
	blog.GET("/of/user", handler.HandleBlogOfUser(blogSvc))
	blog.GET("/of/follow", handler.HandleBlogOfFollow(blogSvc))

	// === 关注模块 ===
	follow := r.Group("/follow")
	follow.GET("/or/not/:id", handler.HandleFollowIsFollow(followSvc))
	follow.GET("/common/:id", handler.HandleFollowCommon(followSvc))

	// === 优惠券模块 ===
	voucher := r.Group("/voucher")
	voucher.POST("", handler.HandleVoucherAdd(voucherSvc))
	voucher.POST("/seckill", handler.HandleVoucherAddSeckill(voucherSvc))
	voucher.GET("/list/:shopId", handler.HandleVoucherList(voucherSvc))

	r.POST("/voucher-order/seckill/:id", handler.HandleSeckill(voucherOrderSvc))
	r.POST("/upload/blog", handler.HandleUploadBlog(uploadSvc))
	r.GET("/upload/blog/delete", handler.HandleUploadDeleteBlog(uploadSvc))

	// === NoRoute 兜底——兼容旧版路径格式 ===
	r.NoRoute(func(c *gin.Context) {
		path := strings.Trim(c.Request.URL.Path, "/")
		parts := strings.Split(path, "/")
		switch {
		case c.Request.Method == http.MethodGet && len(parts) == 2 && parts[0] == "shop":
			c.Params = append(c.Params, gin.Param{Key: "id", Value: parts[1]})
			handler.HandleShopGet(shopSvc)(c)
		case c.Request.Method == http.MethodGet && len(parts) == 2 && parts[0] == "blog":
			c.Params = append(c.Params, gin.Param{Key: "id", Value: parts[1]})
			handler.HandleBlogGet(blogSvc)(c)
		case c.Request.Method == http.MethodGet && len(parts) == 2 && parts[0] == "user":
			c.Params = append(c.Params, gin.Param{Key: "id", Value: parts[1]})
			handler.HandleUserGet(userSvc)(c)
		case c.Request.Method == http.MethodPut && len(parts) == 3 && parts[0] == "follow":
			c.Params = append(c.Params, gin.Param{Key: "id", Value: parts[1]}, gin.Param{Key: "isFollow", Value: parts[2]})
			handler.HandleFollowFollow(followSvc)(c)
		default:
			c.JSON(http.StatusNotFound, gin.H{"success": false, "errorMsg": "接口不存在"})
		}
	})

	return r
}