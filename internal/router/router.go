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
)

func New(cfg *config.Config, rdb *redis.Client, h *handler.Container) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), middleware.Recovery())
	r.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "authorization"},
		AllowCredentials: true,
		AllowOriginFunc:  func(string) bool { return true },
		MaxAge:           12 * time.Hour,
	}))
	r.Use(middleware.Auth(cfg, rdb))

	user := r.Group("/user")
	user.POST("/code", h.Users.SendCode)
	user.POST("/login", h.Users.Login)
	user.POST("/logout", h.Users.Logout)
	user.GET("/me", h.Users.Me)
	user.GET("/info/:id", h.Users.Info)

	shop := r.Group("/shop")
	shop.POST("", h.Shops.Create)
	shop.PUT("", h.Shops.Update)
	shop.GET("/of/type", h.Shops.OfType)
	shop.GET("/of/name", h.Shops.OfName)

	r.GET("/shop-type/list", h.ShopTypes.List)

	blog := r.Group("/blog")
	blog.POST("", h.Blogs.Save)
	blog.PUT("/like/:id", h.Blogs.Like)
	blog.GET("/of/me", h.Blogs.OfMe)
	blog.GET("/hot", h.Blogs.Hot)
	blog.GET("/likes/:id", h.Blogs.Likes)
	blog.GET("/of/user", h.Blogs.OfUser)
	blog.GET("/of/follow", h.Blogs.OfFollow)

	follow := r.Group("/follow")
	follow.GET("/or/not/:id", h.Follows.IsFollow)
	follow.GET("/common/:id", h.Follows.Common)

	voucher := r.Group("/voucher")
	voucher.POST("", h.Vouchers.Add)
	voucher.POST("/seckill", h.Vouchers.AddSeckill)
	voucher.GET("/list/:shopId", h.Vouchers.List)

	r.POST("/voucher-order/seckill/:id", h.VoucherOrders.Seckill)
	r.POST("/upload/blog", h.Uploads.Blog)
	r.GET("/upload/blog/delete", h.Uploads.DeleteBlog)
	r.NoRoute(func(c *gin.Context) {
		path := strings.Trim(c.Request.URL.Path, "/")
		parts := strings.Split(path, "/")
		switch {
		case c.Request.Method == http.MethodGet && len(parts) == 2 && parts[0] == "shop":
			c.Params = append(c.Params, gin.Param{Key: "id", Value: parts[1]})
			h.Shops.Get(c)
		case c.Request.Method == http.MethodGet && len(parts) == 2 && parts[0] == "blog":
			c.Params = append(c.Params, gin.Param{Key: "id", Value: parts[1]})
			h.Blogs.Get(c)
		case c.Request.Method == http.MethodGet && len(parts) == 2 && parts[0] == "user":
			c.Params = append(c.Params, gin.Param{Key: "id", Value: parts[1]})
			h.Users.Get(c)
		case c.Request.Method == http.MethodPut && len(parts) == 3 && parts[0] == "follow":
			c.Params = append(c.Params, gin.Param{Key: "id", Value: parts[1]}, gin.Param{Key: "isFollow", Value: parts[2]})
			h.Follows.Follow(c)
		default:
			c.JSON(http.StatusNotFound, gin.H{"success": false, "errorMsg": "接口不存在"})
		}
	})

	return r
}
