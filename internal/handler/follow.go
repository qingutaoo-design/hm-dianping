package handler

import (
	"github.com/gin-gonic/gin"

	"hm-dianping/internal/ctx"
	"hm-dianping/internal/service"
)

func HandleFollowFollow(svc *service.FollowService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := ctx.CurrentUser(c)
		if err != nil { writeFail(c, err); return }
		id, err := pathUint(c, "id")
		if err != nil { writeFail(c, err); return }
		isFollow := c.Param("isFollow") == "true"
		if err := svc.Follow(c.Request.Context(), user.ID, id, isFollow); err != nil { writeFail(c, err); return }
		writeOK(c, nil)
	}
}

func HandleFollowIsFollow(svc *service.FollowService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := ctx.CurrentUser(c)
		if err != nil { writeFail(c, err); return }
		id, err := pathUint(c, "id")
		if err != nil { writeFail(c, err); return }
		result, err := svc.IsFollow(c.Request.Context(), user.ID, id)
		if err != nil { writeFail(c, err); return }
		writeOK(c, result)
	}
}

func HandleFollowCommon(svc *service.FollowService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := ctx.CurrentUser(c)
		if err != nil { writeFail(c, err); return }
		otherID, err := pathUint(c, "id")
		if err != nil { writeFail(c, err); return }
		users, err := svc.Commons(c.Request.Context(), user.ID, otherID)
		if err != nil { writeFail(c, err); return }
		writeOK(c, users)
	}
}