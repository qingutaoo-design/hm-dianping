package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"hm-dianping/internal/constants"
	"hm-dianping/internal/dto"
	"hm-dianping/internal/model"
)

var phonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

type UserService struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewUserService(db *gorm.DB, rdb *redis.Client) *UserService {
	return &UserService{db: db, rdb: rdb}
}

func (s *UserService) SendCode(ctx context.Context, phone string) (string, error) {
	if !phonePattern.MatchString(phone) {
		return "", errors.New("手机号格式错误")
	}
	code := fmt.Sprintf("%06d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1000000))
	if err := s.rdb.Set(ctx, constants.LoginCodeKey+phone, code, constants.LoginCodeTTL).Err(); err != nil {
		return "", err
	}
	return code, nil
}

func (s *UserService) Login(ctx context.Context, form dto.LoginForm) (string, error) {
	if !phonePattern.MatchString(form.Phone) {
		return "", errors.New("手机号格式错误")
	}
	cacheCode, err := s.rdb.Get(ctx, constants.LoginCodeKey+form.Phone).Result()
	if err == redis.Nil || cacheCode == "" {
		return "", errors.New("验证码不存在或已过期")
	}
	if err != nil {
		return "", err
	}
	if cacheCode != form.Code {
		return "", errors.New("验证码错误")
	}

	var user model.User
	if err := s.db.WithContext(ctx).Where("phone = ?", form.Phone).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
		user = model.User{Phone: form.Phone, NickName: constants.UserNickPrefx + uuid.NewString()[:12]}
		if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
			return "", err
		}
	}

	token := uuid.NewString()
	userDTO := toUserDTO(user)
	key := constants.LoginUserKey + token
	values := map[string]any{
		"id":       strconv.FormatUint(userDTO.ID, 10),
		"nickName": userDTO.NickName,
		"icon":     userDTO.Icon,
	}
	if err := s.rdb.HSet(ctx, key, values).Err(); err != nil {
		return "", err
	}
	if err := s.rdb.Expire(ctx, key, constants.LoginUserTTL).Err(); err != nil {
		return "", err
	}
	_ = s.rdb.Del(ctx, constants.LoginCodeKey+form.Phone).Err()
	return token, nil
}

func (s *UserService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.rdb.Del(ctx, constants.LoginUserKey+token).Err()
}

func (s *UserService) GetDTO(ctx context.Context, id uint64) (dto.UserDTO, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return dto.UserDTO{}, err
	}
	return toUserDTO(user), nil
}

func (s *UserService) GetInfo(ctx context.Context, id uint64) (*model.UserInfo, error) {
	var info model.UserInfo
	if err := s.db.WithContext(ctx).First(&info, "user_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &info, nil
}

func toUserDTO(user model.User) dto.UserDTO {
	return dto.UserDTO{ID: user.ID, NickName: user.NickName, Icon: user.Icon}
}
