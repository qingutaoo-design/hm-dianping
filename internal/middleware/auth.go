package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"hm-dianping/internal/config"
	"hm-dianping/internal/constants"
	userctx "hm-dianping/internal/ctx"
	"hm-dianping/internal/model"
	"hm-dianping/internal/response"
)

func Auth(cfg *config.Config, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		if isPublic(c.Request.Method, path) {
			loadOptionalUser(c, rdb)
			c.Next()
			return
		}

		token := strings.TrimSpace(c.GetHeader("authorization"))
		if token == "" {
			if cfg.Auth.CompatibleMissingToken {
				c.Next()
				return
			}
			c.JSON(http.StatusOK, response.Fail("请先登录"))
			c.Abort()
			return
		}

		if !loadUserByToken(c, rdb, token) {
			c.JSON(http.StatusUnauthorized, response.Fail("登录状态已失效"))
			c.Abort()
			return
		}
		c.Next()
	}
}

func loadOptionalUser(c *gin.Context, rdb *redis.Client) {
	token := strings.TrimSpace(c.GetHeader("authorization"))
	if token != "" {
		loadUserByToken(c, rdb, token)
	}
}

func loadUserByToken(c *gin.Context, rdb *redis.Client, token string) bool {
	values, err := rdb.HGetAll(c.Request.Context(), constants.LoginUserKey+token).Result()
	if err != nil || len(values) == 0 {
		return false
	}
	var user model.UserView
	if id, ok := values["id"]; ok {
		parsed, err := parseUint(id)
		if err != nil { return false }
		user.ID = parsed
	}
	user.NickName = values["nickName"]
	user.Icon = values["icon"]
	if user.ID == 0 { return false }

	userctx.SaveUser(c, user)
	_ = rdb.Expire(c.Request.Context(), constants.LoginUserKey+token, constants.LoginUserTTL).Err()
	return true
}

func isPublic(method, path string) bool {
	switch {
	case method == http.MethodPost && path == "/user/code":
		return true
	case method == http.MethodPost && path == "/user/login":
		return true
	case strings.HasPrefix(path, "/shop") && method == http.MethodGet:
		return true
	case strings.HasPrefix(path, "/shop-type") && method == http.MethodGet:
		return true
	case strings.HasPrefix(path, "/voucher") && method == http.MethodGet:
		return true
	case strings.HasPrefix(path, "/upload"):
		return true
	case path == "/blog/hot" || path == "/blog/:id" || path == "/blog/likes/:id" || path == "/blog/of/user":
		return method == http.MethodGet
	case strings.HasPrefix(path, "/blog/") && method == http.MethodGet:
		parts := strings.Split(strings.Trim(path, "/"), "/")
		return len(parts) == 2
	default:
		return false
	}
}