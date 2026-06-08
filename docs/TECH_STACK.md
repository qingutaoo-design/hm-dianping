# Go 技术栈使用清单

本文记录当前 Go 版 `hm-dianping` 的技术栈、依赖位置和中间件用途。旧 Java/Spring Boot/Maven 项目已经迁移为纯 Go 项目。

## 项目概览

`hm-dianping` 是类似大众点评的本地生活服务后端，包含商户展示、探店博客、秒杀优惠券、用户关注、Feed 推送、图片上传等功能。

当前仓库采用 Go 单体服务结构：

```text
cmd/server       服务启动入口
config           本地配置文件
internal         私有业务代码
migrations       golang-migrate 数据库迁移
docs             教学和迁移文档
```

## 后端技术栈

| 技术 | 用途 | 位置 |
|---|---|---|
| Go 1.25.10 | 运行语言 | `go.mod` |
| Gin | HTTP 路由、中间件、请求响应 | `internal/router`、`internal/handler` |
| GORM | MySQL ORM | `internal/model`、`internal/service` |
| gorm MySQL driver | MySQL 驱动 | `go.mod` |
| go-redis | Redis 客户端 | `internal/service`、`internal/middleware` |
| kafka-go | Kafka Producer/Consumer | `internal/app`、`internal/service/voucher_order.go` |
| Viper | YAML 配置和环境变量覆盖 | `internal/config` |
| golang-migrate | 数据库迁移 | `migrations`、`internal/migration` |
| google/uuid | token、图片文件名、锁 owner | `internal/service` |
| gin-contrib/cors | CORS 中间件 | `internal/router` |
| go:embed | 嵌入 Lua 脚本 | `internal/script` |
| 标准库 log | 应用日志 | `cmd/server`、`internal/app`、`internal/service` |

## 配置

默认配置位于 `config/config.yaml`：

| 配置段 | 说明 |
|---|---|
| `server` | Gin 服务端口、运行模式、读写超时 |
| `mysql` | GORM DSN、连接池 |
| `redis` | Redis 地址、密码、DB |
| `kafka` | brokers、topic、consumer group、是否启用 |
| `upload` | 本地图片目录、公开 URL 前缀 |
| `auth` | 是否兼容旧 Java 缺 token 放行行为 |

Viper 支持 `HMDP_` 环境变量覆盖配置，例如：

```powershell
$env:HMDP_MYSQL_DSN="root:123456@tcp(127.0.0.1:3306)/hmdp?charset=utf8mb4&parseTime=True&loc=Local"
$env:HMDP_REDIS_ADDR="127.0.0.1:6379"
```

## Redis 使用场景

| 场景 | Key 前缀 | 数据结构 | 实现位置 |
|---|---|---|---|
| 登录验证码 | `login:code:` | String | `internal/service/user.go` |
| 登录 token | `login:token:` | Hash | `internal/service/user.go`、`internal/middleware/auth.go` |
| 商铺缓存 | `cache:shop:` | String(JSON) | `internal/service/shop.go` |
| 商铺空值缓存 | `cache:shop:` | String(空串) | `internal/service/shop.go` |
| 店铺类型缓存 | `cache:shop:type:list` | String(JSON) | `internal/service/shop_type.go` |
| 商铺互斥锁 | `lock:shop:` | String(SET NX EX) | `internal/service/shop.go`、`internal/script/unlock.lua` |
| 秒杀库存 | `seckill:stock:` | String | `internal/service/voucher.go`、`internal/script/seckill.lua` |
| 秒杀用户集合 | `seckill:order:` | Set | `internal/script/seckill.lua` |
| 博客点赞 | `blog:liked:` | ZSet | `internal/service/blog.go` |
| 关注列表 | `follows:` | Set | `internal/service/follow.go` |
| Feed 收件箱 | `feed:` | ZSet | `internal/service/blog.go` |
| Redis ID 计数 | `icr:{prefix}:{yyyy:MM:dd}` | String counter | `internal/service/redis_id.go` |

## Kafka

| 配置 | 当前值 |
|---|---|
| Topic | `kafka-orders` |
| Consumer Group | `my-kafka-group` |
| Brokers | `localhost:9094`、`localhost:9095`、`localhost:9096` |

秒杀接口先通过 Redis Lua 原子扣减库存并记录用户，再发送 Kafka 消息异步创建数据库订单。Kafka 未启用时，服务会走同步落库路径。

## 数据库

MySQL 表结构由 `migrations/000001_init_schema.up.sql` 创建：

| 表 | 说明 |
|---|---|
| `tb_user` | 用户 |
| `tb_user_info` | 用户详细信息 |
| `tb_shop` | 商户 |
| `tb_shop_type` | 商户分类 |
| `tb_blog` | 探店博客 |
| `tb_blog_comments` | 博客评论 |
| `tb_follow` | 用户关注 |
| `tb_voucher` | 优惠券 |
| `tb_seckill_voucher` | 秒杀优惠券库存和时间 |
| `tb_voucher_order` | 优惠券订单 |
| `tb_sign` | 签到表，当前保留 schema，业务接口尚未实现 |

## HTTP 与响应

Go 版保留旧前端使用的接口路径和统一响应格式：

```json
{
  "success": true,
  "data": {}
}
```

失败响应：

```json
{
  "success": false,
  "errorMsg": "错误信息"
}
```

主要路由位于 `internal/router/router.go`，业务 Handler 位于 `internal/handler`。

## Lua 脚本

| 脚本 | 用途 |
|---|---|
| `internal/script/seckill.lua` | 秒杀库存检查、一人一单检查、扣 Redis 库存、记录用户集合 |
| `internal/script/unlock.lua` | Redis 锁安全释放，校验 owner 后删除 |

Go 版秒杀 Lua 不再写 Redis Stream，因为当前订单异步链路统一使用 Kafka。

## 文档索引

| 文档 | 内容 |
|---|---|
| `docs/GO_ARCHITECTURE.md` | Go 项目架构模式教学 |
| `docs/CORE_BUSINESS_CHANGES.md` | 核心业务迁移和修正点 |
| `docs/MIGRATIONS.md` | golang-migrate 使用教程 |
