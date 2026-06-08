package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"hm-dianping/internal/dto"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.Printf("panic: %v", recovered)
		c.JSON(http.StatusOK, dto.Fail("服务器异常"))
	})
}
