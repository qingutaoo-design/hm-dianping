package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"hm-dianping/internal/constants"
	"hm-dianping/internal/dto"
	"hm-dianping/internal/model"
)

type BlogService struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewBlogService(db *gorm.DB, rdb *redis.Client) *BlogService {
	return &BlogService{db: db, rdb: rdb}
}

func (s *BlogService) Save(ctx context.Context, blog *model.Blog, userID uint64) error {
	blog.UserID = userID
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(blog).Error; err != nil {
			return err
		}
		var follows []model.Follow
		if err := tx.Where("follow_user_id = ?", userID).Find(&follows).Error; err != nil {
			return err
		}
		score := float64(time.Now().UnixMilli())
		for _, follow := range follows {
			key := constants.FeedKey + strconv.FormatUint(follow.UserID, 10)
			_ = s.rdb.ZAdd(ctx, key, redis.Z{Score: score, Member: blog.ID}).Err()
		}
		return nil
	})
}

func (s *BlogService) QueryHot(ctx context.Context, current int, viewerID uint64) ([]model.Blog, error) {
	if current < 1 {
		current = 1
	}
	var blogs []model.Blog
	if err := s.db.WithContext(ctx).Order("liked DESC").Offset((current - 1) * constants.MaxPageSize).Limit(constants.MaxPageSize).Find(&blogs).Error; err != nil {
		return nil, err
	}
	return s.enrich(ctx, blogs, viewerID)
}

func (s *BlogService) QueryByID(ctx context.Context, id uint64, viewerID uint64) (*model.Blog, error) {
	var blog model.Blog
	if err := s.db.WithContext(ctx).First(&blog, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	blogs, err := s.enrich(ctx, []model.Blog{blog}, viewerID)
	if err != nil {
		return nil, err
	}
	return &blogs[0], nil
}

func (s *BlogService) Like(ctx context.Context, blogID uint64, userID uint64) error {
	key := constants.BlogLikedKey + strconv.FormatUint(blogID, 10)
	member := strconv.FormatUint(userID, 10)
	score, err := s.rdb.ZScore(ctx, key, member).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	_ = score
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err == redis.Nil {
			res := tx.Model(&model.Blog{}).Where("id = ?", blogID).UpdateColumn("liked", gorm.Expr("liked + 1"))
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return errors.New("博客不存在")
			}
			return s.rdb.ZAdd(ctx, key, redis.Z{Score: float64(time.Now().UnixMilli()), Member: member}).Err()
		}
		res := tx.Model(&model.Blog{}).Where("id = ?", blogID).UpdateColumn("liked", gorm.Expr("liked - 1"))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("博客不存在")
		}
		return s.rdb.ZRem(ctx, key, member).Err()
	})
}

func (s *BlogService) QueryLikes(ctx context.Context, blogID uint64) ([]dto.UserDTO, error) {
	key := constants.BlogLikedKey + strconv.FormatUint(blogID, 10)
	members, err := s.rdb.ZRange(ctx, key, 0, 4).Result()
	if err != nil || len(members) == 0 {
		return []dto.UserDTO{}, err
	}
	ids := make([]uint64, 0, len(members))
	for _, member := range members {
		id, err := strconv.ParseUint(member, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	users, err := s.usersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	ordered := make([]dto.UserDTO, 0, len(ids))
	for _, id := range ids {
		if user, ok := users[id]; ok {
			ordered = append(ordered, user)
		}
	}
	return ordered, nil
}

func (s *BlogService) QueryByUser(ctx context.Context, userID uint64, current int, viewerID uint64) ([]model.Blog, error) {
	if current < 1 {
		current = 1
	}
	var blogs []model.Blog
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("create_time DESC").Offset((current - 1) * constants.MaxPageSize).Limit(constants.MaxPageSize).Find(&blogs).Error; err != nil {
		return nil, err
	}
	return s.enrich(ctx, blogs, viewerID)
}

func (s *BlogService) QueryFeed(ctx context.Context, userID uint64, max int64, offset int64) (dto.ScrollResult, error) {
	key := constants.FeedKey + strconv.FormatUint(userID, 10)
	items, err := s.rdb.ZRevRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{Min: "0", Max: strconv.FormatInt(max, 10), Offset: offset, Count: 2}).Result()
	if err != nil || len(items) == 0 {
		return dto.ScrollResult{List: []model.Blog{}, MinTime: 0, Offset: 0}, err
	}
	ids := make([]uint64, 0, len(items))
	minTime := int64(items[0].Score)
	newOffset := int64(1)
	for i, item := range items {
		id, err := strconv.ParseUint(fmt.Sprint(item.Member), 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
		score := int64(item.Score)
		if i == 0 || score < minTime {
			minTime = score
			newOffset = 1
		} else if score == minTime {
			newOffset++
		}
	}
	blogs, err := s.blogsByIDs(ctx, ids, userID)
	if err != nil {
		return dto.ScrollResult{}, err
	}
	return dto.ScrollResult{List: blogs, MinTime: minTime, Offset: newOffset}, nil
}

func (s *BlogService) enrich(ctx context.Context, blogs []model.Blog, viewerID uint64) ([]model.Blog, error) {
	for i := range blogs {
		var user model.User
		if err := s.db.WithContext(ctx).First(&user, blogs[i].UserID).Error; err == nil {
			blogs[i].Name = user.NickName
			blogs[i].Icon = user.Icon
		}
		if viewerID != 0 {
			key := constants.BlogLikedKey + strconv.FormatUint(blogs[i].ID, 10)
			_, err := s.rdb.ZScore(ctx, key, strconv.FormatUint(viewerID, 10)).Result()
			blogs[i].IsLike = err == nil
		}
	}
	return blogs, nil
}

func (s *BlogService) usersByIDs(ctx context.Context, ids []uint64) (map[uint64]dto.UserDTO, error) {
	var users []model.User
	if len(ids) == 0 {
		return map[uint64]dto.UserDTO{}, nil
	}
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	result := make(map[uint64]dto.UserDTO, len(users))
	for _, user := range users {
		result[user.ID] = toUserDTO(user)
	}
	return result, nil
}

func (s *BlogService) blogsByIDs(ctx context.Context, ids []uint64, viewerID uint64) ([]model.Blog, error) {
	if len(ids) == 0 {
		return []model.Blog{}, nil
	}
	var blogs []model.Blog
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&blogs).Error; err != nil {
		return nil, err
	}
	byID := make(map[uint64]model.Blog, len(blogs))
	for _, blog := range blogs {
		byID[blog.ID] = blog
	}
	ordered := make([]model.Blog, 0, len(ids))
	for _, id := range ids {
		if blog, ok := byID[id]; ok {
			ordered = append(ordered, blog)
		}
	}
	return s.enrich(ctx, ordered, viewerID)
}
