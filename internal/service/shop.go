package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"hm-dianping/internal/constants"
	"hm-dianping/internal/model"
	"hm-dianping/internal/script"
)

type ShopService struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewShopService(db *gorm.DB, rdb *redis.Client) *ShopService {
	return &ShopService{db: db, rdb: rdb}
}

func (s *ShopService) GetByID(ctx context.Context, id uint64) (*model.Shop, error) {
	key := constants.CacheShopKey + strconv.FormatUint(id, 10)

	cached, err := s.rdb.Get(ctx, key).Result()
	if err == nil {
		if cached == "" {
			return nil, nil
		}
		var shop model.Shop
		if err := json.Unmarshal([]byte(cached), &shop); err == nil {
			return &shop, nil
		}
	}
	if err != nil && err != redis.Nil {
		return nil, err
	}

	lockKey := constants.LockShopKey + strconv.FormatUint(id, 10)
	owner := uuid.NewString()

	const maxRetries = 5
	for i := 0; i < maxRetries; i++ {
		locked, err := s.rdb.SetNX(ctx, lockKey, owner, constants.LockShopTTL).Result()
		if err != nil { return nil, err }
		if locked { break }
		backoff := time.Duration(50+(i*25)) * time.Millisecond
		time.Sleep(backoff)
	}

	defer s.rdb.Eval(context.Background(), script.UnlockLua, []string{lockKey}, owner)

	var shop model.Shop
	if err := s.db.WithContext(ctx).First(&shop, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = s.rdb.Set(ctx, key, "", constants.CacheNullTTL).Err()
			return nil, nil
		}
		return nil, err
	}
	bytes, _ := json.Marshal(shop)
	_ = s.rdb.Set(ctx, key, bytes, constants.CacheShopTTL).Err()
	return &shop, nil
}

func (s *ShopService) Create(ctx context.Context, shop *model.Shop) error {
	return s.db.WithContext(ctx).Create(shop).Error
}

func (s *ShopService) Update(ctx context.Context, shop *model.Shop) error {
	if shop.ID == 0 {
		return errors.New("店铺id不能为空")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Shop{}).Where("id = ?", shop.ID).Updates(shop).Error; err != nil {
			return err
		}
		return s.rdb.Del(ctx, constants.CacheShopKey+strconv.FormatUint(shop.ID, 10)).Err()
	})
}

func (s *ShopService) PageByType(ctx context.Context, typeID uint64, current int) ([]model.Shop, error) {
	if current < 1 { current = 1 }
	var shops []model.Shop
	err := s.db.WithContext(ctx).Where("type_id = ?", typeID).
		Offset((current - 1) * constants.DefaultPageSize).
		Limit(constants.DefaultPageSize).Find(&shops).Error
	return shops, err
}

func (s *ShopService) PageByName(ctx context.Context, name string, current int) ([]model.Shop, error) {
	if current < 1 { current = 1 }
	var shops []model.Shop
	q := s.db.WithContext(ctx)
	if name != "" {
		q = q.Where("name LIKE ?", fmt.Sprintf("%%%s%%", name))
	}
	err := q.Offset((current - 1) * constants.MaxPageSize).Limit(constants.MaxPageSize).Find(&shops).Error
	return shops, err
}