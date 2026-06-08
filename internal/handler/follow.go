package handler

import (
	"github.com/gin-gonic/gin"

	"hm-dianping/internal/service"
)

type FollowHandler struct {
	service *service.FollowService
}

func NewFollowHandler(service *service.FollowService) *FollowHandler {
	return &FollowHandler{service: service}
}

func (h *FollowHandler) Follow(c *gin.Context) {
	user, err := currentUser(c)
	if err != nil {
		writeFail(c, err)
		return
	}
	followUserID, err := pathUint(c, "id")
	if err != nil {
		writeFail(c, err)
		return
	}
	isFollow := c.Param("isFollow") == "true"
	if err := h.service.Follow(c.Request.Context(), user.ID, followUserID, isFollow); err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, nil)
}

func (h *FollowHandler) IsFollow(c *gin.Context) {
	user, err := currentUser(c)
	if err != nil {
		writeFail(c, err)
		return
	}
	followUserID, err := pathUint(c, "id")
	if err != nil {
		writeFail(c, err)
		return
	}
	result, err := h.service.IsFollow(c.Request.Context(), user.ID, followUserID)
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, result)
}

func (h *FollowHandler) Common(c *gin.Context) {
	user, err := currentUser(c)
	if err != nil {
		writeFail(c, err)
		return
	}
	otherID, err := pathUint(c, "id")
	if err != nil {
		writeFail(c, err)
		return
	}
	users, err := h.service.Commons(c.Request.Context(), user.ID, otherID)
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, users)
}
