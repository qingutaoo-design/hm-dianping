# 核心业务变动教程

本文说明 Java 版核心业务迁移到 Go 版后的实现位置、调用链和主要修正点。

## 登录业务

Java 版：`UserController -> UserServiceImpl -> StringRedisTemplate`

Go 版：`internal/handler/user.go -> internal/service/user.go -> go-redis + GORM`

流程：

```text
POST /user/code
  -> 校验手机号
  -> 生成 6 位验证码
  -> 写入 Redis login:code:{phone}，TTL 2 分钟

POST /user/login
  -> 校验手机号
  -> 读取 Redis 验证码并与用户输入比较
  -> 查询或创建 tb_user
  -> 生成 UUID token
  -> 写入 Redis Hash login:token:{token}
  -> 返回 token
```

重要修正：Java 版登录逻辑只判断 Redis 验证码是否为空，没有比较用户提交的验证码；Go 版已经修正为必须相等。

## 认证中间件

Java 版：`LoginInterceptor + UserHolder(ThreadLocal)`

Go 版：`internal/middleware/auth.go + internal/ctx/user.go`

请求头仍然使用旧项目前端的 `authorization`。中间件从 `login:token:{token}` 读取用户信息，写入 Gin Context，业务 Handler 通过 `currentUser(c)` 读取。

默认行为改为未登录返回失败。若需要兼容 Java 版“缺 token 放行”的旧行为，可在 `config/config.yaml` 中设置：

```yaml
auth:
  compatible_missing_token: true
```

## 商铺缓存

Java 版：`ShopServiceImpl + CacheClient`

Go 版：`internal/service/shop.go`

流程：

```text
GET /shop/{id}
  -> 先读 Redis cache:shop:{id}
  -> 命中空字符串表示缓存穿透空值
  -> 未命中则尝试 SET NX lock:shop:{id}
  -> 持锁查询 MySQL tb_shop
  -> 写入 Redis，TTL 30 分钟
```

店铺更新：

```text
PUT /shop
  -> GORM 更新 tb_shop
  -> 删除 cache:shop:{id}
```

## 店铺类型缓存

Java 版：`ShopTypeServiceImpl`

Go 版：`internal/service/shop_type.go`

缓存 Key 保持：`cache:shop:type:list`。Go 版增加了 24 小时 TTL，避免永久脏缓存。

## 博客点赞

Java 版：`BlogServiceImpl.likeBlog`

Go 版：`internal/service/blog.go`

Redis Key 保持：`blog:liked:{blogId}`，数据结构仍然是 ZSet，member 为用户 id，score 为毫秒时间戳。

点赞：数据库 `liked + 1`，Redis `ZADD`。

取消点赞：数据库 `liked - 1`，Redis `ZREM`。

## 关注和共同关注

Java 版：`FollowServiceImpl`

Go 版：`internal/service/follow.go`

关注关系同时写 MySQL 和 Redis Set：

```text
follows:{userId}
```

共同关注使用 Redis `SINTER`，再按用户 id 查询 `tb_user`。

## Feed 推送

Java 版：`BlogServiceImpl.saveBlog/queryBlogOfFollow`

Go 版：`internal/service/blog.go`

发布博客时查询粉丝列表，把博客 id 推送到粉丝收件箱：

```text
feed:{fansId}
```

数据结构仍然是 ZSet，score 为毫秒时间戳。滚动分页接口仍然是：

```text
GET /blog/of/follow?lastId=&offset=
```

## 优惠券

Java 版：`VoucherServiceImpl + VoucherMapper.xml`

Go 版：`internal/service/voucher.go`

店铺券查询继续使用 `tb_voucher LEFT JOIN tb_seckill_voucher`。

新增秒杀券时会写两张表：

```text
tb_voucher
tb_seckill_voucher
```

并初始化 Redis 库存：

```text
seckill:stock:{voucherId}
```

## 秒杀下单

Java 版：`VoucherOrderServiceImpl + seckill.lua + Kafka`

Go 版：`internal/service/voucher_order.go + internal/script/seckill.lua + kafka-go`

请求流程：

```text
POST /voucher-order/seckill/{id}
  -> RedisIdWorker 生成订单 id
  -> 执行 seckill.lua
  -> Redis 原子判断库存和一人一单
  -> 发送 Kafka 消息
  -> 立即返回订单 id
```

消费流程：

```text
kafka-go Reader
  -> 反序列化 VoucherOrder
  -> GORM Transaction
  -> 检查 DB 一人一单
  -> stock > 0 乐观扣减库存
  -> 创建 tb_voucher_order
  -> Commit Kafka message
```

重要修正：Java 版 `createVoucherOrder()` 发现重复下单或库存不足时只打日志但仍可能保存订单。Go 版已经改为直接返回错误并回滚事务。

可靠性补充：Go 版消费者把“一人一单”和“库存不足”识别为永久业务错误，避免 Kafka 重复投递时无限重试同一条消息。接口侧如果 Kafka 发送失败或同步落库失败，会回补 Redis 秒杀库存并移除 `seckill:order:{voucherId}` 中的用户，降低 Redis 和 MySQL 不一致风险。

## 上传

Java 版：`UploadController` 写本地 Windows 图片目录。

Go 版：`internal/service/upload.go`

图片目录已配置化：

```yaml
upload:
  image_dir: D:\HeiMaDianPing\nginx-1.18.0\html\hmdp\imgs\
  public_prefix: /imgs
```

后续接入 OSS 时，只需要替换 `UploadService.SaveBlogImage` 的存储实现，Handler 和 API 不需要变化。
