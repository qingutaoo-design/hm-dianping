package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"

	"hm-dianping/internal/constants"
	"hm-dianping/internal/model"
	"hm-dianping/internal/script"
)

var (
	ErrSeckillStockNotEnough = errors.New("库存不足！")
	ErrSeckillDuplicateOrder = errors.New("你已经抢过了！")
)

type VoucherOrderService struct {
	db       *gorm.DB
	rdb      *redis.Client
	writer   *kafka.Writer
	idWorker *RedisIDWorker
}

func NewVoucherOrderService(db *gorm.DB, rdb *redis.Client, writer *kafka.Writer) *VoucherOrderService {
	return &VoucherOrderService{db: db, rdb: rdb, writer: writer, idWorker: NewRedisIDWorker(rdb)}
}

func (s *VoucherOrderService) OrderSeckill(ctx context.Context, voucherID, userID uint64) (uint64, error) {
	orderID, err := s.idWorker.NextID(ctx, "order")
	if err != nil {
		return 0, err
	}
	result, err := s.rdb.Eval(ctx, script.SeckillLua, []string{}, voucherID, userID, orderID).Int()
	if err != nil {
		return 0, err
	}
	if result == 1 {
		return 0, ErrSeckillStockNotEnough
	}
	if result == 2 {
		return 0, ErrSeckillDuplicateOrder
	}

	order := model.VoucherOrder{ID: orderID, VoucherID: voucherID, UserID: userID, Status: 1}
	if s.writer == nil {
		if err := s.CreateVoucherOrder(ctx, &order); err != nil {
			s.rollbackRedisSeckill(ctx, voucherID, userID)
			return 0, err
		}
		return orderID, nil
	}
	body, err := json.Marshal(order)
	if err != nil {
		s.rollbackRedisSeckill(ctx, voucherID, userID)
		return 0, err
	}
	if err := s.writer.WriteMessages(ctx, kafka.Message{Key: []byte(strconv.FormatUint(userID, 10)), Value: body, Time: time.Now()}); err != nil {
		s.rollbackRedisSeckill(ctx, voucherID, userID)
		return 0, err
	}
	return orderID, nil
}

func (s *VoucherOrderService) StartConsumer(ctx context.Context, reader *kafka.Reader) {
	go func() {
		for {
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("fetch voucher order message: %v", err)
				continue
			}
			var order model.VoucherOrder
			if err := json.Unmarshal(msg.Value, &order); err != nil {
				log.Printf("decode voucher order message: %v", err)
				_ = reader.CommitMessages(ctx, msg)
				continue
			}
			if err := s.CreateVoucherOrder(ctx, &order); err != nil {
				log.Printf("create voucher order %d: %v", order.ID, err)
				if errors.Is(err, ErrSeckillDuplicateOrder) || errors.Is(err, ErrSeckillStockNotEnough) {
					_ = reader.CommitMessages(ctx, msg)
				}
				continue
			}
			if err := reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("commit voucher order message: %v", err)
			}
		}
	}()
}

func (s *VoucherOrderService) CreateVoucherOrder(ctx context.Context, order *model.VoucherOrder) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.VoucherOrder{}).Where("user_id = ? AND voucher_id = ?", order.UserID, order.VoucherID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrSeckillDuplicateOrder
		}
		res := tx.Model(&model.SeckillVoucher{}).
			Where("voucher_id = ? AND stock > 0", order.VoucherID).
			UpdateColumn("stock", gorm.Expr("stock - 1"))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrSeckillStockNotEnough
		}
		return tx.Create(order).Error
	})
}

func (s *VoucherOrderService) rollbackRedisSeckill(ctx context.Context, voucherID, userID uint64) {
	rollbackCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stockKey := constants.SeckillStock + strconv.FormatUint(voucherID, 10)
	orderKey := "seckill:order:" + strconv.FormatUint(voucherID, 10)
	if err := s.rdb.Incr(rollbackCtx, stockKey).Err(); err != nil {
		log.Printf("rollback seckill stock %d: %v", voucherID, err)
	}
	if err := s.rdb.SRem(rollbackCtx, orderKey, userID).Err(); err != nil {
		log.Printf("rollback seckill order set voucher=%d user=%d: %v", voucherID, userID, err)
	}
}
