package handler

import (
	"github.com/gin-gonic/gin"

	"hm-dianping/internal/service"
)

type UploadHandler struct {
	service *service.UploadService
}

func NewUploadHandler(service *service.UploadService) *UploadHandler {
	return &UploadHandler{service: service}
}

func (h *UploadHandler) Blog(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		writeFail(c, err)
		return
	}
	path, err := h.service.SaveBlogImage(file)
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, path)
}

func (h *UploadHandler) DeleteBlog(c *gin.Context) {
	if err := h.service.DeleteBlogImage(c.Query("name")); err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, nil)
}
