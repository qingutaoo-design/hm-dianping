package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"hm-dianping/internal/model"
	"hm-dianping/internal/service"
)

type BlogHandler struct {
	service *service.BlogService
}

func NewBlogHandler(service *service.BlogService) *BlogHandler {
	return &BlogHandler{service: service}
}

func (h *BlogHandler) Save(c *gin.Context) {
	user, err := currentUser(c)
	if err != nil {
		writeFail(c, err)
		return
	}
	var blog model.Blog
	if err := c.ShouldBindJSON(&blog); err != nil {
		writeFail(c, err)
		return
	}
	if err := h.service.Save(c.Request.Context(), &blog, user.ID); err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, blog.ID)
}

func (h *BlogHandler) Like(c *gin.Context) {
	user, err := currentUser(c)
	if err != nil {
		writeFail(c, err)
		return
	}
	id, err := pathUint(c, "id")
	if err != nil {
		writeFail(c, err)
		return
	}
	if err := h.service.Like(c.Request.Context(), id, user.ID); err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, nil)
}

func (h *BlogHandler) OfMe(c *gin.Context) {
	user, err := currentUser(c)
	if err != nil {
		writeFail(c, err)
		return
	}
	blogs, err := h.service.QueryByUser(c.Request.Context(), user.ID, queryInt(c, "current", 1), user.ID)
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, blogs)
}

func (h *BlogHandler) Hot(c *gin.Context) {
	blogs, err := h.service.QueryHot(c.Request.Context(), queryInt(c, "current", 1), viewerID(c))
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, blogs)
}

func (h *BlogHandler) Get(c *gin.Context) {
	id, err := pathUint(c, "id")
	if err != nil {
		writeFail(c, err)
		return
	}
	blog, err := h.service.QueryByID(c.Request.Context(), id, viewerID(c))
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, blog)
}

func (h *BlogHandler) Likes(c *gin.Context) {
	id, err := pathUint(c, "id")
	if err != nil {
		writeFail(c, err)
		return
	}
	users, err := h.service.QueryLikes(c.Request.Context(), id)
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, users)
}

func (h *BlogHandler) OfUser(c *gin.Context) {
	id, err := queryUint(c, "id")
	if err != nil {
		writeFail(c, err)
		return
	}
	blogs, err := h.service.QueryByUser(c.Request.Context(), id, queryInt(c, "current", 1), viewerID(c))
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, blogs)
}

func (h *BlogHandler) OfFollow(c *gin.Context) {
	user, err := currentUser(c)
	if err != nil {
		writeFail(c, err)
		return
	}
	lastID := int64(time.Now().UnixMilli())
	if v := c.Query("lastId"); v != "" {
		_, _ = fmtSscanf(v, &lastID)
	}
	offset := int64(0)
	if v := c.Query("offset"); v != "" {
		_, _ = fmtSscanf(v, &offset)
	}
	result, err := h.service.QueryFeed(c.Request.Context(), user.ID, lastID, offset)
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, result)
}
