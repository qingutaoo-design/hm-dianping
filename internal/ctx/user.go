package ctx

import (
	"errors"

	"github.com/gin-gonic/gin"

	"hm-dianping/internal/dto"
)

const userKey = "currentUser"

func SaveUser(c *gin.Context, user dto.UserDTO) {
	c.Set(userKey, user)
}

func CurrentUser(c *gin.Context) (dto.UserDTO, error) {
	value, ok := c.Get(userKey)
	if !ok {
		return dto.UserDTO{}, errors.New("用户未登录")
	}
	user, ok := value.(dto.UserDTO)
	if !ok || user.ID == 0 {
		return dto.UserDTO{}, errors.New("用户未登录")
	}
	return user, nil
}
