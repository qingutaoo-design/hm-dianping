package handler

import (
	"github.com/gin-gonic/gin"

	"hm-dianping/internal/service"
)

func HandleUploadBlog(svc *service.UploadService) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil { writeFail(c, err); return }
		path, err := svc.SaveBlogImage(file)
		if err != nil { writeFail(c, err); return }
		writeOK(c, path)
	}
}

func HandleUploadDeleteBlog(svc *service.UploadService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.DeleteBlogImage(c.Query("name")); err != nil { writeFail(c, err); return }
		writeOK(c, nil)
	}
}