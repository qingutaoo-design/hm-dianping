package app

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"hm-dianping/internal/config"
	"hm-dianping/internal/handler"
	"hm-dianping/internal/router"
	"hm-dianping/internal/service"
)

type App struct {
	Router         *gin.Engine
	db             *sql.DB
	rdb            *redis.Client
	writer         *kafka.Writer
	reader         *kafka.Reader
	cancelConsumer context.CancelFunc
}

func New(cfg *config.Config) (*App, error) {
	gin.SetMode(cfg.Server.Mode)

	gormDB, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	db, err := gormDB.DB()
	if err != nil {
		return nil, err
	}
	db.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	db.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	db.SetConnMaxLifetime(time.Hour)

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	var writer *kafka.Writer
	var reader *kafka.Reader
	if cfg.Kafka.Enabled {
		writer = &kafka.Writer{
			Addr:     kafka.TCP(cfg.Kafka.Brokers...),
			Topic:    cfg.Kafka.Topic,
			Balancer: &kafka.LeastBytes{},
		}
		reader = kafka.NewReader(kafka.ReaderConfig{
			Brokers: cfg.Kafka.Brokers,
			Topic:   cfg.Kafka.Topic,
			GroupID: cfg.Kafka.GroupID,
		})
	}

	services := service.NewContainer(cfg, gormDB, rdb, writer)
	var cancelConsumer context.CancelFunc
	if reader != nil {
		consumerCtx, cancel := context.WithCancel(context.Background())
		cancelConsumer = cancel
		services.VoucherOrders.StartConsumer(consumerCtx, reader)
	}

	handlers := handler.NewContainer(cfg, services)
	engine := router.New(cfg, rdb, handlers)

	return &App{Router: engine, db: db, rdb: rdb, writer: writer, reader: reader, cancelConsumer: cancelConsumer}, nil
}

func (a *App) Close() {
	if a.cancelConsumer != nil {
		a.cancelConsumer()
	}
	if a.reader != nil {
		if err := a.reader.Close(); err != nil {
			log.Printf("close kafka reader: %v", err)
		}
	}
	if a.writer != nil {
		if err := a.writer.Close(); err != nil {
			log.Printf("close kafka writer: %v", err)
		}
	}
	if a.rdb != nil {
		if err := a.rdb.Close(); err != nil {
			log.Printf("close redis: %v", err)
		}
	}
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			log.Printf("close db: %v", err)
		}
	}
}
