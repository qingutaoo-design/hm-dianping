package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"hm-dianping/internal/config"
	"hm-dianping/internal/constants"
	userctx "hm-dianping/internal/ctx"
	"hm-dianping/internal/dto"
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
			c.JSON(http.StatusOK, dto.Fail("请先登录"))
			c.Abort()
			return
		}

		if !loadUserByToken(c, rdb, token) {
			c.JSON(http.StatusUnauthorized, dto.Fail("登录状态已失效"))
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
	var user dto.UserDTO
	if id, ok := values["id"]; ok {
		_, _ = fmtSscanf(id, &user.ID)
	}
	user.NickName = values["nickName"]
	user.Icon = values["icon"]
	if user.ID == 0 {
		return false
	}
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
	case strings.HasPrefix(path, "/shop"):
		return method == http.MethodGet
	case strings.HasPrefix(path, "/shop-type"):
		return method == http.MethodGet
	case strings.HasPrefix(path, "/voucher"):
		return method == http.MethodGet
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
