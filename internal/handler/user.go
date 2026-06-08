package handler

import (
	"github.com/gin-gonic/gin"

	"hm-dianping/internal/dto"
	"hm-dianping/internal/service"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) SendCode(c *gin.Context) {
	phone := c.PostForm("phone")
	if phone == "" {
		phone = c.Query("phone")
	}
	code, err := h.service.SendCode(c.Request.Context(), phone)
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, code)
}

func (h *UserHandler) Login(c *gin.Context) {
	var form dto.LoginForm
	if err := c.ShouldBind(&form); err != nil {
		writeFail(c, err)
		return
	}
	token, err := h.service.Login(c.Request.Context(), form)
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, token)
}

func (h *UserHandler) Logout(c *gin.Context) {
	if err := h.service.Logout(c.Request.Context(), c.GetHeader("authorization")); err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, nil)
}

func (h *UserHandler) Me(c *gin.Context) {
	user, err := currentUser(c)
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, user)
}

func (h *UserHandler) Info(c *gin.Context) {
	id, err := pathUint(c, "id")
	if err != nil {
		writeFail(c, err)
		return
	}
	info, err := h.service.GetInfo(c.Request.Context(), id)
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, info)
}

func (h *UserHandler) Get(c *gin.Context) {
	id, err := pathUint(c, "id")
	if err != nil {
		writeFail(c, err)
		return
	}
	user, err := h.service.GetDTO(c.Request.Context(), id)
	if err != nil {
		writeFail(c, err)
		return
	}
	writeOK(c, user)
}
