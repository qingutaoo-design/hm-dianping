package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"hm-dianping/internal/response"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.Printf("panic: %v", recovered)
		c.JSON(http.StatusOK, response.Fail("服务器异常"))
	})
}