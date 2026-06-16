package handler

import (
	"github.com/gin-gonic/gin"

	"hm-dianping/internal/ctx"
	"hm-dianping/internal/service"
)

type sendCodeRequest struct {
	Phone string `json:"phone" form:"phone"`
}

type loginRequest struct {
	Phone string `json:"phone" form:"phone"`
	Code  string `json:"code" form:"code"`
}

func HandleUserSendCode(svc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req sendCodeRequest
		// 兼容 JSON body 和 form-data 两种请求
		if err := c.ShouldBind(&req); err != nil {
			// 如果 ShouldBind 失败（可能没 Content-Type），回退到 PostForm/Query
			req.Phone = c.PostForm("phone")
			if req.Phone == "" {
				req.Phone = c.Query("phone")
			}
		}
		code, err := svc.SendCode(c.Request.Context(), req.Phone)
		if err != nil { writeFail(c, err); return }
		writeOK(c, code)
	}
}

func HandleUserLogin(svc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBind(&req); err != nil {
			// 兼容：从 PostForm 读取
			req.Phone = c.PostForm("phone")
			req.Code = c.PostForm("code")
		}
		token, err := svc.Login(c.Request.Context(), req.Phone, req.Code)
		if err != nil { writeFail(c, err); return }
		writeOK(c, token)
	}
}

func HandleUserLogout(svc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.Logout(c.Request.Context(), c.GetHeader("authorization")); err != nil {
			writeFail(c, err); return
		}
		writeOK(c, nil)
	}
}

func HandleUserMe() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := ctx.CurrentUser(c)
		if err != nil { writeFail(c, err); return }
		writeOK(c, user)
	}
}

func HandleUserInfo(svc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := pathUint(c, "id")
		if err != nil { writeFail(c, err); return }
		info, err := svc.GetInfo(c.Request.Context(), id)
		if err != nil { writeFail(c, err); return }
		writeOK(c, info)
	}
}

func HandleUserGet(svc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := pathUint(c, "id")
		if err != nil { writeFail(c, err); return }
		user, err := svc.GetUserView(c.Request.Context(), id)
		if err != nil { writeFail(c, err); return }
		writeOK(c, user)
	}
}