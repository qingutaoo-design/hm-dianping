package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hm-dianping/internal/response"
)

func writeOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, response.OK(data))
}

func writeFail(c *gin.Context, err error) {
	if err == nil {
		c.JSON(http.StatusOK, response.Fail("服务器异常"))
		return
	}
	c.JSON(http.StatusOK, response.Fail(err.Error()))
}